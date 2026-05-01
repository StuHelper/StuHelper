package auth

import (
	"crypto/subtle"
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

var errOIDCRefreshTokenNotRotated = errors.New("oidc refresh token was not rotated")

type oidcRefreshPayload struct {
	rawIDToken   string
	refreshToken string
	userID       string
}

type oidcSessionRotation struct {
	sessionID       string
	userID          string
	oldRefreshToken string
	payload         oidcRefreshPayload
}

// refreshOIDCToken 处理 OIDC provider 的 refresh 逻辑。
func (h *Handler) refreshOIDCToken(c *gin.Context, refreshTokenStr string) bool {
	payload, ok := h.fetchOIDCRefreshPayload(c, refreshTokenStr)
	if !ok {
		return false
	}

	if err := h.setTokenCookies(c, payload.rawIDToken, payload.refreshToken); err != nil {
		response.InternalError(c, "failed to refresh token")
		return false
	}

	sessionID := h.getSessionID(c, "")
	if sessionID != "" {
		h.setSessionCookie(c, sessionID)
	}
	rotation := oidcSessionRotation{
		sessionID:       sessionID,
		userID:          payload.userID,
		oldRefreshToken: refreshTokenStr,
		payload:         payload,
	}
	if !h.rotateOIDCSession(c, rotation) {
		return false
	}

	response.Success(c, h.buildRefreshResponse(c, payload.rawIDToken, payload.refreshToken))
	return true
}

func (h *Handler) fetchOIDCRefreshPayload(c *gin.Context, oldRefreshToken string) (oidcRefreshPayload, bool) {
	newToken, err := h.oidcClient.RefreshToken(c.Request.Context(), oldRefreshToken)
	if err != nil {
		logger.FromGin(c).Error("OIDC token refresh failed", zap.Error(err))
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

	newClaims, err := h.oidcClient.VerifyIDToken(c.Request.Context(), rawIDToken)
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

	logger.FromGin(c).Error("failed to rotate OIDC session", zap.String("session_id", rotation.sessionID), zap.Error(err))
	h.clearTokenCookies(c)
	if errors.Is(err, token.ErrSessionNotFound) ||
		errors.Is(err, errSessionUserMismatch) ||
		errors.Is(err, errSessionRefreshTokenMismatch) {
		response.Unauthorized(c, "invalid native session id", errs.ErrTokenInvalid)
		return false
	}
	response.InternalError(c, "failed to refresh token")
	return false
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
