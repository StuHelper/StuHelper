package auth

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

func (h *Handler) requireAuthAllowed(c *gin.Context, account string) bool {
	if !h.requireAuthIPAllowed(c) {
		return false
	}
	err := h.authFailureGuard.EnsureAccountAllowed(c.Request.Context(), account)
	return h.handleAuthFailureGuardError(c, err)
}

func (h *Handler) requireAuthIPAllowed(c *gin.Context) bool {
	err := h.authFailureGuard.EnsureAllowed(c.Request.Context(), c.ClientIP())
	return h.handleAuthFailureGuardError(c, err)
}

func (h *Handler) recordAuthFailure(c *gin.Context, account string) bool {
	ipErr := h.authFailureGuard.RecordFailure(c.Request.Context(), c.ClientIP())
	accountErr := h.authFailureGuard.RecordAccountFailure(c.Request.Context(), account)
	h.auditAuthIPLock(c, ipErr)
	h.auditAuthAccountLock(c, account, accountErr)
	return h.handleAuthFailureGuardError(c, errors.Join(ipErr, accountErr))
}

func (h *Handler) clearAuthFailures(c *gin.Context, account string) bool {
	ipErr := h.authFailureGuard.ClearFailures(c.Request.Context(), c.ClientIP())
	accountErr := h.authFailureGuard.ClearAccountFailures(c.Request.Context(), account)
	return h.handleAuthFailureGuardError(c, errors.Join(ipErr, accountErr))
}

func (h *Handler) handleAuthFailureGuardError(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return true
	case isAuthLockError(err):
		response.RateLimitExceeded(c, "too many authentication attempts")
	default:
		logger.FromGin(c).Error("auth failure guard error", zap.Error(err))
		response.ServiceUnavailable(c, "authentication guard unavailable")
	}
	return false
}

func (h *Handler) auditAuthIPLock(c *gin.Context, err error) {
	if !errors.Is(err, ErrAuthIPLocked) {
		return
	}
	audit.LogContext(c.Request.Context(), authIPLockAuditEvent(c))
}

func (h *Handler) auditAuthAccountLock(c *gin.Context, account string, err error) {
	switch {
	case errors.Is(err, ErrAuthAccountHardLocked):
		audit.LogContext(c.Request.Context(), authAccountLockAuditEvent(c, account, "hard"))
	case errors.Is(err, ErrAuthAccountSoftLocked):
		audit.LogContext(c.Request.Context(), authAccountLockAuditEvent(c, account, "soft"))
	}
}

func authIPLockAuditEvent(c *gin.Context) audit.Event {
	return audit.Event{
		Type:         audit.EventType("iam.auth.ip_locked"),
		Category:     "audit",
		ActorType:    "system",
		IP:           c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		RequestID:    middleware.GetRequestID(c),
		ResourceType: "auth.ip",
		ResourceID:   c.ClientIP(),
		Action:       "lock",
		Result:       "failure",
		Reason:       "too many failed authentication attempts",
	}
}

func authAccountLockAuditEvent(c *gin.Context, account string, lockType string) audit.Event {
	maskedAccount := "phone:" + maskPhone(account)
	return audit.Event{
		Type:         audit.EventType("iam.auth.account_locked"),
		Category:     "audit",
		ActorType:    "system",
		IP:           c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		RequestID:    middleware.GetRequestID(c),
		ResourceType: "auth.account",
		ResourceID:   maskedAccount,
		Action:       "lock",
		Result:       "failure",
		Reason:       "too many failed authentication attempts",
		Details: map[string]any{
			"account":   maskedAccount,
			"lock_type": lockType,
		},
	}
}

func isAuthLockError(err error) bool {
	return errors.Is(err, ErrAuthIPLocked) ||
		errors.Is(err, ErrAuthAccountSoftLocked) ||
		errors.Is(err, ErrAuthAccountHardLocked)
}
