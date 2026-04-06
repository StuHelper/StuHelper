package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// parseBody is a helper that unmarshals the recorder body into a Response.
func parseBody(t *testing.T, w *httptest.ResponseRecorder) Response {
	t.Helper()
	var resp Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

// parseBodyRaw returns the raw JSON map for fields that need type-specific checks.
func parseBodyRaw(t *testing.T, w *httptest.ResponseRecorder) map[string]json.RawMessage {
	t.Helper()
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &m))
	return m
}

// ---------------------------------------------------------------------------
// Success
// ---------------------------------------------------------------------------

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/ok", func(c *gin.Context) {
		Success(c, map[string]string{"greeting": "hello"})
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	resp := parseBody(t, w)
	assert.True(t, resp.Success)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
}

func TestSuccess_NilData(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/ok", func(c *gin.Context) {
		Success(c, nil)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	raw := parseBodyRaw(t, w)

	// success must be true
	var success bool
	require.NoError(t, json.Unmarshal(raw["success"], &success))
	assert.True(t, success)

	// data should be omitted (omitempty)
	_, hasData := raw["data"]
	assert.False(t, hasData, "data should be omitted when nil")

	// error should be omitted
	_, hasErr := raw["error"]
	assert.False(t, hasErr)
}

// ---------------------------------------------------------------------------
// Created
// ---------------------------------------------------------------------------

func TestCreated(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.POST("/create", func(c *gin.Context) {
		Created(c, map[string]int{"id": 1})
	})

	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	resp := parseBody(t, w)
	assert.True(t, resp.Success)
	assert.Nil(t, resp.Error)
}

// ---------------------------------------------------------------------------
// Error
// ---------------------------------------------------------------------------

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/err", func(c *gin.Context) {
		Error(c, http.StatusBadRequest, errs.ErrBadRequest, "invalid input")
	})

	req := httptest.NewRequest(http.MethodGet, "/err", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseBody(t, w)
	assert.False(t, resp.Success)
	assert.Nil(t, resp.Data)
	require.NotNil(t, resp.Error)
	assert.Equal(t, string(errs.ErrBadRequest), resp.Error.Code)
	assert.Equal(t, "invalid input", resp.Error.Message)
	assert.Nil(t, resp.Error.Details)
}

// ---------------------------------------------------------------------------
// ErrorWithDetails
// ---------------------------------------------------------------------------

func TestErrorWithDetails(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	details := map[string]string{"field": "email", "reason": "required"}
	r.POST("/err-detail", func(c *gin.Context) {
		ErrorWithDetails(c, http.StatusBadRequest, errs.ErrValidation, "validation failed", details)
	})

	req := httptest.NewRequest(http.MethodPost, "/err-detail", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseBody(t, w)
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)
	assert.Equal(t, string(errs.ErrValidation), resp.Error.Code)
	assert.Equal(t, "validation failed", resp.Error.Message)
	assert.NotNil(t, resp.Error.Details)
}

// ---------------------------------------------------------------------------
// Convenience helpers — verify correct HTTP status and default error code
// ---------------------------------------------------------------------------

func TestBadRequest(t *testing.T) {
	t.Run("default code", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.GET("/t", func(c *gin.Context) { BadRequest(c, "bad") })

		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/t", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)

		resp := parseBody(t, w)
		require.NotNil(t, resp.Error)
		assert.Equal(t, string(errs.ErrBadRequest), resp.Error.Code)
	})

	t.Run("custom code", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.GET("/t", func(c *gin.Context) { BadRequest(c, "bad", errs.ErrInvalidParam) })

		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/t", nil))
		resp := parseBody(t, w)
		require.NotNil(t, resp.Error)
		assert.Equal(t, string(errs.ErrInvalidParam), resp.Error.Code)
	})
}

func TestValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.POST("/t", func(c *gin.Context) {
		ValidationError(c, "invalid", []string{"field1"})
	})

	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/t", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseBody(t, w)
	require.NotNil(t, resp.Error)
	assert.Equal(t, string(errs.ErrValidation), resp.Error.Code)
	assert.NotNil(t, resp.Error.Details)
}

