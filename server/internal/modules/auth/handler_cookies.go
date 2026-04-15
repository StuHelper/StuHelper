package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

const tokenCookieSameSite = http.SameSiteLaxMode
const refreshTokenCookiePath = "/"

// sessionCookieName 服务端 session ID cookie
// OIDC ID Token 由 Zitadel 签发，无法注入自定义 sid claim。
// 因此通过独立的 HttpOnly cookie 传递 session ID，确保 logout/refresh 都能定位到正确的 session。
const sessionCookieName = "session_id"

// setTokenCookies 设置 Token Cookie（OIDC 登录，含 access + refresh）
// 返回 error 而非直接写响应，由调用方统一处理错误响应，避免双重 HTTP 写入
func (h *Handler) setTokenCookies(c *gin.Context, accessToken, refreshToken string) error {
	c.SetSameSite(tokenCookieSameSite)

	csrfToken, err := middleware.GenerateCSRFToken()
	if err != nil {
		logger.FromGin(c).Error("failed to generate CSRF token", zap.Error(err))
		return err
	}
	h.setCSRFCookie(c, csrfToken)

	c.SetCookie(
		middleware.CookieAccessToken,
		accessToken,
		h.tokenConfig.AccessTokenTTL,
		"/",
		h.tokenConfig.CookieDomain,
		h.tokenConfig.CookieSecure,
		true,
	)
	c.SetCookie(
		middleware.CookieRefreshToken,
		refreshToken,
		h.tokenConfig.RefreshTokenTTL,
		refreshTokenCookiePath,
		h.tokenConfig.CookieDomain,
		h.tokenConfig.CookieSecure,
		true,
	)

	return nil
}

// clearTokenCookies 清除 Token Cookie 和 Session Cookie
func (h *Handler) clearTokenCookies(c *gin.Context) {
	c.SetSameSite(tokenCookieSameSite)
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
		refreshTokenCookiePath,
		h.tokenConfig.CookieDomain,
		h.tokenConfig.CookieSecure,
		true,
	)
	h.clearCSRFCookie(c)
	h.clearSessionCookie(c)
}

func (h *Handler) setCSRFCookie(c *gin.Context, token string) {
	c.Header(middleware.CSRFHeaderName, token)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     middleware.CSRFCookieName,
		Value:    token,
		MaxAge:   h.tokenConfig.RefreshTokenTTL,
		Path:     "/",
		Domain:   h.tokenConfig.CookieDomain,
		Secure:   h.tokenConfig.CookieSecure,
		HttpOnly: false,
		SameSite: tokenCookieSameSite,
	})
}

func (h *Handler) clearCSRFCookie(c *gin.Context) {
	c.Header(middleware.CSRFHeaderName, "")
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     middleware.CSRFCookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Domain:   h.tokenConfig.CookieDomain,
		Secure:   h.tokenConfig.CookieSecure,
		HttpOnly: false,
		SameSite: tokenCookieSameSite,
	})
}

// setSessionCookie 写入 session ID cookie（HttpOnly）。
// OIDC 场景下 token 内无 sid claim，通过此 cookie 在 logout/refresh 时定位 session。
func (h *Handler) setSessionCookie(c *gin.Context, sessionID string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		MaxAge:   h.tokenConfig.RefreshTokenTTL,
		Path:     "/",
		Domain:   h.tokenConfig.CookieDomain,
		Secure:   h.tokenConfig.CookieSecure,
		HttpOnly: true,
		SameSite: tokenCookieSameSite,
	})
}

// getSessionID 从请求中获取 session ID。
// 优先从自签名 JWT 的 sid claim 提取（手机登录），
// 回退到 session_id cookie（OIDC 登录）。
func (h *Handler) getSessionID(c *gin.Context, accessToken string) string {
	// 自签名 JWT 优先（手机登录，sid 在 token claim 中）
	if sid := extractSessionID(accessToken); sid != "" {
		return sid
	}
	// OIDC 回退：从 session cookie 读取
	if v, err := c.Cookie(sessionCookieName); err == nil && v != "" {
		return v
	}
	return ""
}

// clearSessionCookie 清除 session ID cookie
func (h *Handler) clearSessionCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Domain:   h.tokenConfig.CookieDomain,
		Secure:   h.tokenConfig.CookieSecure,
		HttpOnly: true,
		SameSite: tokenCookieSameSite,
	})
}
