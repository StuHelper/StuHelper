package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/admission"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/auth"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	platformcasdoor "git.stuhelper.com/StuHelper/StuHelper/internal/platform/casdoor"
)

func (rt *Runtime) initAuthModule(
	api *gin.RouterGroup,
	bgCtx context.Context,
	piiCipher *pii.Cipher,
	roleScopeResolver middleware.RoleScopeResolver,
) (*auth.Handler, gin.HandlerFunc, gin.HandlerFunc, error) {
	userSyncRepo := user.NewUserSyncRepository(rt.database, crypto.GetHMACKey()).
		WithRoleFGAClient(rt.fgaClient)
	rt.warnPendingUserHashBackfill(bgCtx, userSyncRepo)
	oidcSubjectValidator, err := rt.initOIDCSubjectValidator()
	if err != nil {
		return nil, nil, nil, err
	}

	authHandler := auth.NewHandler(
		auth.HandlerConfig{
			Token:                  rt.cfg.Token,
			CORSOrigins:            rt.authRedirectOrigins(),
			OIDCIssuer:             rt.cfg.Casdoor.Issuer,
			AccountSettingsBaseURL: rt.cfg.Casdoor.PublicAuthBaseURL,
			ProviderTokenCipher:    piiCipher,
			OIDCSubjectValidator:   oidcSubjectValidator,
		},
		rt.tokenService,
		rt.redisClient.GetClient(),
		rt.oidcClient,
		userSyncRepo,
		auth.WithAdminAuthorizers(authAdminAuthorizers()),
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

type casdoorOIDCSubjectValidator struct {
	client       *platformcasdoor.UserLookupClient
	organization string
}

func (v casdoorOIDCSubjectValidator) ValidateOIDCSubject(ctx context.Context, subject string) error {
	return v.client.ValidateSubjectOwner(ctx, subject, v.organization)
}

func (rt *Runtime) initOIDCSubjectValidator() (auth.OIDCSubjectValidator, error) {
	cfg := rt.cfg.Casdoor
	if strings.TrimSpace(cfg.UserLookupClientID) == "" &&
		strings.TrimSpace(cfg.UserLookupClientSecret) == "" &&
		strings.TrimSpace(cfg.UserLookupApplication) == "" {
		return nil, nil
	}
	client, err := platformcasdoor.NewUserLookupClient(casdoorUserLookupCredential(cfg))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Casdoor OIDC subject validator: %w", err)
	}
	return casdoorOIDCSubjectValidator{
		client:       client,
		organization: cfg.Organization,
	}, nil
}

func (rt *Runtime) authRedirectOrigins() []string {
	return append([]string(nil), rt.cfg.App.CORSOrigins...)
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
	openPlatformHandler adminRouteRegistrar,
	authMW gin.HandlerFunc,
) {
	adminGroup := api.Group("/admin")
	middlewares := append([]gin.HandlerFunc{authMW}, adminMFAMiddlewares(rt.cfg.App.Env, userRepo)...)
	middlewares = append(middlewares, adminEntryAuthorizer())
	adminGroup.Use(middlewares...)
	authHandler.RegisterAdminRoutes(adminGroup)
	userHandler.RegisterAdminRoutes(adminGroup)
	admissionHandler.RegisterAdminRoutes(adminGroup)
	if openPlatformHandler != nil {
		openPlatformHandler.RegisterAdminRoutes(adminGroup)
	}
}

type adminRouteRegistrar interface {
	RegisterAdminRoutes(admin *gin.RouterGroup)
}

func adminMFAMiddlewares(appEnv string, userRepo user.MFAContextRepository) []gin.HandlerFunc {
	if appEnv == "development" {
		return nil
	}
	return []gin.HandlerFunc{
		user.MFAContextMiddleware(userRepo),
		privilegedMFAAuthorizer(),
	}
}
