package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

// setTokenCookies 设置 Token Cookie
// 返回 error 而非直接写响应，由调用方统一处理错误响应，避免双重 HTTP 写入
func (h *Handler) setTokenCookies(c *gin.Context, accessToken, refreshToken string) error {
	c.SetSameSite(http.SameSiteStrictMode)

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
		"/api/v1/auth/refresh",
		h.tokenConfig.CookieDomain,
		h.tokenConfig.CookieSecure,
		true,
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
