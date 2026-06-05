package auth

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

type unlockAuthAccountRequest struct {
	Phone string `json:"phone" binding:"required"`
}

func (h *Handler) RegisterAdminRoutes(admin *gin.RouterGroup) {
	admin.POST(
		"/auth/account-locks/unlock",
		adminRouteHandlers(
			h.unlockAuthAccount,
			h.adminAuthorizers.AccountLockUpdate,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
}

func adminRouteHandlers(handler gin.HandlerFunc, middlewares ...gin.HandlerFunc) []gin.HandlerFunc {
	handlers := make([]gin.HandlerFunc, 0, len(middlewares)+1)
	for _, mw := range middlewares {
		if mw != nil {
			handlers = append(handlers, mw)
		}
	}
	handlers = append(handlers, handler)
	return handlers
}

func (h *Handler) unlockAuthAccount(c *gin.Context) {
	req, ok := bindUnlockAuthAccountRequest(c)
	if !ok {
		return
	}
	err := h.authFailureGuard.ClearAccountLock(c.Request.Context(), req.Phone)
	if err != nil {
		h.logAuthAccountUnlock(c, req.Phone, err)
		logger.FromGin(c).Error("failed to unlock auth account", zap.Error(err))
		response.ServiceUnavailable(c, "authentication guard unavailable")
		return
	}
	h.logAuthAccountUnlock(c, req.Phone, nil)
	response.Success(c, gin.H{"message": "account lock cleared"})
}

func bindUnlockAuthAccountRequest(c *gin.Context) (unlockAuthAccountRequest, bool) {
	var req unlockAuthAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return unlockAuthAccountRequest{}, false
	}
	if !phoneutil.IsValidMainlandPhone(req.Phone) {
		response.BadRequest(c, "invalid phone number format")
		return unlockAuthAccountRequest{}, false
	}
	return req, true
}

func (h *Handler) logAuthAccountUnlock(c *gin.Context, phone string, err error) {
	audit.LogFromGin(c, authAccountUnlockAuditEvent(phone, err))
}

func authAccountUnlockAuditEvent(phone string, err error) audit.Event {
	result := "success"
	reason := ""
	if err != nil {
		result = "failure"
		reason = errors.New("authentication guard unavailable").Error()
	}
	maskedAccount := "phone:" + maskPhone(phone)
	return audit.Event{
		Type:         audit.EventType("iam.auth.account_unlocked"),
		Category:     "audit",
		ActorType:    "admin",
		ResourceType: "auth.account",
		ResourceID:   maskedAccount,
		Action:       "unlock",
		Result:       result,
		Reason:       reason,
		Details: map[string]any{
			"account": maskedAccount,
		},
	}
}
