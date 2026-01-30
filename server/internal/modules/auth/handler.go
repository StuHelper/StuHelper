package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
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
	refreshLimiter *middleware.RedisRateLimiter
}

// NewHandler 创建认证处理器
func NewHandler(cfg *config.Config, tokenService *token.Service, rdb *redis.Client) *Handler {
	return &Handler{
		ssoClient:    sso.NewClientWithCache(cfg.Casdoor, rdb),
		tokenService: tokenService,
		tokenConfig:  cfg.Token,
		redirectURI:  cfg.Casdoor.RedirectURI,
		appName:      cfg.Casdoor.Application,
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
	url, err := h.ssoClient.GetSigninURL(ctx, h.redirectURI)
	if err != nil {
		logger.FromGin(c).Error("failed to generate login URL", zap.Error(err))
		response.InternalError(c, "failed to generate login URL")
		return
	}
	response.Success(c, gin.H{"url": url})
}

// GetSignupURL 获取注册 URL
func (h *Handler) GetSignupURL(c *gin.Context) {
	ctx := c.Request.Context()
	url, err := h.ssoClient.GetSignupURL(ctx, h.redirectURI)
	if err != nil {
		logger.FromGin(c).Error("failed to generate signup URL", zap.Error(err))
		response.InternalError(c, "failed to generate signup URL")
		return
	}
	response.Success(c, gin.H{"url": url})
}

// HandleCallback 处理 OAuth 回调
func (h *Handler) HandleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	requestID := getRequestID(c)
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
			zap.String("state", state),
		)
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "invalid state")
		response.BadRequest(c, "invalid or expired state parameter")
		return
	}

	// 获取 OAuth Token（state 参数传递给 Casdoor，但我们已经自己验证过了）
	oauthToken, err := h.ssoClient.GetOAuthToken(code, h.appName)
	if err != nil {
		logger.FromGin(c).Error("failed to get OAuth token", zap.Error(err))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "oauth token error")
		response.InternalError(c, "authentication failed")
		return
	}

	// 解析 JWT 获取用户信息
	claims, err := h.ssoClient.ParseJwtToken(oauthToken.AccessToken)
	if err != nil {
		logger.FromGin(c).Error("failed to parse JWT token", zap.Error(err))
		audit.LogFailure(audit.EventUserLoginFailed, c.ClientIP(), c.Request.UserAgent(), requestID, "jwt parse error")
		response.InternalError(c, "failed to retrieve user information")
		return
	}

	// 设置 HttpOnly Cookie
	h.setTokenCookies(c, oauthToken.AccessToken, oauthToken.RefreshToken)

	// 追踪用户 token（用于全设备登出）
	if err := h.tokenService.GetBlacklist().TrackUserToken(
		ctx,
		claims.Id,
		oauthToken.AccessToken,
		h.tokenService.GetRefreshTokenTTL(),
	); err != nil {
		logger.FromGin(c).Warn("failed to track user token",
			zap.String("user_id", claims.Id),
			zap.Error(err),
		)
	}

	// 记录登录成功审计日志
	audit.LogSuccess(audit.EventUserLogin, claims.Id, claims.Name, c.ClientIP(), c.Request.UserAgent(), requestID)

	response.Success(c, gin.H{
		"user": gin.H{
			"id":           claims.Id,
			"name":         claims.Name,
			"display_name": claims.DisplayName,
			"email":        claims.Email,
			"avatar":       claims.Avatar,
		},
	})
}

// GetCurrentUser 获取当前用户信息
func (h *Handler) GetCurrentUser(c *gin.Context) {
	response.Success(c, gin.H{
		"id":           middleware.GetUserID(c),
		"name":         middleware.GetUsername(c),
		"email":        middleware.GetEmail(c),
		"display_name": middleware.GetDisplayName(c),
	})
}

// Logout 登出
func (h *Handler) Logout(c *gin.Context) {
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)
	requestID := getRequestID(c)

	// 获取当前 access token 并加入黑名单
	accessToken, _ := c.Cookie(middleware.CookieAccessToken)
	if accessToken != "" {
		ctx := c.Request.Context()
		if err := h.tokenService.GetBlacklist().Add(ctx, accessToken, h.tokenService.GetAccessTokenTTL()); err != nil {
			logger.FromGin(c).Warn("failed to blacklist token",
				zap.String("user_id", userID),
				zap.Error(err),
			)
		}
	}

	// 清除 Cookie
	h.clearTokenCookies(c)

	// 记录登出审计日志
	audit.LogSuccess(audit.EventUserLogout, userID, username, c.ClientIP(), c.Request.UserAgent(), requestID)

	response.Success(c, gin.H{"message": "logout successful"})
}

// LogoutAll 全设备登出
func (h *Handler) LogoutAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)
	requestID := getRequestID(c)
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
		response.Unauthorized(c, "missing refresh token")
		return
	}

	// 刷新前检查 refresh token 是否已被撤销
	blacklisted, err := h.tokenService.GetBlacklist().IsBlacklisted(c.Request.Context(), refreshToken)
	if err != nil {
		response.ServiceUnavailable(c, "service temporarily unavailable")
		return
	}
	if blacklisted {
		response.Unauthorized(c, "refresh token revoked")
		return
	}

	// 使用 Casdoor SDK 刷新 token
	newToken, err := h.ssoClient.RefreshOAuthToken(refreshToken)
	if err != nil {
		logger.FromGin(c).Error("failed to refresh token", zap.Error(err))
		h.clearTokenCookies(c)
		response.Unauthorized(c, "failed to refresh token")
		return
	}

	// 将旧 refresh token 加入黑名单，避免重放
	_ = h.tokenService.GetBlacklist().Add(c.Request.Context(), refreshToken, h.tokenService.GetRefreshTokenTTL())

	// 设置新的 Cookie
	h.setTokenCookies(c, newToken.AccessToken, newToken.RefreshToken)

	// 追踪新的 access token
	if claims, err := h.ssoClient.ParseJwtToken(newToken.AccessToken); err == nil {
		_ = h.tokenService.GetBlacklist().TrackUserToken(
			c.Request.Context(),
			claims.Id,
			newToken.AccessToken,
			h.tokenService.GetRefreshTokenTTL(),
		)
	}

	response.Success(c, gin.H{"message": "token refreshed successfully"})
}

// setTokenCookies 设置 Token Cookie
func (h *Handler) setTokenCookies(c *gin.Context, accessToken, refreshToken string) {
	// 设置 SameSite 属性防止 CSRF
	c.SetSameSite(http.SameSiteStrictMode)

	csrfToken, err := middleware.GenerateCSRFToken()
	if err == nil {
		h.setCSRFCookie(c, csrfToken)
	}

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
}

// clearTokenCookies 清除 Token Cookie
func (h *Handler) clearTokenCookies(c *gin.Context) {
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
	c.SetCookie(
		middleware.CookieCSRFToken,
		token,
		h.tokenConfig.RefreshTokenTTL,
		"/",
		h.tokenConfig.CookieDomain,
		h.tokenConfig.CookieSecure,
		false,
	)
}

func (h *Handler) clearCSRFCookie(c *gin.Context) {
	c.SetCookie(
		middleware.CookieCSRFToken,
		"",
		-1,
		"/",
		h.tokenConfig.CookieDomain,
		h.tokenConfig.CookieSecure,
		false,
	)
}

func getRequestID(c *gin.Context) string {
	if id, exists := c.Get(middleware.CtxKeyRequestID); exists {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}
