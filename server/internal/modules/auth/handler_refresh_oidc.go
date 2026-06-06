package auth

import (
	"crypto/subtle"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/ctxutil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

const providerRefreshCompensationTimeout = 2 * time.Second

var errOIDCRefreshTokenNotRotated = errors.New("oidc refresh token was not rotated")

type oidcRefreshPayload struct {
	rawIDToken   string
	refreshToken string
	userID       string
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
	csrfToken, ok := h.prepareTokenCookies(c, sessionID)
	if !ok {
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
	logger.FromGin(c).Error("failed to resolve OIDC refresh application", zap.String("session_id", sessionID), zap.Error(err))
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
	return oidcRefreshPayload{rawIDToken: rawIDToken, refreshToken: newToken.RefreshToken, userID: newClaims.GetUserID()}, true
}

func (h *Handler) rotateOIDCSession(c *gin.Context, rotation oidcSessionRotation) bool {
	err := h.svc.RotateSession(
		c.Request.Context(),
		rotation.sessionID,
		rotation.userID,
		rotation.oldRefreshToken,
		rotation.payload.rawIDToken,
		rotation.payload.refreshToken,
	)
	if err == nil {
		return true
	}

	h.revokeNewProviderRefreshTokenAfterRotationFailure(c, rotation)
	logger.FromGin(c).Error("failed to rotate OIDC session", zap.String("session_id", rotation.sessionID), zap.Error(err))
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

func (h *Handler) revokeNewProviderRefreshTokenAfterRotationFailure(c *gin.Context, rotation oidcSessionRotation) {
	if h == nil || h.svc == nil || rotation.payload.refreshToken == "" {
		return
	}
	revokeCtx, cancel := ctxutil.DetachedTimeout(c.Request.Context(), providerRefreshCompensationTimeout)
	defer cancel()
	if err := h.svc.revokeRawProviderRefreshToken(revokeCtx, rotation.appKey, rotation.payload.refreshToken); err != nil {
		logger.FromGin(c).Error("failed to revoke provider refresh token after local rotation failure",
			zap.String("session_id", rotation.sessionID),
			zap.String("provider_app_key", rotation.appKey),
			zap.Error(err),
		)
	}
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
