package model

import "time"

// SeckillRequestMsg Kafka秒杀请求消息结构（补充缺失字段）
type SeckillRequestMsg struct {
	UserID     int64     `json:"user_id"`     // 用户ID
	CouponID   int64     `json:"coupon_id"`   // 优惠券ID
	ShopID     int64     `json:"shop_id"`     // 商铺ID
	Stock      int       `json:"stock"`       // 库存
	BeginTime  time.Time `json:"begin_time"`  // 秒杀开始时间
	EndTime    time.Time `json:"end_time"`    // 秒杀结束时间
	RequestID  string    `json:"request_id"`  // 请求唯一标识（防重复）
	MsgID      string    `json:"msg_id"`      // 消息唯一ID
	CreateTime time.Time `json:"create_time"` // 请求时间
}

// SeckillResponseMsg Kafka秒杀结果消息结构（可选，用于通知用户）
type SeckillResponseMsg struct {
	RequestID string `json:"request_id"` // 关联请求ID
	UserID    int64  `json:"user_id"`    // 用户ID
	CouponID  int64  `json:"coupon_id"`  // 优惠券ID
	OrderID   int64  `json:"order_id"`   // 订单ID（成功则有值）
	Success   bool   `json:"success"`    // 是否成功
	Msg       string `json:"msg"`        // 结果描述
}
