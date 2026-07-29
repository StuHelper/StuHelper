package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	pkgcrypto "github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

const (
	// CSRFCookieName CSRF cookie 名称
	CSRFCookieName = "csrf_token"
	// CSRFHeaderName CSRF 请求头名称
	CSRFHeaderName = "X-CSRF-Token"
)

const csrfTokenVersion = "v1"

var (
	ErrCSRFTokenMissing = errors.New("csrf token missing")
	ErrCSRFTokenInvalid = errors.New("csrf token invalid")
)

// GenerateCSRFToken 生成绑定 sessionID 的 signed double-submit CSRF token。
func GenerateCSRFToken(sessionID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", ErrCSRFTokenInvalid
	}
	b := make([]byte, 32)
	n, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	if n != len(b) {
		return "", fmt.Errorf("crypto/rand: short read (%d/%d bytes)", n, len(b))
	}
	nonce := base64.RawURLEncoding.EncodeToString(b)
	mac, err := signCSRFToken(nonce, sessionID)
	if err != nil {
		return "", err
	}
	return csrfTokenVersion + "." + nonce + "." + mac, nil
}

// ValidateCSRFDoubleSubmit 校验 cookie/header 双重提交 token，并验证其 HMAC 绑定到 sessionID。
func ValidateCSRFDoubleSubmit(cookieToken, headerToken, sessionID string) error {
	cookieToken = strings.TrimSpace(cookieToken)
	headerToken = strings.TrimSpace(headerToken)
	sessionID = strings.TrimSpace(sessionID)
	if cookieToken == "" || headerToken == "" {
		return ErrCSRFTokenMissing
	}
	if sessionID == "" {
		return ErrCSRFTokenInvalid
	}
	if subtle.ConstantTimeCompare([]byte(headerToken), []byte(cookieToken)) != 1 {
		return ErrCSRFTokenInvalid
	}

	parts := strings.Split(cookieToken, ".")
	if len(parts) != 3 || parts[0] != csrfTokenVersion || parts[1] == "" || parts[2] == "" {
		return ErrCSRFTokenInvalid
	}
	expectedMAC, err := signCSRFToken(parts[1], sessionID)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(parts[2]), []byte(expectedMAC)) != 1 {
		return ErrCSRFTokenInvalid
	}
	return nil
}

func signCSRFToken(nonce, sessionID string) (string, error) {
	return pkgcrypto.HMACHashWithKey(csrfTokenVersion+"."+nonce+"."+sessionID, pkgcrypto.GetHMACKey())
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

		cookieToken, cookieErr := c.Cookie(CSRFCookieName)
		if cookieErr != nil || cookieToken == "" {
			response.Error(c, http.StatusForbidden, errs.ErrCSRFTokenMissing, "csrf token missing")
			c.Abort()
			return
		}

		headerToken := c.GetHeader(CSRFHeaderName)
		sessionID, sessionErr := c.Cookie(CookieSessionID)
		if sessionErr != nil || strings.TrimSpace(sessionID) == "" {
			response.Error(c, http.StatusForbidden, errs.ErrCSRFTokenInvalid, "csrf session missing")
			c.Abort()
			return
		}
		if err := ValidateCSRFDoubleSubmit(cookieToken, headerToken, sessionID); err != nil {
			if errors.Is(err, ErrCSRFTokenMissing) {
				response.Error(c, http.StatusForbidden, errs.ErrCSRFTokenMissing, "csrf token missing")
				c.Abort()
				return
			}
			response.Error(c, http.StatusForbidden, errs.ErrCSRFTokenInvalid, "csrf token invalid")
			c.Abort()
			return
		}

		c.Next()
	}
}

func hasCookieSession(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	if accessToken, err := c.Cookie(CookieAccessToken); err == nil && accessToken != "" {
		return true
	}

	if refreshToken, err := c.Cookie(CookieRefreshToken); err == nil && refreshToken != "" {
		return true
	}

	return false
}

func hasBearerAuthorization(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	if len(c.Request.Header.Values("Authorization")) > 1 {
		return false
	}
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	return len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && strings.TrimSpace(parts[1]) != ""
}
