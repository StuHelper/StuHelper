package app

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	gozap "go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/modules/academics"
	"github.com/StuHelper/StuHelper/server/internal/modules/admission"
	"github.com/StuHelper/StuHelper/server/internal/modules/auth"
	"github.com/StuHelper/StuHelper/server/internal/modules/notification"
	"github.com/StuHelper/StuHelper/server/internal/modules/resource"
	"github.com/StuHelper/StuHelper/server/internal/modules/storage"
	"github.com/StuHelper/StuHelper/server/internal/modules/user"
	"github.com/StuHelper/StuHelper/server/internal/pkg/botcredential"
	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto/pii"
	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/sms"
	platformauth "github.com/StuHelper/StuHelper/server/internal/platform/authorization"
	platformcasdoor "github.com/StuHelper/StuHelper/server/internal/platform/casdoor"
	"github.com/StuHelper/StuHelper/server/internal/platform/serviceaccount"
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
	rt.startAuditRetentionCleanup(bgCtx, startBackgroundTask)

	if _, err := rt.initSMSService(r); err != nil {
		return err
	}

	piiCipher, err := pii.NewCipher(rt.cfg.Security.DocAESActiveKeyID, rt.cfg.Security.DocAESKeys)
	if err != nil {
		return fmt.Errorf("failed to initialize PII cipher: %w", err)
	}

	fgaClient, err := fga.NewClient(rt.cfg.OpenFGA)
	if err != nil {
		return fmt.Errorf("failed to create FGA client: %w", err)
	}
	rt.fgaClient = fgaClient

	userRepo := user.NewRepository(rt.database, crypto.GetHMACKey())
	roleScopeResolver, err := platformauth.NewRoleScopeResolver(fgaClient, userRepo.GetInternalUserID)
	if err != nil {
		return err
	}

	authHandler, authMW, optionalAuthMW, err := rt.initAuthModule(api, bgCtx, piiCipher, roleScopeResolver)
	if err != nil {
		return err
	}

	adminMFA := adminMFAMiddlewares(rt.cfg.App.Env, userRepo)
	notifHub := notification.NewHub()
	notifRealtime := notification.NewRealtime(rt.redisClient.GetClient(), notifHub)
	notifRepo := notification.NewRepository(rt.database)
	notifService := notification.NewService(notifRepo, notifRealtime)
	notifHandler := notification.NewHandler(notifService, notifHub, userRepo.GetInternalUserID)
	notifHandler.RegisterRoutes(api, authMW)
	notifRealtime.StartSubscriber(bgCtx, startBackgroundTask)
	rt.addShutdownHook(notifHub.Stop)
	rt.addCleanup(notifRealtime.Stop)

	courseModule := rt.initCourseModule(bgCtx, fgaClient, notifService, userRepo)
	courseModule.RegisterRoutes(api, authMW, optionalAuthMW, adminMFA...)

	storageService := storage.NewService(storage.NewRepository(rt.database), rt.cfg.ObjectStorage)
	if err := storageService.EnsureDefaultMount(bgCtx); err != nil {
		return fmt.Errorf("failed to ensure storage default mount: %w", err)
	}
	schoolEmailSender, err := newSchoolEmailSender(rt.cfg.Email, rt.database)
	if err != nil {
		return fmt.Errorf("failed to initialize school email sender: %w", err)
	}
	var studentEmailSender user.StudentEmailSender
	var admissionEmailSender admission.SchoolEmailSender
	if schoolEmailSender != nil {
		studentEmailSender = schoolEmailSender
		admissionEmailSender = schoolEmailSender
	}

	externalStudentDirectory, err := rt.initExternalStudentDirectory()
	if err != nil {
		return err
	}
	var externalStudentDirectoryOpt user.ServiceOption
	if externalStudentDirectory != nil {
		externalStudentDirectoryOpt = user.WithExternalStudentDirectory(newExternalStudentDirectoryAdapter(externalStudentDirectory))
	}

	userService, err := rt.initUserService(
		bgCtx,
		userRepo,
		piiCipher,
		fgaClient,
		storageService,
		studentEmailSender,
		externalStudentDirectoryOpt,
	)
	if err != nil {
		return err
	}
	userProfileGateway, err := rt.initCasdoorUserProfileGateway(userRepo.GetCasdoorSubject)
	if err != nil {
		return err
	}
	userService.SetProfileIdentitySyncGateway(userProfileGateway)
	academicsHandler := academics.NewHandler(
		academics.NewService(
			academics.NewRepository(rt.database),
			academics.NewRegistry(),
		),
		academics.WithAdminAuthorizers(academicsAdminAuthorizers()),
	)
	academicsHandler.RegisterRoutes(api, authMW, adminMFA...)

	storage.NewHandler(
		storageService,
		storage.WithAdminAuthorizers(storageAdminAuthorizers()),
	).RegisterAdminRoutes(api, authMW, adminMFA...)

	resourceService := resource.NewService(
		resource.NewRepository(rt.database),
		newResourceStorageAdapter(storageService),
	)
	resourceService.StartBackgroundJobs(bgCtx, startBackgroundTask)
	resourceHandler := resource.NewHandler(resourceService)
	resourceHandler.RegisterRoutes(api, authMW, optionalAuthMW)

	var bindPhoneOTP user.OTPGenerator
	var bindPhoneSMS user.SMSSender
	if userProfileGateway != nil {
		if rt.cfg.SMS.Enabled {
			bindPhoneSMS = userProfileGateway
		} else if rt.cfg.App.Env == config.EnvDevelopment {
			bindPhoneSMS = newDevBindPhoneSMSSender(rt.redisClient.GetClient())
		}
		if bindPhoneSMS != nil {
			baseBindPhoneOTP := newBindPhoneOTPAdapter(auth.NewOTPService(rt.redisClient.GetClient()))
			if _, ok := bindPhoneSMS.(*devBindPhoneSMSSender); ok {
				bindPhoneOTP = newDevBindPhoneOTPGenerator(baseBindPhoneOTP, rt.redisClient.GetClient())
			} else {
				bindPhoneOTP = baseBindPhoneOTP
			}
		}
	}
	userHandler := user.NewHandler(
		userService,
		rt.redisClient.GetClient(),
		bindPhoneOTP,
		bindPhoneSMS,
		user.WithAdminAuthorizers(userAdminAuthorizers()),
	)
	openPlatformHandler, _, err := rt.initOpenPlatformModule(
		api,
		authMW,
		userProfileGateway,
		userRepo.GetInternalUserID,
	)
	if err != nil {
		return err
	}
	botCredentialVerifier, err := rt.initBotCredentialVerifier(bgCtx)
	if err != nil {
		return err
	}
	admissionUserGateway := newAdmissionUserGateway(userService)
	admissionService, err := admission.NewService(
		admission.NewRepository(rt.database),
		admissionUserGateway,
		crypto.GetHMACKey(),
		admission.WithAdmissionRedisClient(rt.redisClient.GetClient()),
		admission.WithSchoolEmailSender(admissionEmailSender),
		admission.WithAcademicStudentLookupGateway(admissionUserGateway),
		admission.WithAdmissionMaterialStore(
			newAdmissionMaterialStorageAdapter(storageService, storage.DefaultMountKey),
		),
		admission.WithOperatorAccessGateway(rt.initAdmissionOperatorAccess(userRepo)),
		admission.WithFreshmanProjectionGateway(admissionUserGateway),
		admission.WithSchoolSSOExchanger(admission.NewOIDCSchoolSSOExchanger(rt.oidcClient)),
		admission.WithAdmissionPublicBaseURL(rt.cfg.Admission.PublicBaseURL),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize admission service: %w", err)
	}
	userService.SetAdmissionVerificationProjectionGateway(admissionService)
	admissionHandler := admission.NewHandler(
		admissionService,
		userRepo.GetInternalUserID,
		botCredentialVerifier,
		admission.WithAdminAuthorizers(admissionAdminAuthorizers()),
		admission.WithStreamShutdown(bgCtx),
	)
	admissionHandler.RegisterRoutes(api, authMW)
	admissionHandler.RegisterBotRoutes(api)
	admissionService.StartBackgroundJobs(bgCtx, startBackgroundTask)

	botHandler := user.NewBotHandler(userService, botCredentialVerifier)
	userService.StartBackgroundJobs(bgCtx, startBackgroundTask)
	rt.registerUserRoutes(api, userHandler, authMW)
	botHandler.RegisterRoutes(api)
	rt.registerAdminRoutes(api, userRepo, userHandler, authHandler, admissionHandler, openPlatformHandler, authMW)

	courseModule.StartBackgroundJobs(bgCtx, startBackgroundTask)

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

