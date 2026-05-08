package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis Key前缀常量
const (
	stockKeyPrefix = "xzdp:voucher:stock:%d"
	userKeyPrefix  = "xzdp:voucher:user:%d"
)

// SetCouponStock 初始化库存到Redis
func SetCouponStock(ctx context.Context, voucherId int64, stock int64, expireTime time.Time) error {
	stockKey := fmt.Sprintf(stockKeyPrefix, voucherId)
	ttl := expireTime.Sub(time.Now())
	if ttl < 0 {
		ttl = 0
	}
	return RedisClient.Set(ctx, stockKey, stock, ttl).Err()
}

// ========== ✅ 新方案：原子DECR+SADD操作 ==========

// SeckillDecrAndCheckUser 原子操作：扣库存 + 检查一人一单（Lua脚本保证原子性）
// 返回值：0-成功，1-已过期，2-库存不足，3-用户已下单，-1-系统错误
func SeckillDecrAndCheckUser(ctx context.Context, voucherID int64, userID int64, expireTime time.Time) (int, error) {
	luaScript := `
		local stockKey = KEYS[1]
		local userKey = KEYS[2]
		local expireTs = tonumber(ARGV[1])
		local nowTs = tonumber(ARGV[2])
		local userId = tonumber(ARGV[3])

		-- 1. 校验优惠券是否过期
		if nowTs > expireTs then
			return 1
		end

		-- 2. 校验库存是否充足
		local stock = tonumber(redis.call('get', stockKey) or "0")
		if stock <= 0 then
			return 2
		end

		-- 3. 校验用户是否已下单（一人一单）
		if redis.call('sismember', userKey, userId) == 1 then
			return 3
		end

		-- 4. 原子操作：DECR库存 + SADD用户
		redis.call('decr', stockKey)
		redis.call('sadd', userKey, userId)
		redis.call('expire', userKey, expireTs - nowTs)

		return 0
	`

	stockKey := fmt.Sprintf(stockKeyPrefix, voucherID)
	userKey := fmt.Sprintf(userKeyPrefix, voucherID)
	expireTs := expireTime.Unix()
	nowTs := time.Now().Unix()

	script := redis.NewScript(luaScript)
	result, err := script.Run(ctx, RedisClient, []string{stockKey, userKey}, expireTs, nowTs, userID).Int()
	if err != nil {
		return -1, fmt.Errorf("Lua原子操作失败：%v", err)
	}
	return result, nil
}

// ========== ✅ 回滚脚本：原子回滚（INCR库存 + SREM用户）==========

// RollbackSeckillWithLua 使用Lua脚本原子回滚Redis状态
func RollbackSeckillWithLua(ctx context.Context, voucherID int64, userID int64) error {
	luaScript := `
		local stockKey = KEYS[1]
		local userKey = KEYS[2]
		local userId = tonumber(ARGV[1])

		-- 原子回滚：INCR库存 + SREM用户
		redis.call('incr', stockKey)
		redis.call('srem', userKey, userId)

		return 1
	`

	stockKey := fmt.Sprintf(stockKeyPrefix, voucherID)
	userKey := fmt.Sprintf(userKeyPrefix, voucherID)

	script := redis.NewScript(luaScript)
	_, err := script.Run(ctx, RedisClient, []string{stockKey, userKey}, userID).Result()
	if err != nil {
		return fmt.Errorf("回滚Redis失败：%v", err)
	}
	return nil
}

// DeleteCoupon 删除优惠券缓存
func DeleteCoupon(ctx context.Context, voucherID int64) error {
	stockKey := fmt.Sprintf(stockKeyPrefix, voucherID)
	userKey := fmt.Sprintf(userKeyPrefix, voucherID)
	_, err := RedisClient.Del(ctx, stockKey, userKey).Result()
	return err
}

// GetVoucherStockFromRedis 获取库存
func GetVoucherStockFromRedis(ctx context.Context, voucherID int64) (int64, error) {
	stockKey := fmt.Sprintf(stockKeyPrefix, voucherID)
	return RedisClient.Get(ctx, stockKey).Int64()
}

// DeleteVoucherStockCache 删除库存缓存（Cache Aside 缓存失效策略）
// 消费成功后删除缓存，下次读时从DB重建
func DeleteVoucherStockCache(ctx context.Context, voucherID int64) error {
	stockKey := fmt.Sprintf(stockKeyPrefix, voucherID)
	return RedisClient.Del(ctx, stockKey).Err()
}
