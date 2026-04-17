package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// SeckillLua 秒杀Lua脚本相关操作结构体
type SeckillLua struct {
	redisClient *redis.Client
	luaScript   *redis.Script // 预加载的Lua脚本
}

// 秒杀前置校验的Lua脚本内容（和seckill_pre_check.lua一致）
const seckillPreCheckLua = `
local stockKey = KEYS[1]
local userKey = KEYS[2]
local expireTs = ARGV[1]
local nowTs = ARGV[2]
local userId = ARGV[3]

-- 1. 校验优惠券是否过期
if tonumber(nowTs) > tonumber(expireTs) then
    return 1
end

-- 2. 校验库存
local stock = tonumber(redis.call('get', stockKey) or "0")
if stock <= 0 then
    return 2
end

-- 3. 校验用户是否已下单（核心：防止穿透）
if redis.call('sismember', userKey, userId) == 1 then
    return 3
end

-- 4. 原子操作：扣减库存 + 标记用户下单
redis.call('decr', stockKey)
redis.call('sadd', userKey, userId)
-- 设置用户下单标记过期时间（和优惠券过期时间一致）
redis.call('expire', userKey, tonumber(expireTs) - tonumber(nowTs))

return 0
`

// NewSeckillLua 初始化秒杀Lua脚本执行器
func NewSeckillLua(redisClient *redis.Client) *SeckillLua {
	// 预加载Lua脚本，提升执行效率
	script := redis.NewScript(seckillPreCheckLua)
	return &SeckillLua{
		redisClient: redisClient,
		luaScript:   script,
	}
}

// 秒杀错误码定义
const (
	SeckillErrExpired = 1 // 优惠券过期
	SeckillErrNoStock = 2 // 库存不足
	SeckillErrRepeat  = 3 // 重复下单
	SeckillErrSuccess = 0 // 秒杀成功
)

// SeckillPreCheck 执行秒杀前置校验（封装Lua脚本调用）
// 参数说明：
//
//	stockKey: 库存Redis Key
//	userKey: 用户下单记录Redis Key
//	expireTs: 优惠券过期时间戳（秒）
//	userId: 用户ID
//
// 返回值：
//
//	code: 结果码（SeckillErrXXX）
//	err: 系统错误（如Redis调用失败）
func (s *SeckillLua) SeckillPreCheck(ctx context.Context, stockKey, userKey string, expireTs int64, userId string) (int, error) {
	// 获取当前时间戳（秒）
	nowTs := time.Now().Unix()

	// 执行Lua脚本
	// KEYS: [stockKey, userKey]
	// ARGV: [expireTs, nowTs, userId]
	result, err := s.luaScript.Run(
		ctx,
		s.redisClient,
		[]string{stockKey, userKey}, // KEYS参数
		expireTs, nowTs, userId,     // ARGV参数
	).Int()

	if err != nil {
		return -1, fmt.Errorf("lua script execute failed: %w", err)
	}

	// 校验结果码合法性
	switch result {
	case SeckillErrExpired, SeckillErrNoStock, SeckillErrRepeat, SeckillErrSuccess:
		return result, nil
	default:
		return -1, fmt.Errorf("unknown lua script result: %d", result)
	}
}
