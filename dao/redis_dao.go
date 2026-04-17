package dao

import (
	"context"
	"fmt"
	"time"
	"xzdp-go/utils"

	"github.com/redis/go-redis/v9"
)

type RedisDAO struct {
	client *redis.Client
}

func NewRedisDAO() *RedisDAO {
	return &RedisDAO{client: utils.RedisClient}
}

// ========== Cache Aside 模式：缓存一致性 ==========
// GetVoucherStockFromCache 从缓存获取库存（Cache Aside 读）
func (dao *RedisDAO) GetVoucherStockFromCache(ctx context.Context, voucherID int64) (int64, error) {
	key := fmt.Sprintf("xzdp:voucher:stock:%d", voucherID)
	return dao.client.Get(ctx, key).Int64()
}

// SetVoucherStockToCache 设置缓存库存（Cache Aside 写后更新）
func (dao *RedisDAO) SetVoucherStockToCache(ctx context.Context, voucherID int64, stock int64, expire time.Time) error {
	key := fmt.Sprintf("xzdp:voucher:stock:%d", voucherID)
	return dao.client.Set(ctx, key, stock, expire.Sub(time.Now())).Err()
}

// DelVoucherStockCache 删除缓存（Cache Aside 删前删除）
func (dao *RedisDAO) DelVoucherStockCache(ctx context.Context, voucherID int64) error {
	key := fmt.Sprintf("xzdp:voucher:stock:%d", voucherID)
	return dao.client.Del(ctx, key).Err()
}

// ========== 秒杀防护：防穿透/雪崩/击穿 ==========
// SetUserSeckillFlag 设置用户秒杀标记（防穿透，过期时间和优惠券一致）
func (dao *RedisDAO) SetUserSeckillFlag(ctx context.Context, voucherID, userID int64, expire time.Duration) error {
	key := fmt.Sprintf("xzdp:voucher:user:%d:%d", voucherID, userID)
	return dao.client.SAdd(ctx, key, userID).Err()
}

// CheckUserSeckillFlag 检查用户秒杀标记（防重复下单）
func (dao *RedisDAO) CheckUserSeckillFlag(ctx context.Context, voucherID, userID int64) (bool, error) {
	key := fmt.Sprintf("xzdp:voucher:user:%d:%d", voucherID, userID)
	res, err := dao.client.SIsMember(ctx, key, userID).Result()
	return res, err
}

// ========== Stream 消息队列 ==========
// SendSeckillMsgToStream 发送秒杀消息到Stream
func (dao *RedisDAO) SendSeckillMsgToStream(ctx context.Context, msg map[string]interface{}) error {
	return utils.SendToSeckillStream(ctx, msg)
}

// AckStreamMsg 确认Stream消息消费成功
func (dao *RedisDAO) AckStreamMsg(ctx context.Context, msgID string) error {
	return dao.client.XAck(ctx, utils.SeckillStreamKey, utils.ConsumerGroupName, msgID).Err()
}

// ========== 新增：封装需要的 Redis 操作方法 ==========
// IncrVoucherStock 增加优惠券库存（用于回滚）
func (dao *RedisDAO) IncrVoucherStock(ctx context.Context, voucherID int64) error {
	key := fmt.Sprintf("xzdp:voucher:stock:%d", voucherID)
	return dao.client.Incr(ctx, key).Err()
}

// RemUserSeckillFlag 移除用户秒杀标记（用于回滚）
func (dao *RedisDAO) RemUserSeckillFlag(ctx context.Context, voucherID, userID int64) error {
	key := fmt.Sprintf("xzdp:voucher:user:%d:%d", voucherID, userID)
	return dao.client.SRem(ctx, key, userID).Err()
}
