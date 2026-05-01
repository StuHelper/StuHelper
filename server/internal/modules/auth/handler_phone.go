package auth

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

const phoneLoginRole = "user"

type verifyPhoneOTPRequest struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

// RequestPhoneOTP 发送手机验证码
func (h *Handler) RequestPhoneOTP(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	phone := strings.TrimSpace(req.Phone)
	if !phoneutil.IsValidMainlandPhone(phone) {
		response.BadRequest(c, "invalid phone number format")
		return
	}

	if h.otpService == nil || h.smsService == nil {
		response.ServiceUnavailable(c, "phone login is not configured")
		return
	}

	if err := h.otpService.IssueCode(c.Request.Context(), phone, h.smsService); err != nil {
		switch {
		case errors.Is(err, ErrOTPPhoneRateLimited):
			response.RateLimitExceeded(c, "too many requests for this phone number")
		case errors.Is(err, ErrOTPCooldown):
			response.RateLimitExceeded(c, "please wait before requesting a new code")
		default:
			logger.FromGin(c).Error("failed to send phone OTP", zap.String("phone", maskPhone(phone)), zap.Error(err))
			response.InternalError(c, "failed to send verification code")
		}
		return
	}

	response.Success(c, gin.H{
		"message":  "verification code sent",
		"cooldown": h.otpService.CooldownSeconds(),
	})
}

// VerifyPhoneOTP 验证手机验证码并登录
func (h *Handler) VerifyPhoneOTP(c *gin.Context) {
	phone, code, ok := bindVerifyPhoneOTPRequest(c)
	if !ok {
		return
	}
	if h.otpService == nil {
		response.ServiceUnavailable(c, "phone login is not configured")
		return
	}
	if !h.requireAuthIPAllowed(c) {
		return
	}
	if !h.verifyPhoneOTPCode(c, phone, code) {
		return
	}
	if !h.clearAuthFailures(c) {
		return
	}
	h.completePhoneLogin(c, phone, middleware.GetRequestID(c))
}

func bindVerifyPhoneOTPRequest(c *gin.Context) (string, string, bool) {
	var req verifyPhoneOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return "", "", false
	}
	phone := strings.TrimSpace(req.Phone)
	code := strings.TrimSpace(req.Code)
	if !phoneutil.IsValidMainlandPhone(phone) {
		response.BadRequest(c, "invalid phone number format")
		return "", "", false
	}
	if len(code) != otpLength {
		response.BadRequest(c, fmt.Sprintf("verification code must be %d digits", otpLength))
		return "", "", false
	}
	return phone, code, true
}

func (h *Handler) verifyPhoneOTPCode(c *gin.Context, phone, code string) bool {
	if err := h.otpService.Verify(c.Request.Context(), phone, code); err != nil {
		return h.handlePhoneOTPVerifyError(c, err)
	}
	return true
}

func (h *Handler) handlePhoneOTPVerifyError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, ErrOTPInvalidCode):
		if h.recordAuthFailure(c) {
			response.Unauthorized(c, "invalid verification code", errs.ErrPhoneOTPFailed)
		}
	case errors.Is(err, ErrOTPExpired):
		if h.recordAuthFailure(c) {
			response.Unauthorized(c, "verification code expired", errs.ErrPhoneOTPExpired)
		}
	case errors.Is(err, ErrOTPMaxAttempts):
		if h.recordAuthFailure(c) {
			response.RateLimitExceeded(c, "too many failed attempts, please request a new code")
		}
	default:
		logger.FromGin(c).Error("OTP verification error", zap.Error(err))
		response.InternalError(c, "verification failed")
	}
	return false
}

