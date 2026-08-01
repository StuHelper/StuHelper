package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggingMiddlewaresAndHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("request id and request logger", func(t *testing.T) {
		r := gin.New()
		r.Use(RequestIDMiddleware(), RequestLogger())
		r.GET("/ping", func(c *gin.Context) {
			c.Set(CtxKeyUserID, 12345)
			c.Status(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/ping?token=secret&plain=value", nil)
		req.Header.Set(HeaderRequestID, "valid-request-id")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "valid-request-id", w.Header().Get(HeaderRequestID))
		assert.Equal(t, "", GetRequestID(&gin.Context{}))
	})

	t.Run("invalid request id gets replaced", func(t *testing.T) {
		r := gin.New()
		r.Use(RequestIDMiddleware())
		r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.Header.Set(HeaderRequestID, "bad id with spaces")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.NotEmpty(t, w.Header().Get(HeaderRequestID))
		assert.NotEqual(t, "bad id with spaces", w.Header().Get(HeaderRequestID))
	})

	t.Run("recovery catches panic", func(t *testing.T) {
		r := gin.New()
		r.Use(RequestIDMiddleware(), Recovery())
		r.GET("/panic", func(c *gin.Context) { panic("boom") })

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/panic", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "internal server error")
	})

	assert.Equal(t, "abc", truncateString("abc", 10))
	assert.Equal(t, "你好", truncateString("你好世界", 2))
	assert.Equal(t, "123", toString("123"))
	assert.Equal(t, "", toString(nil))
	assert.Equal(t, "code=%5BREDACTED%5D&plain=value&token=%5BREDACTED%5D", maskSensitiveQueryParams("token=secret&plain=value&code=abc"))
	assert.Equal(t, "[parse_error]", maskSensitiveQueryParams("%%%"))
	assert.Equal(t, "127.0.0.1", extractIP("127.0.0.1:8080"))
	assert.Equal(t, "127.0.0.1", extractIP("127.0.0.1"))
}

func TestAuthContextCollectionHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(CtxKeyGlobalCapabilities, []string{"cap:1"})
	require.Equal(t, []string{"cap:1"}, GetGlobalCapabilities(c))
}

func TestRequestLogRouteNeverReturnsDynamicPathValues(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("matched route uses template before and after handler", func(t *testing.T) {
		var beforeHandler string
		var afterHandler string
		r := gin.New()
		r.Use(func(c *gin.Context) {
			beforeHandler = requestLogRoute(c)
			c.Next()
			afterHandler = requestLogRoute(c)
		})
		r.GET("/api/v1/admission/sessions/:token", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})

		const credential = "admission-bearer-secret"
		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/admission/sessions/"+credential,
			nil,
		)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "/api/v1/admission/sessions/:token", beforeHandler)
		assert.Equal(t, beforeHandler, afterHandler)
		assert.NotContains(t, beforeHandler, credential)
	})

	t.Run("unknown route uses fixed label", func(t *testing.T) {
		var observed string
		r := gin.New()
		r.Use(func(c *gin.Context) {
			observed = requestLogRoute(c)
			c.Next()
		})

		const credential = "unknown-bearer-secret"
		req := httptest.NewRequest(
			http.MethodGet,
			"/not-registered/"+credential,
			nil,
		)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, unmatchedRequestRoute, observed)
		assert.NotContains(t, observed, credential)
	})

	t.Run("method mismatch uses fixed label", func(t *testing.T) {
		var observed string
		r := gin.New()
		r.HandleMethodNotAllowed = true
		r.Use(func(c *gin.Context) {
			observed = requestLogRoute(c)
			c.Next()
		})
		r.GET("/api/v1/admission/sessions/:token", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})

		const credential = "method-mismatch-secret"
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/admission/sessions/"+credential,
			nil,
		)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusMethodNotAllowed, w.Code)
		assert.Equal(t, unmatchedRequestRoute, observed)
		assert.NotContains(t, observed, credential)
	})

	assert.Equal(t, unmatchedRequestRoute, requestLogRoute(nil))
}
