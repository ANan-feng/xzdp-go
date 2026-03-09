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
)

// SeckillPreCheck 秒杀下单资质预检（读取Lua文件执行）
// couponId: 优惠券ID
// userId: 用户ID
// expireTime: 优惠券过期时间（time.Time）
// 返回值：0-成功，1-过期，2-库存不足，3-用户已下单，-1-脚本执行失败
func SeckillPreCheck(ctx context.Context, couponId int64, userId int64, expireTime time.Time) (int, error) {
	// 1. 加载Lua脚本（项目启动时可缓存，避免每次读取文件）
	scriptContent, err := LoadLuaScript(SeckillPreCheckScriptPath)
	if err != nil {
		return -1, fmt.Errorf("load lua script failed: %v", err)
	}

	// 2. 构建Redis Key
	stockKey := fmt.Sprintf("xzdp:coupon:stock:%d", couponId)
	userKey := fmt.Sprintf("xzdp:coupon:user:%d:%d", couponId, userId)

	// 3. 准备入参
	expireTs := expireTime.Unix()
	nowTs := time.Now().Unix()
	keys := []string{stockKey, userKey}
	args := []interface{}{expireTs, nowTs}

	// 4. 执行脚本
	script := redis.NewScript(scriptContent)
	result, err := script.Run(ctx, RedisClient, keys, args...).Int()
	if err != nil {
		return -1, fmt.Errorf("lua script exec failed: %v", err)
	}
	return result, nil
}

// SetCouponStock 初始化优惠券库存（秒杀前调用）
func SetCouponStock(ctx context.Context, couponId int64, stock int64, expireTime time.Time) error {
	stockKey := fmt.Sprintf("xzdp:coupon:stock:%d", couponId)
	// 设置库存 + 过期时间（与优惠券一致）
	err := RedisClient.Set(ctx, stockKey, stock, expireTime.Sub(time.Now())).Err()
	if err != nil {
		return fmt.Errorf("set coupon stock failed: %v", err)
	}
	return nil
}

// DeleteCoupon 下架优惠券（删除库存+用户下单标记）
func DeleteCoupon(ctx context.Context, couponId int64) error {
	// 匹配所有用户下单标记Key
	userKeyPattern := fmt.Sprintf("xzdp:coupon:user:%d:*", couponId)
	keys, err := RedisClient.Keys(ctx, userKeyPattern).Result()
	if err != nil {
		return fmt.Errorf("get user keys failed: %v", err)
	}
	// 批量删除
	if len(keys) > 0 {
		_, err = RedisClient.Del(ctx, append(keys, fmt.Sprintf("xzdp:coupon:stock:%d", couponId))...).Result()
		if err != nil {
			return fmt.Errorf("del coupon keys failed: %v", err)
		}
	}
	return nil
}
