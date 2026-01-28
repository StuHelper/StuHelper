package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/modules/auth"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/modules/course"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/health"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/redis"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		// 使用标准库 log 输出错误，因为此时 logger 可能未初始化或已关闭
		fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 初始化日志系统
	logCfg := logger.Config{
		Level:           cfg.Log.Level,
		Format:          cfg.Log.Format,
		Output:          cfg.Log.Output,
		SamplingEnabled: cfg.Log.SamplingEnabled,
		SamplingInitial: cfg.Log.SamplingInitial,
		SamplingAfter:   cfg.Log.SamplingAfter,
		FileEnabled:     cfg.Log.FileEnabled,
		FilePath:        cfg.Log.FilePath,
		FileMaxSize:     cfg.Log.FileMaxSize,
		FileMaxBackups:  cfg.Log.FileMaxBackups,
		FileMaxAge:      cfg.Log.FileMaxAge,
		FileCompress:    cfg.Log.FileCompress,
	}
	if err := logger.Init(logCfg); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()

	// 初始化 HMAC 密钥（用于用户 ID 哈希等场景）
	crypto.InitHMACKey(cfg.App.HMACSecret)

	// 初始化 Redis 客户端
	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	defer func() { _ = redisClient.Close() }()

	// 初始化 PostgreSQL 连接池
	pgPool, err := db.NewPGPool(cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to Postgres: %w", err)
	}
	defer pgPool.Close()

	// 创建带超时的数据库封装
	database := db.NewDB(pgPool, time.Duration(cfg.Database.QueryTimeout)*time.Second)

	// 初始化 Token 服务
	tokenService := token.NewService(
		redisClient.GetClient(),
		cfg.Token.AccessTokenTTL,
		cfg.Token.RefreshTokenTTL,
	)

	// 根据环境设置 Gin 模式
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建 Gin 路由
	r := gin.New()

	// 配置可信代理列表，防止 IP 欺骗
	if len(cfg.App.TrustedProxies) > 0 {
		if err := r.SetTrustedProxies(cfg.App.TrustedProxies); err != nil {
			return fmt.Errorf("failed to set trusted proxies: %w", err)
		}
	} else {
		// 开发环境：信任所有代理（不推荐用于生产）
		_ = r.SetTrustedProxies(nil)
	}

	r.Use(middleware.Recovery())
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.RequestLogger())
	// 根据环境选择安全头中间件
	if cfg.App.Env == "production" {
		r.Use(middleware.SecurityHeadersWithHSTS())
	} else {
		r.Use(middleware.SecurityHeadersMiddleware())
	}
	r.Use(middleware.MaxBodySize(cfg.App.MaxBodySize)) // 限制请求体大小

	// 配置 CORS
	corsOrigins := cfg.App.CORSOrigins
	for _, origin := range corsOrigins {
		if strings.TrimSpace(origin) == "*" {
			return errors.New("CORS configuration error: wildcard '*' is not allowed when AllowCredentials is true")
		}
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-CSRF-Token", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 注册健康检查端点
	healthHandler := health.NewHandler(pgPool, redisClient.GetClient(), health.BuildInfo{
		Version:   "1.0.0",
		GitCommit: "unknown",
		BuildTime: "unknown",
	})
	healthHandler.RegisterRoutes(r)

	// 注册 API 路由（带版本控制）
	api := r.Group("/api/v1")
	{
		api.Use(middleware.CSRFMiddleware())

		// 注册认证模块路由
		authHandler := auth.NewHandler(cfg, tokenService, redisClient.GetClient())
		authHandler.RegisterRoutes(api)

		// 注册课程模块路由
		courseHandler := course.NewHandler(database, redisClient.GetClient())
		courseHandler.RegisterRoutes(api, middleware.AuthMiddleware(tokenService))
	}

	// 启动服务器
	srv := &http.Server{
		Addr:              ":" + cfg.App.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// 用于接收服务器错误的 channel
	serverErr := make(chan error, 1)

	// 在 goroutine 中启动服务器
	go func() {
		logger.L().Info("Server starting", zap.String("port", cfg.App.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("failed to start server: %w", err)
		}
	}()

	// 等待中断信号或服务器错误
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.L().Info("Shutting down server...")
	case err := <-serverErr:
		return err
	}

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.L().Error("Server forced to shutdown", zap.Error(err))
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	logger.L().Info("Server exited")
	return nil
}
