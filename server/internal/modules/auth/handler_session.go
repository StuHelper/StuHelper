package auth

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// Logout 登出
func (h *Handler) Logout(c *gin.Context) {
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	accessToken, _ := c.Cookie(middleware.CookieAccessToken) //nolint:errcheck // not-found → empty string, skipped below
	if accessToken != "" {
		if err := h.tokenService.GetBlacklist().Add(ctx, accessToken, h.tokenService.GetAccessTokenTTL()); err != nil {
			logger.FromGin(c).Warn("failed to blacklist access token",
				zap.String("user_id", userID),
				zap.Error(err),
			)
		}
	}

	refreshToken, _ := c.Cookie(middleware.CookieRefreshToken) //nolint:errcheck // not-found → empty string, skipped below
	if refreshToken != "" {
		if err := h.tokenService.GetBlacklist().Add(ctx, refreshToken, h.tokenService.GetRefreshTokenTTL()); err != nil {
			logger.FromGin(c).Warn("failed to blacklist refresh token",
				zap.String("user_id", userID),
				zap.Error(err),
			)
		}
	}

	h.clearTokenCookies(c)
	audit.LogSuccess(audit.EventUserLogout, userID, username, c.ClientIP(), c.Request.UserAgent(), requestID)

	response.Success(c, gin.H{
		"message":      "logout successful",
		"ssoLogoutURL": fmt.Sprintf("%s/api/logout", h.ssoEndpoint),
	})
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

// RefreshToken 刷新 Access Token
func (h *Handler) RefreshToken(c *gin.Context) {
	refreshToken, err := c.Cookie(middleware.CookieRefreshToken)
	if err != nil || refreshToken == "" {
		response.Unauthorized(c, "missing refresh token", errs.ErrTokenMissing)
		return
	}

	oldAccessToken, _ := c.Cookie(middleware.CookieAccessToken) //nolint:errcheck // not-found → empty string, best-effort blacklist

	blacklisted, err := h.tokenService.GetBlacklist().IsBlacklisted(c.Request.Context(), refreshToken)
	if err != nil {
		response.ServiceUnavailable(c, "service temporarily unavailable")
		return
	}
	if blacklisted {
		response.Unauthorized(c, "refresh token revoked", errs.ErrTokenRevoked)
		return
	}

	if err := h.tokenService.GetBlacklist().Add(c.Request.Context(), refreshToken, h.tokenService.GetRefreshTokenTTL()); err != nil {
		logger.FromGin(c).Error("failed to blacklist old refresh token, blocking new token issuance", zap.Error(err))
		response.InternalError(c, "failed to refresh token")
		return
	}

	newToken, err := h.ssoClient.RefreshOAuthToken(c.Request.Context(), refreshToken)
	if err != nil {
		logger.FromGin(c).Error("failed to refresh token", zap.Error(err))
		h.clearTokenCookies(c)
		response.Unauthorized(c, "failed to refresh token", errs.ErrRefreshTokenInvalid)
		return
	}

	if err := h.setTokenCookies(c, newToken.AccessToken, newToken.RefreshToken); err != nil {
		response.InternalError(c, "failed to refresh token")
		return
	}

	if claims, err := h.ssoClient.ParseJwtToken(newToken.AccessToken); err == nil {
		h.refreshTrackedTokens(c, claims.Id, refreshToken, oldAccessToken, newToken.AccessToken, newToken.RefreshToken)
	} else {
		logger.FromGin(c).Warn("failed to parse refreshed JWT for token tracking", zap.Error(err))
	}

	response.Success(c, gin.H{
		"message":   "token refreshed successfully",
		"expiresIn": h.tokenConfig.AccessTokenTTL,
	})
}

func (h *Handler) refreshTrackedTokens(c *gin.Context, userID, oldRefreshToken, oldAccessToken, newAccessToken, newRefreshToken string) {
	ctx := c.Request.Context()

	if untrackErr := h.tokenService.GetBlacklist().UntrackUserToken(ctx, userID, oldRefreshToken); untrackErr != nil {
		logger.FromGin(c).Error("failed to untrack old refresh token, may cause token set growth",
			zap.String("user_id", userID),
			zap.Error(untrackErr),
		)
	}
	if oldAccessToken != "" {
		if untrackErr := h.tokenService.GetBlacklist().UntrackUserToken(ctx, userID, oldAccessToken); untrackErr != nil {
			logger.FromGin(c).Error("failed to untrack old access token, may cause token set growth",
				zap.String("user_id", userID),
				zap.Error(untrackErr),
			)
		}
	}
	if trackErr := h.tokenService.GetBlacklist().TrackUserToken(ctx, userID, newAccessToken, h.tokenService.GetRefreshTokenTTL()); trackErr != nil {
		logger.FromGin(c).Warn("failed to track refreshed access token",
			zap.String("user_id", userID),
			zap.Error(trackErr),
		)
	}
	if trackErr := h.tokenService.GetBlacklist().TrackUserToken(ctx, userID, newRefreshToken, h.tokenService.GetRefreshTokenTTL()); trackErr != nil {
		logger.FromGin(c).Warn("failed to track refreshed refresh token",
			zap.String("user_id", userID),
			zap.Error(trackErr),
		)
	}
}
