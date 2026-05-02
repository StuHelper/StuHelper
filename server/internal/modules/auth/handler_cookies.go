package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

const tokenCookieSameSite = http.SameSiteLaxMode
const refreshTokenCookiePath = "/api/v1/auth" //nolint:gosec // Cookie path literal, not a credential.
const nativeSessionIDHeader = "X-Stuhelper-Session-ID"

// sessionCookieName 服务端 session ID cookie
// OIDC ID Token 由外部 provider 签发，无法注入自定义 sid claim。
// 因此通过独立的 HttpOnly cookie 传递 session ID，确保 logout/refresh 都能定位到正确的 session。
const sessionCookieName = "session_id"

func (h *Handler) writeCookie(c *gin.Context, name, value string, maxAge int, path string, httpOnly bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   maxAge,
		Path:     path,
		Domain:   h.tokenConfig.CookieDomain,
		Secure:   h.tokenConfig.CookieSecure,
		HttpOnly: httpOnly,
		SameSite: tokenCookieSameSite,
	})
}

// setTokenCookies 设置 Token Cookie（OIDC 登录，含 access + refresh）
// 返回 error 而非直接写响应，由调用方统一处理错误响应，避免双重 HTTP 写入
func (h *Handler) setTokenCookies(c *gin.Context, accessToken, refreshToken string) error {
	csrfToken, err := middleware.GenerateCSRFToken()
	if err != nil {
		logger.FromGin(c).Error("failed to generate CSRF token", zap.Error(err))
		return err
	}
	h.setTokenCookiesWithCSRF(c, accessToken, refreshToken, csrfToken)

	return nil
}

func (h *Handler) prepareTokenCookies(c *gin.Context) (string, bool) {
	csrfToken, err := middleware.GenerateCSRFToken()
	if err != nil {
		logger.FromGin(c).Error("failed to generate CSRF token", zap.Error(err))
		response.InternalError(c, "failed to refresh token")
		return "", false
	}
	return csrfToken, true
}

func (h *Handler) setTokenCookiesWithCSRF(c *gin.Context, accessToken, refreshToken, csrfToken string) {
	h.setCSRFCookie(c, csrfToken)
	h.writeCookie(c, middleware.CookieAccessToken, accessToken, h.currentAccessTokenTTLSeconds(), "/", true)
	h.writeCookie(c, middleware.CookieRefreshToken, refreshToken, h.tokenConfig.RefreshTokenTTL, refreshTokenCookiePath, true)
}

// clearTokenCookies 清除 Token Cookie 和 Session Cookie
func (h *Handler) clearTokenCookies(c *gin.Context) {
	h.writeCookie(c, middleware.CookieAccessToken, "", -1, "/", true)
	h.writeCookie(c, middleware.CookieRefreshToken, "", -1, refreshTokenCookiePath, true)
	h.clearCSRFCookie(c)
	h.clearSessionCookie(c)
}

func (h *Handler) setCSRFCookie(c *gin.Context, token string) {
	c.Header(middleware.CSRFHeaderName, token)
	h.writeCookie(c, middleware.CSRFCookieName, token, h.tokenConfig.RefreshTokenTTL, "/", false)
}

func (h *Handler) clearCSRFCookie(c *gin.Context) {
	c.Header(middleware.CSRFHeaderName, "")
	h.writeCookie(c, middleware.CSRFCookieName, "", -1, "/", false)
}

// setSessionCookie 写入 session ID cookie（HttpOnly）。
// OIDC 场景下 token 内无 sid claim，通过此 cookie 在 logout/refresh 时定位 session。
func (h *Handler) setSessionCookie(c *gin.Context, sessionID string) {
	h.writeCookie(c, sessionCookieName, sessionID, h.tokenConfig.RefreshTokenTTL, "/", true)
}

// getSessionID 从请求中获取 session ID。
// 优先级：
//  1. 自签名 JWT 的 sid claim（手机登录）
//  2. X-Stuhelper-Session-ID header（原生 OIDC）
//  3. session_id cookie（浏览器 OIDC）
func (h *Handler) getSessionID(c *gin.Context, accessToken string) string {
	// 自签名 JWT 优先（手机登录，sid 在 token claim 中）
	if sid := extractSessionID(accessToken); sid != "" {
		return sid
	}
	// 原生客户端：从显式 header 读取 session ID
	if v := c.GetHeader(nativeSessionIDHeader); v != "" {
		return v
	}
	// OIDC 回退：从 session cookie 读取
	if v, err := c.Cookie(sessionCookieName); err == nil && v != "" {
		return v
	}
	return ""
}

// clearSessionCookie 清除 session ID cookie
func (h *Handler) clearSessionCookie(c *gin.Context) {
	h.writeCookie(c, sessionCookieName, "", -1, "/", true)
}
