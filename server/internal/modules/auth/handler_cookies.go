package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

const tokenCookieSameSite = http.SameSiteLaxMode
const nativeSessionIDHeader = "X-Stuhelper-Session-ID"

// sessionCookieName 服务端 session ID cookie
// OIDC ID Token 由外部 provider 签发，无法注入自定义 sid claim。
// 因此通过独立的 HttpOnly cookie 传递 session ID，确保 logout/refresh 都能定位到正确的 session。
const sessionCookieName = middleware.CookieSessionID

func (h *Handler) writeHTTPOnlyCookie(c *gin.Context, name, value string, maxAge int, path string) {
	http.SetCookie(c.Writer, &http.Cookie{ //nolint:gosec // cookie Secure flag is environment-configured through TokenConfig.
		Name:     name,
		Value:    value,
		MaxAge:   maxAge,
		Path:     path,
		Domain:   h.tokenConfig.CookieDomain,
		Secure:   h.tokenConfig.CookieSecure,
		HttpOnly: true,
		SameSite: tokenCookieSameSite,
	})
}

func (h *Handler) writeCSRFCookie(c *gin.Context, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{ //nolint:gosec // CSRF double-submit cookie must be readable by the browser client.
		Name:     middleware.CSRFCookieName,
		Value:    value,
		MaxAge:   maxAge,
		Path:     "/",
		Domain:   h.tokenConfig.CookieDomain,
		Secure:   h.tokenConfig.CookieSecure,
		HttpOnly: false,
		SameSite: tokenCookieSameSite,
	})
}

// setTokenCookies 设置 Token Cookie（OIDC 登录，含 access + refresh）
// 返回 error 而非直接写响应，由调用方统一处理错误响应，避免双重 HTTP 写入
func (h *Handler) setTokenCookies(c *gin.Context, accessToken, refreshToken, sessionID string) error {
	csrfToken, err := middleware.GenerateCSRFToken(sessionID)
	if err != nil {
		logger.FromGin(c).Error("failed to generate CSRF token", zap.Error(err))
		return err
	}
	h.setTokenCookiesWithCSRF(c, accessToken, refreshToken, csrfToken)

	return nil
}

func (h *Handler) prepareTokenCookies(c *gin.Context, sessionID string) (string, bool) {
	csrfToken, err := middleware.GenerateCSRFToken(sessionID)
	if err != nil {
		logger.FromGin(c).Error("failed to generate CSRF token", zap.Error(err))
		response.InternalError(c, "failed to refresh token")
		return "", false
	}
	return csrfToken, true
}

func (h *Handler) setTokenCookiesWithCSRF(c *gin.Context, accessToken, refreshToken, csrfToken string) {
	h.setCSRFCookie(c, csrfToken)
	h.writeHTTPOnlyCookie(c, middleware.CookieAccessToken, accessToken, h.currentAccessTokenTTLSeconds(), middleware.CookieAccessTokenPath)
	h.writeHTTPOnlyCookie(c, middleware.CookieRefreshToken, refreshToken, h.tokenConfig.RefreshTokenTTL, middleware.CookieRefreshTokenPath)
}

// clearTokenCookies 清除 Token Cookie 和 Session Cookie
func (h *Handler) clearTokenCookies(c *gin.Context) {
	h.writeHTTPOnlyCookie(c, middleware.CookieAccessToken, "", -1, middleware.CookieAccessTokenPath)
	h.writeHTTPOnlyCookie(c, middleware.CookieRefreshToken, "", -1, middleware.CookieRefreshTokenPath)
	h.clearCSRFCookie(c)
	h.clearSessionCookie(c)
}

func (h *Handler) setCSRFCookie(c *gin.Context, token string) {
	c.Header(middleware.CSRFHeaderName, token)
	h.writeCSRFCookie(c, token, h.tokenConfig.RefreshTokenTTL)
}

func (h *Handler) clearCSRFCookie(c *gin.Context) {
	c.Header(middleware.CSRFHeaderName, "")
	h.writeCSRFCookie(c, "", -1)
}

// setSessionCookie 写入 session ID cookie（HttpOnly）。
// OIDC 场景下 token 内无 sid claim，通过此 cookie 在 logout/refresh 时定位 session。
func (h *Handler) setSessionCookie(c *gin.Context, sessionID string) {
	h.writeHTTPOnlyCookie(c, sessionCookieName, sessionID, h.tokenConfig.RefreshTokenTTL, "/")
}

// getSessionID 从请求中获取 session ID。
// 优先级：
//  1. X-Stuhelper-Session-ID header（原生 OIDC）
//  2. session_id cookie（浏览器 OIDC）
func (h *Handler) getSessionID(c *gin.Context, _ string) string {
	sessionID, _ := h.resolveSessionID(c)
	return sessionID
}

func (h *Handler) resolveSessionID(c *gin.Context) (string, bool) {
	headerSessionID, hasHeader, ok := nativeSessionIDFromHeader(c)
	if !ok {
		response.BadRequest(c, "invalid native session id", errs.ErrInvalidParam)
		return "", false
	}

	cookieSessionID, hasCookie := sessionIDFromCookie(c)
	if hasHeader && hasCookie {
		response.BadRequest(c, "session id source is ambiguous", errs.ErrInvalidParam)
		return "", false
	}
	if hasHeader {
		return headerSessionID, true
	}
	if hasCookie {
		return cookieSessionID, true
	}
	return "", true
}

func nativeSessionIDFromHeader(c *gin.Context) (string, bool, bool) {
	if c == nil || c.Request == nil {
		return "", false, true
	}

	values := c.Request.Header.Values(nativeSessionIDHeader)
	if len(values) == 0 {
		return "", false, true
	}
	if len(values) != 1 {
		return "", true, false
	}

	sessionID := strings.TrimSpace(values[0])
	if sessionID == "" || strings.Contains(sessionID, ",") {
		return "", true, false
	}
	return sessionID, true, true
}

func sessionIDFromCookie(c *gin.Context) (string, bool) {
	if c == nil {
		return "", false
	}
	v, err := c.Cookie(sessionCookieName)
	if err != nil {
		return "", false
	}
	sessionID := strings.TrimSpace(v)
	return sessionID, sessionID != ""
}

// clearSessionCookie 清除 session ID cookie
func (h *Handler) clearSessionCookie(c *gin.Context) {
	h.writeHTTPOnlyCookie(c, sessionCookieName, "", -1, "/")
}
