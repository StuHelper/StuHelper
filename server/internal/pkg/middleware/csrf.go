package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

const (
	// CSRFCookieName CSRF cookie 名称
	CSRFCookieName = "csrf_token"
	// CSRFHeaderName CSRF 请求头名称
	CSRFHeaderName = "X-CSRF-Token"
)

// GenerateCSRFToken 生成 CSRF token
func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	n, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	if n != len(b) {
		return "", fmt.Errorf("crypto/rand: short read (%d/%d bytes)", n, len(b))
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// CSRFMiddleware 双重提交 CSRF 校验
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if cookieToken, err := c.Cookie(CSRFCookieName); err == nil && cookieToken != "" {
			c.Header(CSRFHeaderName, cookieToken)
		}

		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		// 仅对 Cookie 会话执行双重提交校验：
		// - Bearer 请求不依赖浏览器自动携带的 cookie，可直接放行
		// - 纯 Bearer / 无 cookie 的请求交给后续认证中间件处理
		if hasBearerAuthorization(c) || !hasCookieSession(c) {
			c.Next()
			return
		}

		cookieToken, err := c.Cookie(CSRFCookieName)
		if err != nil || cookieToken == "" {
			response.Error(c, http.StatusForbidden, errs.ErrCSRFTokenMissing, "csrf token missing")
			c.Abort()
			return
		}

		headerToken := c.GetHeader(CSRFHeaderName)
		// 使用常量时间比较防止时序攻击
		if headerToken == "" || subtle.ConstantTimeCompare([]byte(headerToken), []byte(cookieToken)) != 1 {
			response.Error(c, http.StatusForbidden, errs.ErrCSRFTokenInvalid, "csrf token invalid")
			c.Abort()
			return
		}

		c.Next()
	}
}

func hasCookieSession(c *gin.Context) bool {
	if accessToken, err := c.Cookie(CookieAccessToken); err == nil && accessToken != "" {
		return true
	}

	if refreshToken, err := c.Cookie(CookieRefreshToken); err == nil && refreshToken != "" {
		return true
	}

	return false
}

func hasBearerAuthorization(c *gin.Context) bool {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	return len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && strings.TrimSpace(parts[1]) != ""
}
