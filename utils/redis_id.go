package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisIDGenerator Redis全局唯一ID生成器
// 结构：时间戳(32位) + 机器ID(10位) + 序列号(22位)
type RedisIDGenerator struct {
	redisClient *redis.Client
	machineID   int64 // 机器ID（分布式部署时区分不同机器，避免ID冲突）
}

// NewRedisIDGenerator 初始化ID生成器
func NewRedisIDGenerator(machineID int64) *RedisIDGenerator {
	if machineID < 0 || machineID > 1023 { // 10位机器ID最大1023
		panic("machineID must be between 0 and 1023")
	}
	return &RedisIDGenerator{
		redisClient: RedisClient,
		machineID:   machineID,
	}
}

// Generate 生成全局唯一ID
func (g *RedisIDGenerator) Generate(ctx context.Context, keyPrefix string) (int64, error) {
	// 1. 生成日期前缀（避免序列号溢出，每天一个key）
	now := time.Now()
	dateKey := now.Format("20060102")
	fullKey := fmt.Sprintf("xzdp:seq:%s:%s", keyPrefix, dateKey)

	// 2. Redis自增获取序列号（原子操作，步长1）
	seq, err := g.redisClient.Incr(ctx, fullKey).Result()
	if err != nil {
		return 0, err
	}

	// 3. 设置key过期时间（2天，避免冗余）
	_, err = g.redisClient.Expire(ctx, fullKey, 2*24*time.Hour).Result()
	if err != nil {
		return 0, err
	}

	// 4. 拼接ID：时间戳(秒级) << 32 | 机器ID << 22 | 序列号
	timestamp := now.Unix()
	id := (timestamp << 32) | (g.machineID << 22) | seq
	return id, nil
}

// 全局ID生成器实例（全局唯一，初始化时指定机器ID）
var IDGenerator = NewRedisIDGenerator(1) // 测试环境机器ID=1，生产环境可从配置读取
