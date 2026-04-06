package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
	api.POST("/auth/phone/request-otp", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	body := bytes.NewBufferString(`{"phone":"123"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/phone/request-otp", body)
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
	api.POST("/auth/phone/request-otp", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	body := bytes.NewBufferString(`{"phone":"13800138000"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/phone/request-otp", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
