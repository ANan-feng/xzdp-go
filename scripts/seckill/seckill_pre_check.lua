-- scripts/seckill/seckill_pre_check.lua
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