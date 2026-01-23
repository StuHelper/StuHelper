package auth

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/sso"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

// Handler 认证处理器
type Handler struct {
	ssoClient     *sso.Client
	tokenService  *token.Service
	tokenConfig   config.TokenConfig
	redirectURI   string
	appName       string
	refreshLimiter *middleware.RateLimiter
}

// NewHandler 创建认证处理器
func NewHandler(cfg *config.Config, tokenService *token.Service) *Handler {
	return &Handler{
		ssoClient:     sso.NewClient(cfg.Casdoor),
		tokenService:  tokenService,
		tokenConfig:   cfg.Token,
		redirectURI:   cfg.Casdoor.RedirectURI,
		appName:       cfg.Casdoor.Application,
		// RefreshToken 限制: 每分钟最多 10 次
		refreshLimiter: middleware.NewRateLimiter(10, time.Minute),
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
	url := h.ssoClient.GetSigninURL(h.redirectURI)
	c.JSON(http.StatusOK, gin.H{
		"url": url,
	})
}

// GetSignupURL 获取注册 URL
func (h *Handler) GetSignupURL(c *gin.Context) {
	url := h.ssoClient.GetSignupURL(h.redirectURI)
	c.JSON(http.StatusOK, gin.H{
		"url": url,
	})
}

// HandleCallback 处理 OAuth 回调
func (h *Handler) HandleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing authorization code",
		})
		return
	}

	// 验证 state（Casdoor SDK 使用 ApplicationName 作为 state）
	if state != h.appName {
		log.Printf("invalid state: expected %s, got %s", h.appName, state)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid state parameter",
		})
		return
	}

	// 获取 OAuth Token
	oauthToken, err := h.ssoClient.GetOAuthToken(code, state)
	if err != nil {
		log.Printf("failed to get OAuth token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "authentication failed",
		})
		return
	}

	// 解析 JWT 获取用户信息
	claims, err := h.ssoClient.ParseJwtToken(oauthToken.AccessToken)
	if err != nil {
		log.Printf("failed to parse JWT token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve user information",
		})
		return
	}

	// 设置 HttpOnly Cookie
	h.setTokenCookies(c, oauthToken.AccessToken, oauthToken.RefreshToken)

	// 追踪用户 token（用于全设备登出）
	ctx := c.Request.Context()
	if err := h.tokenService.GetBlacklist().TrackUserToken(
		ctx,
		claims.User.Id,
		oauthToken.AccessToken,
		h.tokenService.GetRefreshTokenTTL(),
	); err != nil {
		log.Printf("warning: failed to track user token for user %s: %v", claims.User.Id, err)
		// 不阻止登录，但记录警告
	}

	log.Printf("user %s logged in successfully", claims.User.Id)
	c.JSON(http.StatusOK, gin.H{
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
	c.JSON(http.StatusOK, gin.H{
		"id":           middleware.GetUserID(c),
		"name":         middleware.GetUsername(c),
		"email":        middleware.GetEmail(c),
		"display_name": middleware.GetDisplayName(c),
	})
}

// Logout 登出
func (h *Handler) Logout(c *gin.Context) {
	userID := middleware.GetUserID(c)
	// 获取当前 access token 并加入黑名单
	accessToken, _ := c.Cookie(middleware.CookieAccessToken)
	if accessToken != "" {
		ctx := c.Request.Context()
		if err := h.tokenService.GetBlacklist().Add(ctx, accessToken, h.tokenService.GetAccessTokenTTL()); err != nil {
			log.Printf("warning: failed to blacklist token for user %s: %v", userID, err)
			// 继续登出流程，但记录警告
		}
	}

	// 清除 Cookie
	h.clearTokenCookies(c)

	c.JSON(http.StatusOK, gin.H{
		"message": "logout successful",
	})
}

// LogoutAll 全设备登出
func (h *Handler) LogoutAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	ctx := c.Request.Context()

	// 撤销用户所有 token
	if err := h.tokenService.GetBlacklist().RevokeAllUserTokens(
		ctx,
		userID,
		h.tokenService.GetRefreshTokenTTL(),
	); err != nil {
		log.Printf("error: failed to revoke all tokens for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to logout from all devices",
		})
		return
	}

	// 清除当前设备的 Cookie
	h.clearTokenCookies(c)

	c.JSON(http.StatusOK, gin.H{
		"message": "logged out from all devices",
	})
}

// RefreshToken 刷新 Access Token
func (h *Handler) RefreshToken(c *gin.Context) {
	refreshToken, err := c.Cookie(middleware.CookieRefreshToken)
	if err != nil || refreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "missing refresh token",
		})
		return
	}

	// 使用 Casdoor SDK 刷新 token
	newToken, err := h.ssoClient.RefreshOAuthToken(refreshToken)
	if err != nil {
		log.Printf("failed to refresh token: %v", err)
		h.clearTokenCookies(c)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "failed to refresh token",
		})
		return
	}

	// 设置新的 Cookie
	h.setTokenCookies(c, newToken.AccessToken, newToken.RefreshToken)

	c.JSON(http.StatusOK, gin.H{
		"message": "token refreshed successfully",
	})
}

// setTokenCookies 设置 Token Cookie
func (h *Handler) setTokenCookies(c *gin.Context, accessToken, refreshToken string) {
	// 设置 SameSite 属性防止 CSRF
	c.SetSameSite(http.SameSiteLaxMode)

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
		"/auth/refresh",
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
		"/auth/refresh",
		h.tokenConfig.CookieDomain,
		h.tokenConfig.CookieSecure,
		true,
	)
}
