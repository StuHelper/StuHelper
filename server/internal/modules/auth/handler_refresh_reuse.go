package auth

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/metrics"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
	"github.com/StuHelper/StuHelper/server/internal/pkg/token"
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

	session, err := loadTrackedSession(
		c.Request.Context(),
		h.tokenService.GetSessionStore(),
		ref.SessionID,
	)
	if err != nil {
		if errors.Is(err, token.ErrSessionNotFound) {
			// Logout and logout-all remember the revoked refresh hash before
			// deleting the tracked session. A delayed browser refresh must not
			// turn that ordinary revocation into a false family-reuse incident.
			response.Unauthorized(c, "refresh token revoked", errs.ErrTokenRevoked)
			return
		}
		logger.FromGin(c).Error("failed to load refresh token family",
			zap.Bool("session_present", ref.SessionID != ""),
			zap.Error(err),
		)
		response.ServiceUnavailable(c, "service temporarily unavailable")
		return
	}
	if session.UserID != ref.UserID {
		logger.FromGin(c).Error("refresh token attribution user mismatch",
			zap.String("user_id", ref.UserID),
			zap.Bool("session_present", ref.SessionID != ""),
		)
		response.ServiceUnavailable(c, "service temporarily unavailable")
		return
	}
	if session.RefreshTokenHash == "" || session.RefreshTokenHash == ref.RefreshTokenHash {
		// The current hash can still be visible while logout has blacklisted
		// the token but has not yet deleted the session. Only a different,
		// non-empty current hash proves that this token was superseded by a
		// successful rotation.
		response.Unauthorized(c, "refresh token revoked", errs.ErrTokenRevoked)
		return
	}

	metrics.ObserveRefreshTokenReuse(refreshTokenFamily(refreshToken))
	if err := h.svc.RevokeAllSessions(c.Request.Context(), ref.UserID); err != nil {
		logger.FromGin(c).Error("failed to revoke sessions after refresh token reuse",
			zap.String("user_id", ref.UserID),
			zap.Bool("session_present", ref.SessionID != ""),
			zap.Error(err),
		)
		response.ServiceUnavailable(c, "service temporarily unavailable")
		return
	}

	audit.LogContext(c.Request.Context(), audit.Event{
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

func refreshTokenFamily(refreshToken string) string {
	if token.IsSelfSignedToken(refreshToken) {
		return "self_signed"
	}
	return "oidc"
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
	return &tokenRefreshRef{
		SessionID:        ref.SessionID,
		UserID:           ref.UserID,
		RefreshTokenHash: hash,
	}, nil
}

type tokenRefreshRef struct {
	SessionID        string
	UserID           string
	RefreshTokenHash string
}
