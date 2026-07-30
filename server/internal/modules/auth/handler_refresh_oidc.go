package auth

import (
	"crypto/subtle"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/ctxutil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/oidc"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
	"github.com/StuHelper/StuHelper/server/internal/pkg/token"
)

const providerRefreshCompensationTimeout = 2 * time.Second

var errOIDCRefreshTokenNotRotated = errors.New("oidc refresh token was not rotated")

type oidcRefreshPayload struct {
	rawIDToken           string
	providerAccessToken  string
	refreshToken         string
	userID               string
	accessTokenExpiresAt int64
	userSync             UserSyncInput
}

type oidcSessionRotation struct {
	sessionID       string
	appKey          string
	userID          string
	oldRefreshToken string
	payload         oidcRefreshPayload
}

// refreshOIDCToken 处理 OIDC provider 的 refresh 逻辑。
func (h *Handler) refreshOIDCToken(c *gin.Context, refreshTokenStr string, includeTokensInResponse bool) bool {
	sessionID := h.getSessionID(c, "")
	appKey, ok := h.resolveOIDCRefreshApplication(c, sessionID, refreshTokenStr)
	if !ok {
		return false
	}
	payload, ok := h.fetchOIDCRefreshPayload(c, appKey, refreshTokenStr)
	if !ok {
		return false
	}
	if !h.syncOIDCRefreshUser(c, appKey, sessionID, refreshTokenStr, payload) {
		return false
	}
	csrfToken, ok := h.prepareTokenCookies(c, sessionID)
	if !ok {
		h.revokeIssuedProviderTokenFamily(
			c,
			appKey,
			sessionID,
			refreshTokenStr,
			payload.providerAccessToken,
			payload.refreshToken,
			"cookie preparation failed",
		)
		return false
	}

	rotation := oidcSessionRotation{
		sessionID:       sessionID,
		appKey:          appKey,
		userID:          payload.userID,
		oldRefreshToken: refreshTokenStr,
		payload:         payload,
	}
	if !h.rotateOIDCSession(c, rotation) {
		return false
	}

	h.setTokenCookiesWithCSRF(c, payload.rawIDToken, payload.refreshToken, csrfToken)
	if sessionID != "" {
		h.setSessionCookie(c, sessionID)
	}
	response.Success(c, h.buildRefreshResponse(payload.rawIDToken, payload.refreshToken, includeTokensInResponse))
	return true
}

func (h *Handler) resolveOIDCRefreshApplication(c *gin.Context, sessionID, oldRefreshToken string) (string, bool) {
	appKey, err := h.svc.OIDCApplicationForRefresh(c.Request.Context(), sessionID, oldRefreshToken)
	if err == nil {
		return appKey, true
	}
	logger.FromGin(c).Error("failed to resolve OIDC refresh application", zap.Bool("session_present", sessionID != ""), zap.Error(err))
	h.clearTokenCookies(c)
	if errors.Is(err, token.ErrSessionNotFound) ||
		errors.Is(err, errSessionIDRequired) ||
		errors.Is(err, errSessionRefreshTokenMismatch) {
		response.Unauthorized(c, "invalid native session id", errs.ErrTokenInvalid)
		return "", false
	}
	response.InternalError(c, "failed to refresh token")
	return "", false
}

func (h *Handler) fetchOIDCRefreshPayload(c *gin.Context, appKey, oldRefreshToken string) (oidcRefreshPayload, bool) {
	newToken, err := h.oidcClient.RefreshTokenForApplication(c.Request.Context(), appKey, oldRefreshToken)
	if err != nil {
		logger.FromGin(c).Error("OIDC token refresh failed", zap.Error(err))
		if errors.Is(err, oidc.ErrProviderUnavailable) {
			response.ServiceUnavailable(c, "refresh service temporarily unavailable")
			return oidcRefreshPayload{}, false
		}
		h.clearTokenCookies(c)
		response.Unauthorized(c, "failed to refresh token", errs.ErrRefreshTokenInvalid)
		return oidcRefreshPayload{}, false
	}
	issuedAccessToken := newToken.AccessToken
	revokeUncommittedRefresh := true
	defer func() {
		if revokeUncommittedRefresh {
			h.revokeIssuedProviderTokenFamily(
				c,
				appKey,
				"",
				oldRefreshToken,
				issuedAccessToken,
				newToken.RefreshToken,
				"provider refresh validation failed",
			)
		}
	}()

	rawIDToken := oidc.ExtractIDToken(newToken)
	if rawIDToken == "" {
		logger.FromGin(c).Error("no id_token in refresh response")
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return oidcRefreshPayload{}, false
	}
	if err := validateOIDCRefreshRotation(oldRefreshToken, newToken.RefreshToken); err != nil {
		logger.FromGin(c).Error("OIDC refresh token rotation unavailable", zap.Error(err))
		h.clearTokenCookies(c)
		response.ServiceUnavailable(c, "refresh token rotation unavailable")
		return oidcRefreshPayload{}, false
	}

	newClaims, err := h.oidcClient.VerifyIDTokenForApplication(c.Request.Context(), appKey, rawIDToken)
	if err != nil {
		logger.FromGin(c).Error("new ID token verification failed after refresh", zap.Error(err))
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return oidcRefreshPayload{}, false
	}
	revokeUncommittedRefresh = false
	return oidcRefreshPayload{
		rawIDToken:           rawIDToken,
		providerAccessToken:  issuedAccessToken,
		refreshToken:         newToken.RefreshToken,
		userID:               newClaims.GetUserID(),
		accessTokenExpiresAt: newClaims.ExpiresAt,
		userSync: UserSyncInput{
			CasdoorSubject:     newClaims.GetUserID(),
			Username:           newClaims.GetUsername(),
			Email:              newClaims.GetEmail(),
			AvatarURL:          newClaims.GetAvatar(),
			Roles:              newClaims.Roles,
			RolesAuthoritative: newClaims.RolesClaimPresent,
		},
	}, true
}

