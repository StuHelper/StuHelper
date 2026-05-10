package app

import (
	"context"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/admission"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/auth"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/rbac"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/sms"
)

func (rt *Runtime) initAuthModule(
	api *gin.RouterGroup,
	bgCtx context.Context,
	piiCipher *pii.Cipher,
	smsSvc *sms.Service,
	roleScopeResolver middleware.RoleScopeResolver,
) (*auth.Handler, gin.HandlerFunc, gin.HandlerFunc, error) {
	userSyncRepo := user.NewUserSyncRepository(rt.database, piiCipher, crypto.GetHMACKey())
	rt.warnPendingUserHashBackfill(bgCtx, userSyncRepo)

	authHandler := auth.NewHandler(
		auth.HandlerConfig{
			Token:               rt.cfg.Token,
			CORSOrigins:         rt.cfg.App.CORSOrigins,
			OIDCIssuer:          rt.cfg.Casdoor.Issuer,
			ProviderTokenCipher: piiCipher,
		},
		rt.tokenService,
		rt.redisClient.GetClient(),
		rt.oidcClient,
		userSyncRepo,
		smsSvc,
	)
	authHandler.RegisterPublicRoutes(api)

	api.Use(middleware.CSRFMiddleware())
	authCookieConfig := middleware.OptionalAuthConfig{
		CookieDomain: rt.cfg.Token.CookieDomain,
		CookieSecure: rt.cfg.Token.CookieSecure,
	}
	authMW := middleware.AuthMiddlewareWithConfigAndRoleScopeResolver(rt.oidcClient, rt.tokenService, authCookieConfig, roleScopeResolver)
	authHandler.RegisterRoutesWithAuthMiddleware(api, authMW)

	optionalAuthMW := middleware.OptionalAuthMiddlewareWithRoleScopeResolver(rt.oidcClient, rt.tokenService, authCookieConfig, roleScopeResolver)
	return authHandler, authMW, optionalAuthMW, nil
}

func (rt *Runtime) registerUserRoutes(api *gin.RouterGroup, userHandler *user.Handler, authMW gin.HandlerFunc) {
	userHandler.RegisterRoutes(api, authMW)
}

func (rt *Runtime) registerAdminRoutes(
	api *gin.RouterGroup,
	userRepo user.MFAContextRepository,
	userHandler *user.Handler,
	authHandler *auth.Handler,
	admissionHandler *admission.Handler,
	authMW gin.HandlerFunc,
) {
	adminGroup := api.Group("/admin")
	middlewares := append([]gin.HandlerFunc{authMW}, adminMFAMiddlewares(rt.cfg.App.Env, userRepo)...)
	middlewares = append(middlewares, rbac.RequireAnyCapability(capability.AdminEntryCapabilities...))
	adminGroup.Use(middlewares...)
	authHandler.RegisterAdminRoutes(adminGroup)
	userHandler.RegisterAdminRoutes(adminGroup)
	admissionHandler.RegisterAdminRoutes(adminGroup)
}

func adminMFAMiddlewares(appEnv string, userRepo user.MFAContextRepository) []gin.HandlerFunc {
	if appEnv == "development" {
		return nil
	}
	return []gin.HandlerFunc{
		user.MFAContextMiddleware(userRepo),
		rbac.RequirePrivilegedMFA(),
	}
}
