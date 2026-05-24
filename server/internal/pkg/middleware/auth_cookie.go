package middleware

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

func resolveCookieToken(c *gin.Context, oidcClient *oidc.Client, tokenService *token.Service, rawToken string) (*authResult, error) {
	if token.IsSelfSignedToken(rawToken) {
		metrics.ObserveAuthSessionValidationFailure("self_signed_unsupported")
		return nil, errSessionInvalid
	}
	return resolveOIDCCookieToken(c, oidcClient, tokenService, rawToken)
}

func resolveOIDCCookieToken(c *gin.Context, oidcClient *oidc.Client, tokenService *token.Service, rawIDToken string) (*authResult, error) {
	if oidcClient == nil || tokenService == nil {
		metrics.ObserveAuthSessionValidationFailure("dependency_missing")
		return nil, errSessionUnavailable
	}
	session, err := loadOIDCCookieSession(c, tokenService, rawIDToken)
	if err != nil {
		return nil, err
	}
	claims, err := oidcClient.VerifyIDTokenForApplication(c.Request.Context(), session.ProviderAppKey, rawIDToken)
	if err != nil {
		logger.L().Debug("OIDC token verification failed", zap.Error(err))
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	if session.UserID != "" && session.UserID != claims.GetUserID() {
		metrics.ObserveAuthSessionValidationFailure("user_mismatch")
		return nil, errSessionInvalid
	}
	return authResultFromOIDCClaims(claims, session.CreatedAt), nil
}

func loadOIDCCookieSession(c *gin.Context, tokenService *token.Service, rawIDToken string) (*token.SessionData, error) {
	sessionID, err := c.Cookie(CookieSessionID)
	if err != nil || sessionID == "" {
		metrics.ObserveAuthSessionValidationFailure("missing_session")
		return nil, errSessionInvalid
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), authSessionLookupTimeout)
	defer cancel()
	session, err := tokenService.GetSessionStore().Get(ctx, sessionID)
	if err != nil {
		metrics.ObserveAuthSessionValidationFailure(sessionLookupFailureReason(ctx))
		return nil, fmt.Errorf("%w: %v", errSessionUnavailable, err)
	}
	return validateOIDCCookieSession(session, rawIDToken)
}

func validateOIDCCookieSession(session *token.SessionData, rawIDToken string) (*token.SessionData, error) {
	if session == nil {
		metrics.ObserveAuthSessionValidationFailure("not_found")
		return nil, errSessionInvalid
	}
	if session.ProviderAppKey == "" {
		metrics.ObserveAuthSessionValidationFailure("missing_app")
		return nil, errSessionInvalid
	}
	if !sessionAccessTokenMatches(rawIDToken, session.AccessTokenHash) {
		metrics.ObserveAuthSessionValidationFailure("access_mismatch")
		return nil, errSessionInvalid
	}
	return session, nil
}

func sessionLookupFailureReason(ctx context.Context) string {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "timeout"
	}
	return "unavailable"
}

func sessionAccessTokenMatches(rawToken, storedHash string) bool {
	if rawToken == "" || storedHash == "" {
		return false
	}
	hash, err := crypto.HMACHash(rawToken)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(hash), []byte(storedHash)) == 1
}

func authResultFromOIDCClaims(claims *oidc.Claims, fallbackUnix int64) *authResult {
	return &authResult{
		userID:         claims.GetUserID(),
		appID:          claims.GetAppID(),
		username:       claims.GetUsername(),
		email:          claims.GetEmail(),
		displayName:    claims.GetDisplayName(),
		avatar:         claims.GetAvatar(),
		roles:          claims.Roles,
		orgScopedRoles: claims.OrgScopedRoles,
		authTime:       oidcAuthTime(claims.AuthTime, fallbackUnix),
		mfaProofAt:     claims.MFAProofVerifiedAt(),
	}
}

func oidcAuthTime(authTimeUnix, fallbackUnix int64) time.Time {
	if authTime := timeFromUnix(authTimeUnix); !authTime.IsZero() {
		return authTime
	}
	return timeFromUnix(fallbackUnix)
}

func timeFromUnix(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	return time.Unix(value, 0).UTC()
}
