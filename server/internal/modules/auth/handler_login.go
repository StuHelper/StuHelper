package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// GetLoginURL 获取登录 URL
func (h *Handler) GetLoginURL(c *gin.Context) {
	ctx := c.Request.Context()
	url, state, err := h.ssoClient.GetSigninURL(ctx, h.redirectURI)
	if err != nil {
		logger.FromGin(c).Error("failed to generate login URL", zap.Error(err))
		response.InternalError(c, "failed to generate login URL")
		return
	}
	response.Success(c, gin.H{"url": url, "state": state})
}

// GetSignupURL 获取注册 URL
func (h *Handler) GetSignupURL(c *gin.Context) {
	ctx := c.Request.Context()
	url, state, err := h.ssoClient.GetSignupURL(ctx, h.redirectURI)
	if err != nil {
		logger.FromGin(c).Error("failed to generate signup URL", zap.Error(err))
		response.InternalError(c, "failed to generate signup URL")
		return
	}
	response.Success(c, gin.H{"url": url, "state": state})
}

// HandleCallback 处理 OAuth 回调
func (h *Handler) HandleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	if code == "" {
		response.BadRequest(c, "missing authorization code")
		return
	}

	valid, err := h.ssoClient.ValidateState(ctx, state)
	if err != nil {
		logger.FromGin(c).Error("failed to validate state", zap.Error(err))
		response.InternalError(c, "authentication failed")
		return
	}
	if !valid {
		logger.FromGin(c).Warn("invalid or expired state parameter", zap.Int("state_len", len(state)))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "invalid state")
		response.BadRequest(c, "invalid or expired state parameter", errs.ErrOAuthStateInvalid)
		return
	}

	oauthToken, err := h.ssoClient.GetOAuthToken(ctx, code, state)
	if err != nil {
		logger.FromGin(c).Error("SSO token exchange failed", zap.Error(err))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "oauth token error")
		response.Error(c, http.StatusUnauthorized, errs.ErrOAuthFailed, "authentication failed")
		return
	}

	claims, err := h.ssoClient.ParseJwtToken(oauthToken.AccessToken)
	if err != nil {
		logger.FromGin(c).Error("failed to parse JWT token",
			zap.Error(err),
			zap.String("error_type", fmt.Sprintf("%T", err)),
		)
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "jwt parse error")
		response.InternalError(c, "failed to retrieve user information")
		return
	}

	if expectedOrg := h.ssoClient.GetOrganization(); claims.Owner != expectedOrg {
		logger.FromGin(c).Warn("user organization mismatch",
			zap.String("expected", expectedOrg),
			zap.String("actual", claims.Owner),
			zap.String("username", claims.Name),
		)
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "organization mismatch")

		if err := h.ssoClient.DeleteUserSession(claims.Owner, claims.Name); err != nil {
			logger.FromGin(c).Warn("failed to delete casdoor session for mismatched user",
				zap.String("username", claims.Name),
				zap.Error(err),
			)
		}

		ssoLogoutURL := fmt.Sprintf("%s/api/logout", h.ssoEndpoint)
		response.ErrorWithDetails(c, http.StatusForbidden, errs.ErrForbidden,
			"user does not belong to this application",
			gin.H{"ssoLogoutURL": ssoLogoutURL},
		)
		return
	}

	// 先组装 userInfo — 如果这一步降级失败，不写 token 也不写 cookie，保持纯失败
	warmCtx, warmCancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer warmCancel()
	if _, err := h.ssoClient.GetCachedUserByID(warmCtx, claims.Id); err != nil {
		logger.FromGin(c).Warn("failed to pre-warm user cache",
			zap.String("user_id", claims.Id),
			zap.Error(err),
		)
	}

	userInfo, err := h.buildUserInfo(ctx, claims.Id, claims.Name, claims.DisplayName, claims.Email, optionalString(claims.Avatar), claims.IsAdmin)
	if err != nil {
		logger.FromGin(c).Error("failed to build current user info after callback", zap.Error(err))
		response.ServiceUnavailable(c, "identity service temporarily unavailable")
		return
	}

	// userInfo 组装成功后才 track token 和写 cookie
	if err := h.trackLoginTokens(ctx, claims.Id, oauthToken.AccessToken, oauthToken.RefreshToken); err != nil {
		logger.FromGin(c).Error("failed to track login tokens",
			zap.String("user_id", claims.Id),
			zap.Error(err),
		)
		response.InternalError(c, "authentication failed")
		return
	}

	if err := h.setTokenCookies(c, oauthToken.AccessToken, oauthToken.RefreshToken); err != nil {
		// cookie 写入失败，回滚已 track 的 token，避免 Redis 中残留幽灵会话
		h.rollbackTrackedTokens(ctx, claims.Id, oauthToken.AccessToken, oauthToken.RefreshToken)
		response.InternalError(c, "authentication failed")
		return
	}

	audit.LogSuccess(audit.EventUserLogin, claims.Id, claims.Name, c.ClientIP(), c.Request.UserAgent(), requestID)

	response.Success(c, gin.H{
		"user":      userInfo,
		"expiresIn": h.tokenConfig.AccessTokenTTL,
	})
}

func (h *Handler) trackLoginTokens(ctx context.Context, userID, accessToken, refreshToken string) error {
	if err := h.tokenService.GetBlacklist().TrackUserToken(
		ctx,
		userID,
		accessToken,
		h.tokenService.GetRefreshTokenTTL(),
	); err != nil {
		return err
	}

	if err := h.tokenService.GetBlacklist().TrackUserToken(
		ctx,
		userID,
		refreshToken,
		h.tokenService.GetRefreshTokenTTL(),
	); err != nil {
		if rollbackErr := h.tokenService.GetBlacklist().UntrackUserToken(ctx, userID, accessToken); rollbackErr != nil {
			logger.L().Error("CRITICAL: failed to rollback token tracking, orphan token in Redis",
				zap.String("user_id", userID),
				zap.Error(rollbackErr),
			)
		}
		return err
	}

	return nil
}

// rollbackTrackedTokens 回滚已 track 的 access + refresh token，用于后续步骤（如 cookie 写入）失败时清理
func (h *Handler) rollbackTrackedTokens(ctx context.Context, userID, accessToken, refreshToken string) {
	if err := h.tokenService.GetBlacklist().UntrackUserToken(ctx, userID, accessToken); err != nil {
		logger.L().Error("failed to rollback tracked access token",
			zap.String("user_id", userID),
			zap.Error(err),
		)
	}
	if err := h.tokenService.GetBlacklist().UntrackUserToken(ctx, userID, refreshToken); err != nil {
		logger.L().Error("failed to rollback tracked refresh token",
			zap.String("user_id", userID),
			zap.Error(err),
		)
	}
}
