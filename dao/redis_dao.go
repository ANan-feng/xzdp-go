package dao

import (
	"context"
	"fmt"
	"strconv"
	"time"
	"xzdp-go/model"
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

// ========== 博客点赞相关 ==========
// BlogLikeAdd 添加博客点赞
func (dao *RedisDAO) BlogLikeAdd(ctx context.Context, blogId, userId int64) error {
	key := fmt.Sprintf(utils.BlogLikeKey, blogId)
	return dao.client.ZAdd(ctx, key, redis.Z{Score: float64(time.Now().Unix()), Member: userId}).Err()
}

// BlogLikeRemove 移除博客点赞
func (dao *RedisDAO) BlogLikeRemove(ctx context.Context, blogId, userId int64) error {
	key := fmt.Sprintf(utils.BlogLikeKey, blogId)
	return dao.client.ZRem(ctx, key, userId).Err()
}

// IsBlogLiked 检查是否已点赞博客
func (dao *RedisDAO) IsBlogLiked(ctx context.Context, blogId, userId int64) (bool, error) {
	key := fmt.Sprintf(utils.BlogLikeKey, blogId)
	score, err := dao.client.ZScore(ctx, key, fmt.Sprintf("%d", userId)).Result()
	if err == redis.Nil {
		return false, nil
	}
	return score > 0, err
}

// GetBlogLikeTopN 获取博客点赞排行榜前N
func (dao *RedisDAO) GetBlogLikeTopN(ctx context.Context, blogId int64, n int64) ([]int64, error) {
	key := fmt.Sprintf(utils.BlogLikeKey, blogId)
	result, err := dao.client.ZRevRangeWithScores(ctx, key, 0, n-1).Result()
	if err != nil {
		return nil, err
	}
	userIds := make([]int64, len(result))
	for i, z := range result {
		if member, ok := z.Member.(string); ok {
			// 假设member是string格式的int64
			if id, err := strconv.ParseInt(member, 10, 64); err == nil {
				userIds[i] = id
			}
		}
	}
	return userIds, nil
}

// GetBlogLikeCount 获取博客点赞数
func (dao *RedisDAO) GetBlogLikeCount(ctx context.Context, blogId int64) (int64, error) {
	key := fmt.Sprintf(utils.BlogLikeKey, blogId)
	return dao.client.ZCard(ctx, key).Result()
}

// ========== 评论点赞相关 ==========
// CommentLikeAdd 添加评论点赞
func (dao *RedisDAO) CommentLikeAdd(ctx context.Context, commentId, userId int64) error {
	key := fmt.Sprintf(utils.CommentLikeKey, commentId)
	return dao.client.SAdd(ctx, key, userId).Err()
}

// CommentLikeRemove 移除评论点赞
func (dao *RedisDAO) CommentLikeRemove(ctx context.Context, commentId, userId int64) error {
	key := fmt.Sprintf(utils.CommentLikeKey, commentId)
	return dao.client.SRem(ctx, key, userId).Err()
}

// IsCommentLiked 检查是否已点赞评论
func (dao *RedisDAO) IsCommentLiked(ctx context.Context, commentId, userId int64) (bool, error) {
	key := fmt.Sprintf(utils.CommentLikeKey, commentId)
	return dao.client.SIsMember(ctx, key, userId).Result()
}

// ========== 博客详情缓存 ==========
// GetBlogDetailCache 获取博客详情缓存
func (dao *RedisDAO) GetBlogDetailCache(ctx context.Context, blogId int64) (*model.Blog, error) {
	key := fmt.Sprintf(utils.BlogDetailCacheKey, blogId)
	result, err := dao.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, redis.Nil
	}

	blog := &model.Blog{}
	if idStr, ok := result["id"]; ok {
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			blog.Id = id
		}
	}
	if shopIdStr, ok := result["shop_id"]; ok {
		if shopId, err := strconv.ParseInt(shopIdStr, 10, 64); err == nil {
			blog.ShopId = shopId
		}
	}
	if userIdStr, ok := result["user_id"]; ok {
		if userId, err := strconv.ParseInt(userIdStr, 10, 64); err == nil {
			blog.UserId = userId
		}
	}
	blog.Title = result["title"]
	blog.Images = result["images"]
	blog.Content = result["content"]
	if likeStr, ok := result["liked_count"]; ok {
		if likedCount, err := strconv.Atoi(likeStr); err == nil {
			blog.LikedCount = likedCount
		}
	}
	if commentStr, ok := result["comment_count"]; ok {
		if commentCount, err := strconv.Atoi(commentStr); err == nil {
			blog.CommentCount = commentCount
		}
	}
	if createTimeStr, ok := result["create_time"]; ok {
		if ts, err := strconv.ParseInt(createTimeStr, 10, 64); err == nil {
			blog.CreateTime = time.Unix(ts, 0)
		}
	}
	if updateTimeStr, ok := result["update_time"]; ok {
		if ts, err := strconv.ParseInt(updateTimeStr, 10, 64); err == nil {
			blog.UpdateTime = time.Unix(ts, 0)
		}
	}
	return blog, nil
}

