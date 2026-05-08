package utils

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"
	"xzdp-go/model"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	RedisClient *redis.Client
	ctx         = context.Background()
)

// Redis Key 常量定义
const (
	// 博客点赞 SortedSet: score=时间戳, member=userId
	BlogLikeKey = "blog:like:%d"
	// 评论点赞 Set: member=userId
	CommentLikeKey = "comment:like:%d"
	// 博客详情缓存 Hash: field为blog各字段
	BlogDetailCacheKey = "blog:detail:%d"
	// 热门一级评论ID列表 List: 只缓存第1页10条
	BlogHotCommentsKey = "blog:comments:%d:hot"
	// 最新一级评论ID列表 List: 只缓存第1页10条
	BlogRecentCommentsKey = "blog:comments:%d:recent"
	// 单条评论详情缓存 Hash: field为comment各字段
	CommentDetailCacheKey = "comment:detail:%d"
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

// GetRedisClient 获取全局Redis客户端（供其他包调用）
func GetRedisClient() *redis.Client {
	return RedisClient
}

// GetDB 获取全局数据库连接（供其他包调用）
func GetDB() *gorm.DB {
	return DB
}

// InitSeckillCouponCache 初始化秒杀优惠券缓存（含库存+过期时间）
func InitSeckillCouponCache(ctx context.Context) error {
	redisClient := GetRedisClient()
	db := GetDB()

	var seckillVouchers []model.SeckillVoucher
	if err := db.Find(&seckillVouchers).Error; err != nil {
		return err
	}

	for _, sv := range seckillVouchers {
		// 1. 初始化库存（使用统一的Key前缀）
		stockKey := fmt.Sprintf("xzdp:voucher:stock:%d", sv.VoucherID)
		if err := redisClient.Set(ctx, stockKey, sv.Stock, 0).Err(); err != nil {
			return err
		}

		// 2. 初始化用户下单集合（清空旧数据）
		userKey := fmt.Sprintf("xzdp:voucher:user:%d:%d", sv.VoucherID, 0) // 仅初始化结构，用户ID动态填充
		if err := redisClient.Del(ctx, userKey).Err(); err != nil {
			return err
		}
	}
	return nil
}

// RedisSetNXWithExpire 原子性执行 SET NX EX 命令
func RedisSetNXWithExpire(ctx context.Context, key, value string, expire time.Duration) (bool, error) {
	return RedisClient.SetNX(ctx, key, value, expire).Result()
}

// RedisEval 执行Lua脚本
func RedisEval(ctx context.Context, script string, keys []string, args []string) (interface{}, error) {
	// 新增：将 []string 转换为 []interface{}
	argsInterface := make([]interface{}, len(args))
	for i, v := range args {
		argsInterface[i] = v
	}
	// 传入转换后的 []interface{} 类型参数
	return RedisClient.Eval(ctx, script, keys, argsInterface...).Result()
}
