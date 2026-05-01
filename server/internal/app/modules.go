package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	gozap "go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/academics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/auth"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/course"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/course/review"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/notification"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/rbac"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/resource"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/storage"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/sms"
)

func (rt *Runtime) registerAPIRoutes(r *gin.Engine, bgCtx context.Context) error {
	api := r.Group("/api/v1")
	if err := rt.configureAPICommonMiddleware(api); err != nil {
		return err
	}
	rt.registerMetricsRoutes(api)
	startBackgroundTask := func(name string, run func(context.Context)) {
		rt.startBackgroundTask(bgCtx, name, run)
	}

	smsSvc, err := rt.initSMSService()
	if err != nil {
		return err
	}

	piiCipher, err := pii.NewCipher(rt.cfg.Security.DocAESActiveKeyID, rt.cfg.Security.DocAESKeys)
	if err != nil {
		return fmt.Errorf("failed to initialize PII cipher: %w", err)
	}

	authMW, optionalAuthMW, err := rt.initAuthModule(api, bgCtx, piiCipher, smsSvc)
	if err != nil {
		return err
	}

	fgaClient, err := fga.NewClient(rt.cfg.OpenFGA)
	if err != nil {
		return fmt.Errorf("failed to create FGA client: %w", err)
	}

	notifHub := notification.NewHub(rt.redisClient.GetClient())
	notifRepo := notification.NewRepository(rt.database)
	notifService := notification.NewService(notifRepo, notifHub, rt.redisClient.GetClient())
	notifHandler := notification.NewHandler(notifService, notifHub)
	notifHandler.RegisterRoutes(api, authMW)
	notifHub.StartRedisSubscriber(bgCtx, startBackgroundTask)
	rt.addCleanup(notifHub.Stop)

	userRepo := user.NewRepository(rt.database, crypto.GetHMACKey())
	courseHandler := rt.initCourseModule(fgaClient, notifService, userRepo)
	courseHandler.RegisterRoutes(api, authMW, optionalAuthMW)

	storageService := storage.NewService(storage.NewRepository(rt.database), rt.cfg.ObjectStorage)
	if err := storageService.EnsureDefaultMount(bgCtx); err != nil {
		return fmt.Errorf("failed to ensure storage default mount: %w", err)
	}

	userService, err := rt.initUserService(userRepo, piiCipher, fgaClient, storageService)
	if err != nil {
		return err
	}
	academicsHandler := academics.NewHandler(academics.NewService(
		academics.NewRepository(rt.database),
		academics.NewRegistry(),
	))
	academicsHandler.RegisterRoutes(api, authMW)

	storage.NewHandler(storageService).RegisterAdminRoutes(api, authMW)

	resourceService := resource.NewService(
		resource.NewRepository(rt.database),
		storageService,
	)
	resourceService.StartBackgroundJobs(bgCtx, startBackgroundTask)
	resourceHandler := resource.NewHandler(resourceService)
	resourceHandler.RegisterRoutes(api, authMW, optionalAuthMW)

	var bindPhoneOTP user.OTPGenerator
	var bindPhoneSMS user.SMSSender
	if smsSvc != nil {
		bindPhoneOTP = auth.NewOTPService(rt.redisClient.GetClient())
		bindPhoneSMS = smsSvc
	}
	userHandler := user.NewHandler(userService, rt.redisClient.GetClient(), bindPhoneOTP, bindPhoneSMS)
	botHandler := user.NewBotHandler(userService, rt.cfg.Bot.ServiceToken)
	userService.StartBackgroundJobs(bgCtx, startBackgroundTask)
	rt.registerUserRoutes(api, userHandler, authMW)
	botHandler.RegisterRoutes(api)
	rt.registerAdminRoutes(api, userHandler, authMW)

	courseHandler.StartBackgroundJobs(bgCtx, startBackgroundTask)

	return nil
}

