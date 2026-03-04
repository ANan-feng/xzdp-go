package utils

import (
	"context"
	"os"
	"strconv"

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
