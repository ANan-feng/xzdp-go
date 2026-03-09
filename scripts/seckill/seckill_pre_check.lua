-- 秒杀下单预检脚本
-- 入参：
-- KEYS[1] = 优惠券库存Key（xzdp:coupon:stock:{couponId}）
-- KEYS[2] = 用户下单标记Key（xzdp:coupon:user:{couponId}:{userId}）
-- ARGV[1] = 优惠券过期时间戳（秒级）
-- ARGV[2] = 当前时间戳（秒级）
-- 返回值：
-- 0: 成功
-- 1: 优惠券已过期
-- 2: 库存不足
-- 3: 用户已下单

-- 1. 检查优惠券是否过期
if tonumber(ARGV[2]) > tonumber(ARGV[1]) then
    return 1
end

-- 2. 检查库存
local stock = tonumber(redis.call('get', KEYS[1]))
if not stock or stock <= 0 then
    return 2
end

-- 3. 检查用户是否已下单
if redis.call('exists', KEYS[2]) == 1 then
    return 3
end

-- 4. 扣减库存 + 标记用户已下单（原子操作）
redis.call('decr', KEYS[1])
redis.call('set', KEYS[2], 1)
-- 标记用户下单Key的过期时间（与优惠券有效期一致）
redis.call('expire', KEYS[2], tonumber(ARGV[1]) - tonumber(ARGV[2]) + 86400)
return 0