func TestUnauthorized(t *testing.T) {
	t.Run("default code", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.GET("/t", func(c *gin.Context) { Unauthorized(c, "login required") })

		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/t", nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		resp := parseBody(t, w)
		require.NotNil(t, resp.Error)
		assert.Equal(t, string(errs.ErrLoginRequired), resp.Error.Code)
	})

	t.Run("custom code", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.GET("/t", func(c *gin.Context) { Unauthorized(c, "expired", errs.ErrTokenExpired) })

		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/t", nil))
		resp := parseBody(t, w)
		require.NotNil(t, resp.Error)
		assert.Equal(t, string(errs.ErrTokenExpired), resp.Error.Code)
	})
}

func TestForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/t", func(c *gin.Context) { Forbidden(c, "no access") })

	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/t", nil))
	assert.Equal(t, http.StatusForbidden, w.Code)

	resp := parseBody(t, w)
	require.NotNil(t, resp.Error)
	assert.Equal(t, string(errs.ErrForbidden), resp.Error.Code)
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/t", func(c *gin.Context) { NotFound(c, "not found") })

	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/t", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := parseBody(t, w)
	require.NotNil(t, resp.Error)
	assert.Equal(t, string(errs.ErrNotFound), resp.Error.Code)
}

func TestConflict(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.POST("/t", func(c *gin.Context) { Conflict(c, "conflict") })

	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/t", nil))
	assert.Equal(t, http.StatusConflict, w.Code)

	resp := parseBody(t, w)
	require.NotNil(t, resp.Error)
	assert.Equal(t, string(errs.ErrConflict), resp.Error.Code)
}

func TestInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/t", func(c *gin.Context) { InternalError(c, "oops") })

	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/t", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	resp := parseBody(t, w)
	require.NotNil(t, resp.Error)
	assert.Equal(t, string(errs.ErrInternal), resp.Error.Code)
}

func TestRateLimitExceeded(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/t", func(c *gin.Context) { RateLimitExceeded(c, "slow down") })

	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/t", nil))
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	resp := parseBody(t, w)
	require.NotNil(t, resp.Error)
	assert.Equal(t, string(errs.ErrRateLimited), resp.Error.Code)
}

func TestServiceUnavailable(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/t", func(c *gin.Context) { ServiceUnavailable(c, "unavailable") })

	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/t", nil))
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	resp := parseBody(t, w)
	require.NotNil(t, resp.Error)
	assert.Equal(t, string(errs.ErrServiceUnavailable), resp.Error.Code)
}

// ---------------------------------------------------------------------------
// Error aborts handler chain
// ---------------------------------------------------------------------------

func TestError_AbortsHandlerChain(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	reached := false
	r.GET("/abort",
		func(c *gin.Context) {
			Error(c, http.StatusForbidden, errs.ErrForbidden, "denied")
		},
		func(c *gin.Context) {
			reached = true
			c.Status(http.StatusOK)
		},
	)

	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/abort", nil))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, reached, "second handler should not have been reached after Error")
}

// ---------------------------------------------------------------------------
// JSON envelope structure
// ---------------------------------------------------------------------------

func TestResponse_JSONStructure_Success(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/s", func(c *gin.Context) {
		Success(c, map[string]int{"count": 5})
	})

	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/s", nil))

	raw := parseBodyRaw(t, w)

	// Must have "success" key
	_, ok := raw["success"]
	assert.True(t, ok, "response must contain 'success' key")

	// Must have "data" key
	_, ok = raw["data"]
	assert.True(t, ok, "success response must contain 'data' key")

	// Must not have "error" key (omitempty)
	_, ok = raw["error"]
	assert.False(t, ok, "success response should not contain 'error' key")
}

func TestResponse_JSONStructure_Error(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/e", func(c *gin.Context) {
		Error(c, http.StatusBadRequest, errs.ErrBadRequest, "bad")
	})

	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/e", nil))

	raw := parseBodyRaw(t, w)

	// Must have "success" key
	_, ok := raw["success"]
	assert.True(t, ok, "response must contain 'success' key")

	// Must have "error" key
	_, ok = raw["error"]
	assert.True(t, ok, "error response must contain 'error' key")

	// Must not have "data" key (omitempty)
	_, ok = raw["data"]
	assert.False(t, ok, "error response should not contain 'data' key")

	// Error object must have code and message
	var apiErr APIError
	require.NoError(t, json.Unmarshal(raw["error"], &apiErr))
	assert.NotEmpty(t, apiErr.Code)
	assert.NotEmpty(t, apiErr.Message)
}
