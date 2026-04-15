package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

// Logout 登出当前设备
func (h *Handler) Logout(c *gin.Context) {
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)
	requestID := middleware.GetRequestID(c)

	// 与 AuthMiddleware 保持一致：优先从 Authorization: Bearer 读 token，
	// 再回退 cookie。确保 Bearer 客户端调用 /logout 时也能正确撤销 access token。
	accessToken := middleware.GetAccessToken(c)

	var refreshToken string
	if v, err := c.Cookie(middleware.CookieRefreshToken); err == nil {
		refreshToken = v
	}

	if err := h.svc.RevokeCurrentSession(c.Request.Context(), userID, accessToken, refreshToken); err != nil {
		logger.FromGin(c).Error("failed to revoke current session",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		// 仍然清除客户端 cookie（减少攻击面），但不向客户端承诺撤销成功
		h.clearTokenCookies(c)
		audit.LogFailure(audit.EventUserLogout, c.ClientIP(), c.Request.UserAgent(), requestID, "server-side revocation failed")
		response.InternalError(c, "logout partially failed: server-side revocation unsuccessful")
		return
	}

	h.clearTokenCookies(c)
	audit.LogSuccess(audit.EventUserLogout, userID, username, c.ClientIP(), c.Request.UserAgent(), requestID)

	response.Success(c, gin.H{"message": "logout successful"})
}

// LogoutAll 全设备登出
func (h *Handler) LogoutAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)
	requestID := middleware.GetRequestID(c)

	if err := h.svc.RevokeAllSessions(c.Request.Context(), userID); err != nil {
		logger.FromGin(c).Error("failed to revoke all tokens",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		response.InternalError(c, "failed to logout from all devices")
		return
	}

	h.clearTokenCookies(c)
	audit.LogSuccess(audit.EventUserLogoutAll, userID, username, c.ClientIP(), c.Request.UserAgent(), requestID)
	response.Success(c, gin.H{"message": "logged out from all devices"})
}

// RefreshToken 使用 refresh token 获取新的 token 对
func (h *Handler) RefreshToken(c *gin.Context) {
	refreshTokenStr, err := c.Cookie(middleware.CookieRefreshToken)
	if err != nil || refreshTokenStr == "" {
		response.Unauthorized(c, "missing refresh token", errs.ErrTokenMissing)
		return
	}

	// 检查 refresh token 是否已被吊销或已被并发请求消费
	blacklisted, err := h.tokenService.GetBlacklist().IsBlacklisted(c.Request.Context(), refreshTokenStr)
	if err != nil {
		response.ServiceUnavailable(c, "service temporarily unavailable")
		return
	}
	if blacklisted {
		response.Unauthorized(c, "refresh token revoked", errs.ErrTokenRevoked)
		return
	}

	releaseConsume, ok := h.consumeRefreshToken(c, refreshTokenStr)
	if !ok {
		return
	}
	success := false
	defer func() {
		if success {
			return
		}
		releaseConsume()
	}()

	// 自签名 JWT refresh token（手机验证码登录签发）
	if token.IsSelfSignedToken(refreshTokenStr) {
		success = h.refreshSelfSignedToken(c, refreshTokenStr)
		return
	}

	// Zitadel OIDC refresh token
	success = h.refreshZitadelToken(c, refreshTokenStr)
}

func (h *Handler) consumeRefreshToken(c *gin.Context, refreshToken string) (func(), bool) {
	consumed, err := h.tokenService.GetBlacklist().TryConsumeRefreshToken(c.Request.Context(), refreshToken, h.tokenService.GetRefreshTokenTTL())
	if err != nil {
		logger.FromGin(c).Warn("failed to reserve refresh token for rotation", zap.Error(err))
		response.ServiceUnavailable(c, "service temporarily unavailable")
		return nil, false
	}
	if !consumed {
		response.Unauthorized(c, "refresh token revoked", errs.ErrTokenRevoked)
		return nil, false
	}

	return func() {
		if err := h.tokenService.GetBlacklist().ReleaseConsumedRefreshToken(c.Request.Context(), refreshToken); err != nil {
			logger.FromGin(c).Warn("failed to release refresh token reservation", zap.Error(err))
		}
	}, true
}

