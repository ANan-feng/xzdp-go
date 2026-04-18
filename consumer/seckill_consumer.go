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

// Start 启动消费者（正常消费 + 重试协程）
func (c *SeckillConsumer) Start(ctx context.Context) {
	// 1. 启动正常消费循环（读取新消息）
	go c.consumeLoop(ctx)

	// 2. 启动重试协程：处理超时未确认的 Pending 消息
	go c.retryPendingMessages(ctx)
}

// consumeLoop 正常消费新消息
func (c *SeckillConsumer) consumeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			c.consumeOnce(ctx)
		}
	}
}

// consumeOnce 读取一条新消息并处理
func (c *SeckillConsumer) consumeOnce(ctx context.Context) {
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
	for _, msg := range streams[0].Messages {
		if err := c.skService.ConsumeSeckillMsg(ctx, msg); err != nil {
			fmt.Printf("消费消息失败：%v, msgID=%s\n", err, msg.ID)
			// 失败：不 ACK，消息会留在 Pending 列表，等待重试
			continue
		}
		// 成功：显式 ACK，从 Pending 列表中移除消息
		if err := utils.RedisClient.XAck(ctx, utils.SeckillStreamKey, utils.ConsumerGroupName, msg.ID).Err(); err != nil {
			fmt.Printf("ACK 失败：%v, msgID=%s\n", err, msg.ID)
		}
	}
}

// retryPendingMessages 定期认领超时未确认的消息
func (c *SeckillConsumer) retryPendingMessages(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.claimPendingMessages(ctx)
		}
	}
}

// claimPendingMessages 查询并认领超时的 Pending 消息
func (c *SeckillConsumer) claimPendingMessages(ctx context.Context) {
	// 1. 获取 Pending 消息摘要
	pending, err := utils.RedisClient.XPending(ctx, utils.SeckillStreamKey, utils.ConsumerGroupName).Result()
	if err != nil || pending.Count == 0 {
		return
	}

	// 2. 遍历每个消费者组中的 Pending 消息（简化：直接获取所有 Pending 消息的 ID 列表）
	//    使用 XPendingExt 获取详细信息
	pendingExt, err := utils.RedisClient.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: utils.SeckillStreamKey,
		Group:  utils.ConsumerGroupName,
		Start:  "-",
		End:    "+",
		Count:  100,
	}).Result()
	if err != nil {
		fmt.Printf("获取 pending 详情失败：%v\n", err)
		return
	}

	// 3. 筛选 idle 超过 1 分钟的消息，使用 XClaim 重新分配给当前消费者
	for _, p := range pendingExt {
		if p.Idle > time.Minute {
			// 认领消息，重置 idle 时间
			claimed, err := utils.RedisClient.XClaim(ctx, &redis.XClaimArgs{
				Stream:   utils.SeckillStreamKey,
				Group:    utils.ConsumerGroupName,
				Consumer: c.consumerName,
				Messages: []string{p.ID},
				MinIdle:  time.Minute,
			}).Result()
			if err != nil {
				fmt.Printf("认领消息 %s 失败：%v\n", p.ID, err)
				continue
			}
			// 重新消费认领到的消息
			for _, msg := range claimed {
				fmt.Printf("重试消费 pending 消息：%s\n", msg.ID)
				if err := c.skService.ConsumeSeckillMsg(ctx, msg); err != nil {
					fmt.Printf("重试消费失败：%v\n", err)
				}
			}
		}
	}
}
