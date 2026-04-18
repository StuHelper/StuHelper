package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhoneHandlers_ValidationAndUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{}
	r := gin.New()
	r.POST("/otp/request", h.RequestPhoneOTP)
	r.POST("/otp/verify", h.VerifyPhoneOTP)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/otp/request", bytes.NewBufferString(`{"phone":"bad"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid phone number format")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/otp/request", bytes.NewBufferString(`{"phone":"13800138000"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "phone login is not configured")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/otp/verify", bytes.NewBufferString(`{"phone":"bad","code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid phone number format")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/otp/verify", bytes.NewBufferString(`{"phone":"13800138000","code":"123"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "verification code must be 6 digits")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/otp/verify", bytes.NewBufferString(`{"phone":"13800138000","code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "phone login is not configured")
}

func TestMaskPhone(t *testing.T) {
	assert.Equal(t, "138****8000", maskPhone("13800138000"))
	assert.Equal(t, "***", maskPhone("12345"))
}
