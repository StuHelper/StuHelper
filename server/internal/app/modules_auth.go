package app

import (
	"context"

	"github.com/gin-gonic/gin"

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
	authHandler.RegisterRoutes(api, rt.oidcClient, rt.tokenService)

	authMW := middleware.AuthMiddleware(rt.oidcClient, rt.tokenService)
	optionalAuthMW := middleware.OptionalAuthMiddleware(rt.oidcClient, rt.tokenService, middleware.OptionalAuthConfig{
		CookieDomain: rt.cfg.Token.CookieDomain,
		CookieSecure: rt.cfg.Token.CookieSecure,
	})
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
	authMW gin.HandlerFunc,
) {
	adminGroup := api.Group("/admin")
	middlewares := append([]gin.HandlerFunc{authMW}, adminMFAMiddlewares(userRepo)...)
	middlewares = append(middlewares, rbac.RequireAnyCapability(capability.AdminEntryCapabilities...))
	adminGroup.Use(middlewares...)
	authHandler.RegisterAdminRoutes(adminGroup)
	userHandler.RegisterAdminRoutes(adminGroup)
}

func adminMFAMiddlewares(userRepo user.MFAContextRepository) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		user.MFAContextMiddleware(userRepo),
		rbac.RequirePrivilegedMFA(),
	}
}
