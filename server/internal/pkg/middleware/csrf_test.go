package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGenerateCSRFToken(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-middleware-csrf-secret-32char", false))

	token1, err := GenerateCSRFToken("sid-1")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(token1, "v1."))

	token2, err := GenerateCSRFToken("sid-1")
	require.NoError(t, err)
	assert.NotEqual(t, token1, token2, "每次生成的 token 应该不同")
}

func TestValidateCSRFDoubleSubmit_BindsTokenToSession(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-middleware-csrf-secret-32char", false))

	token, err := GenerateCSRFToken("sid-1")
	require.NoError(t, err)

	assert.NoError(t, ValidateCSRFDoubleSubmit(token, token, "sid-1"))
	assert.ErrorIs(t, ValidateCSRFDoubleSubmit(token, token, "sid-2"), ErrCSRFTokenInvalid)
	assert.ErrorIs(t, ValidateCSRFDoubleSubmit(token, "", "sid-1"), ErrCSRFTokenMissing)
	assert.ErrorIs(t, ValidateCSRFDoubleSubmit(token, token, ""), ErrCSRFTokenInvalid)
	assert.ErrorIs(t, ValidateCSRFDoubleSubmit("csrf-legacy", "csrf-legacy", "sid-1"), ErrCSRFTokenInvalid)
}

func TestCSRFMiddleware_AllowsSafeMethod(t *testing.T) {
	safeMethods := []string{http.MethodGet, http.MethodHead, http.MethodOptions}

	for _, method := range safeMethods {
		t.Run(method, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.Use(CSRFMiddleware())
			r.Handle(method, "/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			c.Request = httptest.NewRequest(method, "/test", nil)
			r.ServeHTTP(w, c.Request)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestCSRFMiddleware_EchoesCSRFCookieOnSafeRequests(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(CSRFMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "safe-csrf-token"})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "safe-csrf-token", w.Header().Get(CSRFHeaderName))
}

func TestCSRFMiddleware_AllowsAnonymousRequestWithoutCookies(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(CSRFMiddleware())
	r.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFMiddleware_AllowsBearerRequestWithoutCookies(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(CSRFMiddleware())
	r.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFMiddleware_AllowsBearerRequestWithCookiesWithoutCSRF(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(CSRFMiddleware())
	r.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: "access.jwt.token"})
	req.AddCookie(&http.Cookie{Name: CookieRefreshToken, Value: "refresh.jwt.token"})
	req.AddCookie(&http.Cookie{Name: CookieSessionID, Value: "sid-1"})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFMiddleware_BlocksCookieSessionWithMalformedAuthorizationWithoutCSRF(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
	}{
		{
			name:    "blank bearer",
			headers: []string{"Bearer   "},
		},
		{
			name:    "repeated authorization",
			headers: []string{"Bearer test-token", "Bearer other-token"},
		},
		{
			name:    "unsupported scheme",
			headers: []string{"Basic Y2xpZW50OnNlY3JldA=="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(CSRFMiddleware())
			r.POST("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			for _, header := range tt.headers {
				req.Header.Add("Authorization", header)
			}
			req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: "access.jwt.token"})
			req.AddCookie(&http.Cookie{Name: CookieRefreshToken, Value: "refresh.jwt.token"})
			req.AddCookie(&http.Cookie{Name: CookieSessionID, Value: "sid-1"})
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}
}

func TestCSRFHelpersAreNilSafe(t *testing.T) {
	assert.False(t, hasCookieSession(nil))
	assert.False(t, hasBearerAuthorization(nil))

	assert.False(t, hasCookieSession(&gin.Context{}))
	assert.False(t, hasBearerAuthorization(&gin.Context{}))
}

func TestCSRFMiddleware_BlocksCookieSessionWithoutCSRFToken(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(CSRFMiddleware())
	r.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: "access.jwt.token"})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRFMiddleware_AllowsSignedCookieSessionCSRF(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-middleware-csrf-secret-32char", false))

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(CSRFMiddleware())
	r.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	csrfToken, err := GenerateCSRFToken("sid-1")
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: "access.jwt.token"})
	req.AddCookie(&http.Cookie{Name: CookieSessionID, Value: "sid-1"})
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: csrfToken})
	req.Header.Set(CSRFHeaderName, csrfToken)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFMiddleware_BlocksSignedCSRFForDifferentSession(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-middleware-csrf-secret-32char", false))

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(CSRFMiddleware())
	r.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	csrfToken, err := GenerateCSRFToken("sid-1")
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: "access.jwt.token"})
	req.AddCookie(&http.Cookie{Name: CookieSessionID, Value: "sid-2"})
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: csrfToken})
	req.Header.Set(CSRFHeaderName, csrfToken)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRFMiddleware_BlocksCookieSessionWithoutSessionID(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-middleware-csrf-secret-32char", false))

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(CSRFMiddleware())
	r.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	csrfToken, err := GenerateCSRFToken("sid-1")
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: "access.jwt.token"})
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: csrfToken})
	req.Header.Set(CSRFHeaderName, csrfToken)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
