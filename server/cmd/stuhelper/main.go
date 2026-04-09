package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"

	apidocs "git.stuhelper.com/StuHelper/StuHelper/api"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/auth"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/course"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/course/review"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/ldap"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/notification"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/rbac"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/health"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/objectstorage"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/observability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/redis"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/sms"
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
		ServiceName:     cfg.Log.ServiceName,
		Environment:     cfg.Log.Environment,
		ServiceVersion:  version,
	}
	logger.Init(logCfg)
	defer func() {
		if err := logger.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "logger close error: %v\n", err)
		}
	}()

	// bgWg 跟踪后台 goroutine（如 backfill），确保清理资源前已退出。
	// bgCancelFn 在 bgCtx 创建后赋值；早期失败路径中可能保持 nil。
	var bgWg sync.WaitGroup
	var bgCancelFn context.CancelFunc

	// cleanups 累积已初始化资源的清理函数，初始化失败时按逆序释放
	var cleanups []func()
	runCleanups := func() {
		if bgCancelFn != nil {
			bgCancelFn()
		}
		bgWg.Wait()
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}

	// 初始化 HMAC 密钥（用于用户 ID 哈希等场景）
	isProduction := cfg.App.Env == "production"
	if err := crypto.InitHMACKey(cfg.App.HMACSecret, isProduction); err != nil {
		return fmt.Errorf("failed to initialize HMAC key: %w", err)
	}

	obsShutdown, err := observability.Setup(context.Background(), cfg.Observability, cfg.App.Env, observability.BuildInfo{
		Version:   version,
		GitCommit: gitCommit,
		BuildTime: buildTime,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize observability: %w", err)
	}
	cleanups = append(cleanups, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if shutdownErr := obsShutdown(ctx); shutdownErr != nil {
			logger.L().Warn("observability shutdown error", zap.Error(shutdownErr))
		}
	})

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

	// 创建带超时的数据库封装
	database := db.NewDB(pgPool, time.Duration(cfg.Database.QueryTimeout)*time.Second)
	cleanups = append(cleanups, func() {
		database.Close()
	})

	// 初始化 Token 管理服务
	tokenService, err := token.NewService(token.ServiceConfig{
		RedisClient: redisClient.GetClient(),
		AccessTTL:   cfg.Token.AccessTokenTTL,
		RefreshTTL:  cfg.Token.RefreshTokenTTL,
	})
	if err != nil {
		runCleanups()
		return fmt.Errorf("failed to initialize token service: %w", err)
	}
	cleanups = append(cleanups, tokenService.Close)

	// 初始化 Zitadel OIDC 客户端
	oidcClient, err := oidc.NewClient(context.Background(), cfg.Zitadel)
	if err != nil {
		runCleanups()
		return fmt.Errorf("failed to initialize OIDC client: %w", err)
	}

	// 根据环境设置 Gin 模式
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建 Gin 路由
	r := gin.New()

	// 配置可信代理列表
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

	r.Use(middleware.RequestIDMiddleware())
	r.Use(otelgin.Middleware(cfg.Observability.ServiceName))
	r.Use(middleware.Recovery())
	r.Use(metrics.Middleware())
	r.Use(middleware.RequestLogger())
	// 根据环境选择安全头中间件
	if cfg.App.Env == "production" {
		r.Use(middleware.SecurityHeadersWithHSTS())
	} else {
		r.Use(middleware.SecurityHeadersMiddleware())
	}
	r.Use(middleware.MaxBodySize(cfg.App.MaxBodySize)) // 限制请求体大小

	// 配置 CORS — 先校验配置合法性，再注册中间件
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
		ExposeHeaders:    []string{"Content-Length", "X-CSRF-Token"},
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

	// bgCtx/bgCancel 管理后台任务生命周期。
	// runCleanups 会执行 cancel -> wait -> close，初始化失败和正常 shutdown 共用同一清理顺序。
	bgCtx, bgCancel := context.WithCancel(context.Background()) //nolint:gosec // G118: managed by runCleanups and explicit shutdown cancel
	bgCancelFn = bgCancel

	// 注册 API 路由（带版本控制）
	api := r.Group("/api/v1")
	{
		// 全局限流（覆盖所有公开和认证端点）
		globalLimiter := middleware.NewRedisRateLimiter(redisClient.GetClient(), cfg.App.APIGlobalLimit, time.Minute)
		ipLimiter := middleware.NewRedisRateLimiter(redisClient.GetClient(), cfg.App.APIIPRateLimit, time.Minute)
		api.Use(middleware.GlobalRateLimitMiddleware(globalLimiter))
		api.Use(middleware.RateLimitMiddleware(ipLimiter))

		// Web Vitals 上报：注册在限流之后、CSRF 之前（sendBeacon 无法携带自定义 header）
		api.POST("/metrics/vitals", metrics.VitalsHandler())
		api.POST("/metrics/frontend-errors", metrics.FrontendErrorHandler())

		openapiValidationMW, err := middleware.NewOpenAPIRequestValidationMiddleware()
		if err != nil {
			runCleanups()
			return fmt.Errorf("failed to initialize OpenAPI request validator: %w", err)
		}
		api.Use(openapiValidationMW)

		// 初始化 SMS 服务（可选，配置了腾讯云密钥时启用手机验证码登录）
		var smsSvc *sms.Service
		if cfg.SMS.SecretID != "" && cfg.SMS.SecretKey != "" {
			smsSvc = sms.NewService(sms.Config{
				SecretID:    cfg.SMS.SecretID,
				SecretKey:   cfg.SMS.SecretKey,
				AppID:       cfg.SMS.AppID,
				SignName:    cfg.SMS.SignName,
				TemplateID:  cfg.SMS.TemplateID,
				Region:      cfg.SMS.Region,
				InternalKey: cfg.SMS.InternalKey,
			}, logger.L())

			// 启动内部 SMS 转发服务（Zitadel Action 调用）
			smsMux := http.NewServeMux()
			smsSvc.RegisterInternalHandler(smsMux)
			smsSrv := &http.Server{
				Addr:              "127.0.0.1:" + cfg.SMS.InternalPort,
				Handler:           otelhttp.NewHandler(smsMux, "sms_internal"),
				ReadHeaderTimeout: 5 * time.Second,
				ReadTimeout:       10 * time.Second,
				WriteTimeout:      10 * time.Second,
			}
			listener, listenErr := net.Listen("tcp", smsSrv.Addr)
			if listenErr != nil {
				runCleanups()
				return fmt.Errorf("failed to start SMS internal server: %w", listenErr)
			}
			go func() {
				logger.L().Info("SMS internal server starting", zap.String("addr", smsSrv.Addr))
				if smsErr := smsSrv.Serve(listener); smsErr != nil && !errors.Is(smsErr, http.ErrServerClosed) {
					logger.L().Error("SMS internal server error", zap.Error(smsErr))
				}
			}()
			cleanups = append(cleanups, func() {
				smsCtx, smsCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer smsCancel()
				if smsErr := smsSrv.Shutdown(smsCtx); smsErr != nil {
					logger.L().Warn("SMS server shutdown error", zap.Error(smsErr))
				}
			})
		} else {
			logger.L().Info("SMS service not configured (SMS_SECRET_ID/SMS_SECRET_KEY missing), phone login and Zitadel SMS callback unavailable")
		}

		// 构造 PII 加密器（密钥已在 config.Load() 阶段完成解码和校验）
		piiCipher, err := pii.NewCipher(cfg.Security.DocAESActiveKeyID, cfg.Security.DocAESKeys)
		if err != nil {
			runCleanups()
			return fmt.Errorf("failed to initialize PII cipher: %w", err)
		}

		userSyncRepo := auth.NewUserSyncRepository(database, piiCipher, crypto.GetHMACKey())

		// 异步回填缺失的 user_hash（幂等，增量执行），不阻塞启动。
		// goroutine 受 bgCtx 管理，并在 runCleanups 中等待退出后再关闭 DB。
		bgWg.Add(1)
		go func() {
			defer bgWg.Done()
			if backfilled, backfillErr := userSyncRepo.BackfillUserHashes(bgCtx); backfillErr != nil {
				logger.L().Warn("user_hash backfill failed (non-fatal)", zap.Error(backfillErr))
			} else if backfilled > 0 {
				logger.L().Info("backfilled user_hash for existing users", zap.Int64("count", backfilled))
			}
		}()

		authHandler := auth.NewHandler(
			cfg,
			tokenService,
			redisClient.GetClient(),
			oidcClient,
			userSyncRepo,
			smsSvc,
		)

		// OTP 路由注册在 CSRF 之前（匿名用户无 CSRF cookie，已有独立限流保护）
		authHandler.RegisterPublicRoutes(api)

		api.Use(middleware.CSRFMiddleware())

		// 注册认证模块路由（CSRF 保护下）
		authHandler.RegisterRoutes(api, oidcClient, tokenService)

		// 注册课程模块路由
		authMW := middleware.AuthMiddleware(oidcClient, tokenService)
		optionalAuthMW := middleware.OptionalAuthMiddleware(oidcClient, tokenService)
		fgaClient, err := fga.NewClient(cfg.OpenFGA)
		if err != nil {
			runCleanups()
			return fmt.Errorf("failed to create FGA client: %w", err)
		}
		if fgaClient == nil {
			logger.L().Info("OpenFGA not configured, relational authorization disabled")
		}

		// 初始化通知模块（SSE + Redis Pub/Sub）— 在 courseHandler 之前，因为 review 模块需要 notification.Sender
		notifHub := notification.NewHub(redisClient.GetClient())
		notifRepo := notification.NewRepository(database)
		notifService := notification.NewService(notifRepo, notifHub, redisClient.GetClient())

		courseHandler := course.NewHandler(database, redisClient.GetClient(), cfg, fgaClient, notifService)
		courseHandler.RegisterRoutes(api, authMW, optionalAuthMW)

		// 初始化 LDAP 客户端（可选，仅在配置了 LDAP_URL 时启用）
		var ldapClient *ldap.Client
		if cfg.LDAP.URL != "" {
			ldapCfg := ldap.Config{
				URL:                cfg.LDAP.URL,
				BaseDN:             cfg.LDAP.BaseDN,
				SystemBindDN:       cfg.LDAP.SystemBindDN,
				SystemBindPassword: cfg.LDAP.SystemBindPassword,
				UseTLS:             cfg.LDAP.UseTLS,
				InsecureSkipVerify: cfg.LDAP.InsecureSkipVerify,
			}
			var ldapErr error
			ldapClient, ldapErr = ldap.NewClient(ldapCfg)
			if ldapErr != nil {
				logger.L().Warn("LDAP client initialization failed, student verification via LDAP unavailable",
					zap.Error(ldapErr),
				)
				ldapClient = nil
			} else {
				logger.L().Info("LDAP client initialized", zap.String("url", cfg.LDAP.URL))
			}
		}

		// 注册用户中心模块路由
		userRepo := user.NewRepository(database)

		userService, err := user.NewService(userRepo, ldapClient, crypto.GetHMACKey(), piiCipher)
		if err != nil {
			runCleanups()
			return fmt.Errorf("failed to initialize user service: %w", err)
		}
		if cfg.ObjectStorage.Endpoint != "" {
			photoStore, err := objectstorage.New(context.Background(), objectstorage.Config{
				Endpoint:        cfg.ObjectStorage.Endpoint,
				Region:          cfg.ObjectStorage.Region,
				Bucket:          cfg.ObjectStorage.Bucket,
				AccessKeyID:     cfg.ObjectStorage.AccessKeyID,
				SecretAccessKey: cfg.ObjectStorage.SecretAccessKey,
				UseSSL:          cfg.ObjectStorage.UseSSL,
				ForcePathStyle:  cfg.ObjectStorage.ForcePathStyle,
				PresignTTL:      time.Duration(cfg.ObjectStorage.PresignTTL) * time.Second,
			})
			if err != nil {
				runCleanups()
				return fmt.Errorf("failed to initialize object storage: %w", err)
			}
			if err := photoStore.EnsureBucket(context.Background()); err != nil {
				runCleanups()
				return fmt.Errorf("failed to ensure identity photo bucket: %w", err)
			}
			userService.SetIdentityPhotoStore(photoStore)
		}
		// 注册角色同步回调：认证状态变化时异步同步 Zitadel Project Role
		mgmtClient := oidc.NewManagementClient(cfg.Zitadel)
		roleSyncFn := oidc.BuildRoleSyncFunc(mgmtClient, func(ctx context.Context, userID int64) (string, error) {
			return userRepo.GetExternalID(ctx, userID)
		})
		userService.SetRoleSyncFunc(roleSyncFn)
		// 构建绑定手机 OTP 依赖（复用 SMS 服务与 OTP 服务）
		var bindPhoneOTP user.OTPGenerator
		var bindPhoneSMS user.SMSSender
		if smsSvc != nil {
			bindPhoneOTP = auth.NewOTPService(redisClient.GetClient())
			bindPhoneSMS = smsSvc
		}
		userHandler := user.NewHandler(userService, fgaClient, redisClient.GetClient(), bindPhoneOTP, bindPhoneSMS)
		userHandler.RegisterRoutes(api, authMW)

		// 管理后台路由组：认证 + 管理入口能力校验
		adminGroup := api.Group("/admin")
		adminGroup.Use(authMW, rbac.RequireAnyCapability(capability.AdminEntryCapabilities...))
		userHandler.RegisterAdminRoutes(adminGroup)

		// 启动后台定时任务（日志清理等）
		courseHandler.StartBackgroundJobs(bgCtx)

		// 注册通知模块路由 + 启动 Redis 订阅
		notifHandler := notification.NewHandler(notifService, notifHub)
		notifHandler.RegisterRoutes(api, authMW)
		notifHub.StartRedisSubscriber(bgCtx)
		cleanups = append(cleanups, notifHub.Stop)
	}

	// 启动服务器
	srv := &http.Server{
		Addr:              ":" + cfg.App.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
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
	// 信号 channel 缓冲区设为 2，收到首个信号后恢复默认行为
	quit := make(chan os.Signal, 2)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// serverStartErr 记录启动阶段错误，shutdown 完成后返回给调用方，确保非零退出码
	var serverStartErr error

	select {
	case sig := <-quit:
		logger.L().Info("Received shutdown signal", zap.String("signal", sig.String()))
		// 收到第一个信号后恢复默认行为，第二个信号将直接终止进程
		signal.Stop(quit)
	case err := <-serverErr:
		// 启动失败时也走 graceful shutdown
		logger.L().Error("Server startup error, initiating shutdown", zap.Error(err))
		serverStartErr = err
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
		// 强制关闭时记录活跃连接数等指标
		logger.L().Error("Server forced to shutdown",
			zap.Error(err),
			zap.Duration("timeout", shutdownTimeout),
		)
	} else {
		logger.L().Info("HTTP server stopped gracefully")
	}

	// Step 1.5: 先取消后台任务，通知 backfill / 定时任务尽快退出。
	if bgCancelFn != nil {
		bgCancelFn()
	}

	// Step 1.6: 等待所有 in-flight FGA 异步写入完成，避免 DB 关闭后 goroutine 仍访问数据库。
	review.WaitFGAWrites()

	// Step 2-4: 按逆序关闭 PostgreSQL → Redis（runCleanups 内部执行 cancel -> wait -> close）
	runCleanups()
	logger.L().Info("All resources released, server exited")
	return serverStartErr
}
