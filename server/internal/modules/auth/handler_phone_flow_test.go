package auth

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

func TestVerifyPhoneOTP_SuccessAndFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newRefreshTestHandler(t, &fakeUserSyncRepo{})
	h.otpService = NewOTPService(h.redisClient)

	code, err := h.otpService.Generate(t.Context(), "13800138000")
	require.NoError(t, err)

	r := gin.New()
	r.POST("/otp/verify", h.VerifyPhoneOTP)

	body, err := json.Marshal(map[string]string{"phone": "13800138000", "code": code})
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/otp/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "capabilities")
	assert.Contains(t, w.Body.String(), "canAccessAdmin")
	assert.GreaterOrEqual(t, len(w.Result().Cookies()), 2)

	// same code is one-time use; repeated verify should now be treated as expired
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/otp/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "verification code expired")
}

func TestVerifyPhoneOTP_InvalidCodeAndMaxAttempts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newRefreshTestHandler(t, &fakeUserSyncRepo{})
	h.otpService = NewOTPService(h.redisClient)

	_, err := h.otpService.Generate(t.Context(), "13800138001")
	require.NoError(t, err)

	r := gin.New()
	r.POST("/otp/verify", h.VerifyPhoneOTP)
	badBody := []byte(`{"phone":"13800138001","code":"000000"}`)

	for i := 0; i < otpMaxAttempts; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/otp/verify", bytes.NewReader(badBody))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/otp/verify", bytes.NewReader(badBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "too many failed attempts")
}