func (rt *Runtime) configureAPICommonMiddleware(api *gin.RouterGroup) error {
	globalLimiter := middleware.NewRedisRateLimiter(rt.redisClient.GetClient(), rt.cfg.App.APIGlobalLimit, time.Minute)
	ipLimiter := middleware.NewRedisRateLimiter(rt.redisClient.GetClient(), rt.cfg.App.APIIPRateLimit, time.Minute)
	api.Use(middleware.GlobalRateLimitMiddleware(globalLimiter))
	api.Use(middleware.RateLimitMiddleware(ipLimiter))

	openapiValidationMW, err := middleware.NewOpenAPIRequestValidationMiddleware()
	if err != nil {
		return fmt.Errorf("failed to initialize OpenAPI request validator: %w", err)
	}
	api.Use(openapiValidationMW)
	return nil
}

func (rt *Runtime) registerMetricsRoutes(api *gin.RouterGroup) {
	metricsGroup := api.Group("/metrics")
	metricsGroup.Use(metrics.OriginValidationMiddleware(rt.metricsAllowedOrigins()))
	metricsGroup.POST("/vitals", metrics.VitalsHandler())
	metricsGroup.POST("/frontend-errors", metrics.FrontendErrorHandler())
}

func (rt *Runtime) initAuthModule(api *gin.RouterGroup, bgCtx context.Context, piiCipher *pii.Cipher, smsSvc *sms.Service) (gin.HandlerFunc, gin.HandlerFunc, error) {
	userSyncRepo := user.NewUserSyncRepository(rt.database, piiCipher, crypto.GetHMACKey())
	rt.warnPendingUserHashBackfill(bgCtx, userSyncRepo)

	authHandler := auth.NewHandler(
		auth.HandlerConfig{
			Token:       rt.cfg.Token,
			CORSOrigins: rt.cfg.App.CORSOrigins,
			OIDCIssuer:  rt.cfg.Casdoor.Issuer,
		},
		rt.tokenService,
		rt.redisClient.GetClient(),
		rt.oidcClient,
		userSyncRepo,
		smsSvc,
	)
	authHandler.RegisterPublicRoutes(api)

	api.Use(middleware.CSRFMiddleware())
	authHandler.RegisterRoutes(api, rt.oidcClient, rt.tokenService)

	authMW := middleware.AuthMiddleware(rt.oidcClient, rt.tokenService)
	optionalAuthMW := middleware.OptionalAuthMiddleware(rt.oidcClient, rt.tokenService, middleware.OptionalAuthConfig{
		CookieDomain: rt.cfg.Token.CookieDomain,
		CookieSecure: rt.cfg.Token.CookieSecure,
	})
	return authMW, optionalAuthMW, nil
}

func (rt *Runtime) initCourseModule(authorizer review.AuthorizationProvider, notifSender notification.Sender, accessReader review.ReviewAccessReader) *course.Handler {
	courseCache := cache.NewHelper(rt.redisClient.GetClient())
	reviewRepo := review.NewRepository(rt.database)
	reviewService := review.NewService(rt.database, reviewRepo, notifSender, authorizer, accessReader)
	reviewHandler := review.NewHandler(courseCache, reviewService, rt.redisClient.GetClient(), rt.cfg.RateLimit, authorizer)

	courseRepo := course.NewRepository(rt.database)
	courseService := course.NewService(courseRepo, logger.L().Named("course_service"))
	return course.NewHandler(courseCache, courseService, reviewHandler)
}

func (rt *Runtime) registerUserRoutes(api *gin.RouterGroup, userHandler *user.Handler, authMW gin.HandlerFunc) {
	userHandler.RegisterRoutes(api, authMW)
}

func (rt *Runtime) registerAdminRoutes(api *gin.RouterGroup, userHandler *user.Handler, authMW gin.HandlerFunc) {
	adminGroup := api.Group("/admin")
	adminGroup.Use(authMW, rbac.RequireAnyCapability(capability.AdminEntryCapabilities...))
	userHandler.RegisterAdminRoutes(adminGroup)
}

