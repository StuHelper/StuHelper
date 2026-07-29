package auth

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/oidc"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

func (h *Handler) validateOIDCSubjectForLogin(
	c *gin.Context,
	ctx context.Context,
	claims *oidc.Claims,
	requestID string,
	failureReason string,
) bool {
	if h.oidcSubjectValidator == nil {
		return true
	}
	subject := claims.GetUserID()
	if err := h.oidcSubjectValidator.ValidateOIDCSubject(ctx, subject); err != nil {
		logger.FromGin(c).Warn("OIDC subject validation failed",
			zap.String("user_id", subject),
			zap.Error(err),
		)
		audit.LogFailureContext(ctx, audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, failureReason)
		response.Unauthorized(c, "authentication failed", errs.ErrOAuthFailed)
		return false
	}
	return true
}
