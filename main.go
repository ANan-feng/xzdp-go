package main

import (
	"xzdp-go/controller"
	"xzdp-go/middleware"
	"xzdp-go/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 初始化数据库
	utils.InitDB()
	// 初始化Redis
	utils.InitRedis()

	// 2. 创建Gin引擎
	r := gin.Default()

	// 3. 初始化控制器
	userController := controller.NewUserController()

	// 白名单路由（无需登录）
	noAuthGroup := r.Group("/user")
	{
		noAuthGroup.GET("/send-email", userController.SendEmailCodeHandler) // 发送验证码
		noAuthGroup.POST("/email-login", userController.EmailLoginHandler)  // 登录
	}

	// 需要登录的路由（双重拦截器）
	authGroup := r.Group("/user")
	authGroup.Use(middleware.LoginInterceptor(), middleware.TokenRefreshInterceptor())
	{
		authGroup.GET("/info", userController.GetUserInfoHandler) // 获取用户信息
		authGroup.POST("/logout", userController.LogoutHandler)   // 登出
	}
	// 4. 启动服务
	r.Run(":8080") // 监听8080端口
}