// refreshSelfSignedToken 处理自签名 JWT 的 refresh 逻辑
func (h *Handler) refreshSelfSignedToken(c *gin.Context, refreshTokenStr string) bool {
	hmacKey := crypto.GetHMACKey()
	if len(hmacKey) == 0 {
		logger.FromGin(c).Error("HMAC key not initialized for refresh")
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return false
	}

	oldClaims, err := token.VerifyJWTWithType(hmacKey, refreshTokenStr, token.JWTTokenTypeRefresh)
	if err != nil {
		logger.FromGin(c).Debug("self-signed refresh token verification failed", zap.Error(err))
		h.clearTokenCookies(c)
		response.Unauthorized(c, "invalid refresh token", errs.ErrRefreshTokenInvalid)
		return false
	}

	accessTTL := time.Duration(h.tokenConfig.AccessTokenTTL) * time.Second
	refreshTTL := time.Duration(h.tokenConfig.RefreshTokenTTL) * time.Second

	exists, err := h.svc.UserExistsByExternalID(c.Request.Context(), oldClaims.Sub)
	if err != nil {
		logger.FromGin(c).Error("failed to verify phone-login user during refresh", zap.Error(err))
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return false
	}
	if !exists {
		logger.FromGin(c).Warn("phone-login refresh rejected for missing user", zap.String("user_id", oldClaims.Sub))
		h.clearTokenCookies(c)
		response.Unauthorized(c, "failed to refresh token", errs.ErrRefreshTokenInvalid)
		return false
	}

	newAccessClaims := token.JWTClaims{
		Sub:         oldClaims.Sub,
		Name:        oldClaims.Name,
		Email:       oldClaims.Email,
		DisplayName: oldClaims.DisplayName,
		Avatar:      oldClaims.Avatar,
		Roles:       []string{"user"},
		Typ:         token.JWTTokenTypeAccess,
	}

	newAccessToken, err := token.SignJWT(hmacKey, newAccessClaims, accessTTL)
	if err != nil {
		logger.FromGin(c).Error("failed to sign new access JWT", zap.Error(err))
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return false
	}

	newRefreshClaims := token.JWTClaims{
		Sub:   oldClaims.Sub,
		Name:  oldClaims.Name,
		Roles: []string{"user"},
		Typ:   token.JWTTokenTypeRefresh,
	}
	newRefreshToken, err := token.SignJWT(hmacKey, newRefreshClaims, refreshTTL)
	if err != nil {
		logger.FromGin(c).Error("failed to sign new refresh JWT", zap.Error(err))
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return false
	}

	if err := h.setTokenCookies(c, newAccessToken, newRefreshToken); err != nil {
		response.InternalError(c, "failed to refresh token")
		return false
	}

	// 新 token 签发成功后，旋转 token 对（黑名单旧 + 跟踪新）
	h.svc.RotateTokenPair(c.Request.Context(), oldClaims.Sub, refreshTokenStr, newAccessToken, newRefreshToken)

	response.Success(c, gin.H{
		"message":   "token refreshed successfully",
		"expiresIn": h.tokenConfig.AccessTokenTTL,
	})
	return true
}

// refreshZitadelToken 处理 Zitadel OIDC 的 refresh 逻辑
func (h *Handler) refreshZitadelToken(c *gin.Context, refreshTokenStr string) bool {
	// 调 Zitadel token endpoint 刷新
	newToken, err := h.oidcClient.RefreshToken(c.Request.Context(), refreshTokenStr)
	if err != nil {
		logger.FromGin(c).Error("OIDC token refresh failed", zap.Error(err))
		h.clearTokenCookies(c)
		response.Unauthorized(c, "failed to refresh token", errs.ErrRefreshTokenInvalid)
		return false
	}

	rawIDToken := oidc.ExtractIDToken(newToken)
	if rawIDToken == "" {
		logger.FromGin(c).Error("no id_token in refresh response")
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return false
	}

	// 验证新 ID Token 并提取用户信息（用于 Token 跟踪）
	newClaims, err := h.oidcClient.VerifyIDToken(c.Request.Context(), rawIDToken)
	if err != nil {
		logger.FromGin(c).Error("new ID token verification failed after refresh", zap.Error(err))
		h.clearTokenCookies(c)
		response.InternalError(c, "failed to refresh token")
		return false
	}

	if err := h.setTokenCookies(c, rawIDToken, newToken.RefreshToken); err != nil {
		response.InternalError(c, "failed to refresh token")
		return false
	}

	// 新 token 签发成功后，旋转 token 对（黑名单旧 + 跟踪新）
	h.svc.RotateTokenPair(c.Request.Context(), newClaims.GetUserID(), refreshTokenStr, rawIDToken, newToken.RefreshToken)

	response.Success(c, gin.H{
		"message":   "token refreshed successfully",
		"expiresIn": h.tokenConfig.AccessTokenTTL,
	})
	return true
}
