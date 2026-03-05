package utils

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"
)

// Token相关Redis前缀
const (
	UserTokenPrefix = "xzdp:user:token:" // 用户Token Key：xzdp:user:token:{token}
	TokenExpireTime = 2 * time.Hour      // Token基础过期时间（2小时）
	RefreshExpire   = 7 * 24 * time.Hour // 刷新最大有效期（7天）
)

// GenerateCustomToken 生成32位随机自定义Token（无JWT，纯随机字符串）
func GenerateCustomToken() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// 降级：用时间戳+随机数生成
		return fmt.Sprintf("%d%06d", time.Now().Unix(), rand.Intn(999999))
	}
	return hex.EncodeToString(b)
}

// SetTokenToRedis 存储Token到Redis（关联用户ID+用户信息）
func SetTokenToRedis(token string, userId int64, userInfo map[string]interface{}) error {
	mainKey := UserTokenPrefix + token
	// 存储用户信息Hash
	for k, v := range userInfo {
		if err := RedisClient.HSet(ctx, mainKey, k, v).Err(); err != nil {
			return fmt.Errorf("redis hset failed: %v", err)
		}
	}
	// 设置主Token过期时间（2小时）
	if err := RedisClient.Expire(ctx, mainKey, TokenExpireTime).Err(); err != nil {
		return fmt.Errorf("redis expire failed: %v", err)
	}
	// 刷新Key（7天有效期，用于判断是否可刷新）
	refreshKey := UserTokenPrefix + "refresh:" + token
	RedisClient.Set(ctx, refreshKey, userId, RefreshExpire)
	return nil
}

// RefreshTokenExpire 刷新Token过期时间
func RefreshTokenExpire(token string) error {
	mainKey := UserTokenPrefix + token
	if RedisClient.Exists(ctx, mainKey).Val() == 0 {
		return fmt.Errorf("token expired")
	}
	return RedisClient.Expire(ctx, mainKey, TokenExpireTime).Err()
}

// GetUserInfoByToken 从Redis获取用户信息
func GetUserInfoByToken(token string) (map[string]interface{}, error) {
	mainKey := UserTokenPrefix + token
	// 检查Token是否存在
	if RedisClient.Exists(ctx, mainKey).Val() == 0 {
		return nil, fmt.Errorf("token invalid or expired")
	}
	// 获取Hash并转换类型
	strMap, err := RedisClient.HGetAll(ctx, mainKey).Result()
	if err != nil {
		return nil, fmt.Errorf("redis hgetall failed: %v", err)
	}
	// 转换为interface{} map
	infoMap := make(map[string]interface{}, len(strMap))
	for k, v := range strMap {
		infoMap[k] = v
	}
	return infoMap, nil
}

// DeleteToken 删除Token（登出）
func DeleteToken(token string) error {
	mainKey := UserTokenPrefix + token
	refreshKey := UserTokenPrefix + "refresh:" + token
	_, err := RedisClient.Del(ctx, mainKey, refreshKey).Result()
	return err
}

// GetUserIdByToken 从Token获取用户ID（简化版）
func GetUserIdByToken(token string) (int64, error) {
	info, err := GetUserInfoByToken(token)
	if err != nil {
		return 0, err
	}
	userIdStr, ok := info["id"].(string)
	if !ok {
		return 0, fmt.Errorf("user id not found in token info")
	}
	// 转换为int64
	var userId int64
	_, err = fmt.Sscanf(userIdStr, "%d", &userId)
	if err != nil {
		return 0, fmt.Errorf("user id format error: %v", err)
	}
	return userId, nil
}
