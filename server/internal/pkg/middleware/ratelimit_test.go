package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

func TestProgressiveEndpointRateLimitMiddleware_AnonymousVsAuthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fixture := redisfixture.Start(t)

	anonLimiter := NewRedisRateLimiter(fixture.Client, 1, time.Minute)
	userLimiter := NewRedisRateLimiter(fixture.Client, 2, time.Minute)

	router := gin.New()
	router.GET("/search",
		func(c *gin.Context) {
			if c.GetHeader("X-Test-User") != "" {
				c.Set(CtxKeyUserID, c.GetHeader("X-Test-User"))
			}
			c.Next()
		},
		ProgressiveEndpointRateLimitMiddleware(anonLimiter, userLimiter, "review-search"),
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		},
	)

	t.Run("anonymous requests use stricter IP quota", func(t *testing.T) {
		for i := range 2 {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/search", nil)
			req.RemoteAddr = "203.0.113.10:12345"
			router.ServeHTTP(w, req)

			if i == 0 {
				assert.Equal(t, http.StatusOK, w.Code)
			} else {
				assert.Equal(t, http.StatusTooManyRequests, w.Code)
			}
		}
	})

	t.Run("authenticated requests use user quota instead of IP quota", func(t *testing.T) {
		for i := range 3 {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/search", nil)
			req.RemoteAddr = "203.0.113.11:12345"
			req.Header.Set("X-Test-User", "user-123")
			router.ServeHTTP(w, req)

			if i < 2 {
				assert.Equal(t, http.StatusOK, w.Code)
			} else {
				assert.Equal(t, http.StatusTooManyRequests, w.Code)
			}
		}
	})
}
