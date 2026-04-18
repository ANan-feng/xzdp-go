// service/seckill_service.go
package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
	"xzdp-go/dao"
	"xzdp-go/model"
	"xzdp-go/utils"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type SeckillService struct {
	skDAO    *dao.SeckillDAO
	redisDAO *dao.RedisDAO
}

func NewSeckillService(db *gorm.DB) *SeckillService {
	return &SeckillService{
		skDAO:    dao.NewSeckillDAO(utils.DB), // 传递DB参数
		redisDAO: dao.NewRedisDAO(),
	}
}

func (s *SeckillService) CreateSeckillOrder(ctx context.Context, userID, voucherID int64) (int64, error) {
	// 1. 查询秒杀优惠券（无需分布式锁）
	skVoucher, voucher, err := s.skDAO.GetSeckillVoucherByID(ctx, voucherID)
	if err != nil {
		return 0, fmt.Errorf("查询优惠券失败：%v", err)
	}

	// 2. 校验秒杀时间
	now := time.Now()
	if now.Before(skVoucher.BeginTime) || now.After(skVoucher.EndTime) {
		return 0, errors.New("秒杀未开始或已结束")
	}
	// 校验优惠券状态
	if voucher.Status != 1 {
		return 0, errors.New("优惠券已下架")
	}

	// 3. 数据库二次校验是否已下单（兜底，可选保留）
	exist, err := s.skDAO.CheckUserOrderExist(ctx, userID, voucherID)
	if err != nil {
		return 0, fmt.Errorf("校验用户订单失败：%v", err)
	}
	if exist {
		return 0, errors.New("您已参与过该秒杀（一人一单）")
	}

	// 4. Redis Lua脚本预检（原子扣库存+标记用户）
	result, err := utils.SeckillPreCheckAndDeduct(ctx, voucherID, userID, skVoucher.EndTime)
	if err != nil {
		return 0, fmt.Errorf("Lua预检失败：%v", err)
	}
	switch result {
	case 1:
		return 0, errors.New("优惠券已过期")
	case 2:
		return 0, errors.New("库存不足")
	case 3:
		return 0, errors.New("您已参与过该秒杀（一人一单）")
	case -1:
		return 0, errors.New("系统错误")
	}

	// 5. 生成订单ID
	orderID, err := utils.IDGenerator.Generate(ctx, "order")
	if err != nil {
		// 回滚Redis库存+用户标记（Lua已扣减，需补偿）
		s.rollbackRedisSeckillStatus(ctx, voucherID, userID)
		return 0, fmt.Errorf("生成订单ID失败：%v", err)
	}

	// 6. 发送异步消息
	msg := map[string]interface{}{
		"userId":    strconv.FormatInt(userID, 10),
		"couponId":  strconv.FormatInt(voucherID, 10),
		"orderId":   strconv.FormatInt(orderID, 10),
		"shopId":    strconv.FormatInt(voucher.ShopID, 10),
		"timestamp": strconv.FormatInt(now.Unix(), 10),
	}
	if err := s.redisDAO.SendSeckillMsgToStream(ctx, msg); err != nil {
		s.rollbackRedisSeckillStatus(ctx, voucherID, userID)
		return 0, fmt.Errorf("发送异步消息失败：%v", err)
	}

	return orderID, nil
}

// ========== 消费Stream消息（数据库持久化，乐观锁防超卖） ==========
func (s *SeckillService) ConsumeSeckillMsg(ctx context.Context, msg redis.XMessage) error {
	// 解析消息：先手动解析string类型，再转换为int64
	data := struct {
		UserID   int64 `json:"userId"`
		CouponID int64 `json:"couponId"`
		OrderID  int64 `json:"orderId"`
		ShopID   int64 `json:"shopId"`
	}{}

	// 方式1：手动解析每个字段（推荐，避免JSON序列化/反序列化损耗）
	var err error
	// 解析userId
	userIDStr, ok := msg.Values["userId"].(string)
	if !ok {
		return errors.New("userId字段类型错误")
	}
	data.UserID, err = strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("解析userId失败：%v", err)
	}

	// 解析couponId（核心修复点）
	couponIDStr, ok := msg.Values["couponId"].(string)
	if !ok {
		return errors.New("couponId字段类型错误")
	}
	data.CouponID, err = strconv.ParseInt(couponIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("解析couponId失败：%v", err)
	}

	// 解析orderId
	orderIDStr, ok := msg.Values["orderId"].(string)
	if !ok {
		return errors.New("orderId字段类型错误")
	}
	data.OrderID, err = strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("解析orderId失败：%v", err)
	}

	// 解析shopId
	shopIDStr, ok := msg.Values["shopId"].(string)
	if !ok {
		return errors.New("shopId字段类型错误")
	}
	data.ShopID, err = strconv.ParseInt(shopIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("解析shopId失败：%v", err)
	}

	//==========解析完成==========
	// ✅ 幂等性检查：根据订单ID查询是否已存在
	existingOrder, err := s.skDAO.GetOrderById(ctx, data.OrderID)
	if err == nil && existingOrder != nil && existingOrder.ID > 0 {
		// 订单已存在，直接确认消息（无需重复处理）
		_ = s.redisDAO.AckStreamMsg(ctx, msg.ID)
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// 查询出错（非记录不存在），返回错误让消息重试
		return fmt.Errorf("查询订单失败：%v", err)
	}

	// 数据库事务：乐观锁扣库存+创建订单
	order := &model.SeckillOrders{
		ID:        data.OrderID,
		UserID:    data.UserID,
		VoucherID: data.CouponID,
		ShopID:    data.ShopID,
	}
	if err := s.skDAO.CreateSeckillOrder(ctx, order, data.CouponID); err != nil {
		// 消费失败：回滚Redis状态（关键！修复数据不一致）
		s.rollbackRedisSeckillStatus(ctx, data.CouponID, data.UserID)
		return fmt.Errorf("创建订单失败：%v", err)
	}

	// 缓存更新策略改为：删除缓存（而不是更新）
	_ = s.redisDAO.DelVoucherStockCache(ctx, data.CouponID)

	// 确认消息消费成功
	return s.redisDAO.AckStreamMsg(ctx, msg.ID)
}

// 新增：消费失败时回滚Redis状态（库存+用户标记）
func (s *SeckillService) rollbackRedisSeckillStatus(ctx context.Context, voucherID, userID int64) {
	// 1. 恢复Redis库存（+1）
	_ = s.redisDAO.IncrVoucherStock(ctx, voucherID)

	// 2. 删除用户秒杀标记（允许重新下单）
	_ = s.redisDAO.RemUserSeckillFlag(ctx, voucherID, userID)
}

// ========== 查询秒杀结果 ==========
// GetSeckillOrderById 根据订单ID查询订单
func (s *SeckillService) GetSeckillOrderById(ctx context.Context, orderId int64) (*model.SeckillOrders, error) {
	return s.skDAO.GetOrderById(ctx, orderId)
}

// ========== 初始化秒杀库存（Cache Aside 写） ==========
func (s *SeckillService) InitSeckillStock(ctx context.Context, voucherID int64, stock int, expireTime time.Time) error {
	// 1. 先更数据库
	skVoucher := &model.SeckillVouchers{VoucherID: voucherID, Stock: stock}
	err := utils.DB.WithContext(ctx).Create(skVoucher).Error
	if err != nil {
		return err
	}
	// 2. 再更缓存（Cache Aside 写后更新）
	return s.redisDAO.SetVoucherStockToCache(ctx, voucherID, int64(stock), expireTime)
}
