package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityHeadersMiddleware_APIRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(SecurityHeadersMiddleware())
	r.GET("/api/v1/ping", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	assert.Equal(t, "geolocation=(), microphone=(), camera=()", w.Header().Get("Permissions-Policy"))
	assert.Equal(t, "none", w.Header().Get("X-Permitted-Cross-Domain-Policies"))
	assert.Equal(t, "same-origin", w.Header().Get("Cross-Origin-Resource-Policy"))
	assert.Equal(t, "same-origin", w.Header().Get("Cross-Origin-Opener-Policy"))
	assert.Equal(t, "default-src 'none'; frame-ancestors 'none'", w.Header().Get("Content-Security-Policy"))
	assert.Empty(t, w.Header().Get("Strict-Transport-Security"))
}

func TestSecurityHeadersWithHSTS_DocsRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(SecurityHeadersWithHSTS())
	r.GET("/docs/index.html", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs/index.html", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "script-src 'self' 'unsafe-inline'")
	assert.Equal(t, "max-age=31536000; includeSubDomains", w.Header().Get("Strict-Transport-Security"))
}

func TestMaxBodySize_ContentLengthTooLarge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(MaxBodySize(4))
	r.POST("/upload", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("12345"))
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestMaxBodySize_UnknownContentLengthReadOverflow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(MaxBodySize(4))
	r.POST("/upload", func(c *gin.Context) {
		_, err := io.ReadAll(c.Request.Body)
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		require.NoError(t, err)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/upload", io.NopCloser(strings.NewReader("12345")))
	req.ContentLength = -1
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}
