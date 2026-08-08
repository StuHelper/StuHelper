package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/modules/admission"
	"github.com/StuHelper/StuHelper/server/internal/modules/auth"
	authorizationmodule "github.com/StuHelper/StuHelper/server/internal/modules/authorization"
	"github.com/StuHelper/StuHelper/server/internal/modules/course/review"
	"github.com/StuHelper/StuHelper/server/internal/modules/rbac"
	"github.com/StuHelper/StuHelper/server/internal/modules/studentverification"
	"github.com/StuHelper/StuHelper/server/internal/modules/user"
	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto/pii"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/oidc"
	platformcasdoor "github.com/StuHelper/StuHelper/server/internal/platform/casdoor"
)

func (rt *Runtime) initAuthModule(
	api *gin.RouterGroup,
	bgCtx context.Context,
	piiCipher *pii.Cipher,
	accessResolver middleware.AccessSnapshotResolver,
	oidcSubjectValidator auth.OIDCSubjectValidator,
	organizationAdminSync auth.OrganizationAdminSynchronizer,
) (*auth.Handler, gin.HandlerFunc, gin.HandlerFunc, error) {
	if oidcSubjectValidator == nil {
		// Development may intentionally omit the Casdoor management credential.
		// Without an authoritative IsAdmin lookup, do not interpret the default
		// false value as a demotion signal.
		organizationAdminSync = nil
	}
	userSyncRepo := user.NewUserSyncRepository(rt.database, crypto.GetHMACKey())
	rt.warnPendingUserHashBackfill(bgCtx, userSyncRepo)
	authHandler, err := auth.NewHandler(
		auth.HandlerConfig{
			Token:                  rt.cfg.Token,
			CORSOrigins:            rt.authRedirectOrigins(),
			OIDCIssuer:             rt.cfg.Casdoor.Issuer,
			AccountSettingsBaseURL: rt.cfg.Casdoor.PublicAuthBaseURL,
			ProviderTokenCipher:    piiCipher,
			OIDCSubjectValidator:   oidcSubjectValidator,
			OrganizationAdminSync:  organizationAdminSync,
		},
		rt.tokenService,
		rt.redisClient.GetClient(),
		rt.oidcClient,
		userSyncRepo,
		auth.WithAdminAuthorizers(authAdminAuthorizers()),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize auth handler: %w", err)
	}
	authHandler.RegisterPublicRoutes(api)

	api.Use(middleware.CSRFMiddleware())
	authCookieConfig := middleware.OptionalAuthConfig{
		CookieDomain: rt.cfg.Token.CookieDomain,
		CookieSecure: rt.cfg.Token.CookieSecure,
	}
	authMW := middleware.AuthMiddlewareWithConfigAndAccessSnapshotResolver(
		rt.oidcClient,
		rt.tokenService,
		authCookieConfig,
		accessResolver,
	)
	authHandler.RegisterRoutesWithAuthMiddleware(api, authMW)

	optionalAuthMW := middleware.OptionalAuthMiddlewareWithAccessSnapshotResolver(
		rt.oidcClient,
		rt.tokenService,
		authCookieConfig,
		accessResolver,
	)
	return authHandler, authMW, optionalAuthMW, nil
}

type casdoorOIDCSubjectValidator struct {
	client       *platformcasdoor.UserLookupClient
	organization string
}

func (v casdoorOIDCSubjectValidator) ValidateOIDCSubject(ctx context.Context, subject string) (bool, error) {
	identity, err := v.client.ResolveSubject(ctx, subject, v.organization)
	if err != nil {
		return false, normalizeCasdoorSubjectLookupError(err)
	}
	return identity.OrganizationAdmin, nil
}

func normalizeCasdoorSubjectLookupError(err error) error {
	if errors.Is(err, platformcasdoor.ErrUserLookupUnavailable) {
		return fmt.Errorf("%w: %v", oidc.ErrProviderUnavailable, err)
	}
	return err
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
	authorizationHandler *authorizationmodule.Handler,
	studentVerificationHandler *studentverification.Handler,
	admissionHandler *admission.Handler,
	openPlatformHandler adminRouteRegistrar,
	authMW gin.HandlerFunc,
) {
	adminGroup := api.Group("/admin")
	middlewares := append([]gin.HandlerFunc{authMW}, adminMFAMiddlewares(rt.cfg.App.Env, userRepo)...)
	middlewares = append(middlewares, adminEntryAuthorizer())
	adminGroup.Use(middlewares...)
	authHandler.RegisterAdminRoutes(adminGroup)
	authorizationHandler.RegisterAdminRoutes(adminGroup)
	studentVerificationHandler.RegisterAdminRoutes(adminGroup)
	userHandler.RegisterAdminRoutes(adminGroup)
	admissionHandler.RegisterAdminRoutes(adminGroup)
	if openPlatformHandler != nil {
		openPlatformHandler.RegisterAdminRoutes(adminGroup)
	}
}

type adminRouteRegistrar interface {
	RegisterAdminRoutes(admin *gin.RouterGroup)
}

func adminMFAMiddlewares(appEnv string, repo user.MFAContextRepository) []gin.HandlerFunc {
	if appEnv == config.EnvDevelopment {
		return nil
	}
	return []gin.HandlerFunc{
		user.MFAContextMiddleware(repo),
		rbac.RequirePrivilegedMFA(),
	}
}

func adminReviewRouteSecurity(appEnv string, repo user.MFAContextRepository) review.AdminRouteSecurity {
	if appEnv == config.EnvDevelopment {
		return review.AdminRouteSecurity{}
	}
	return review.AdminRouteSecurity{
		Dashboard: []gin.HandlerFunc{
			user.MFAContextMiddleware(repo),
			rbac.RequireMFAProof(),
		},
		Privileged: adminMFAMiddlewares(appEnv, repo),
	}
}
