package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims 自定义JWT载荷（包含用户ID、手机号）
type CustomClaims struct {
	UserId int64  `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT token（短信登录核心：生成登录凭证）
func GenerateToken(userId int64, email string) (string, error) {
	// 过期时间
	expireTime := time.Now().Add(time.Second * time.Duration(GetEnvInt("JWT_EXPIRE", 7200)))

	// 构建载荷
	claims := CustomClaims{
		UserId: userId,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "xzdp-go",
		},
	}

	// 生成token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

// GetEnvInt 读取环境变量（转int，带默认值）
func GetEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	// 简单转换（实习可优化：用strconv.Atoi）
	var num int
	_, err := fmt.Sscanf(value, "%d", &num)
	if err != nil {
		return defaultValue
	}
	return num
}
