// consumer/seckill_consumer.go
package consumer

import (
	"context"
	"fmt"
	"time"
	"xzdp-go/service"
	"xzdp-go/utils"

	"github.com/redis/go-redis/v9"
)

type SeckillConsumer struct {
	consumerName string
	skService    *service.SeckillService
}

func NewSeckillConsumer() *SeckillConsumer {
	return &SeckillConsumer{
		consumerName: utils.ConsumerNamePrefix + utils.GenerateUUID()[:8],
		skService:    service.NewSeckillService(utils.DB),
	}
}

// Start 启动消费者
func (c *SeckillConsumer) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			c.consumeOnce(ctx)
		}
	}
}

func (c *SeckillConsumer) consumeOnce(ctx context.Context) {
	// 读取Stream消息
	streams, err := utils.RedisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    utils.ConsumerGroupName,
		Consumer: c.consumerName,
		Streams:  []string{utils.SeckillStreamKey, ">"},
		Count:    1,
		Block:    2 * time.Second,
	}).Result()

	if err != nil || len(streams) == 0 {
		return
	}

	// 处理消息
	for _, msg := range streams[0].Messages {
		if err := c.skService.ConsumeSeckillMsg(ctx, msg); err != nil {
			fmt.Printf("消费消息失败：%v, msgID=%s\n", err, msg.ID)
			continue
		}
	}
}