func (rt *Runtime) initSMSService(r *gin.Engine) (*sms.Service, error) {
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

	registerSMSInternalRoute(r, smsSvc)
	logger.L().Info("SMS internal route registered", gozap.String("path", "/internal/sms/send"))
	return smsSvc, nil
}

func registerSMSInternalRoute(r *gin.Engine, smsSvc *sms.Service) {
	smsMux := http.NewServeMux()
	smsSvc.RegisterInternalHandler(smsMux)
	r.POST("/internal/sms/send", gin.WrapH(smsMux))
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

func (rt *Runtime) initUserService(
	ctx context.Context,
	userRepo *user.Repository,
	piiCipher *pii.Cipher,
	fgaClient *fga.Client,
	storageService *storage.Service,
	schoolEmailSender user.StudentEmailSender,
	extraOptions ...user.ServiceOption,
) (*user.Service, error) {
	var photoStore user.ServiceOption
	if objectStorageConfigured(rt.cfg.ObjectStorage) {
		initCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if _, err := storageService.ValidateMountByKey(initCtx, storage.DefaultMountKey); err != nil {
			return nil, fmt.Errorf("identity photo storage mount validation failed: %w", err)
		}
		photoStore = user.WithIdentityPhotoStore(newIdentityPhotoStorageAdapter(storageService, storage.DefaultMountKey))
	}

	roleSyncFn, err := rt.initCasdoorRoleSync(userRepo)
	if err != nil {
		return nil, err
	}
	options := []user.ServiceOption{
		user.WithProfileFGAClient(fgaClient),
		user.WithRoleSyncFunc(roleSyncFn),
		user.WithStudentEmailOTP(rt.redisClient.GetClient(), schoolEmailSender),
		user.WithLDAPClientFactory(newLDAPAuthClient),
		photoStore,
	}
	options = append(options, extraOptions...)
	userService, err := user.NewService(userRepo, crypto.GetHMACKey(), piiCipher, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize user service: %w", err)
	}
	if err := userService.LoadSystemConfigSnapshots(ctx); err != nil {
		return nil, fmt.Errorf("failed to load user system config snapshots: %w", err)
	}
	return userService, nil
}

func (rt *Runtime) initCasdoorUserProfileGateway(
	casdoorSubjectByUserID func(context.Context, int64) (string, error),
) (*casdoorUserProfileGateway, error) {
	client, err := rt.newCasdoorUserProfileClient()
	if err != nil {
		return nil, err
	}
	return newCasdoorUserProfileGateway(client, casdoorSubjectByUserID), nil
}

func (rt *Runtime) initCasdoorRoleSync(userRepo *user.Repository) (user.RoleSyncFunc, error) {
	client, err := rt.newCasdoorRoleSyncClient()
	if err != nil {
		return nil, err
	}
	return platformcasdoor.BuildRoleSyncFunc(client, userRepo.GetCasdoorSubject), nil
}

func (rt *Runtime) initBotCredentialVerifier(ctx context.Context) (*serviceaccount.Verifier, error) {
	if strings.TrimSpace(rt.cfg.Bot.ServiceToken) == "" {
		return nil, nil
	}
	verifier, err := serviceaccount.NewVerifier(rt.database, crypto.GetHMACKey())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize bot service credential verifier: %w", err)
	}
	_, err = verifier.EnsureBootstrapCredential(ctx, serviceaccount.BootstrapCredential{
		Name:     botcredential.KoishiRuntimeCredentialName,
		RawToken: rt.cfg.Bot.ServiceToken,
		Audience: []string{botcredential.AudienceBotAPI},
		Scopes:   botcredential.KoishiRuntimeScopes(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to bootstrap bot service credential: %w", err)
	}
	return verifier, nil
}

func (rt *Runtime) newCasdoorRoleSyncClient() (*platformcasdoor.RoleSyncClient, error) {
	cfg := rt.cfg.Casdoor
	if !casdoorRoleSyncConfigured(cfg) {
		return nil, nil
	}
	client, err := platformcasdoor.NewRoleSyncClient(casdoorRoleSyncCredential(cfg), casdoorUserLookupCredential(cfg))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Casdoor role sync client: %w", err)
	}
	return client, nil
}

func casdoorRoleSyncConfigured(cfg config.CasdoorConfig) bool {
	return !configStringMissing(cfg.RoleSyncClientID) ||
		!configStringMissing(cfg.RoleSyncClientSecret) ||
		!configStringMissing(cfg.RoleSyncApplication) ||
		!configStringMissing(cfg.UserLookupClientID) ||
		!configStringMissing(cfg.UserLookupClientSecret) ||
		!configStringMissing(cfg.UserLookupApplication)
}

func configStringMissing(value string) bool {
	return strings.TrimSpace(value) == ""
}

func objectStorageConfigured(cfg config.ObjectStorageConfig) bool {
	return !configStringMissing(cfg.Endpoint)
}

func casdoorRoleSyncCredential(cfg config.CasdoorConfig) platformcasdoor.Credential {
	return platformcasdoor.Credential{
		Purpose:      platformcasdoor.PurposeRoleSync,
		Endpoint:     cfg.Issuer,
		ClientID:     cfg.RoleSyncClientID,
		ClientSecret: cfg.RoleSyncClientSecret,
		Certificate:  cfg.RoleSyncCertificate,
		Organization: cfg.Organization,
		Application:  cfg.RoleSyncApplication,
	}
}

func casdoorUserLookupCredential(cfg config.CasdoorConfig) platformcasdoor.Credential {
	return platformcasdoor.Credential{
		Purpose:      platformcasdoor.PurposeUserLookup,
		Endpoint:     cfg.Issuer,
		ClientID:     cfg.UserLookupClientID,
		ClientSecret: cfg.UserLookupClientSecret,
		Certificate:  cfg.UserLookupCertificate,
		Organization: cfg.Organization,
		Application:  cfg.UserLookupApplication,
	}
}
