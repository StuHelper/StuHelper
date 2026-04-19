package auth

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

// requireTrackedNativeRefreshSession 强制 native OIDC refresh 命中已追踪的服务端 session。
func (h *Handler) requireTrackedNativeRefreshSession(c *gin.Context, fromBody bool, refreshToken string) bool {
	if !fromBody || token.IsSelfSignedToken(refreshToken) {
		return true
	}

	return h.requireTrackedSession(c, h.getSessionID(c, ""), "missing native session id", "invalid native session id")
}

// requireTrackedNativeLogoutSession 防止 native OIDC 在缺失 tracked session 时仅撤销 access token。
func (h *Handler) requireTrackedNativeLogoutSession(c *gin.Context, accessToken, refreshToken, sessionID string) bool {
	if refreshToken != "" || accessToken == "" || token.IsSelfSignedToken(accessToken) {
		return true
	}

	return h.requireTrackedSession(c, sessionID, "missing native session id", "invalid native session id")
}

func (h *Handler) requireTrackedSession(c *gin.Context, sessionID, missingMessage, invalidMessage string) bool {
	if sessionID == "" {
		response.Unauthorized(c, missingMessage, errs.ErrTokenInvalid)
		return false
	}

	session, err := h.tokenService.GetSessionStore().Get(c.Request.Context(), sessionID)
	if err != nil {
		logger.FromGin(c).Error("failed to load tracked session",
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
		response.ServiceUnavailable(c, "service temporarily unavailable")
		return false
	}
	if session == nil {
		response.Unauthorized(c, invalidMessage, errs.ErrTokenInvalid)
		return false
	}

	return true
}
