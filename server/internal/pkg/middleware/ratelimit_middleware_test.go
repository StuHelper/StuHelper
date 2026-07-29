package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

func newLimiter(t *testing.T, limit int) *RedisRateLimiter {
	t.Helper()
	fixture := redisfixture.Start(t)
	return NewRedisRateLimiter(fixture.Client, limit, time.Minute)
}

func TestRateLimitMiddlewares(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("ip rate limit", func(t *testing.T) {
		limiter := newLimiter(t, 1)
		r := gin.New()
		r.Use(RateLimitMiddleware(limiter))
		r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		for i, want := range []int{http.StatusNoContent, http.StatusTooManyRequests} {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "127.0.0.1:1234"
			r.ServeHTTP(w, req)
			assert.Equal(t, want, w.Code, "req %d", i)
		}
	})

	t.Run("global rate limit", func(t *testing.T) {
		limiter := newLimiter(t, 1)
		r := gin.New()
		r.Use(GlobalRateLimitMiddleware(limiter))
		r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Equal(t, http.StatusNoContent, w.Code)

		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("user and endpoint rate limit", func(t *testing.T) {
		userLimiter := newLimiter(t, 1)
		endpointLimiter := newLimiter(t, 1)

		rUser := gin.New()
		rUser.Use(func(c *gin.Context) {
			c.Set(CtxKeyUserID, "user-rl")
			c.Next()
		}, UserRateLimitMiddleware(userLimiter))
		rUser.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		w := httptest.NewRecorder()
		rUser.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Equal(t, http.StatusNoContent, w.Code)
		w = httptest.NewRecorder()
		rUser.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Equal(t, http.StatusTooManyRequests, w.Code)

		rEndpoint := gin.New()
		rEndpoint.Use(func(c *gin.Context) {
			c.Set(CtxKeyUserID, "user-rl")
			c.Next()
		}, EndpointRateLimitMiddleware(endpointLimiter, "write-review"))
		rEndpoint.POST("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		w = httptest.NewRecorder()
		rEndpoint.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", nil))
		assert.Equal(t, http.StatusNoContent, w.Code)
		w = httptest.NewRecorder()
		rEndpoint.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", nil))
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})

	t.Run("anonymous user middleware passes through", func(t *testing.T) {
		limiter := newLimiter(t, 1)
		r := gin.New()
		r.Use(UserRateLimitMiddleware(limiter))
		r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		for range 2 {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
			assert.Equal(t, http.StatusNoContent, w.Code)
		}
	})

	t.Run("endpoint rate limit falls back to ip for anonymous", func(t *testing.T) {
		limiter := newLimiter(t, 1)
		r := gin.New()
		r.Use(EndpointRateLimitMiddleware(limiter, "public-write"))
		r.POST("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "203.0.113.20:4567"
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)

		w = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "203.0.113.20:4567"
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, "60", w.Header().Get("Retry-After"))
	})
}
