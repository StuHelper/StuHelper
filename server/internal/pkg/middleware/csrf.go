package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
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
	n, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	if n != len(b) {
		return "", fmt.Errorf("crypto/rand: short read (%d/%d bytes)", n, len(b))
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// CSRFSetCookieMiddleware 在响应中自动设置 CSRF cookie（如果尚未存在）
// 前端从 cookie 读取 token 并在请求头中回传，实现双重提交校验
//
// 安全权衡: HttpOnly=false 是双重提交模式的必要条件——前端 JS 必须能读取 cookie 值
// 以便将其放入 X-CSRF-Token 请求头。这意味着 XSS 攻击可以读取该 token。
// 缓解措施:
//   - SameSite=Strict 阻止跨站请求携带 cookie
//   - 依赖 CSP (Content-Security-Policy) 头限制脚本执行来源，降低 XSS 风险
//   - token 仅用于 CSRF 校验，不包含敏感信息
func CSRFSetCookieMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := c.Cookie(CookieCSRFToken); err != nil {
			token, genErr := GenerateCSRFToken()
			if genErr == nil {
				// SameSite=Strict 防止跨站携带；HttpOnly=false 允许前端 JS 读取
				c.SetSameSite(http.SameSiteStrictMode)
				c.SetCookie(CookieCSRFToken, token, 0, "/", "", false, false)
			}
		}
		c.Next()
	}
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
			response.Error(c, http.StatusForbidden, errs.ErrCSRFTokenMissing, "csrf token missing")
			c.Abort()
			return
		}

		headerToken := c.GetHeader(HeaderCSRFToken)
		// 使用常量时间比较防止时序攻击
		if headerToken == "" || subtle.ConstantTimeCompare([]byte(headerToken), []byte(cookieToken)) != 1 {
			response.Error(c, http.StatusForbidden, errs.ErrCSRFTokenBad, "csrf token invalid")
			c.Abort()
			return
		}

		c.Next()
	}
}