// SetBlogDetailCache 设置博客详情缓存
func (dao *RedisDAO) SetBlogDetailCache(ctx context.Context, blog *model.Blog, ttl time.Duration) error {
	key := fmt.Sprintf(utils.BlogDetailCacheKey, blog.Id)
	fields := map[string]interface{}{
		"id":            blog.Id,
		"shop_id":       blog.ShopId,
		"user_id":       blog.UserId,
		"title":         blog.Title,
		"images":        blog.Images,
		"content":       blog.Content,
		"liked_count":   blog.LikedCount,
		"comment_count": blog.CommentCount,
		"create_time":   blog.CreateTime.Unix(),
		"update_time":   blog.UpdateTime.Unix(),
	}
	if err := dao.client.HSet(ctx, key, fields).Err(); err != nil {
		return err
	}
	if ttl > 0 {
		return dao.client.Expire(ctx, key, ttl).Err()
	}
	return nil
}

// DelBlogDetailCache 删除博客详情缓存
func (dao *RedisDAO) DelBlogDetailCache(ctx context.Context, blogId int64) error {
	key := fmt.Sprintf(utils.BlogDetailCacheKey, blogId)
	return dao.client.Del(ctx, key).Err()
}

// ========== 评论列表缓存 ==========
// GetHotCommentIds 获取热门评论ID列表
func (dao *RedisDAO) GetHotCommentIds(ctx context.Context, blogId int64, start, stop int64) ([]string, error) {
	key := fmt.Sprintf(utils.BlogHotCommentsKey, blogId)
	return dao.client.LRange(ctx, key, start, stop).Result()
}

// GetRecentCommentIds 获取最新评论ID列表
func (dao *RedisDAO) GetRecentCommentIds(ctx context.Context, blogId int64, start, stop int64) ([]string, error) {
	key := fmt.Sprintf(utils.BlogRecentCommentsKey, blogId)
	return dao.client.LRange(ctx, key, start, stop).Result()
}

// SetHotCommentIds 设置热门评论ID列表
func (dao *RedisDAO) SetHotCommentIds(ctx context.Context, blogId int64, ids []string, ttl time.Duration) error {
	key := fmt.Sprintf(utils.BlogHotCommentsKey, blogId)
	if err := dao.client.Del(ctx, key).Err(); err != nil {
		return err
	}
	if len(ids) > 0 {
		if err := dao.client.RPush(ctx, key, ids).Err(); err != nil {
			return err
		}
	}
	if ttl > 0 {
		return dao.client.Expire(ctx, key, ttl).Err()
	}
	return nil
}

// SetRecentCommentIds 设置最新评论ID列表
func (dao *RedisDAO) SetRecentCommentIds(ctx context.Context, blogId int64, ids []string, ttl time.Duration) error {
	key := fmt.Sprintf(utils.BlogRecentCommentsKey, blogId)
	if err := dao.client.Del(ctx, key).Err(); err != nil {
		return err
	}
	if len(ids) > 0 {
		if err := dao.client.RPush(ctx, key, ids).Err(); err != nil {
			return err
		}
	}
	if ttl > 0 {
		return dao.client.Expire(ctx, key, ttl).Err()
	}
	return nil
}

// DelCommentListCache 删除评论列表缓存
func (dao *RedisDAO) DelCommentListCache(ctx context.Context, blogId int64) error {
	hotKey := fmt.Sprintf(utils.BlogHotCommentsKey, blogId)
	recentKey := fmt.Sprintf(utils.BlogRecentCommentsKey, blogId)
	if err := dao.client.Del(ctx, hotKey).Err(); err != nil {
		return err
	}
	return dao.client.Del(ctx, recentKey).Err()
}
