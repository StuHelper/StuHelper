package middleware

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"

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
		return resolveSelfSignedCookieToken(rawToken)
	}
	return resolveOIDCCookieToken(c, oidcClient, tokenService, rawToken)
}

func resolveSelfSignedCookieToken(rawToken string) (*authResult, error) {
	hmacKey := crypto.GetHMACKey()
	if len(hmacKey) == 0 {
		return nil, fmt.Errorf("self-signed token present but HMAC key not initialized")
	}
	claims, err := token.VerifyJWTWithType(hmacKey, rawToken, token.JWTTokenTypeAccess)
	if err != nil {
		logger.L().Debug("self-signed JWT verification failed", zap.Error(err))
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	return authResultFromSelfSignedClaims(claims), nil
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
	return authResultFromOIDCClaims(claims), nil
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

func authResultFromSelfSignedClaims(claims *token.JWTClaims) *authResult {
	var avatarPtr *string
	if claims.Avatar != "" {
		avatarPtr = &claims.Avatar
	}
	return &authResult{
		userID:      claims.Sub,
		username:    claims.Name,
		email:       claims.Email,
		displayName: claims.DisplayName,
		avatar:      avatarPtr,
		roles:       claims.Roles,
	}
}

func authResultFromOIDCClaims(claims *oidc.Claims) *authResult {
	return &authResult{
		userID:         claims.GetUserID(),
		appID:          claims.GetAppID(),
		username:       claims.GetUsername(),
		email:          claims.GetEmail(),
		displayName:    claims.GetDisplayName(),
		avatar:         claims.GetAvatar(),
		roles:          claims.Roles,
		orgScopedRoles: claims.OrgScopedRoles,
		mfaProofAt:     claims.MFAProofVerifiedAt(),
	}
}
