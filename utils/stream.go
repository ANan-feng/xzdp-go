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

// SendToSeckillStream 发送消息到秒杀Stream（去掉DAO套娃）
func SendToSeckillStream(ctx context.Context, msg map[string]interface{}) error {
	if RedisClient == nil {
		return nil
	}
	// XAdd往Redis Stream中发送消息
	return RedisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: SeckillStreamKey,
		Values: msg,
		ID:     "*", // 自动生成ID
	}).Err()
}

// AckStreamMsg 确认消息消费
func AckStreamMsg(ctx context.Context, streamKey, groupName, msgID string) error {
	return RedisClient.XAck(ctx, streamKey, groupName, msgID).Err()
}
