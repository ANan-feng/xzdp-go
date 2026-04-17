package main

import (
	"context"
	"fmt"
	"log"
	"os" // 新增：读取环境变量
	"runtime"
	"strconv" // 新增：类型转换
	"strings"
	"time"
	"xzdp-go/consumer"
	"xzdp-go/controller"
	"xzdp-go/middleware"
	"xzdp-go/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv" // 需先安装：go get github.com/joho/godotenv
)

func main() {
	fmt.Println("=== 项目启动开始 ===")
	// 开启Gin多核模式
	gin.SetMode(gin.ReleaseMode)         // 生产环境建议开启release模式
	runtime.GOMAXPROCS(runtime.NumCPU()) // 利用所有CPU核心
	fmt.Println("✅ Gin 模式设置完成")

	// 加载.env文件
	err := godotenv.Load()
	if err != nil {
		fmt.Println("❌ 加载 .env 失败：", err.Error())
		panic("load .env file failed: " + err.Error())
	}
	fmt.Println("✅ .env 配置加载完成")

	// 1. 初始化组件
	fmt.Println("开始初始化 DB...")
	utils.InitDB()
	fmt.Println("✅ DB 初始化完成")

	fmt.Println("开始初始化 Redis...")
	utils.InitRedis()
	fmt.Println("✅ Redis 初始化完成")

	// 初始化ID生成器（必须在Redis初始化之后）
	utils.InitIDGenerator(1) // 测试环境机器ID=1，生产环境可从env读取

	// 初始化Lua脚本缓存
	fmt.Println("开始初始化 Lua 脚本...")
	if err := utils.InitScriptCache(); err != nil {
		fmt.Println("❌ Lua 脚本初始化失败：", err.Error())
		panic("init lua script cache failed")
	}
	fmt.Println("✅ Lua 脚本初始化完成")

	// 业务缓存初始化（加超时，防止无限卡住）
	fmt.Println("开始初始化秒杀优惠券缓存...")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := utils.InitSeckillCouponCache(ctx); err != nil {
		fmt.Println("⚠️ 缓存初始化失败（不影响启动）：", err.Error())
		// 不 panic，只打印警告，让项目能继续启动
	} else {
		fmt.Println("✅ 秒杀优惠券缓存初始化完成")
	}

	// 初始化 Stream 消费者组
	err = utils.InitStreamGroup(context.Background(), utils.RedisClient)
	if err != nil {
		// 如果组已经存在，不报错
		if strings.Contains(err.Error(), "BUSYGROUP") {
			println("✅ 消费者组已存在")
		} else {
			log.Fatalf("创建消费者组失败: %v", err)
		}
	}
	// 启动消费者
	consumer := consumer.NewSeckillConsumer()
	go consumer.Start(context.Background())

	fmt.Println("=== 所有初始化完成，启动 Gin 服务 ===")

	// 2. 创建Gin引擎
	r := gin.Default()
	fmt.Println("✅ Gin 引擎创建成功")

	// 3. 初始化控制器
	userController := controller.NewUserController()
	shopController := controller.NewShopController()
	shopTypeController := controller.NewShopTypeController()
	seckillController := controller.NewSeckillController() // 秒杀控制器

	// ========== 用户路由：只定义一次 /user 组！==========
	userGroup := r.Group("/user") // 只写这一次！
	{
		// 👇 公开接口（不用登录）
		userGroup.GET("/send-email", userController.SendEmailCodeHandler)
		userGroup.POST("/email-login", userController.EmailLoginHandler)

		// 👇 下面这些才需要登录（单独套拦截器）
		auth := userGroup.Group("/") // 继承 /user 前缀
		auth.Use(middleware.LoginInterceptor(), middleware.TokenRefreshInterceptor())
		auth.GET("/info", userController.GetUserInfoHandler)
		auth.POST("/logout", userController.LogoutHandler)
	}

	// ========== 白名单路由（无需登录） ==========

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

	// 秒杀相关
	seckillGroup := r.Group("/seckill")
	seckillGroup.Use(middleware.LoginInterceptor(), middleware.TokenRefreshInterceptor()) // 登录校验
	{
		seckillGroup.POST("/:couponId", seckillController.SeckillOrderHandler)            // 秒杀下单
		seckillGroup.GET("/result/:orderId", seckillController.QuerySeckillResultHandler) // 查询秒杀结果
	}

	// 4. 启动服务
	portStr := os.Getenv("SERVER_PORT")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Println("❌ 端口配置错误，使用默认 8080")
		port = 8080
	}
	addr := ":" + strconv.Itoa(port)

	fmt.Printf("🚀 服务启动成功，监听地址：%s\n", addr)
	fmt.Println("=== 启动完成，可以访问接口 ===")

	if err := r.Run(addr); err != nil {
		fmt.Println("❌ 服务启动失败：", err.Error())
		panic("server start failed")
	}
}
