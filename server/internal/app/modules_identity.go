package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/auth"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/identityserver"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/openplatform"
)

func (rt *Runtime) initIdentityServerRoutes(
	r *gin.Engine,
	openPlatformService *openplatform.Service,
	authHandler *auth.Handler,
	optionalAuthMW gin.HandlerFunc,
	userIDResolver func(context.Context, string) (int64, error),
) (*identityserver.Service, error) {
	issuer := rt.identityIssuer()
	signer, err := identityserver.NewSigner(
		issuer,
		rt.cfg.Identity.SigningKeyID,
		rt.cfg.Identity.SigningPrivateKeyPEM,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize identity signer: %w", err)
	}
	service, err := identityserver.NewService(
		openPlatformService,
		rt.redisClient.GetClient(),
		signer,
		issuer,
		time.Duration(rt.cfg.Identity.AuthorizationCodeTTL)*time.Second,
		time.Duration(rt.cfg.Identity.AccessTokenTTL)*time.Second,
		time.Duration(rt.cfg.Identity.RefreshTokenTTL)*time.Second,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize identity server: %w", err)
	}
	handler := identityserver.NewHandler(service, openPlatformService, issuer, issuer, userIDResolver)
	if authHandler != nil {
		handler.SetSessionRevoker(
			authHandler.SessionRevoker(),
			rt.cfg.Token.CookieDomain,
			rt.cfg.Token.CookieSecure,
		)
	}
	handler.RegisterRoutes(r, optionalAuthMW)
	return service, nil
}

func (rt *Runtime) identityIssuer() string {
	if issuer := strings.TrimRight(strings.TrimSpace(rt.cfg.Identity.Issuer), "/"); issuer != "" {
		return issuer
	}
	for _, origin := range rt.cfg.App.CORSOrigins {
		if trimmed := strings.TrimRight(strings.TrimSpace(origin), "/"); trimmed != "" {
			return trimmed
		}
	}
	return "http://localhost:" + rt.cfg.App.Port
}
