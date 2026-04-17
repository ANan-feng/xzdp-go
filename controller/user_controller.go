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

// 发送验证码接口：参数解析逻辑简化（工具函数内已做校验）
func (c *UserController) SendEmailCodeHandler(ctx *gin.Context) {
	// 直接传递 ctx 给业务层，参数解析交给工具函数
	email := ctx.Query("email")
	if err := c.userService.SendEmailCode(ctx.Request.Context(), email); err != nil {
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

// 邮箱登录接口
func (c *UserController) EmailLoginHandler(ctx *gin.Context) {
	// 1. 从 URL 查询参数 + POST 表单中获取参数（优先取 POST 表单，兼容 URL 参数）
	email := ctx.DefaultPostForm("email", ctx.Query("email"))
	code := ctx.DefaultPostForm("code", ctx.Query("code"))

	// 2. 调用 Service 层
	token, userDTO, err := c.userService.EmailLogin(ctx.Request.Context(), email, code)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	// 3. 返回结果
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "登录成功",
		"data": gin.H{
			"token": token,
			"user":  userDTO,
		},
	})
}

// GetUserInfoHandler 获取用户信息（需登录）
func (c *UserController) GetUserInfoHandler(ctx *gin.Context) {
	// 从Context获取用户ID（拦截器中已存入）
	userIdAny, exists := ctx.Get("userId")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "未登录",
		})
		return
	}

	// 类型断言：将 any 转为 int64（增加类型校验，避免断言失败 panic）
	userId, ok := userIdAny.(int64)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "用户ID类型错误",
		})
		return
	}

	// 查询用户信息
	user, err := c.userService.GetUserInfo(ctx, userId)
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
	if err := utils.DeleteToken(ctx, token); err != nil {
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
