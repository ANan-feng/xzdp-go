package main

import (
	"xzdp-go/controller"
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

	// 3. 注册路由（替换为邮箱接口）
	userController := controller.NewUserController()
	userGroup := r.Group("/user")
	{
		userGroup.GET("/send-email", userController.SendEmailCodeHandler) // 发送邮箱验证码
		userGroup.POST("/email-login", userController.EmailLoginHandler)  // 邮箱登录
	}

	// 4. 启动服务
	r.Run(":8080") // 监听8080端口
}
