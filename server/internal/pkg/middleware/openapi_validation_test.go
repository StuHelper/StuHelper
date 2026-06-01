package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAPIRequestValidationMiddleware_RejectsInvalidPathParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1")
	validator, err := NewOpenAPIRequestValidationMiddleware()
	require.NoError(t, err)
	api.Use(validator)
	api.GET("/course/courses/:courseID", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/course/courses/not-a-number", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["success"])
}

func TestOpenAPIRequestValidationMiddleware_RejectsInvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1")
	validator, err := NewOpenAPIRequestValidationMiddleware()
	require.NoError(t, err)
	api.Use(validator)
	api.POST("/user/profile/bind-phone", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	body := bytes.NewBufferString(`{"phone":"123","otpCode":"123"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/profile/bind-phone", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["success"])
}

func TestOpenAPIRequestValidationMiddleware_AllowsValidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1")
	validator, err := NewOpenAPIRequestValidationMiddleware()
	require.NoError(t, err)
	api.Use(validator)
	api.POST("/user/profile/bind-phone", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	body := bytes.NewBufferString(`{"phone":"13800138000","otpCode":"123456"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/profile/bind-phone", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOpenAPIRequestValidationMiddleware_AllowsSendBeaconVitalsTextPlain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1")
	validator, err := NewOpenAPIRequestValidationMiddleware()
	require.NoError(t, err)
	api.Use(validator)
	api.POST("/metrics/vitals", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	body := bytes.NewBufferString(`{"name":"LCP","value":123,"rating":"good"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/vitals", body)
	req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestOpenAPIRequestValidationMiddleware_RejectsMethodNotAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	writeOpenAPIValidationError(c, &openapi3filter.ValidationError{
		Status: http.StatusMethodNotAllowed,
		Title:  "method not allowed",
	})

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestOpenAPIRequestValidationMiddleware_RejectsUnsupportedMediaType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1")
	validator, err := NewOpenAPIRequestValidationMiddleware()
	require.NoError(t, err)
	api.Use(validator)
	api.POST("/user/profile/bind-phone", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	body := bytes.NewBufferString("phone=13800138000&otpCode=123456")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/profile/bind-phone", body)
	req.Header.Set("Content-Type", "text/plain")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
}

func TestNewOpenAPIRequestValidationMiddleware_HelperEdges(t *testing.T) {
	t.Run("nil swagger is rejected", func(t *testing.T) {
		_, err := newOpenAPIRequestValidationMiddleware(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "openapi spec is nil")
	})

	t.Run("nil auth func defaults to noop", func(t *testing.T) {
		swagger := &openapi3.T{
			OpenAPI: "3.0.0",
			Info: &openapi3.Info{
				Title:   "test",
				Version: "1.0.0",
			},
			Paths: openapi3.NewPaths(
				openapi3.WithPath("/ping", &openapi3.PathItem{
					Get: &openapi3.Operation{
						Responses: openapi3.NewResponses(
							openapi3.WithStatus(200, &openapi3.ResponseRef{Value: &openapi3.Response{Description: ptr("ok")}}),
						),
					},
				}),
			),
		}
		handler, err := newOpenAPIRequestValidationMiddleware(swagger, nil)
		require.NoError(t, err)
		require.NotNil(t, handler)
	})
}

func TestWriteOpenAPIValidationError_FallbackShapes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("default validation status falls back to 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		writeOpenAPIValidationError(c, &openapi3filter.ValidationError{
			Status: http.StatusBadRequest,
			Title:  "bad request",
			Detail: "detail",
			Source: &openapi3filter.ValidationErrorSource{Parameter: "courseID", Pointer: "/courseID"},
		})

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "courseID")
		assert.Contains(t, w.Body.String(), "/courseID")
	})

	t.Run("non validation errors still return validation envelope", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		writeOpenAPIValidationError(c, assert.AnError)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "request validation failed")
	})
}

func ptr[T any](v T) *T { return &v }