func (h *Handler) completePhoneLogin(c *gin.Context, phone, requestID string) {
	user, err := h.svc.SyncPhoneUser(c.Request.Context(), phone)
	if err != nil {
		logger.FromGin(c).Error("failed to upsert phone user", zap.String("phone", maskPhone(phone)), zap.Error(err))
		response.InternalError(c, "login failed")
		return
	}

	// 手机登录只授予基础 "user" 角色。
	// 设计决策：手机验证码登录用于快速访问，不授予管理权限。
	// 需要 admin/moderator 等高级角色的用户应使用 Casdoor SSO 登录，
	// 角色由 Casdoor token claims 提供。
	roles := []string{phoneLoginRole}
	accessToken, refreshToken, ok := h.issuePhoneSession(c, user, roles)
	if !ok {
		return
	}
	if err := h.setTokenCookies(c, accessToken, refreshToken); err != nil {
		response.InternalError(c, "login failed")
		return
	}
	h.writePhoneLoginSuccess(c, user, roles, phone, requestID)
}

func (h *Handler) issuePhoneSession(c *gin.Context, user *PhoneUser, roles []string) (string, string, bool) {
	sessionID, sessIDErr := token.GenerateSessionID()
	if sessIDErr != nil {
		logger.FromGin(c).Error("failed to generate session ID", zap.Error(sessIDErr))
		response.InternalError(c, "login failed")
		return "", "", false
	}

	accessToken, refreshToken, err := h.svc.SignPhoneTokenPair(user, roles, sessionID)
	if err != nil {
		logger.FromGin(c).Error("failed to sign phone JWT pair", zap.Error(err))
		response.InternalError(c, "login failed")
		return "", "", false
	}

	deviceInfo := c.Request.UserAgent()
	if _, sessErr := h.svc.CreateSession(c.Request.Context(), sessionID, user.CasdoorSubject, accessToken, refreshToken, "phone", deviceInfo); sessErr != nil {
		logger.FromGin(c).Error("failed to create session",
			zap.String("user_id", user.CasdoorSubject),
			zap.Error(sessErr),
		)
		response.InternalError(c, "login failed")
		return "", "", false
	}
	return accessToken, refreshToken, true
}

func (h *Handler) writePhoneLoginSuccess(c *gin.Context, user *PhoneUser, roles []string, phone, requestID string) {
	maskedID := "phone:" + maskPhone(phone)
	audit.LogSuccess(audit.EventUserLogin, maskedID, user.Username, c.ClientIP(), c.Request.UserAgent(), requestID)

	snapshot := buildAccessSnapshotForRoles(roles, nil)

	response.Success(c, gin.H{
		"user": gin.H{
			"id":                 user.CasdoorSubject,
			"name":               user.Username,
			"displayName":        user.Username,
			"email":              user.Email,
			"avatar":             user.AvatarURL,
			"roles":              roles,
			"capabilities":       snapshot.Capabilities,
			"globalCapabilities": snapshot.GlobalCapabilities,
			"capabilityGrants":   snapshot.CapabilityGrants,
			"isPlatformAdmin":    false,
			"canAccessAdmin":     capability.CanAccessAdmin(snapshot.Capabilities),
		},
		"expiresIn": h.currentAccessTokenTTLSeconds(),
	})
}

func (h *Handler) requireAuthIPAllowed(c *gin.Context) bool {
	err := h.authFailureGuard.EnsureAllowed(c.Request.Context(), c.ClientIP())
	return h.handleAuthFailureGuardError(c, err)
}

func (h *Handler) recordAuthFailure(c *gin.Context) bool {
	err := h.authFailureGuard.RecordFailure(c.Request.Context(), c.ClientIP())
	if errors.Is(err, ErrAuthIPLocked) {
		audit.Log(audit.Event{
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
		})
	}
	return h.handleAuthFailureGuardError(c, err)
}

func (h *Handler) clearAuthFailures(c *gin.Context) bool {
	err := h.authFailureGuard.ClearFailures(c.Request.Context(), c.ClientIP())
	return h.handleAuthFailureGuardError(c, err)
}

func (h *Handler) handleAuthFailureGuardError(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return true
	case errors.Is(err, ErrAuthIPLocked):
		response.RateLimitExceeded(c, "too many authentication attempts")
	default:
		logger.FromGin(c).Error("auth failure guard error", zap.Error(err))
		response.ServiceUnavailable(c, "authentication guard unavailable")
	}
	return false
}

// maskPhone 隐藏手机号中间四位
func maskPhone(phone string) string {
	if len(phone) < 7 {
		return "***"
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}
