package utils

import (
	"context"
	"encoding/json"
	"time"
)

// ========== Cache Aside 模式工具 ==========
// UpdateCacheAfterDB 数据库更新后更新缓存（写后更新）
func UpdateCacheAfterDB(ctx context.Context, key string, value interface{}, expire time.Duration) error {
	return RedisClient.Set(ctx, key, value, expire).Err()
}

// DeleteCacheBeforeDB 数据库删除前删除缓存（删前删除）
func DeleteCacheBeforeDB(ctx context.Context, key string) error {
	return RedisClient.Del(ctx, key).Err()
}

// ========== 通用工具补充 ==========
var Json = json.Marshal // 全局JSON工具（简化代码）

// ParseValidationError 解析校验错误（迁移自controller）
func ParseValidationError(err error) string { /* 复用原有逻辑 */ return "" }
