package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/sso"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

// Handler 认证处理器
type Handler struct {
	ssoClient      *sso.Client
	tokenService   *token.Service
	tokenConfig    config.TokenConfig
	redirectURI    string
	appName        string
	ssoEndpoint    string
	refreshLimiter *middleware.RedisRateLimiter
}

// NewHandler 创建认证处理器
func NewHandler(cfg *config.Config, tokenService *token.Service, rdb *redis.Client, ssoClient *sso.Client) *Handler {
	return &Handler{
		ssoClient:    ssoClient,
		tokenService: tokenService,
		tokenConfig:  cfg.Token,
		redirectURI:  cfg.Casdoor.RedirectURI,
		appName:      cfg.Casdoor.Application,
		ssoEndpoint:  cfg.Casdoor.Endpoint,
		// RefreshToken 限制: 每分钟最多 10 次
		refreshLimiter: middleware.NewRedisRateLimiter(rdb, 10, time.Minute),
	}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.GET("/login", h.GetLoginURL)
		auth.GET("/signup", h.GetSignupURL)
		auth.GET("/callback", h.HandleCallback)
		auth.POST("/refresh", middleware.RateLimitMiddleware(h.refreshLimiter), h.RefreshToken)
		auth.GET("/me", middleware.AuthMiddleware(h.tokenService), h.GetCurrentUser)
		auth.POST("/logout", middleware.AuthMiddleware(h.tokenService), h.Logout)
		auth.POST("/logout-all", middleware.AuthMiddleware(h.tokenService), h.LogoutAll)
	}
}

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

	// 验证并消费 state（一次性使用，防止 CSRF 和回放攻击）
	valid, err := h.ssoClient.ValidateState(ctx, state)
	if err != nil {
		logger.FromGin(c).Error("failed to validate state", zap.Error(err))
		response.InternalError(c, "authentication failed")
		return
	}
	if !valid {
		logger.FromGin(c).Warn("invalid or expired state parameter",
			zap.Int("state_len", len(state)),
		)
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "invalid state")
		response.BadRequest(c, "invalid or expired state parameter", errs.ErrOAuthStateInvalid)
		return
	}

	// 获取 OAuth Token（state 参数传递给 Casdoor，但我们已经自己验证过了）
	oauthToken, err := h.ssoClient.GetOAuthToken(c.Request.Context(), code, h.appName)
	if err != nil {
		logger.FromGin(c).Error("SSO token exchange failed", zap.Error(err))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "oauth token error")
		response.Error(c, http.StatusUnauthorized, errs.ErrOAuthFailed, "authentication failed")
		return
	}

	// 解析 JWT 获取用户信息
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

	// 校验用户所属组织，拒绝非本应用组织的用户登录
	if expectedOrg := h.ssoClient.GetOrganization(); claims.Owner != expectedOrg {
		logger.FromGin(c).Warn("user organization mismatch",
			zap.String("expected", expectedOrg),
			zap.String("actual", claims.Owner),
			zap.String("username", claims.Name),
		)
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "organization mismatch")

		// 删除该用户的 Casdoor 会话，防止下次登录自动跳过
		if err := h.ssoClient.DeleteUserSession(claims.Owner, claims.Name); err != nil {
			logger.FromGin(c).Warn("failed to delete casdoor session for mismatched user",
				zap.String("username", claims.Name),
				zap.Error(err),
			)
		}

		// 返回 SSO 登出 URL，让前端通过顶级导航清除浏览器中的 Casdoor session cookie
		ssoLogoutURL := fmt.Sprintf("%s/api/logout", h.ssoEndpoint)
		response.ErrorWithDetails(c, http.StatusForbidden, errs.ErrForbidden,
			"user does not belong to this application",
			gin.H{"ssoLogoutURL": ssoLogoutURL},
		)
		return
	}

	// 追踪用户 token（用于全设备登出）
	// 硬错误：追踪失败阻断登录，否则 LogoutAll 对该用户无效
	if err := h.tokenService.GetBlacklist().TrackUserToken(
		ctx,
		claims.Id,
		oauthToken.AccessToken,
		h.tokenService.GetRefreshTokenTTL(),
	); err != nil {
		logger.FromGin(c).Error("failed to track user token, blocking login",
			zap.String("user_id", claims.Id),
			zap.Error(err),
		)
		response.InternalError(c, "authentication failed")
		return
	}
	// 同时追踪 refresh token，确保 LogoutAll 可撤销
	if err := h.tokenService.GetBlacklist().TrackUserToken(
		ctx,
		claims.Id,
		oauthToken.RefreshToken,
		h.tokenService.GetRefreshTokenTTL(),
	); err != nil {
		logger.FromGin(c).Error("failed to track user refresh token, blocking login",
			zap.String("user_id", claims.Id),
			zap.Error(err),
		)
		// 回滚已追踪的 access token
		_ = h.tokenService.GetBlacklist().UntrackUserToken(ctx, claims.Id, oauthToken.AccessToken)
		response.InternalError(c, "authentication failed")
		return
	}

	// 设置 HttpOnly Cookie
	if err := h.setTokenCookies(c, oauthToken.AccessToken, oauthToken.RefreshToken); err != nil {
		response.InternalError(c, "authentication failed")
		return
	}

	// 预热用户缓存：使用独立超时避免 Casdoor 响应慢时延迟整个登录响应
	// 使用 context.WithoutCancel 保留追踪信息，但不受请求取消影响
	// 预热失败不阻断登录，仅记录警告
	warmCtx, warmCancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer warmCancel()
	if _, err := h.ssoClient.GetCachedUserByID(warmCtx, claims.Id); err != nil {
		logger.FromGin(c).Warn("failed to pre-warm user cache",
			zap.String("user_id", claims.Id),
			zap.Error(err),
		)
	}

	// 记录登录成功审计日志
	audit.LogSuccess(audit.EventUserLogin, claims.Id, claims.Name, c.ClientIP(), c.Request.UserAgent(), requestID)

	response.Success(c, gin.H{
		"user": gin.H{
			"id":          claims.Id,
			"name":        claims.Name,
			"displayName": claims.DisplayName,
			"email":       claims.Email,
			"avatar":      claims.Avatar,
			"isAdmin":     claims.IsAdmin,
		},
		"expiresIn": h.tokenConfig.AccessTokenTTL,
	})
}