func (h *Handler) syncOIDCRefreshUser(
	c *gin.Context,
	appKey,
	sessionID,
	oldRefreshToken string,
	payload oidcRefreshPayload,
) bool {
	if err := h.svc.SyncOIDCUser(c.Request.Context(), payload.userSync); err != nil {
		h.revokeIssuedProviderTokenFamily(
			c,
			appKey,
			sessionID,
			oldRefreshToken,
			payload.providerAccessToken,
			payload.refreshToken,
			"user sync failed",
		)
		logger.FromGin(c).Error("user sync failed during OIDC refresh",
			zap.String("user_id", payload.userID),
			zap.Error(err),
		)
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return false
	}
	return true
}

func (h *Handler) rotateOIDCSession(c *gin.Context, rotation oidcSessionRotation) bool {
	err := h.svc.RotateSession(
		c.Request.Context(),
		rotation.sessionID,
		rotation.userID,
		rotation.oldRefreshToken,
		rotation.payload.rawIDToken,
		rotation.payload.accessTokenExpiresAt,
		rotation.payload.providerAccessToken,
		rotation.payload.refreshToken,
	)
	if err == nil {
		return true
	}

	if sessionRotationCommitted(err) {
		logger.FromGin(c).Error("OIDC session rotated with post-commit cleanup failure",
			zap.Bool("session_present", rotation.sessionID != ""),
			zap.Error(err),
		)
		return true
	}

	h.revokeIssuedProviderTokenFamily(
		c,
		rotation.appKey,
		rotation.sessionID,
		rotation.oldRefreshToken,
		rotation.payload.providerAccessToken,
		rotation.payload.refreshToken,
		"local session rotation failed before commit",
	)
	logger.FromGin(c).Error("failed to rotate OIDC session", zap.Bool("session_present", rotation.sessionID != ""), zap.Error(err))
	h.clearTokenCookies(c)
	if errors.Is(err, token.ErrSessionNotFound) ||
		errors.Is(err, errSessionIDRequired) ||
		errors.Is(err, errSessionUserRequired) ||
		errors.Is(err, errSessionUserMismatch) ||
		errors.Is(err, errSessionRefreshTokenMismatch) {
		response.Unauthorized(c, "invalid native session id", errs.ErrTokenInvalid)
		return false
	}
	response.InternalError(c, "failed to refresh token")
	return false
}

func (h *Handler) revokeIssuedProviderTokenFamily(
	c *gin.Context,
	appKey,
	sessionID,
	oldRefreshToken,
	newAccessToken,
	newRefreshToken,
	reason string,
) {
	if h == nil || h.svc == nil || !shouldCompensateProviderRefreshToken(oldRefreshToken, newRefreshToken) {
		return
	}
	revokeCtx, cancel := ctxutil.DetachedTimeout(c.Request.Context(), providerRefreshCompensationTimeout)
	defer cancel()
	if err := h.svc.revokeRawProviderTokenFamily(
		revokeCtx,
		appKey,
		newAccessToken,
		newRefreshToken,
	); err != nil {
		logger.FromGin(c).Error("failed to revoke uncommitted provider token family",
			zap.Bool("session_present", sessionID != ""),
			zap.String("provider_app_key", appKey),
			zap.String("reason", reason),
			zap.Error(err),
		)
	}
}

func shouldCompensateProviderRefreshToken(oldRefreshToken, newRefreshToken string) bool {
	oldRefreshToken = normalizeProviderRefreshToken(oldRefreshToken)
	newRefreshToken = normalizeProviderRefreshToken(newRefreshToken)
	if newRefreshToken == "" {
		return false
	}
	if oldRefreshToken != "" && subtle.ConstantTimeCompare([]byte(oldRefreshToken), []byte(newRefreshToken)) == 1 {
		return false
	}
	return true
}

func validateOIDCRefreshRotation(oldRefreshToken, newRefreshToken string) error {
	if oldRefreshToken == "" || newRefreshToken == "" {
		return errOIDCRefreshTokenNotRotated
	}
	if subtle.ConstantTimeCompare([]byte(oldRefreshToken), []byte(newRefreshToken)) == 1 {
		return errOIDCRefreshTokenNotRotated
	}
	return nil
}
