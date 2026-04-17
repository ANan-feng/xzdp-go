package utils

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const (
	ConsumerNamePrefix = "seckill-consumer-" // 根据业务自定义
	SeckillStreamKey   = "stream:seckill:order"
	ConsumerGroupName  = "seckillGroup"
)

// 初始化消费者组 → 自动创建流（MKSTREAM）
func InitStreamGroup(ctx context.Context, client *redis.Client) error {
	// 关键：XGroupCreateMkStream = 自动创建空流 + 创建组
	return client.XGroupCreateMkStream(ctx, SeckillStreamKey, ConsumerGroupName, "0").Err()
}

// 补充SendToSeckillStream实现：发送消息到秒杀Stream
func SendToSeckillStream(ctx context.Context, msg map[string]interface{}) error {
	// XAdd往Redis Stream中发送消息
	return RedisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: SeckillStreamKey, // 需提前定义这个常量
		Values: msg,
		ID:     "*", // 自动生成ID
	}).Err()
}
