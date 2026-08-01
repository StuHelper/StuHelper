package auth

import (
	"context"
	"errors"

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
) (organizationAdmin bool, ok bool) {
	if h.oidcSubjectValidator == nil {
		return false, true
	}
	subject := claims.GetUserID()
	organizationAdmin, err := h.oidcSubjectValidator.ValidateOIDCSubject(ctx, subject)
	if err != nil {
		logger.FromGin(c).Warn("OIDC subject validation failed",
			zap.String("user_id", subject),
			zap.Error(err),
		)
		audit.LogFailureContext(ctx, audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, failureReason)
		if errors.Is(err, oidc.ErrProviderUnavailable) {
			response.ServiceUnavailable(c, "authentication provider temporarily unavailable")
			return false, false
		}
		response.Unauthorized(c, "authentication failed", errs.ErrOAuthFailed)
		return false, false
	}
	return organizationAdmin, true
}

func (h *Handler) syncOIDCIdentity(
	ctx context.Context,
	input UserSyncInput,
	organizationAdmin bool,
) error {
	if err := h.svc.SyncOIDCUser(ctx, input); err != nil {
		return err
	}
	if h.organizationAdminSync == nil {
		return nil
	}
	return h.organizationAdminSync.SyncCasdoorOrganizationAdmin(
		ctx,
		input.CasdoorSubject,
		organizationAdmin,
	)
}
