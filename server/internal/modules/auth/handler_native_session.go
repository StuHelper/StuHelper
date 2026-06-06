package auth

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

// requireTrackedNativeRefreshSession 强制 native OIDC refresh 命中已追踪的服务端 session。
func (h *Handler) requireTrackedNativeRefreshSession(c *gin.Context, fromBody bool, refreshToken string) bool {
	if !fromBody {
		return true
	}

	return h.requireTrackedSession(
		c,
		h.getSessionID(c, ""),
		"missing native session id",
		"invalid native session id",
		trackedSessionExpectation{refreshToken: refreshToken},
	)
}

// requireTrackedNativeLogoutSession 防止 native OIDC 在缺失 tracked session 时仅撤销 access token。
func (h *Handler) requireTrackedNativeLogoutSession(c *gin.Context, accessToken, refreshToken, sessionID string) bool {
	if refreshToken != "" || accessToken == "" {
		return true
	}

	return h.requireTrackedSession(
		c,
		sessionID,
		"missing native session id",
		"invalid native session id",
		trackedSessionExpectation{
			userID:      middleware.GetUserID(c),
			accessToken: accessToken,
		},
	)
}

func (h *Handler) requireTrackedSession(
	c *gin.Context,
	sessionID, missingMessage, invalidMessage string,
	expectation trackedSessionExpectation,
) bool {
	var err error
	sessionID, err = normalizeRequiredSessionID(sessionID)
	if err != nil {
		response.Unauthorized(c, missingMessage, errs.ErrTokenInvalid)
		return false
	}

	session, err := loadTrackedSession(c.Request.Context(), h.tokenService.GetSessionStore(), sessionID)
	if err != nil {
		if errors.Is(err, token.ErrSessionNotFound) {
			response.Unauthorized(c, invalidMessage, errs.ErrTokenInvalid)
			return false
		}
		logger.FromGin(c).Error("failed to load tracked session",
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
		response.ServiceUnavailable(c, "service temporarily unavailable")
		return false
	}
	if err := validateTrackedSession(session, expectation); err != nil {
		if errors.Is(err, errSessionUserMismatch) ||
			errors.Is(err, errSessionAccessTokenMismatch) ||
			errors.Is(err, errSessionRefreshTokenMismatch) {
			response.Unauthorized(c, invalidMessage, errs.ErrTokenInvalid)
			return false
		}
		logger.FromGin(c).Error("failed to validate tracked session",
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
		response.ServiceUnavailable(c, "service temporarily unavailable")
		return false
	}

	return true
}
