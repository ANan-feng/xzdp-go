package utils

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	RedisClient *redis.Client
	ctx         = context.Background()
)

// InitRedis 初始化Redis连接（和InitDB同级，在main.go中调用）
func InitRedis() {
	// 从.env读取Redis配置（先在.env中添加）
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB, _ := strconv.Atoi(os.Getenv("REDIS_DB"))

	// 创建Redis客户端
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: redisPassword, // 无密码则为空字符串
		DB:       redisDB,       // 使用第0个数据库
	})

	// 测试连接
	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		panic("Redis connect failed: " + err.Error())
	}
}

// InitSeckillCouponCache 初始化秒杀优惠券库存缓存
func InitSeckillCouponCache(ctx context.Context) error {
	// 这里可以从数据库/配置文件读取优惠券配置，而非硬编码
	couponConfigs := []struct {
		couponId int64
		stock    int64
		expire   time.Duration
	}{
		{1, 100, 1 * time.Hour},
		{2, 50, 1 * time.Hour},
		// 更多优惠券配置...
	}

	for _, cfg := range couponConfigs {
		if err := SetCouponStock(ctx, cfg.couponId, cfg.stock, time.Now().Add(cfg.expire)); err != nil {
			return err
		}
	}
	return nil
}