func (rt *Runtime) metricsAllowedOrigins() []string {
	if len(rt.cfg.App.CORSOrigins) > 0 {
		return rt.cfg.App.CORSOrigins
	}
	if rt.isProduction {
		return nil
	}
	return []string{
		"http://localhost:3000",
		"http://localhost:5173",
		"http://localhost:4173",
	}
}

func (rt *Runtime) initSMSService() (*sms.Service, error) {
	if !rt.cfg.SMS.Enabled {
		logger.L().Info("SMS service disabled (SMS_ENABLED=false), phone login and Casdoor SMS callback unavailable")
		return nil, nil
	}

	smsSvc := sms.NewService(sms.Config{
		SecretID:    rt.cfg.SMS.SecretID,
		SecretKey:   rt.cfg.SMS.SecretKey,
		AppID:       rt.cfg.SMS.AppID,
		SignName:    rt.cfg.SMS.SignName,
		TemplateID:  rt.cfg.SMS.TemplateID,
		Region:      rt.cfg.SMS.Region,
		InternalKey: rt.cfg.SMS.InternalKey,
	}, logger.L())

	smsMux := http.NewServeMux()
	smsSvc.RegisterInternalHandler(smsMux)
	smsSrv := &http.Server{
		Addr:              "127.0.0.1:" + rt.cfg.SMS.InternalPort,
		Handler:           otelhttp.NewHandler(smsMux, "sms_internal"),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
	listenCfg := net.ListenConfig{}
	listener, err := listenCfg.Listen(context.Background(), "tcp", smsSrv.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to start SMS internal server: %w", err)
	}
	go func() {
		logger.L().Info("SMS internal server starting", gozap.String("addr", smsSrv.Addr))
		if serveErr := smsSrv.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.L().Error("SMS internal server error", gozap.Error(serveErr))
		}
	}()
	rt.addCleanup(func() {
		smsCtx, smsCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer smsCancel()
		if shutdownErr := smsSrv.Shutdown(smsCtx); shutdownErr != nil {
			logger.L().Warn("SMS server shutdown error", gozap.Error(shutdownErr))
		}
	})

	return smsSvc, nil
}

// warnPendingUserHashBackfill 在启动时检查是否仍有未回填的 user_hash 记录。
// 仅记录警告日志，不再自动执行回填——回填应作为运维任务显式执行。
func (rt *Runtime) warnPendingUserHashBackfill(ctx context.Context, repo *user.UserSyncRepository) {
	count, err := repo.CountMissingUserHashes(ctx)
	if err != nil {
		logger.L().Warn("failed to check pending user_hash backfill", gozap.Error(err))
		return
	}
	if count > 0 {
		logger.L().Warn("users with missing user_hash detected; run backfill ops task",
			gozap.Int64("pending_count", count),
		)
	}
}

func (rt *Runtime) initUserService(userRepo *user.Repository, piiCipher *pii.Cipher, fgaClient *fga.Client, storageService *storage.Service) (*user.Service, error) {
	var photoStore user.ServiceOption
	if rt.cfg.ObjectStorage.Endpoint != "" {
		initCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, err := storageService.ValidateMountByKey(initCtx, storage.DefaultMountKey); err != nil {
			return nil, fmt.Errorf("identity photo storage mount validation failed: %w", err)
		}
		photoStore = user.WithIdentityPhotoStorageService(storageService, storage.DefaultMountKey)
	}

	mgmtClient := oidc.NewManagementClient(rt.cfg.Casdoor)
	roleSyncFn := oidc.BuildRoleSyncFunc(mgmtClient, func(ctx context.Context, userID int64) (string, error) {
		return userRepo.GetExternalID(ctx, userID)
	})

	userService, err := user.NewService(
		userRepo,
		crypto.GetHMACKey(),
		piiCipher,
		user.WithProfileFGAClient(fgaClient),
		user.WithRoleSyncFunc(roleSyncFn),
		photoStore,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize user service: %w", err)
	}
	if err := userService.LoadSystemConfigSnapshots(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to load user system config snapshots: %w", err)
	}
	return userService, nil
}
