package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// Cookie 名称常量
const (
	CookieAccessToken  = "access_token"
	CookieRefreshToken = "refresh_token"
	CookieSessionID    = "session_id"
)

// Cookie path constants. Browser deletion must use the same (name, domain,
// path) tuple as cookie creation.
const (
	CookieAccessTokenPath  = "/"
	CookieRefreshTokenPath = "/api/v1/auth" //nolint:gosec // cookie path constant, not token secret material.
)

// tokenSource 标记 Token 来源
type tokenSource int

const (
	tokenSourceNone   tokenSource = iota
	tokenSourceCookie             // Cookie: 本地 JWKS 验证（快速）
	tokenSourceBearer             // Bearer: introspection 验证（即时吊销）
)

// GetAccessToken 从请求中提取 access token（优先级：Authorization Header > Cookie）。
// 用于 Logout 等需要获取当前 token 原始值的场景。
func GetAccessToken(c *gin.Context) string {
	accessToken, _ := getTokenWithSource(c)
	return accessToken
}

// getTokenWithSource 从请求中获取 access token 及其来源。
// 优先级：Authorization Header > Cookie。
// 当客户端显式携带 Bearer token 时，应优先按 API 客户端语义处理，
// 避免浏览器 cookie 覆盖 bearerAuth 契约。
func getTokenWithSource(c *gin.Context) (string, tokenSource) {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			accessToken := strings.TrimSpace(parts[1])
			if accessToken != "" {
				return accessToken, tokenSourceBearer
			}
		}
	}

	if accessToken, err := c.Cookie(CookieAccessToken); err == nil && accessToken != "" {
		return accessToken, tokenSourceCookie
	}

	return "", tokenSourceNone
}
