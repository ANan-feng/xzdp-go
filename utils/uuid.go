// utils/uuid.go
package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateUUID 生成唯一UUID（用于分布式锁value）
func GenerateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic("生成UUID失败: " + err.Error())
	}
	return hex.EncodeToString(b)
}
