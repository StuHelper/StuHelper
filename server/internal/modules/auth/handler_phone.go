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
		if errors.Is(err, ErrOTPPhoneRateLimited) {
			response.RateLimitExceeded(c, "too many requests for this phone number")
		} else if errors.Is(err, ErrOTPCooldown) {
			response.RateLimitExceeded(c, "please wait before requesting a new code")
		} else {
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
	var req struct {
		Phone string `json:"phone" binding:"required"`
		Code  string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	phone := strings.TrimSpace(req.Phone)
	code := strings.TrimSpace(req.Code)
	requestID := middleware.GetRequestID(c)

	if !phoneutil.IsValidMainlandPhone(phone) {
		response.BadRequest(c, "invalid phone number format")
		return
	}
	if len(code) != otpLength {
		response.BadRequest(c, fmt.Sprintf("verification code must be %d digits", otpLength))
		return
	}

	if h.otpService == nil {
		response.ServiceUnavailable(c, "phone login is not configured")
		return
	}

	// 验证 OTP
	if err := h.otpService.Verify(c.Request.Context(), phone, code); err != nil {
		switch {
		case errors.Is(err, ErrOTPInvalidCode):
			response.Unauthorized(c, "invalid verification code", errs.ErrPhoneOTPFailed)
		case errors.Is(err, ErrOTPExpired):
			response.Unauthorized(c, "verification code expired", errs.ErrPhoneOTPExpired)
		case errors.Is(err, ErrOTPMaxAttempts):
			response.RateLimitExceeded(c, "too many failed attempts, please request a new code")
		default:
			logger.FromGin(c).Error("OTP verification error", zap.Error(err))
			response.InternalError(c, "verification failed")
		}
		return
	}

	// 查找或创建用户
	user, err := h.svc.SyncPhoneUser(c.Request.Context(), phone)
	if err != nil {
		logger.FromGin(c).Error("failed to upsert phone user", zap.String("phone", maskPhone(phone)), zap.Error(err))
		response.InternalError(c, "login failed")
		return
	}

	// 手机登录只授予基础 "user" 角色。
	// 设计决策：手机验证码登录用于快速访问，不授予管理权限。
	// 需要 admin/moderator 等高级角色的用户应使用 Zitadel SSO 登录，
	// 角色由 Zitadel ID Token claims 提供。
	roles := []string{"user"}

	// 创建 session（Token Family）
	sessionID, sessIDErr := token.GenerateSessionID()
	if sessIDErr != nil {
		logger.FromGin(c).Error("failed to generate session ID", zap.Error(sessIDErr))
		response.InternalError(c, "login failed")
		return
	}

	// 签发自签名 JWT 对（含 session ID）
	accessToken, refreshToken, err := h.svc.SignPhoneTokenPair(user, roles, sessionID)
	if err != nil {
		logger.FromGin(c).Error("failed to sign phone JWT pair", zap.Error(err))
		response.InternalError(c, "login failed")
		return
	}

	// 创建服务端 Session —— 必须传入签发 JWT 时使用的同一 sessionID，
	// 否则 JWT 中的 Sid claim 与 session store key 不一致，refresh/revoke 会失效。
	deviceInfo := c.Request.UserAgent()
	if _, sessErr := h.svc.CreateSession(c.Request.Context(), sessionID, user.ExternalID, accessToken, refreshToken, "phone", deviceInfo); sessErr != nil {
		logger.FromGin(c).Error("failed to create session",
			zap.String("user_id", user.ExternalID),
			zap.Error(sessErr),
		)
		response.InternalError(c, "login failed")
		return
	}

	// 设置 access + refresh cookie
	if err := h.setTokenCookies(c, accessToken, refreshToken); err != nil {
		response.InternalError(c, "login failed")
		return
	}

	// 审计日志中使用脱敏的手机号，避免泄露完整号码
	maskedID := "phone:" + maskPhone(phone)
	audit.LogSuccess(audit.EventUserLogin, maskedID, user.Username, c.ClientIP(), c.Request.UserAgent(), requestID)

	// 构建用户响应
	snapshot := buildAccessSnapshotForRoles(roles, nil)

	response.Success(c, gin.H{
		"user": gin.H{
			"id":                 user.ExternalID,
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
		"expiresIn": h.tokenConfig.AccessTokenTTL,
	})
}

// maskPhone 隐藏手机号中间四位
func maskPhone(phone string) string {
	if len(phone) < 7 {
		return "***"
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}
