package controller

import (
	"net/http"

	"xzdp-go/service"

	"github.com/gin-gonic/gin"
)

// UserController 用户控制器（邮箱登录）
type UserController struct {
	userService *service.UserService
}

func NewUserController() *UserController {
	return &UserController{
		userService: service.NewUserService(),
	}
}

// SendEmailCodeHandler 发送邮箱验证码接口
// @Summary 发送邮箱验证码
// @Param email query string true "邮箱"
// @Success 200 {string} string "success"
// @Router /user/send-email [get]
func (c *UserController) SendEmailCodeHandler(ctx *gin.Context) {
	// 1. 获取参数（email替换phone）
	email := ctx.Query("email")
	if email == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "email is required",
		})
		return
	}

	// 2. 调用业务逻辑
	err := c.userService.SendEmailCode(email)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "send email failed: " + err.Error(),
		})
		return
	}

	// 3. 返回响应
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "send email code success",
	})
}

// EmailLoginHandler 邮箱登录接口
// @Summary 邮箱登录
// @Param email formData string true "邮箱"
// @Param code formData string true "验证码"
// @Success 200 {object} gin.H{"code":200,"msg":"success","data":{"token":"xxx","user":"xxx"}}
// @Router /user/email-login [post]
func (c *UserController) EmailLoginHandler(ctx *gin.Context) {
	// 1. 获取参数（email替换phone）
	var req struct {
		Email string `form:"email" binding:"required,email"` // 增加email格式校验
		Code  string `form:"code" binding:"required,len=6"`
	}
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid params: " + err.Error(),
		})
		return
	}

	// 2. 调用业务逻辑
	token, userId, err := c.userService.EmailLogin(req.Email, req.Code)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "login failed: " + err.Error(),
		})
		return
	}

	// 3. 查询用户信息（临时复用phone字段）
	user, err := c.userService.GetUserInfo(userId) // 简化版：实际需从token解析userId，这里先注释，后续完善
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "get user info failed: " + err.Error(),
		})
		return
	}

	// 4. 返回响应
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "login success",
		"data": gin.H{
			"token": token,
			"user":  user,
		},
	})
}
