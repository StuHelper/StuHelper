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

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/auth"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/course"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/ldap"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/notification"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/rbac"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/objectstorage"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/sms"
)

func (rt *Runtime) registerAPIRoutes(r *gin.Engine, bgCtx context.Context) error {
	api := r.Group("/api/v1")

	globalLimiter := middleware.NewRedisRateLimiter(rt.redisClient.GetClient(), rt.cfg.App.APIGlobalLimit, time.Minute)
	ipLimiter := middleware.NewRedisRateLimiter(rt.redisClient.GetClient(), rt.cfg.App.APIIPRateLimit, time.Minute)
	api.Use(middleware.GlobalRateLimitMiddleware(globalLimiter))
	api.Use(middleware.RateLimitMiddleware(ipLimiter))

	api.POST("/metrics/vitals", metrics.VitalsHandler())
	api.POST("/metrics/frontend-errors", metrics.FrontendErrorHandler())

	openapiValidationMW, err := middleware.NewOpenAPIRequestValidationMiddleware()
	if err != nil {
		return fmt.Errorf("failed to initialize OpenAPI request validator: %w", err)
	}
	api.Use(openapiValidationMW)

	smsSvc, err := rt.initSMSService()
	if err != nil {
		return err
	}

	piiCipher, err := pii.NewCipher(rt.cfg.Security.DocAESActiveKeyID, rt.cfg.Security.DocAESKeys)
	if err != nil {
		return fmt.Errorf("failed to initialize PII cipher: %w", err)
	}

	userSyncRepo := auth.NewUserSyncRepository(rt.database, piiCipher, crypto.GetHMACKey())
	rt.startUserHashBackfill(bgCtx, userSyncRepo)

	authHandler := auth.NewHandler(
		rt.cfg,
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
	optionalAuthMW := middleware.OptionalAuthMiddleware(rt.oidcClient, rt.tokenService)

	fgaClient, err := fga.NewClient(rt.cfg.OpenFGA)
	if err != nil {
		return fmt.Errorf("failed to create FGA client: %w", err)
	}
	if fgaClient == nil {
		logger.L().Info("OpenFGA not configured, relational authorization disabled")
	}

	notifHub := notification.NewHub(rt.redisClient.GetClient())
	notifRepo := notification.NewRepository(rt.database)
	notifService := notification.NewService(notifRepo, notifHub, rt.redisClient.GetClient())

	courseCache := cache.NewHelper(rt.redisClient.GetClient())
	courseHandler := course.NewHandler(rt.database, courseCache, rt.redisClient.GetClient(), rt.cfg, fgaClient, notifService)
	courseHandler.RegisterRoutes(api, authMW, optionalAuthMW)

	ldapClient, err := rt.initLDAPClient()
	if err != nil {
		logger.L().Warn("LDAP client initialization failed, student verification via LDAP unavailable",
			gozap.Error(err),
		)
		ldapClient = nil
	}

	userRepo := user.NewRepository(rt.database, crypto.GetHMACKey())
	userService, err := user.NewService(userRepo, ldapClient, crypto.GetHMACKey(), piiCipher)
	if err != nil {
		return fmt.Errorf("failed to initialize user service: %w", err)
	}
	if err := rt.configureUserService(userService, userRepo); err != nil {
		return err
	}

	var bindPhoneOTP user.OTPGenerator
	var bindPhoneSMS user.SMSSender
	if smsSvc != nil {
		bindPhoneOTP = auth.NewOTPService(rt.redisClient.GetClient())
		bindPhoneSMS = smsSvc
	}
	userHandler := user.NewHandler(userService, fgaClient, rt.redisClient.GetClient(), bindPhoneOTP, bindPhoneSMS)
	userHandler.RegisterRoutes(api, authMW)

	adminGroup := api.Group("/admin")
	adminGroup.Use(authMW, rbac.RequireAnyCapability(capability.AdminEntryCapabilities...))
	userHandler.RegisterAdminRoutes(adminGroup)

	courseHandler.StartBackgroundJobs(bgCtx)

	notifHandler := notification.NewHandler(notifService, notifHub)
	notifHandler.RegisterRoutes(api, authMW)
	notifHub.StartRedisSubscriber(bgCtx)
	rt.addCleanup(notifHub.Stop)

	return nil
}

func (rt *Runtime) initSMSService() (*sms.Service, error) {
	if rt.cfg.SMS.SecretID == "" || rt.cfg.SMS.SecretKey == "" {
		logger.L().Info("SMS service not configured (SMS_SECRET_ID/SMS_SECRET_KEY missing), phone login and Zitadel SMS callback unavailable")
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
	listener, err := net.Listen("tcp", smsSrv.Addr)
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

func (rt *Runtime) startUserHashBackfill(bgCtx context.Context, repo *auth.UserSyncRepository) {
	rt.bgWg.Add(1)
	go func() {
		defer rt.bgWg.Done()
		if backfilled, backfillErr := repo.BackfillUserHashes(bgCtx); backfillErr != nil {
			logger.L().Warn("user_hash backfill failed (non-fatal)", gozap.Error(backfillErr))
		} else if backfilled > 0 {
			logger.L().Info("backfilled user_hash for existing users", gozap.Int64("count", backfilled))
		}
	}()
}

func (rt *Runtime) initLDAPClient() (*ldap.Client, error) {
	if rt.cfg.LDAP.URL == "" {
		return nil, nil
	}
	ldapClient, err := ldap.NewClient(ldap.Config{
		URL:                rt.cfg.LDAP.URL,
		BaseDN:             rt.cfg.LDAP.BaseDN,
		SystemBindDN:       rt.cfg.LDAP.SystemBindDN,
		SystemBindPassword: rt.cfg.LDAP.SystemBindPassword,
		UseTLS:             rt.cfg.LDAP.UseTLS,
		InsecureSkipVerify: rt.cfg.LDAP.InsecureSkipVerify,
	})
	if err != nil {
		return nil, err
	}
	logger.L().Info("LDAP client initialized", gozap.String("url", rt.cfg.LDAP.URL))
	return ldapClient, nil
}

func (rt *Runtime) configureUserService(userService *user.Service, userRepo *user.Repository) error {
	if rt.cfg.ObjectStorage.Endpoint != "" {
		photoStore, err := objectstorage.New(context.Background(), objectstorage.Config{
			Endpoint:        rt.cfg.ObjectStorage.Endpoint,
			Region:          rt.cfg.ObjectStorage.Region,
			Bucket:          rt.cfg.ObjectStorage.Bucket,
			AccessKeyID:     rt.cfg.ObjectStorage.AccessKeyID,
			SecretAccessKey: rt.cfg.ObjectStorage.SecretAccessKey,
			UseSSL:          rt.cfg.ObjectStorage.UseSSL,
			ForcePathStyle:  rt.cfg.ObjectStorage.ForcePathStyle,
			PresignTTL:      time.Duration(rt.cfg.ObjectStorage.PresignTTL) * time.Second,
		})
		if err != nil {
			return fmt.Errorf("failed to initialize object storage: %w", err)
		}
		if err := photoStore.EnsureBucket(context.Background()); err != nil {
			return fmt.Errorf("failed to ensure identity photo bucket: %w", err)
		}
		userService.SetIdentityPhotoStore(photoStore)
	}

	mgmtClient := oidc.NewManagementClient(rt.cfg.Zitadel)
	roleSyncFn := oidc.BuildRoleSyncFunc(mgmtClient, func(ctx context.Context, userID int64) (string, error) {
		return userRepo.GetExternalID(ctx, userID)
	})
	userService.SetRoleSyncFunc(roleSyncFn)
	return nil
}
