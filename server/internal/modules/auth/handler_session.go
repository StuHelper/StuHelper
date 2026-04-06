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
	ctx := c.Request.Context()

	// 将当前 access token 加入黑名单并从跟踪集合移除
	if accessToken, err := c.Cookie(middleware.CookieAccessToken); err == nil && accessToken != "" {
		if blErr := h.tokenService.GetBlacklist().Add(ctx, accessToken, h.tokenService.GetAccessTokenTTL()); blErr != nil {
			logger.FromGin(c).Warn("failed to blacklist access token",
				zap.String("user_id", userID),
				zap.Error(blErr),
			)
		}
		if untrackErr := h.tokenService.GetBlacklist().UntrackUserToken(ctx, userID, accessToken, token.TokenTypeAccess); untrackErr != nil {
			logger.FromGin(c).Warn("failed to untrack access token",
				zap.String("user_id", userID),
				zap.Error(untrackErr),
			)
		}
	}

	// 将当前 refresh token 加入黑名单并从跟踪集合移除
	if refreshToken, err := c.Cookie(middleware.CookieRefreshToken); err == nil && refreshToken != "" {
		if blErr := h.tokenService.GetBlacklist().Add(ctx, refreshToken, h.tokenService.GetRefreshTokenTTL()); blErr != nil {
			logger.FromGin(c).Warn("failed to blacklist refresh token",
				zap.String("user_id", userID),
				zap.Error(blErr),
			)
		}
		if untrackErr := h.tokenService.GetBlacklist().UntrackUserToken(ctx, userID, refreshToken, token.TokenTypeRefresh); untrackErr != nil {
			logger.FromGin(c).Warn("failed to untrack refresh token",
				zap.String("user_id", userID),
				zap.Error(untrackErr),
			)
		}
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
	ctx := c.Request.Context()

	if err := h.tokenService.GetBlacklist().RevokeAllUserTokens(
		ctx,
		userID,
		h.tokenService.GetRefreshTokenTTL(),
	); err != nil {
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

	newAccessClaims := token.JWTClaims{
		Sub:         oldClaims.Sub,
		Name:        oldClaims.Name,
		Email:       oldClaims.Email,
		DisplayName: oldClaims.DisplayName,
		Avatar:      oldClaims.Avatar,
		Roles:       oldClaims.Roles,
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
		Sub:  oldClaims.Sub,
		Name: oldClaims.Name,
		Typ:  token.JWTTokenTypeRefresh,
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

	// 新 token 签发成功后，将旧 refresh token 加入黑名单（一次性使用）
	if blErr := h.tokenService.GetBlacklist().Add(c.Request.Context(), refreshTokenStr, h.tokenService.GetRefreshTokenTTL()); blErr != nil {
		logger.FromGin(c).Warn("failed to blacklist old refresh token", zap.Error(blErr))
	}
	if untrackErr := h.tokenService.GetBlacklist().UntrackUserToken(c.Request.Context(), oldClaims.Sub, refreshTokenStr, token.TokenTypeRefresh); untrackErr != nil {
		logger.FromGin(c).Warn("failed to untrack old refresh token",
			zap.String("user_id", oldClaims.Sub),
			zap.Error(untrackErr),
		)
	}

	// 跟踪新 token
	if trackErr := h.tokenService.GetBlacklist().TrackUserToken(
		c.Request.Context(), oldClaims.Sub, newAccessToken, token.TokenTypeAccess, time.Now().Add(h.tokenService.GetAccessTokenTTL()),
	); trackErr != nil {
		logger.FromGin(c).Warn("failed to track refreshed phone token",
			zap.String("user_id", oldClaims.Sub),
			zap.Error(trackErr),
		)
	}
	if trackErr := h.tokenService.GetBlacklist().TrackUserToken(
		c.Request.Context(), oldClaims.Sub, newRefreshToken, token.TokenTypeRefresh, time.Now().Add(h.tokenService.GetRefreshTokenTTL()),
	); trackErr != nil {
		logger.FromGin(c).Warn("failed to track refreshed phone refresh token",
			zap.String("user_id", oldClaims.Sub),
			zap.Error(trackErr),
		)
	}

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

	// 新 token 签发成功后，将旧 refresh token 加入黑名单（一次性使用）
	if blErr := h.tokenService.GetBlacklist().Add(c.Request.Context(), refreshTokenStr, h.tokenService.GetRefreshTokenTTL()); blErr != nil {
		logger.FromGin(c).Warn("failed to blacklist old refresh token", zap.Error(blErr))
	}
	if untrackErr := h.tokenService.GetBlacklist().UntrackUserToken(c.Request.Context(), newClaims.GetUserID(), refreshTokenStr, token.TokenTypeRefresh); untrackErr != nil {
		logger.FromGin(c).Warn("failed to untrack old refresh token",
			zap.String("user_id", newClaims.GetUserID()),
			zap.Error(untrackErr),
		)
	}

	// 跟踪新 Token，支持 LogoutAll 批量撤销
	if trackErr := h.tokenService.GetBlacklist().TrackUserToken(
		c.Request.Context(), newClaims.GetUserID(), rawIDToken, token.TokenTypeAccess, time.Now().Add(h.tokenService.GetAccessTokenTTL()),
	); trackErr != nil {
		logger.FromGin(c).Warn("failed to track refreshed token",
			zap.String("user_id", newClaims.GetUserID()),
			zap.Error(trackErr),
		)
	}
	if newToken.RefreshToken != "" {
		if trackErr := h.tokenService.GetBlacklist().TrackUserToken(
			c.Request.Context(), newClaims.GetUserID(), newToken.RefreshToken, token.TokenTypeRefresh, time.Now().Add(h.tokenService.GetRefreshTokenTTL()),
		); trackErr != nil {
			logger.FromGin(c).Warn("failed to track refreshed refresh token",
				zap.String("user_id", newClaims.GetUserID()),
				zap.Error(trackErr),
			)
		}
	}

	response.Success(c, gin.H{
		"message":   "token refreshed successfully",
		"expiresIn": h.tokenConfig.AccessTokenTTL,
	})
	return true
}
