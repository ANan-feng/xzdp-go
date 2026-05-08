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
	skDAO *dao.SeckillDAO
}

func NewSeckillService(db *gorm.DB) *SeckillService {
	return &SeckillService{
		skDAO: dao.NewSeckillDAO(utils.DB),
	}
}

// ✅ CreateSeckillOrder 秒杀下单（优化版：只查一次DB，一人一单交Redis）
func (s *SeckillService) CreateSeckillOrder(ctx context.Context, userID, voucherID int64) (int64, error) {
	// ✅ 只查一次DB：获取秒杀券信息 + 主券信息
	skVoucher, voucher, err := s.skDAO.GetSeckillVoucherByID(ctx, voucherID)
	if err != nil {
		return 0, fmt.Errorf("查询优惠券失败：%v", err)
	}

	// 校验秒杀时间
	now := time.Now()
	if now.Before(skVoucher.BeginTime) || now.After(skVoucher.EndTime) {
		return 0, errors.New("秒杀未开始或已结束")
	}

	// 校验优惠券状态
	if voucher.Status != 1 {
		return 0, errors.New("优惠券已下架")
	}

	// ✅ 使用Redis原子操作（DECR+SADD）检查一人一单 + 扣库存
	result, err := utils.SeckillDecrAndCheckUser(ctx, voucherID, userID, skVoucher.EndTime)
	if err != nil {
		return 0, fmt.Errorf("Redis操作失败：%v", err)
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

	// 生成订单ID
	orderID, err := utils.IDGenerator.Generate(ctx, "order")
	if err != nil {
		// 回滚Redis（Lua原子回滚）
		_ = utils.RollbackSeckillWithLua(ctx, voucherID, userID)
		return 0, fmt.Errorf("生成订单ID失败：%v", err)
	}

	// ✅ 去掉DAO套娃，直接调utils发送消息
	msg := map[string]interface{}{
		"userId":    strconv.FormatInt(userID, 10),
		"couponId":  strconv.FormatInt(voucherID, 10),
		"orderId":   strconv.FormatInt(orderID, 10),
		"shopId":    strconv.FormatInt(voucher.ShopID, 10),
		"timestamp": strconv.FormatInt(now.Unix(), 10),
	}
	if err := utils.SendToSeckillStream(ctx, msg); err != nil {
		// 消息发送失败，回滚Redis
		_ = utils.RollbackSeckillWithLua(ctx, voucherID, userID)
		return 0, fmt.Errorf("发送异步消息失败：%v", err)
	}

	return orderID, nil
}

// ========== MQ消费者：添加乐观锁 WHERE stock > 0 兜底 ==========
func (s *SeckillService) ConsumeSeckillMsg(ctx context.Context, msg redis.XMessage) error {
	// 解析消息
	data := struct {
		UserID   int64
		CouponID int64
		OrderID  int64
		ShopID   int64
	}{}

	// 手动解析字段
	var err error
	userIDStr, ok := msg.Values["userId"].(string)
	if !ok {
		return errors.New("userId字段类型错误")
	}
	data.UserID, err = strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("解析userId失败：%v", err)
	}

	couponIDStr, ok := msg.Values["couponId"].(string)
	if !ok {
		return errors.New("couponId字段类型错误")
	}
	data.CouponID, err = strconv.ParseInt(couponIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("解析couponId失败：%v", err)
	}

	orderIDStr, ok := msg.Values["orderId"].(string)
	if !ok {
		return errors.New("orderId字段类型错误")
	}
	data.OrderID, err = strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("解析orderId失败：%v", err)
	}

	shopIDStr, ok := msg.Values["shopId"].(string)
	if !ok {
		return errors.New("shopId字段类型错误")
	}
	data.ShopID, err = strconv.ParseInt(shopIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("解析shopId失败：%v", err)
	}

	// ✅ 幂等性检查：订单是否已存在
	existingOrder, err := s.skDAO.GetOrderById(ctx, data.OrderID)
	if err == nil && existingOrder != nil && existingOrder.ID > 0 {
		// 订单已存在，确认消息
		return s.AckStreamMsg(ctx, msg.ID)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询订单失败：%v", err)
	}

	// ✅ 创建订单（乐观锁 WHERE stock > 0 做最终兜底）
	order := &model.SeckillOrder{
		ID:        data.OrderID,
		UserID:    data.UserID,
		VoucherID: data.CouponID,
		ShopID:    data.ShopID,
	}

	if err := s.skDAO.CreateSeckillOrderWithOptimisticLock(ctx, order, data.CouponID); err != nil {
		// 创建订单失败：使用Lua脚本原子回滚Redis
		_ = utils.RollbackSeckillWithLua(ctx, data.CouponID, data.UserID)
		return fmt.Errorf("创建订单失败：%v", err)
	}

	// ✅ 删除库存缓存（Cache Aside 失效策略）
	// 下次读时从DB重建缓存，保证与DB一致
	_ = utils.DeleteVoucherStockCache(ctx, data.CouponID)

	// 确认消息
	return s.AckStreamMsg(ctx, msg.ID)
}

// AckStreamMsg 确认消息消费
func (s *SeckillService) AckStreamMsg(ctx context.Context, msgID string) error {
	return utils.RedisClient.XAck(ctx, utils.SeckillStreamKey, utils.ConsumerGroupName, msgID).Err()
}

// GetSeckillOrderById 查询订单
func (s *SeckillService) GetSeckillOrderById(ctx context.Context, orderId int64) (*model.SeckillOrder, error) {
	return s.skDAO.GetOrderById(ctx, orderId)
}

// InitSeckillStock 初始化库存
func (s *SeckillService) InitSeckillStock(ctx context.Context, voucherID int64, stock int, expireTime time.Time) error {
	// 1. 数据库
	skVoucher := &model.SeckillVoucher{VoucherID: voucherID, Stock: stock}
	if err := utils.DB.WithContext(ctx).Create(skVoucher).Error; err != nil {
		return err
	}
	// 2. Redis缓存
	return utils.SetCouponStock(ctx, voucherID, int64(stock), expireTime)
}
