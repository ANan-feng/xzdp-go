package utils

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strconv"
	"time"
	"xzdp-go/model"
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
func SetTokenToRedis(ctx context.Context, token string, userId int64, dto *model.UserDTO) error {
	mainKey := UserTokenPrefix + token
	// 存储用户信息Hash
	// 转为 map 存入 Redis Hash
	data := map[string]interface{}{
		"id":       dto.ID,
		"nickname": dto.Nickname,
		"avatar":   dto.Avatar,
	}
	for k, v := range data {
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
func RefreshTokenExpire(ctx context.Context, token string) error {
	mainKey := UserTokenPrefix + token
	if RedisClient.Exists(ctx, mainKey).Val() == 0 {
		return fmt.Errorf("token expired")
	}
	return RedisClient.Expire(ctx, mainKey, TokenExpireTime).Err()
}

// GetUserInfoByToken 从Redis获取用户信息
func GetUserInfoByToken(ctx context.Context, token string) (map[string]interface{}, error) {
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
func DeleteToken(ctx context.Context, token string) error {
	mainKey := UserTokenPrefix + token
	refreshKey := UserTokenPrefix + "refresh:" + token
	_, err := RedisClient.Del(ctx, mainKey, refreshKey).Result()
	return err
}

// GetUserIdByToken 从Token获取用户ID（简化版）
func GetUserIdByToken(ctx context.Context, token string) (int64, error) {
	info, err := GetUserInfoByToken(ctx, token)
	if err != nil {
		return 0, err
	}
	userIdStr, ok := info["id"].(string)
	if !ok {
		return 0, fmt.Errorf("user id not found in token info")
	}
	// 转换为int64
	userId, err := strconv.ParseInt(userIdStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("user id format error: %v", err)
	}
	return userId, nil
}
func GetUserDTOByToken(ctx context.Context, token string) (*model.UserDTO, error) {
	info, err := GetUserInfoByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	userIdStr, ok := info["id"].(string)
	if !ok {
		return nil, fmt.Errorf("user id not found in token info")
	}
	userId, err := strconv.ParseInt(userIdStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("user id format error: %v", err)
	}
	nickname, _ := info["nickname"].(string)
	avatar, _ := info["avatar"].(string)
	return &model.UserDTO{
		ID:       userId,
		Nickname: nickname,
		Avatar:   avatar,
	}, nil
}
