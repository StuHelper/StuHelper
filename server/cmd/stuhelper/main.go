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

	"github.com/joho/godotenv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	apidocs "git.stuhelper.com/StuHelper/StuHelper/api"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/auth"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/course"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/ldap"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/rbac"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/health"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/redis"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/sso"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

// 构建信息，通过 -ldflags 注入
var (
	version   = "dev"
	gitCommit = "unknown"
	buildTime = "unknown"
)

func main() {
	if err := run(); err != nil {
		// 使用标准库 log 输出错误，因为此时 logger 可能未初始化或已关闭
		fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// 仅在非生产环境加载 .env 文件，生产环境由 docker-compose 注入环境变量
	if os.Getenv("APP_ENV") != "production" && os.Getenv("GIN_MODE") != "release" {
		if err := godotenv.Load("../.env"); err != nil {
			fmt.Fprintf(os.Stderr, "debug: .env not loaded: %v\n", err)
		}
	}

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Logger 必须在 config 之后初始化：日志级别/格式/输出目标由配置决定，
	// 因此 config.Load() 阶段的错误只能输出到 stderr。
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
	logger.Init(logCfg)
	defer func() {
		if err := logger.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "logger close error: %v\n", err)
		}
	}()

	// cleanups 累积已初始化资源的清理函数，初始化失败时按逆序释放，防止资源泄漏
	var cleanups []func()
	runCleanups := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}

	// 初始化 HMAC 密钥（用于用户 ID 哈希等场景）
	isProduction := cfg.App.Env == "production"
	if err := crypto.InitHMACKey(cfg.App.HMACSecret, isProduction); err != nil {
		return fmt.Errorf("failed to initialize HMAC key: %w", err)
	}

	// 初始化 Redis 客户端
	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	cleanups = append(cleanups, func() {
		if err := redisClient.Close(); err != nil {
			logger.L().Warn("redis client close error", zap.Error(err))
		}
	})

	// 初始化 PostgreSQL 连接池
	pgPool, err := db.NewPGPool(cfg.Database)
	if err != nil {
		runCleanups()
		return fmt.Errorf("failed to connect to Postgres: %w", err)
	}
	cleanups = append(cleanups, func() {
		pgPool.Close()
	})

	// 创建带超时的数据库封装
	database := db.NewDB(pgPool, time.Duration(cfg.Database.QueryTimeout)*time.Second)

	// 初始化 Token 服务（包含增强的 JWT 验证器）
	tokenService, err := token.NewService(token.ServiceConfig{
		RedisClient:    redisClient.GetClient(),
		AccessTTL:      cfg.Token.AccessTokenTTL,
		RefreshTTL:     cfg.Token.RefreshTokenTTL,
		JWTIssuer:      cfg.Casdoor.Endpoint,
		JWTAudience:    cfg.Casdoor.ClientID,
		JWTCertificate: cfg.Casdoor.Certificate,
	})
	if err != nil {
		runCleanups()
		return fmt.Errorf("failed to initialize token service: %w", err)
	}

	// 初始化 SSO 客户端
	ssoClient, err := sso.NewClientWithCache(cfg.Casdoor, redisClient.GetClient())
	if err != nil {
		runCleanups()
		return fmt.Errorf("failed to initialize SSO client: %w", err)
	}
	cleanups = append(cleanups, func() {
		ssoClient.Close()
	})

	rbacRepo := rbac.NewRepository(database)
	rbacService := rbac.NewService(rbacRepo)

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
		if err := r.SetTrustedProxies(nil); err != nil {
			logger.L().Warn("failed to set trusted proxies to nil", zap.Error(err))
		}
	}

	r.Use(middleware.Recovery())
	r.Use(metrics.Middleware())
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.RequestLogger())
	// 根据环境选择安全头中间件
	if cfg.App.Env == "production" {
		r.Use(middleware.SecurityHeadersWithHSTS())
	} else {
		r.Use(middleware.SecurityHeadersMiddleware())
	}
	r.Use(middleware.MaxBodySize(cfg.App.MaxBodySize)) // 限制请求体大小

	// 配置 CORS — 先校验配置合法性，再注册中间件（H-25: 校验必须在 cors.New() 之前）
	corsOrigins := cfg.App.CORSOrigins
	for _, origin := range corsOrigins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "*" {
			return errors.New("CORS configuration error: wildcard '*' is not allowed when AllowCredentials is true")
		}
		if trimmed == "" {
			return errors.New("CORS configuration error: empty origin is not allowed")
		}
		if strings.HasSuffix(trimmed, "/") {
			return fmt.Errorf("CORS configuration error: origin %q must not have a trailing slash", trimmed)
		}
	}
	corsConfig := cors.Config{
		AllowOrigins:     corsOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-CSRF-Token", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	r.Use(cors.New(corsConfig))

	// 注册健康检查端点
	healthHandler := health.NewHandler(pgPool, redisClient.GetClient(), health.BuildInfo{
		Version:   version,
		GitCommit: gitCommit,
		BuildTime: buildTime,
	}, isProduction, time.Duration(cfg.App.HealthCheckTimeout)*time.Second)
	healthHandler.RegisterRoutes(r)

	// 注册 API 文档（仅开发/测试环境）
	apidocs.RegisterDocs(r, isProduction)

	// 注册 Prometheus 指标端点（仅限内部访问，生产环境应通过网络策略限制）
	metricsGroup := r.Group("/metrics")
	if isProduction {
		if cfg.App.MetricsPassword == "" {
			runCleanups()
			return fmt.Errorf("METRICS_PASSWORD must be set in production to protect the metrics endpoint")
		}
		metricsGroup.Use(gin.BasicAuth(gin.Accounts{
			cfg.App.MetricsUser: cfg.App.MetricsPassword,
		}))
	}
	metricsGroup.GET("", gin.WrapH(promhttp.Handler()))

	// 注册 API 路由（带版本控制）
	api := r.Group("/api/v1")
	{
		// 全局限流：防止单 IP 或整体流量过载（覆盖所有公开和认证端点）
		globalLimiter := middleware.NewRedisRateLimiter(redisClient.GetClient(), cfg.App.APIGlobalLimit, time.Minute)
		ipLimiter := middleware.NewRedisRateLimiter(redisClient.GetClient(), cfg.App.APIIPRateLimit, time.Minute)
		api.Use(middleware.GlobalRateLimitMiddleware(globalLimiter))
		api.Use(middleware.RateLimitMiddleware(ipLimiter))

		// Web Vitals 上报：注册在限流之后、CSRF 之前（sendBeacon 无法携带自定义 header）
		api.POST("/metrics/vitals", metrics.VitalsHandler())

		api.Use(middleware.CSRFMiddleware())

		// 注册认证模块路由
		authHandler := auth.NewHandler(
			cfg,
			tokenService,
			redisClient.GetClient(),
			ssoClient,
			auth.NewUserSyncRepository(database),
			rbacService,
		)
		authHandler.RegisterRoutes(api)

		// 注册课程模块路由
		authMW := middleware.AuthMiddleware(tokenService)
		optionalAuthMW := middleware.OptionalAuthMiddleware(tokenService)
		courseHandler := course.NewHandler(database, redisClient.GetClient(), rbacService, cfg)
		courseHandler.RegisterRoutes(api, authMW, optionalAuthMW)

		// 初始化 LDAP 客户端（可选，仅在配置了 LDAP_URL 时启用）
		var ldapClient *ldap.Client
		if ldapURL := os.Getenv("LDAP_URL"); ldapURL != "" {
			ldapCfg := ldap.Config{
				URL:                ldapURL,
				BaseDN:             os.Getenv("LDAP_BASE_DN"),
				SystemBindDN:       os.Getenv("LDAP_SYSTEM_BIND_DN"),
				SystemBindPassword: os.Getenv("LDAP_SYSTEM_BIND_PASSWORD"),
				UseTLS:             os.Getenv("LDAP_USE_TLS") == "true",
				InsecureSkipVerify: os.Getenv("LDAP_INSECURE_SKIP_VERIFY") == "true",
			}
			var ldapErr error
			ldapClient, ldapErr = ldap.NewClient(ldapCfg)
			if ldapErr != nil {
				logger.L().Warn("LDAP client initialization failed, student verification via LDAP unavailable",
					zap.Error(ldapErr),
				)
				ldapClient = nil
			} else {
				logger.L().Info("LDAP client initialized", zap.String("url", ldapURL))
			}
		}

		// 注册用户中心模块路由
		userRepo := user.NewRepository(database)

		// 构造 PII 加密器（密钥已在 config.Load() 阶段完成解码和校验）
		piiCipher, err := pii.NewCipher(cfg.Security.DocAESActiveKeyID, cfg.Security.DocAESKeys)
		if err != nil {
			runCleanups()
			return fmt.Errorf("failed to initialize PII cipher: %w", err)
		}

		userService, err := user.NewService(userRepo, ldapClient, crypto.GetHMACKey(), piiCipher)
		if err != nil {
			runCleanups()
			return fmt.Errorf("failed to initialize user service: %w", err)
		}
		userHandler := user.NewHandler(userService)
		rbacHandler := rbac.NewHandler(rbacService)
		userHandler.RegisterRoutes(api, authMW)

		// 管理后台路由组先做认证和管理入口能力校验，具体路由继续做细粒度授权。
		adminGroup := api.Group("/admin")
		adminGroup.Use(authMW, rbac.RequireAnyPermission(rbacService, capability.AdminEntryCapabilities...))
		rbacHandler.RegisterAdminRoutes(adminGroup, rbacService)
		userHandler.RegisterAdminRoutes(adminGroup, rbacService)

		// 启动后台定时任务（日志清理等）
		bgCtx, bgCancel := context.WithCancel(context.Background()) //nolint:gosec // G118: bgCancel is appended to cleanups and called on shutdown
		cleanups = append(cleanups, bgCancel)
		courseHandler.StartBackgroundJobs(bgCtx)
	}

	// 启动服务器
	srv := &http.Server{
		Addr:              ":" + cfg.App.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB，防止超大请求头消耗内存
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
	// H-32: 缓冲区设为 2，防止第二个信号丢失；收到首个信号后恢复默认行为
	quit := make(chan os.Signal, 2)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.L().Info("Received shutdown signal", zap.String("signal", sig.String()))
		// 收到第一个信号后恢复默认行为，第二个信号将直接终止进程
		signal.Stop(quit)
	case err := <-serverErr:
		// M-87: 服务器启动失败也执行 graceful shutdown，确保资源正确释放
		logger.L().Error("Server startup error, initiating shutdown", zap.Error(err))
		// 继续执行下方的 graceful shutdown 流程（而非直接 return）
		_ = err // 记录错误但不直接返回，走统一的 shutdown 路径
	}

	// 优雅关闭 — 按依赖关系逆序释放资源：
	// 1. HTTP server: 先停止接收新请求并排空现有连接
	// 2. 应用级服务（token/SSO 等）: 依赖 Redis/PG，需在底层连接关闭前完成
	// 3. Redis: 被 token blacklist、缓存、限流等依赖
	// 4. PostgreSQL: 最底层存储，最后关闭
	// shutdown 超时 = 事务超时（查询超时×3）+ 15 秒缓冲，确保长事务和数据库连接有足够时间排空
	shutdownTimeout := time.Duration(cfg.Database.QueryTimeout)*time.Second*3 + 15*time.Second
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	// Step 1: 关闭 HTTP server，排空连接
	if err := srv.Shutdown(shutdownCtx); err != nil {
		// L-57: 强制关闭时记录活跃连接数等指标
		logger.L().Error("Server forced to shutdown",
			zap.Error(err),
			zap.Duration("timeout", shutdownTimeout),
		)
	} else {
		logger.L().Info("HTTP server stopped gracefully")
	}

	// H-26: 在 shutdown 完成后显式调用 cleanups（使用 fresh context），
	// 而非 defer，避免 shutdown 超时后 cleanups 使用已过期的 context
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cleanupCancel()
	_ = cleanupCtx // cleanups 当前不需要 context，预留给未来需要 context 的清理操作

	// Step 2-4: 按逆序关闭 PostgreSQL → Redis（cleanups 逆序执行）
	runCleanups()
	logger.L().Info("All resources released, server exited")
	return nil
}
