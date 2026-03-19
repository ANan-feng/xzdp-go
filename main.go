package main

import (
	"context"
	"os" // 新增：读取环境变量
	"runtime"
	"strconv" // 新增：类型转换
	"xzdp-go/controller"
	"xzdp-go/middleware"
	"xzdp-go/service"
	"xzdp-go/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv" // 需先安装：go get github.com/joho/godotenv
)

func main() {

	// 新增：开启Gin多核模式
	gin.SetMode(gin.ReleaseMode)         // 生产环境建议开启release模式
	runtime.GOMAXPROCS(runtime.NumCPU()) // 利用所有CPU核心

	// 新增：加载.env文件
	err := godotenv.Load()
	if err != nil {
		panic("load .env file failed: " + err.Error())
	}

	// 1. 初始化组件
	utils.InitDB()
	utils.InitRedis()

	// 初始化Kafka
	utils.InitKafka()
	defer utils.CloseKafka()

	// 启动Kafka秒杀消费者（后台运行）
	seckillConsumer := service.NewSeckillConsumer()
	go seckillConsumer.StartConsume()

	// 初始化Lua脚本缓存
	if err := utils.InitScriptCache(); err != nil {
		panic("init lua script cache failed: " + err.Error())
	}

	// 业务缓存初始化
	ctx := context.Background()
	if err := utils.InitSeckillCouponCache(ctx); err != nil {
		panic("init seckill coupon cache failed: " + err.Error())
	}

	// 2. 创建Gin引擎
	r := gin.Default()

	// 3. 初始化控制器
	userController := controller.NewUserController()
	shopController := controller.NewShopController()
	shopTypeController := controller.NewShopTypeController()
	seckillController := controller.NewSeckillController() // 秒杀控制器

	// ========== 白名单路由（无需登录） ==========
	// 用户相关
	noAuthGroup := r.Group("/user")
	{
		noAuthGroup.GET("/send-email", userController.SendEmailCodeHandler) // 发送验证码
		noAuthGroup.POST("/email-login", userController.EmailLoginHandler)  // 登录
	}

	// 商户相关
	shopGroup := r.Group("/shop")
	{
		shopGroup.GET("/:id", shopController.GetShopByIdHandler)        // 通用商户查询
		shopGroup.GET("/hot/:id", shopController.GetHotShopByIdHandler) // 热点商户查询
		shopGroup.GET("/type", shopController.ListShopByTypeHandler)    // 按类型分页查询
		shopGroup.POST("", shopController.UpdateShopHandler)            // 更新商户
	}

	// 商户类型相关
	shopTypeGroup := r.Group("/shop-type")
	{
		shopTypeGroup.GET("/:id", shopTypeController.GetShopTypeByIdHandler)   // 根据ID查询
		shopTypeGroup.GET("", shopTypeController.ListAllShopTypesHandler)      // 查询所有
		shopTypeGroup.POST("", shopTypeController.CreateShopTypeHandler)       // 新增
		shopTypeGroup.PUT("", shopTypeController.UpdateShopTypeHandler)        // 更新
		shopTypeGroup.DELETE("/:id", shopTypeController.DeleteShopTypeHandler) // 删除
	}

	// 优惠券添加接口（无需登录，可根据实际需求添加权限校验）
	voucherGroup := r.Group("/voucher")
	{
		voucherGroup.POST("/add", seckillController.AddVoucher) // 注册添加优惠券接口
	}

	// ========== 需要登录的路由 ==========
	// 用户相关
	authGroup := r.Group("/user")
	authGroup.Use(middleware.LoginInterceptor(), middleware.TokenRefreshInterceptor())
	{
		authGroup.GET("/info", userController.GetUserInfoHandler) // 获取用户信息
		authGroup.POST("/logout", userController.LogoutHandler)   // 登出
	}

	// 秒杀相关
	seckillGroup := r.Group("/seckill")
	seckillGroup.Use(middleware.LoginInterceptor()) // 登录校验
	{
		seckillGroup.POST("/:couponId", seckillController.SeckillOrderHandler) // 秒杀下单
	}

	// 4. 启动服务（读取.env中的端口）
	portStr := os.Getenv("SERVER_PORT")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		panic("invalid SERVER_PORT in .env: " + err.Error())
	}
	addr := ":" + strconv.Itoa(port)
	if err := r.Run(addr); err != nil {
		panic("server start failed: " + err.Error())
	}
}
