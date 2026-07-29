package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

func TestOriginValidationMiddleware_AllowsMatchingOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(OriginValidationMiddleware([]string{"http://localhost:5173"}))
	router.POST("/metrics", func(c *gin.Context) { response.NoContent(c) })

	req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestOriginValidationMiddleware_AllowsRefererFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(OriginValidationMiddleware([]string{"http://localhost:5173"}))
	router.POST("/metrics", func(c *gin.Context) { response.NoContent(c) })

	req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	req.Header.Set("Referer", "http://localhost:5173/course/review")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestOriginValidationMiddleware_AllowsSameOriginFetchMetadataFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(OriginValidationMiddleware([]string{"https://join.stuhelper.com"}))
	router.POST("/metrics", func(c *gin.Context) { response.NoContent(c) })

	req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	req.Host = "join.stuhelper.com"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestOriginValidationMiddleware_RejectsSameSiteFetchMetadataFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(OriginValidationMiddleware([]string{"https://join.stuhelper.com"}))
	router.POST("/metrics", func(c *gin.Context) { response.NoContent(c) })

	req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	req.Host = "join.stuhelper.com"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assertForbiddenResponse(t, w)
}

func TestOriginValidationMiddleware_RejectsUnknownOrEmptyAllowlist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("unknown origin", func(t *testing.T) {
		router := gin.New()
		router.Use(OriginValidationMiddleware([]string{"http://localhost:5173"}))
		router.POST("/metrics", func(c *gin.Context) { response.NoContent(c) })

		req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
		req.Header.Set("Origin", "https://evil.example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assertForbiddenResponse(t, w)
	})

	t.Run("empty allowlist", func(t *testing.T) {
		router := gin.New()
		router.Use(OriginValidationMiddleware(nil))
		router.POST("/metrics", func(c *gin.Context) { response.NoContent(c) })

		req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assertForbiddenResponse(t, w)
	})
}

func assertForbiddenResponse(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	var body response.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotNil(t, body.Error)
	assert.False(t, body.Success)
	assert.Equal(t, string(errs.ErrForbidden), body.Error.Code)
	assert.Equal(t, "forbidden", body.Error.Message)
}
