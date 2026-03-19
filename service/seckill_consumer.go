package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"xzdp-go/model"
	"xzdp-go/utils"

	"gorm.io/gorm"
)

// SeckillConsumer 秒杀消费者
type SeckillConsumer struct{}

// NewSeckillConsumer 创建消费者实例
func NewSeckillConsumer() *SeckillConsumer {
	return &SeckillConsumer{}
}

// StartConsume 启动消费者（阻塞运行）
func (sc *SeckillConsumer) StartConsume() {
	ctx := context.Background()
	reader := utils.KafkaReader
	defer reader.Close()

	fmt.Println("Kafka秒杀消费者已启动，开始监听消息...")

	for {
		// 消费消息（阻塞）
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			fmt.Printf("读取Kafka消息失败：%v，5秒后重试\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// 反序列化消息
		var reqMsg model.SeckillRequestMsg
		if err := json.Unmarshal(msg.Value, &reqMsg); err != nil {
			fmt.Printf("消息反序列化失败：%v，消息内容：%s\n", err, string(msg.Value))
			continue // 跳过无效消息
		}

		// 处理秒杀请求（异步处理，防止阻塞消费）
		go sc.handleSeckillRequest(ctx, reqMsg)
	}
}

// handleSeckillRequest 处理单个秒杀请求（核心逻辑）
func (sc *SeckillConsumer) handleSeckillRequest(ctx context.Context, reqMsg model.SeckillRequestMsg) {
	userId := reqMsg.UserID
	couponId := reqMsg.CouponID
	requestID := reqMsg.RequestID

	// ========== 1. 分布式锁（防重复下单/超卖） ==========
	lockKey := fmt.Sprintf("seckill:lock:%d_%d", userId, couponId)
	lock := utils.NewRedisLock(ctx, utils.GetRedisClient(), lockKey, 5*time.Second)
	lockSuccess, err := lock.Lock()
	if err != nil {
		fmt.Printf("[%s] 获取分布式锁失败：%v\n", requestID, err)
		return
	}
	if !lockSuccess {
		fmt.Printf("[%s] 分布式锁被占用，用户%d重复请求优惠券%d\n", requestID, userId, couponId)
		return
	}
	defer lock.Unlock() // 确保锁释放
	// ========== 2. 再次Redis预检（防止消息重复消费/库存已耗尽） ==========
	voucher, ok := sc.getSeckillVoucher(ctx, couponId)
	if !ok {
		fmt.Printf("[%s] 优惠券%d不存在\n", requestID, couponId)
		return
	}

	// Lua脚本：扣减库存 + 标记用户已下单（原子操作）
	result, err := utils.SeckillPreCheckAndDeduct(ctx, couponId, userId, voucher.EndTime)
	if err != nil {
		fmt.Printf("[%s] Redis扣减库存失败：%v\n", requestID, err)
		return
	}
	if result != 0 {
		fmt.Printf("[%s] 秒杀失败，结果码：%d，用户%d，优惠券%d\n", requestID, result, userId, couponId)
		return
	}

	// ========== 3. 数据库下单  ==========
	tx := utils.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		fmt.Printf("[%s] 开启事务失败：%v\n", requestID, tx.Error)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			fmt.Printf("[%s] 程序panic：%v\n", requestID, r)
		}
	}()

	orderId, ok := sc.createSeckillOrder(tx, userId, couponId, requestID)
	if !ok {
		tx.Rollback()
		// 回滚Redis库存
		// 在数据库下单失败时回滚
		stockKey := fmt.Sprintf("xzdp:voucher:stock:%d", couponId)
		utils.RedisClient.Incr(ctx, stockKey)

		userKey := fmt.Sprintf("xzdp:voucher:user:%d:%d", couponId, userId) // 修正Key格式
		utils.RedisClient.Del(ctx, userKey)
		return
	}

	// ========== 4. 写入秒杀结果到Redis（供查询） ==========
	resultKey := fmt.Sprintf("seckill:result:%s", requestID)
	// 设置结果缓存（过期时间10分钟，避免冗余）
	utils.RedisClient.Set(ctx, resultKey, orderId, 10*time.Minute)
	fmt.Printf("[%s] 秒杀成功，订单ID：%d，结果已写入Redis\n", requestID, orderId)
}

// 复用/改造原有工具函数（getSeckillVoucher/createSeckillOrder）
func (sc *SeckillConsumer) getSeckillVoucher(ctx context.Context, couponId int64) (*model.SeckillVoucher, bool) {
	voucher := &model.SeckillVoucher{}
	if err := utils.DB.WithContext(ctx).Where("voucher_id = ?", couponId).First(voucher).Error; err != nil {
		return nil, false
	}
	return voucher, true
}

func (sc *SeckillConsumer) createSeckillOrder(tx *gorm.DB, userId, couponId int64, requestID string) (int64, bool) {
	// 1. 扣减库存（乐观锁）
	voucher := &model.SeckillVoucher{VoucherID: couponId}
	if err := tx.Model(voucher).Where("stock > 0").Update("stock", gorm.Expr("stock - ?", 1)).Error; err != nil {
		fmt.Printf("[%s] 扣减库存失败：%v\n", requestID, err)
		return 0, false
	}

	// 2. 检查库存
	var count int64
	tx.Model(&model.SeckillVoucher{}).Where("voucher_id = ?", couponId).Select("stock").Scan(&count)
	if count < 0 {
		fmt.Printf("[%s] 库存不足，优惠券%d\n", requestID, couponId)
		return 0, false
	}

	// 3. 创建订单
	order := &model.SeckillOrder{
		UserID:     userId,
		VoucherID:  couponId,
		CreateTime: time.Now(),
	}
	if err := tx.Create(order).Error; err != nil {
		fmt.Printf("[%s] 创建订单失败：%v\n", requestID, err)
		return 0, false
	}

	// 4. 提交事务
	if err := tx.Commit().Error; err != nil {
		fmt.Printf("[%s] 提交事务失败：%v\n", requestID, err)
		return 0, false
	}

	return order.ID, true
}
