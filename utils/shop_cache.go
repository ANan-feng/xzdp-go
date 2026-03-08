package utils

import (
	"context"
	"encoding/json"
	"math/rand"
	"time"
	"xzdp-go/model"

	"github.com/go-redis/redis/v8"
)

// 商户缓存常量
const (
	ShopCachePrefix     = "xzdp:shop:"     // 商户缓存Key前缀
	ShopCacheBasicTTL   = 30 * time.Minute // 基础过期时间30分钟
	ShopCacheRandTTLMax = 10 * time.Minute // 随机过期时间最大值10分钟
	ShopNullCacheTTL    = 5 * time.Minute  // 空对象缓存过期时间5分钟
)

// 逻辑过期缓存结构体（热点Key专用）
type ShopCacheWithExpire struct {
	Shop   *model.Shop `json:"shop"`
	Expire int64       `json:"expire"` // 逻辑过期时间戳（秒）
}

// GetShopByIdWithCache 缓存穿透+缓存雪崩+Cache Aside
func GetShopByIdWithCache(ctx context.Context, shopId int64) (*model.Shop, error) {
	// 1. 构建缓存Key
	cacheKey := ShopCachePrefix + string(rune(shopId))

	// 2. 先查缓存
	shopJson, err := RedisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		// 2.1 缓存命中：判断是否是空对象
		if shopJson == "null" {
			return nil, nil // 空对象，返回nil
		}
		// 2.2 反序列化返回
		var shop model.Shop
		if err := json.Unmarshal([]byte(shopJson), &shop); err != nil {
			return nil, err
		}
		return &shop, nil
	}
	// 缓存未命中（非redis.ErrNil错误则直接返回）
	if err != redis.Nil {
		return nil, err
	}

	// 3. 缓存未命中：查数据库
	var shop model.Shop
	if err := DB.WithContext(ctx).Where("id = ?", shopId).First(&shop).Error; err != nil {
		// 3.1 数据库也不存在：缓存空对象（解决穿透）
		if err.Error() == "record not found" {
			// 存储空对象到缓存
			if err := RedisClient.Set(ctx, cacheKey, "null", ShopNullCacheTTL).Err(); err != nil {
				return nil, err
			}
			return nil, nil
		}
		return nil, err
	}

	// 3.2 数据库存在：回写缓存（Cache Aside）+ 防雪崩（TTL加随机值）
	shopBytes, err := json.Marshal(&shop)
	if err != nil {
		return nil, err
	}
	// 计算带随机值的TTL
	randTTL := time.Duration(rand.Int63n(int64(ShopCacheRandTTLMax)))
	ttl := ShopCacheBasicTTL + randTTL
	if err := RedisClient.Set(ctx, cacheKey, shopBytes, ttl).Err(); err != nil {
		return nil, err
	}

	return &shop, nil
}

// UpdateShopWithCache Cache Aside模式：更新数据库后删除缓存
func UpdateShopWithCache(ctx context.Context, shop *model.Shop) error {
	// 1. 先更新数据库
	if err := DB.WithContext(ctx).Save(shop).Error; err != nil {
		return err
	}

	// 2. 再删除缓存（避免更新缓存的并发问题）
	cacheKey := ShopCachePrefix + string(rune(shop.ID))
	if err := RedisClient.Del(ctx, cacheKey).Err(); err != nil {
		return err
	}

	return nil
}

// GetShopByIdWithLogicalExpire 热点Key逻辑过期方案（防缓存雪崩）
// 适用场景：高频访问的商户（如热门店铺），牺牲一致性换性能
func GetShopByIdWithLogicalExpire(ctx context.Context, shopId int64) (*model.Shop, error) {
	cacheKey := ShopCachePrefix + "hot:" + string(rune(shopId))

	// 1. 查缓存
	cacheBytes, err := RedisClient.Get(ctx, cacheKey).Result()
	if err != nil {
		if err == redis.Nil {
			// 缓存未命中：走数据库查询并异步更新缓存
			return loadShopAndSetLogicalExpire(ctx, shopId)
		}
		return nil, err
	}

	// 2. 解析逻辑过期缓存
	var cacheData ShopCacheWithExpire
	if err := json.Unmarshal([]byte(cacheBytes), &cacheData); err != nil {
		return nil, err
	}

	// 3. 判断是否过期
	now := time.Now().Unix()
	if cacheData.Expire > now {
		// 未过期：直接返回
		return cacheData.Shop, nil
	}

	// 4. 已过期：异步更新缓存，当前请求返回旧数据（避免阻塞）
	go func() {
		loadShopAndSetLogicalExpire(ctx, shopId)
	}()

	return cacheData.Shop, nil
}

// loadShopAndSetLogicalExpire 加载商户并设置逻辑过期缓存
func loadShopAndSetLogicalExpire(ctx context.Context, shopId int64) (*model.Shop, error) {
	// 1. 查数据库
	var shop model.Shop
	if err := DB.WithContext(ctx).Where("id = ?", shopId).First(&shop).Error; err != nil {
		return nil, err
	}

	// 2. 设置逻辑过期（基础TTL+随机值）
	randTTL := time.Duration(rand.Int63n(int64(ShopCacheRandTTLMax)))
	expireTime := time.Now().Add(ShopCacheBasicTTL + randTTL).Unix()
	cacheData := ShopCacheWithExpire{
		Shop:   &shop,
		Expire: expireTime,
	}
	cacheBytes, err := json.Marshal(&cacheData)
	if err != nil {
		return nil, err
	}

	// 3. 写入缓存（永不过期，靠逻辑过期控制）
	cacheKey := ShopCachePrefix + "hot:" + string(rune(shopId))
	if err := RedisClient.Set(ctx, cacheKey, cacheBytes, 0).Err(); err != nil {
		return nil, err
	}

	return &shop, nil
}
