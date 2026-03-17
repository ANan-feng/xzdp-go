package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// 定义脚本路径常量（统一管理，便于修改）
const (
	SeckillPreCheckScriptPath = "scripts/seckill/seckill_pre_check.lua"
	stockKeyPrefix            = "xzdp:voucher:stock:%d"   // 原xzdp:coupon:stock:%d
	userKeyPrefix             = "xzdp:voucher:user:%d:%d" // 原xzdp:coupon:user:%d:%d
)

// SeckillPreCheck 秒杀下单资质预检（读取Lua文件执行）
// voucherId: 优惠券ID
// userId: 用户ID
// expireTime: 优惠券过期时间（time.Time）
// 返回值：0-成功，1-过期，2-库存不足，3-用户已下单，-1-脚本执行失败
func SeckillPreCheck(ctx context.Context, voucherId int64, userId int64, expireTime time.Time) (int, error) {
	scriptContent, err := LoadLuaScript(SeckillPreCheckScriptPath)
	if err != nil {
		return -1, fmt.Errorf("load lua script failed: %v", err)
	}

	// 替换Key前缀（与Lua脚本中使用的Key统一）
	stockKey := fmt.Sprintf(stockKeyPrefix, voucherId)
	userKey := fmt.Sprintf(userKeyPrefix, voucherId, userId)

	expireTs := expireTime.Unix()
	nowTs := time.Now().Unix()
	keys := []string{stockKey, userKey}
	// 修正参数：ARGV[1]=过期时间, ARGV[2]=当前时间, ARGV[3]=用户ID（匹配Lua脚本）
	args := []interface{}{expireTs, nowTs, userId}

	script := redis.NewScript(scriptContent)
	result, err := script.Run(ctx, RedisClient, keys, args...).Int()
	if err != nil {
		return -1, fmt.Errorf("lua script exec failed: %v", err)
	}
	return result, nil
}

// // SetCouponStock 初始化优惠券库存（秒杀前调用）
// func SetCouponStock(ctx context.Context, voucherId int64, stock int64, expireTime time.Time) error {
// 	stockKey := fmt.Sprintf(stockKeyPrefix, voucherId)
// 	err := RedisClient.Set(ctx, stockKey, stock, expireTime.Sub(time.Now())).Err()
// 	if err != nil {
// 		return fmt.Errorf("set voucher stock failed: %v", err)
// 	}
// 	return nil
// }

// DeleteCoupon 下架优惠券（删除库存+用户下单标记）
func DeleteCoupon(ctx context.Context, voucherId int64) error {
	userKeyPattern := fmt.Sprintf("xzdp:voucher:user:%d:*", voucherId)
	keys, err := RedisClient.Keys(ctx, userKeyPattern).Result()
	if err != nil {
		return fmt.Errorf("get user keys failed: %v", err)
	}
	if len(keys) > 0 {
		_, err = RedisClient.Del(ctx, append(keys, fmt.Sprintf(stockKeyPrefix, voucherId))...).Result()
		if err != nil {
			return fmt.Errorf("del voucher keys failed: %v", err)
		}
	}
	return nil
}