// GetCurrentUser 获取当前用户信息
func (h *Handler) GetCurrentUser(c *gin.Context) {
	response.Success(c, gin.H{
		"id":          middleware.GetUserID(c),
		"name":        middleware.GetUsername(c),
		"email":       middleware.GetEmail(c),
		"displayName": middleware.GetDisplayName(c),
		"isAdmin":     middleware.GetIsAdmin(c),
	})
}

// Logout 登出
func (h *Handler) Logout(c *gin.Context) {
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)
	requestID := middleware.GetRequestID(c)

	ctx := c.Request.Context()

	// 将 access token 加入黑名单
	accessToken, _ := c.Cookie(middleware.CookieAccessToken)
	if accessToken != "" {
		if err := h.tokenService.GetBlacklist().Add(ctx, accessToken, h.tokenService.GetAccessTokenTTL()); err != nil {
			logger.FromGin(c).Warn("failed to blacklist access token",
				zap.String("user_id", userID),
				zap.Error(err),
			)
		}
	}

	// 将 refresh token 加入黑名单，防止攻击者用旧 refresh token 获取新 access token
	refreshToken, _ := c.Cookie(middleware.CookieRefreshToken)
	if refreshToken != "" {
		if err := h.tokenService.GetBlacklist().Add(ctx, refreshToken, h.tokenService.GetRefreshTokenTTL()); err != nil {
			logger.FromGin(c).Warn("failed to blacklist refresh token",
				zap.String("user_id", userID),
				zap.Error(err),
			)
		}
	}

	// 清除 Cookie
	h.clearTokenCookies(c)

	// 记录登出审计日志
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

	// 撤销用户所有 token
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

	// 清除当前设备的 Cookie
	h.clearTokenCookies(c)

	// 记录全设备登出审计日志
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

	// 提前读取旧 access token，在 setTokenCookies 覆盖响应 Cookie 之前捕获
	oldAccessToken, _ := c.Cookie(middleware.CookieAccessToken)

	// 刷新前检查 refresh token 是否已被撤销
	blacklisted, err := h.tokenService.GetBlacklist().IsBlacklisted(c.Request.Context(), refreshToken)
	if err != nil {
		response.ServiceUnavailable(c, "service temporarily unavailable")
		return
	}
	if blacklisted {
		response.Unauthorized(c, "refresh token revoked", errs.ErrTokenRevoked)
		return
	}

	// 先将旧 refresh token 加入黑名单，再签发新 token
	// 顺序很重要：防止旧 token 在新 token 签发后、黑名单写入前被重放
	if err := h.tokenService.GetBlacklist().Add(c.Request.Context(), refreshToken, h.tokenService.GetRefreshTokenTTL()); err != nil {
		logger.FromGin(c).Error("failed to blacklist old refresh token, blocking new token issuance", zap.Error(err))
		response.InternalError(c, "failed to refresh token")
		return
	}

	// 旧 token 已入黑名单，安全签发新 token
	newToken, err := h.ssoClient.RefreshOAuthToken(c.Request.Context(), refreshToken)
	if err != nil {
		logger.FromGin(c).Error("failed to refresh token", zap.Error(err))
		h.clearTokenCookies(c)
		response.Unauthorized(c, "failed to refresh token", errs.ErrRefreshTokenInvalid)
		return
	}

	// 设置新的 Cookie
	if err := h.setTokenCookies(c, newToken.AccessToken, newToken.RefreshToken); err != nil {
		response.InternalError(c, "failed to refresh token")
		return
	}

	// 追踪新的 access token 和 refresh token
	if claims, err := h.ssoClient.ParseJwtToken(newToken.AccessToken); err == nil {
		// 从用户 token 集合中移除旧的 refresh token，防止集合无限增长
		if untrackErr := h.tokenService.GetBlacklist().UntrackUserToken(
			c.Request.Context(),
			claims.Id,
			refreshToken,
		); untrackErr != nil {
			logger.FromGin(c).Warn("failed to untrack old refresh token",
				zap.String("user_id", claims.Id),
				zap.Error(untrackErr),
			)
		}
		// 同时移除旧的 access token（在函数开头已从请求 Cookie 中读取）
		if oldAccessToken != "" {
			if untrackErr := h.tokenService.GetBlacklist().UntrackUserToken(
				c.Request.Context(),
				claims.Id,
				oldAccessToken,
			); untrackErr != nil {
				logger.FromGin(c).Warn("failed to untrack old access token",
					zap.String("user_id", claims.Id),
					zap.Error(untrackErr),
				)
			}
		}
		if trackErr := h.tokenService.GetBlacklist().TrackUserToken(
			c.Request.Context(),
			claims.Id,
			newToken.AccessToken,
			h.tokenService.GetRefreshTokenTTL(),
		); trackErr != nil {
			logger.FromGin(c).Warn("failed to track refreshed access token",
				zap.String("user_id", claims.Id),
				zap.Error(trackErr),
			)
		}
		// 同时追踪新 refresh token，确保 LogoutAll 可撤销
		if trackErr := h.tokenService.GetBlacklist().TrackUserToken(
			c.Request.Context(),
			claims.Id,
			newToken.RefreshToken,
			h.tokenService.GetRefreshTokenTTL(),
		); trackErr != nil {
			logger.FromGin(c).Warn("failed to track refreshed refresh token",
				zap.String("user_id", claims.Id),
				zap.Error(trackErr),
			)
		}
	} else {
		logger.FromGin(c).Warn("failed to parse refreshed JWT for token tracking", zap.Error(err))
	}

	response.Success(c, gin.H{
		"message":   "token refreshed successfully",
		"expiresIn": h.tokenConfig.AccessTokenTTL,
	})
}

