package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"y-ui/internal/config"
	"y-ui/internal/database"
	"y-ui/internal/handlers"
	"y-ui/internal/middleware"
	"y-ui/internal/scheduler"
	"y-ui/internal/services"
	"y-ui/internal/xray"

	"github.com/gin-gonic/gin"
)

var (
	configPath = flag.String("config", "config.yaml", "配置文件路径")
	Version    = "1.4.0" // 通过 -ldflags 注入
)

func main() {
	flag.Parse()

	// 加载配置（自动生成安全的 JWT 密钥）
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("错误: 无法加载配置文件 %s: %v\n", *configPath, err)
		fmt.Println("请确保配置文件存在且格式正确")
		os.Exit(1)
	}

	// 初始化日志
	if err := middleware.InitLogger(cfg.Log.Level, cfg.Log.Output); err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
	}

	// 初始化数据库
	if err := database.Init(&cfg.Database); err != nil {
		fmt.Printf("初始化数据库失败: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// 创建默认出站
	services.NewOutboundService().CreateDefaultOutbounds()

	// 初始化 Xray 管理器
	xrayManager := xray.NewManager(&cfg.Xray)

	// 启动 Xray（带重试）
	var xrayStartErr error
	for i := 0; i < 3; i++ {
		xrayStartErr = xrayManager.Start()
		if xrayStartErr == nil {
			fmt.Println("Xray 启动成功")
			break
		}
		fmt.Printf("Xray 启动失败 (尝试 %d/3): %v\n", i+1, xrayStartErr)
		time.Sleep(time.Second)
	}
	if xrayStartErr != nil {
		fmt.Printf("Xray 启动失败，请通过界面手动启动: %v\n", xrayStartErr)
	}

	// 初始化调度器
	sched := scheduler.NewScheduler(xrayManager, cfg.Database.DSN)
	sched.Start()
	defer sched.Stop()

	// 设置 Gin 模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	router := setupRouter(xrayManager)

	// 启动服务器
	srv := &http.Server{
		Addr:    cfg.Server.Addr,
		Handler: router,
	}

	// 优雅关闭
	go func() {
		fmt.Printf("Y-UI %s 启动成功，监听: %s\n", Version, cfg.Server.Addr)
		fmt.Println("管理命令: y-ui")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("服务器错误: %v\n", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("服务器关闭错误: %v\n", err)
	}

	// 停止 Xray
	xrayManager.Stop()

	fmt.Println("服务器已关闭")
}

func setupRouter(xrayManager *xray.Manager) *gin.Engine {
	router := gin.New()

	// 中间件
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.CORS())
	router.Use(middleware.Security())

	// 静态文件（前端）
	router.Static("/assets", "./dist/assets")
	router.StaticFile("/", "./dist/index.html")
	router.NoRoute(func(c *gin.Context) {
		c.File("./dist/index.html")
	})

	// API 路由
	api := router.Group("/api/v1")

	// 处理器
	authHandler := handlers.NewAuthHandler()
	userHandler := handlers.NewUserHandler()
	clientHandler := handlers.NewClientHandler()
	inboundHandler := handlers.NewInboundHandler(xrayManager)
	outboundHandler := handlers.NewOutboundHandler(xrayManager)
	trafficHandler := handlers.NewTrafficHandler()
	auditHandler := handlers.NewAuditHandler()
	systemHandler := handlers.NewSystemHandler(xrayManager)
	certHandler := handlers.NewCertificateHandler()
	subHandler := handlers.NewSubscriptionHandler()

	// 公开路由 - 添加速率限制防止暴力破解
	loginRateLimit := middleware.RateLimitLogin(5, 1*time.Minute) // 每分钟最多5次
	api.GET("/auth/check", authHandler.CheckInit)
	api.POST("/auth/login", loginRateLimit, authHandler.Login)
	api.POST("/auth/init", middleware.InitAdminProtection(), authHandler.InitAdmin)

	// 订阅链接 (公开，通过 UUID 验证，添加速率限制防止枚举)
	subRateLimit := middleware.RateLimit(30, 1*time.Minute) // 每分钟最多30次
	api.GET("/sub/:uuid", subRateLimit, subHandler.GetSubscription)

	// 需要认证的路由
	auth := api.Group("")
	auth.Use(middleware.JWTAuth())
	{
		// 认证相关
		auth.POST("/auth/logout", authHandler.Logout)
		auth.GET("/auth/profile", authHandler.GetProfile)
		auth.PUT("/auth/password", authHandler.ChangePassword)

		// 用户管理 (仅管理员)
		users := auth.Group("/users")
		users.Use(middleware.RequireAdmin())
		{
			users.GET("", userHandler.List)
			users.GET("/:id", userHandler.Get)
			users.POST("", userHandler.Create)
			users.PUT("/:id", userHandler.Update)
			users.DELETE("/:id", userHandler.Delete)
		}

		// 客户端管理 (操作员以上)
		clients := auth.Group("/clients")
		clients.Use(middleware.RequireOperator())
		{
			clients.GET("", clientHandler.List)
			clients.GET("/:id", clientHandler.Get)
			clients.POST("", clientHandler.Create)
			clients.PUT("/:id", clientHandler.Update)
			clients.DELETE("/:id", clientHandler.Delete)
			clients.POST("/:id/reset-traffic", clientHandler.ResetTraffic)
			clients.GET("/:id/links", subHandler.GetClientLinks)
		}

		// 入站管理 (操作员以上)
		inbounds := auth.Group("/inbounds")
		inbounds.Use(middleware.RequireOperator())
		{
			inbounds.GET("", inboundHandler.List)
			inbounds.GET("/:id", inboundHandler.Get)
			inbounds.POST("", inboundHandler.Create)
			inbounds.PUT("/:id", inboundHandler.Update)
			inbounds.DELETE("/:id", inboundHandler.Delete)
			inbounds.GET("/:id/clients", inboundHandler.GetClients)
			inbounds.POST("/:id/clients", inboundHandler.AddClient)
			inbounds.DELETE("/:id/clients/:client_id", inboundHandler.RemoveClient)
		}

		// 出站管理 (仅管理员)
		outbounds := auth.Group("/outbounds")
		outbounds.Use(middleware.RequireAdmin())
		{
			outbounds.GET("", outboundHandler.List)
			outbounds.GET("/:id", outboundHandler.Get)
			outbounds.POST("", outboundHandler.Create)
			outbounds.PUT("/:id", outboundHandler.Update)
			outbounds.DELETE("/:id", outboundHandler.Delete)
		}

		// 流量统计
		stats := auth.Group("/stats")
		{
			stats.GET("/summary", trafficHandler.GetSummary)
			stats.GET("/daily", trafficHandler.GetDailyStats)
			stats.GET("/client/:id", trafficHandler.GetClientTraffic)
			stats.GET("/inbound/:id", trafficHandler.GetInboundTraffic)
		}

		// 证书管理 (仅管理员)
		certs := auth.Group("/certificates")
		certs.Use(middleware.RequireAdmin())
		{
			certs.GET("", certHandler.List)
			certs.GET("/:id", certHandler.Get)
			certs.POST("", certHandler.Request)
			certs.POST("/:id/renew", certHandler.Renew)
			certs.PUT("/:id/auto-renew", certHandler.UpdateAutoRenew)
			certs.DELETE("/:id", certHandler.Delete)
		}

		// 系统管理
		system := auth.Group("/system")
		{
			system.GET("/status", systemHandler.GetStatus)
			system.GET("/check-port", systemHandler.CheckPort)
			system.GET("/check-update", systemHandler.CheckUpdate)
			system.POST("/reload", middleware.RequireAdmin(), systemHandler.ReloadXray)
			system.POST("/restart", middleware.RequireAdmin(), systemHandler.RestartXray)
			system.GET("/config", middleware.RequireAdmin(), systemHandler.GetXrayConfig)
		}

		// 审计日志 (仅管理员)
		audits := auth.Group("/audits")
		audits.Use(middleware.RequireAdmin())
		{
			audits.GET("", auditHandler.List)
		}
	}

	return router
}
