package auth

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

func (h *Handler) rejectRefreshReuse(c *gin.Context, refreshToken string) {
	ref, err := h.lookupRefreshTokenRef(c, refreshToken)
	if err != nil {
		logger.FromGin(c).Error("failed to resolve refresh token reuse",
			zap.Error(err),
		)
		response.ServiceUnavailable(c, "service temporarily unavailable")
		return
	}
	if ref == nil {
		response.Unauthorized(c, "refresh token revoked", errs.ErrTokenRevoked)
		return
	}

	if err := h.svc.RevokeAllSessions(c.Request.Context(), ref.UserID); err != nil {
		logger.FromGin(c).Error("failed to revoke sessions after refresh token reuse",
			zap.String("user_id", ref.UserID),
			zap.String("session_id", ref.SessionID),
			zap.Error(err),
		)
		response.ServiceUnavailable(c, "service temporarily unavailable")
		return
	}

	audit.Log(audit.Event{
		Type:         audit.EventTokenRevoked,
		ActorType:    "system",
		UserID:       ref.UserID,
		IP:           c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		RequestID:    middleware.GetRequestID(c),
		ResourceType: "auth.session",
		ResourceID:   ref.SessionID,
		Action:       "refresh_reuse_detected",
		Result:       "failure",
		Reason:       "refresh token reuse detected",
	})
	response.Unauthorized(c, "refresh token reuse detected", errs.ErrTokenRevoked)
}

func (h *Handler) lookupRefreshTokenRef(c *gin.Context, refreshToken string) (*tokenRefreshRef, error) {
	hash, err := hashTokenForSession(refreshToken)
	if err != nil {
		return nil, err
	}
	ref, err := h.tokenService.GetSessionStore().LookupRefreshTokenHash(c.Request.Context(), hash)
	if err != nil {
		return nil, err
	}
	if ref == nil {
		return nil, nil
	}
	if ref.SessionID == "" || ref.UserID == "" {
		return nil, errors.New("refresh token ref is incomplete")
	}
	return &tokenRefreshRef{SessionID: ref.SessionID, UserID: ref.UserID}, nil
}

type tokenRefreshRef struct {
	SessionID string
	UserID    string
}
