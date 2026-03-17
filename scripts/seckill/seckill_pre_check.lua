-- SeckillPreCheckLua 秒杀预检Lua脚本（带nil兜底）
-- 参数：KEYS[1]=库存Key, KEYS[2]=用户下单标记Key
-- ARGV[1]=优惠券过期时间戳(秒), ARGV[2]=当前时间戳(秒)

-- 1. 校验优惠券过期时间（nil兜底）
if tonumber(ARGV[1]) < tonumber(ARGV[2]) then
    return 1  -- 过期
end

-- 2. 校验库存（nil兜底）
local stock = redis.call('get', KEYS[1])
if not stock or tonumber(stock) <= 0 then
    return 2  -- 库存不足
end

-- 3. 校验用户是否已下单
if redis.call('sismember', KEYS[2], ARGV[3]) == 1 then
    return 3  -- 已下单
end

-- 4. 扣减库存+标记用户下单（原子操作）
redis.call('decr', KEYS[1])
redis.call('sadd', KEYS[2], ARGV[3])
return 0  -- 成功