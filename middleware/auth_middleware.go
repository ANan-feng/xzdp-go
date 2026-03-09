package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"xzdp-go/utils"

	"github.com/gin-gonic/gin"
)

// LoginInterceptor 登录校验拦截器（纯Redis Token）
func LoginInterceptor() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 1. 获取Token
		token := ctx.GetHeader("Authorization")
		// 兼容Bearer格式：Bearer xxxx
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}
		fmt.Printf("处理后的Token：%s\n", token) // 加日志
		if token == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "请先登录",
			})
			return
		}
		// 2. 校验Token（从Redis查询）
		userId, err := utils.GetUserIdByToken(token)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "登录态已过期，请重新登录",
			})
			return
		}
		// 3. 存入Context供后续使用
		ctx.Set("token", token)
		ctx.Set("userId", userId)
		// 放行
		ctx.Next()
	}
}

// TokenRefreshInterceptor Token刷新拦截器
func TokenRefreshInterceptor() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 仅登录校验通过后刷新
		token, exists := ctx.Get("token")
		if !exists {
			ctx.Next()
			return
		}
		// 刷新Token过期时间（延长2小时）
		_ = utils.RefreshTokenExpire(token.(string))
		ctx.Next()
	}
}
