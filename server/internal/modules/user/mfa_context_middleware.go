package user

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

type MFAContextRepository interface {
	GetInternalUserID(ctx context.Context, casdoorSubject string) (int64, error)
	GetMFAEnrollment(ctx context.Context, userID int64) (*MFAEnrollment, error)
}

func MFAContextMiddleware(repo MFAContextRepository) gin.HandlerFunc {
	if repo == nil {
		panic("user.MFAContextMiddleware: repo is required")
	}
	return func(c *gin.Context) {
		if loadMFAContext(c, repo) {
			c.Next()
		}
	}
}

func loadMFAContext(c *gin.Context, repo MFAContextRepository) bool {
	casdoorSubject := middleware.GetUserID(c)
	if casdoorSubject == "" {
		response.Unauthorized(c, "authentication required", errs.ErrLoginRequired)
		c.Abort()
		return false
	}

	userID, err := repo.GetInternalUserID(c.Request.Context(), casdoorSubject)
	if err != nil {
		return respondMFAContextUserError(c, casdoorSubject, err)
	}
	if userID <= 0 {
		logger.FromGin(c).Warn("resolved invalid MFA internal user ID",
			zap.String("casdoor_subject", casdoorSubject),
			zap.Int64("user_id", userID),
		)
		response.Forbidden(c, "user has not completed provisioning", errs.ErrUserNotFound)
		c.Abort()
		return false
	}
	enrollment, err := repo.GetMFAEnrollment(c.Request.Context(), userID)
	if err != nil {
		logger.FromGin(c).Error("failed to load MFA enrollment",
			zap.Int64("internal_user_id", userID),
			zap.Error(err),
		)
		response.ServiceUnavailable(c, "mfa enrollment unavailable")
		c.Abort()
		return false
	}

	middleware.SetMFAContext(c, middleware.MFAContext{
		EnrollmentActive: mfaEnrollmentUsable(enrollment),
		ProofVerifiedAt:  middleware.GetMFAProofVerifiedAt(c),
	})
	return true
}

func respondMFAContextUserError(c *gin.Context, casdoorSubject string, err error) bool {
	if errors.Is(err, pgx.ErrNoRows) {
		response.Forbidden(c, "user has not completed provisioning", errs.ErrUserNotFound)
		c.Abort()
		return false
	}
	logger.FromGin(c).Error("failed to resolve MFA user",
		zap.String("casdoor_subject", casdoorSubject),
		zap.Error(err),
	)
	response.ServiceUnavailable(c, "mfa enrollment unavailable")
	c.Abort()
	return false
}

func mfaEnrollmentUsable(enrollment *MFAEnrollment) bool {
	return enrollment != nil && enrollment.Active && !enrollment.ResetRequired
}
