package middleware

import (
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
		// 1. 从 Redis 获取用户信息（不仅仅是 ID，而是完整 DTO）
		//    我们需要在 utils/token.go 里加一个 GetUserDTOByToken 方法
		userDTO, err := utils.GetUserDTOByToken(ctx.Request.Context(), token)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"msg": "请登录"})
			return
		}

		// 2. 存入 Context，后续的 Handler 直接用 c.Get("user") 拿到全部信息，不用再查库！
		ctx.Set("user", userDTO)
		ctx.Set("userId", userDTO.ID)

		// 3. 刷新 Token 有效期（续期）
		_ = utils.RefreshTokenExpire(ctx.Request.Context(), token)

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
		_ = utils.RefreshTokenExpire(ctx.Request.Context(), token.(string))
		ctx.Next()
	}
}
