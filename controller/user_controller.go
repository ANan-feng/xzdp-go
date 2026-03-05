package controller

import (
	"net/http"

	"xzdp-go/service"
	"xzdp-go/utils"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userService *service.UserService
}

func NewUserController() *UserController {
	return &UserController{
		userService: service.NewUserService(),
	}
}

// SendEmailCodeHandler 发送邮箱验证码
func (c *UserController) SendEmailCodeHandler(ctx *gin.Context) {
	email := ctx.Query("email")
	if email == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "邮箱不能为空",
		})
		return
	}
	// 调用业务逻辑
	if err := c.userService.SendEmailCode(email); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "验证码已发送，请注意查收",
	})
}

// EmailLoginHandler 邮箱登录（返回Redis Token）
func (c *UserController) EmailLoginHandler(ctx *gin.Context) {
	// 1. 参数校验
	var req struct {
		Email string `form:"email" binding:"required,email"`
		Code  string `form:"code" binding:"required,len=6"`
	}
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "参数错误：" + err.Error(),
		})
		return
	}
	// 2. 登录获取用户ID
	token, userId, err := c.userService.EmailLogin(req.Email, req.Code)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}
	// 3. 查询用户信息（脱敏）
	user, err := c.userService.GetUserInfo(userId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "获取用户信息失败：" + err.Error(),
		})
		return
	}
	// 5. 构建脱敏用户信息
	userInfo := map[string]interface{}{
		"id":       user.Id,
		"nickname": user.Nickname,
		"avatar":   user.Avatar,
	}
	// 6. 存储Token到Redis
	if err := utils.SetTokenToRedis(token, userId, userInfo); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "登录失败：" + err.Error(),
		})
		return
	}
	// 7. 返回响应（仅Redis Token，无JWT）
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "登录成功",
		"data": gin.H{
			"token": token,    // 前端存储此Token，后续请求携带
			"user":  userInfo, // 脱敏用户信息
		},
	})
}

// GetUserInfoHandler 获取用户信息（需登录）
func (c *UserController) GetUserInfoHandler(ctx *gin.Context) {
	// 从Context获取用户ID（拦截器中已存入）
	userId, exists := ctx.Get("userId")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "未登录",
		})
		return
	}
	// 查询用户信息
	user, err := c.userService.GetUserInfo(userId.(int64))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "获取用户信息失败",
		})
		return
	}
	// 脱敏返回
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": gin.H{
			"id":       user.Id,
			"nickname": user.Nickname,
			"avatar":   user.Avatar,
		},
	})
}

// LogoutHandler 登出（删除Redis Token）
func (c *UserController) LogoutHandler(ctx *gin.Context) {
	// 从Header获取Token
	token := ctx.GetHeader("Authorization")
	if token == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "Token不能为空",
		})
		return
	}
	// 删除Token
	if err := utils.DeleteToken(token); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "登出失败：" + err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "登出成功",
	})
}