// setTokenCookies 设置 Token Cookie
// 返回 error 而非直接写响应，由调用方统一处理错误响应，避免双重 HTTP 写入
func (h *Handler) setTokenCookies(c *gin.Context, accessToken, refreshToken string) error {
	// 设置 SameSite 属性防止 CSRF
	c.SetSameSite(http.SameSiteStrictMode)

	csrfToken, err := middleware.GenerateCSRFToken()
	if err != nil {
		logger.FromGin(c).Error("failed to generate CSRF token", zap.Error(err))
		return err
	}
	h.setCSRFCookie(c, csrfToken)

	// Access Token Cookie
	c.SetCookie(
		middleware.CookieAccessToken,
		accessToken,
		h.tokenConfig.AccessTokenTTL,
		"/",
		h.tokenConfig.CookieDomain,
		h.tokenConfig.CookieSecure,
		true, // HttpOnly
	)

	// Refresh Token Cookie
	c.SetCookie(
		middleware.CookieRefreshToken,
		refreshToken,
		h.tokenConfig.RefreshTokenTTL,
		"/api/v1/auth/refresh",
		h.tokenConfig.CookieDomain,
		h.tokenConfig.CookieSecure,
		true, // HttpOnly
	)

	return nil
}

// clearTokenCookies 清除 Token Cookie
func (h *Handler) clearTokenCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		middleware.CookieAccessToken,
		"",
		-1,
		"/",
		h.tokenConfig.CookieDomain,
		h.tokenConfig.CookieSecure,
		true,
	)
	c.SetCookie(
		middleware.CookieRefreshToken,
		"",
		-1,
		"/api/v1/auth/refresh",
		h.tokenConfig.CookieDomain,
		h.tokenConfig.CookieSecure,
		true,
	)
	h.clearCSRFCookie(c)
}

func (h *Handler) setCSRFCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     middleware.CookieCSRFToken,
		Value:    token,
		MaxAge:   h.tokenConfig.RefreshTokenTTL,
		Path:     "/",
		Domain:   h.tokenConfig.CookieDomain,
		Secure:   h.tokenConfig.CookieSecure,
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *Handler) clearCSRFCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     middleware.CookieCSRFToken,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Domain:   h.tokenConfig.CookieDomain,
		Secure:   h.tokenConfig.CookieSecure,
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
	})
}
