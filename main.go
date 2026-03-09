package main

import (
	"context"
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

	//初始化Lua脚本缓存
	if err := utils.InitScriptCache(); err != nil {
		panic("init lua script cache failed: " + err.Error())
	}

	// 2. 业务缓存初始化（统一入口）
	ctx := context.Background()
	if err := utils.InitSeckillCouponCache(ctx); err != nil {
		panic("init seckill coupon cache failed: " + err.Error())
	}

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

	// 初始化商户控制器
	shopController := controller.NewShopController()

	// 商户路由（无需登录）
	shopGroup := r.Group("/shop")
	{
		shopGroup.GET("/:id", shopController.GetShopByIdHandler)        // 通用商户查询
		shopGroup.GET("/hot/:id", shopController.GetHotShopByIdHandler) // 热点商户查询
		shopGroup.GET("/type", shopController.ListShopByTypeHandler)    // 按类型分页查询
		shopGroup.POST("", shopController.UpdateShopHandler)            // 更新商户
	}

	// 初始化商户类型控制器
	shopTypeController := controller.NewShopTypeController()

	// 商户类型路由（无需登录）
	shopTypeGroup := r.Group("/shop-type")
	{
		shopTypeGroup.GET("/:id", shopTypeController.GetShopTypeByIdHandler)   // 根据ID查询
		shopTypeGroup.GET("", shopTypeController.ListAllShopTypesHandler)      // 查询所有
		shopTypeGroup.POST("", shopTypeController.CreateShopTypeHandler)       // 新增
		shopTypeGroup.PUT("", shopTypeController.UpdateShopTypeHandler)        // 更新
		shopTypeGroup.DELETE("/:id", shopTypeController.DeleteShopTypeHandler) // 删除
	}

	// 初始化秒杀控制器
	seckillController := controller.NewSeckillController()

	// 秒杀路由（需要登录）
	seckillGroup := r.Group("/seckill")
	seckillGroup.Use(middleware.LoginInterceptor())
	{
		seckillGroup.POST("/:couponId", seckillController.SeckillOrderHandler) // 秒杀下单
	}

	// 4. 启动服务
	r.Run(":8080") // 监听8080端口
}
