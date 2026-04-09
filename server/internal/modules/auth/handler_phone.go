package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
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

	if err := h.otpService.CheckPhoneRateLimit(c.Request.Context(), phone); err != nil {
		if errors.Is(err, ErrOTPPhoneRateLimited) {
			response.RateLimitExceeded(c, "too many requests for this phone number")
		} else {
			logger.FromGin(c).Error("failed to check phone OTP rate limit", zap.String("phone", maskPhone(phone)), zap.Error(err))
			response.InternalError(c, "failed to send verification code")
		}
		return
	}

	code, err := h.otpService.Generate(c.Request.Context(), phone)
	if err != nil {
		if errors.Is(err, ErrOTPCooldown) {
			response.RateLimitExceeded(c, "please wait before requesting a new code")
		} else {
			logger.FromGin(c).Error("failed to generate OTP", zap.String("phone", maskPhone(phone)), zap.Error(err))
			response.InternalError(c, "failed to send verification code")
		}
		return
	}

	// 发送短信
	internationalPhone := "+86" + phone
	if err := h.smsService.Send(c.Request.Context(), internationalPhone, code); err != nil {
		if cleanupErr := h.otpService.CleanupCodeOnly(c.Request.Context(), phone); cleanupErr != nil {
			logger.FromGin(c).Warn("failed to cleanup OTP after SMS send failure", zap.String("phone", maskPhone(phone)), zap.Error(cleanupErr))
		}
		logger.FromGin(c).Error("failed to send SMS", zap.String("phone", maskPhone(phone)), zap.Error(err))
		response.InternalError(c, "failed to send verification code")
		return
	}

	response.Success(c, gin.H{
		"message":  "verification code sent",
		"cooldown": int(otpCooldown.Seconds()),
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
		if errors.Is(err, ErrOTPInvalidCode) {
			response.Unauthorized(c, "invalid verification code", errs.ErrPhoneOTPFailed)
		} else if errors.Is(err, ErrOTPExpired) {
			response.Unauthorized(c, "verification code expired", errs.ErrPhoneOTPExpired)
		} else if errors.Is(err, ErrOTPMaxAttempts) {
			response.RateLimitExceeded(c, "too many failed attempts, please request a new code")
		} else {
			logger.FromGin(c).Error("OTP verification error", zap.Error(err))
			response.InternalError(c, "verification failed")
		}
		return
	}

	// 查找或创建用户
	user, err := h.userSyncRepo.UpsertByPhone(c.Request.Context(), phone)
	if err != nil {
		logger.FromGin(c).Error("failed to upsert phone user", zap.String("phone", maskPhone(phone)), zap.Error(err))
		response.InternalError(c, "login failed")
		return
	}

	// 签发自签名 JWT
	hmacKey := crypto.GetHMACKey()
	if len(hmacKey) == 0 {
		logger.FromGin(c).Error("HMAC key not initialized")
		response.InternalError(c, "login failed")
		return
	}

	// 手机登录只授予基础 "user" 角色。
	// 设计决策：手机验证码登录用于快速访问，不授予管理权限。
	// 需要 admin/moderator 等高级角色的用户应使用 Zitadel SSO 登录，
	// 角色由 Zitadel ID Token claims 提供。
	roles := []string{"user"}
	accessTTL := time.Duration(h.tokenConfig.AccessTokenTTL) * time.Second
	refreshTTL := time.Duration(h.tokenConfig.RefreshTokenTTL) * time.Second

	claims := token.JWTClaims{
		Sub:         user.ExternalID,
		Name:        user.Username,
		Email:       user.Email,
		DisplayName: user.Username,
		Roles:       roles,
		Typ:         token.JWTTokenTypeAccess,
	}
	if user.AvatarURL != nil {
		claims.Avatar = *user.AvatarURL
	}

	accessToken, err := token.SignJWT(hmacKey, claims, accessTTL)
	if err != nil {
		logger.FromGin(c).Error("failed to sign JWT", zap.Error(err))
		response.InternalError(c, "login failed")
		return
	}

	// 签发自签名 refresh token（更长有效期，用于刷新 access token）
	// Roles 必须写入 refresh token，refreshSelfSignedToken 刷新时从中复制到新 access token
	refreshClaims := token.JWTClaims{
		Sub:   user.ExternalID,
		Name:  user.Username,
		Roles: roles,
		Typ:   token.JWTTokenTypeRefresh,
	}
	refreshToken, err := token.SignJWT(hmacKey, refreshClaims, refreshTTL)
	if err != nil {
		logger.FromGin(c).Error("failed to sign refresh JWT", zap.Error(err))
		response.InternalError(c, "login failed")
		return
	}

	// 设置 access + refresh cookie
	if err := h.setTokenCookies(c, accessToken, refreshToken); err != nil {
		response.InternalError(c, "login failed")
		return
	}

	// Token 跟踪（支持 LogoutAll）
	if trackErr := h.tokenService.GetBlacklist().TrackUserToken(
		c.Request.Context(), user.ExternalID, accessToken, token.TokenTypeAccess, time.Now().Add(h.tokenService.GetAccessTokenTTL()),
	); trackErr != nil {
		logger.FromGin(c).Warn("failed to track phone login token",
			zap.String("user_id", user.ExternalID),
			zap.Error(trackErr),
		)
	}
	if trackErr := h.tokenService.GetBlacklist().TrackUserToken(
		c.Request.Context(), user.ExternalID, refreshToken, token.TokenTypeRefresh, time.Now().Add(h.tokenService.GetRefreshTokenTTL()),
	); trackErr != nil {
		logger.FromGin(c).Warn("failed to track phone login refresh token",
			zap.String("user_id", user.ExternalID),
			zap.Error(trackErr),
		)
	}

	// 审计日志中使用脱敏的手机号，避免泄露完整号码
	maskedID := "phone:" + maskPhone(phone)
	audit.LogSuccess(audit.EventUserLogin, maskedID, user.Username, c.ClientIP(), c.Request.UserAgent(), requestID)

	// 构建用户响应
	capabilities := capability.ExpandRoles(roles)
	snapshot := buildAccessSnapshot(capabilities)

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
