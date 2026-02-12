package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	CookieCSRFToken = "csrf_token"
	HeaderCSRFToken = "X-CSRF-Token" //nolint:gosec // G101: this is a header name, not a credential
)

// GenerateCSRFToken 生成 CSRF token
func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// CSRFMiddleware 双重提交 CSRF 校验
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		cookieToken, err := c.Cookie(CookieCSRFToken)
		if err != nil || cookieToken == "" {
			response.Error(c, http.StatusForbidden, "CSRF_TOKEN_MISSING", "csrf token missing")
			c.Abort()
			return
		}

		headerToken := c.GetHeader(HeaderCSRFToken)
		// 使用常量时间比较防止时序攻击
		if headerToken == "" || subtle.ConstantTimeCompare([]byte(headerToken), []byte(cookieToken)) != 1 {
			response.Error(c, http.StatusForbidden, "CSRF_TOKEN_INVALID", "csrf token invalid")
			c.Abort()
			return
		}

		c.Next()
	}
}
