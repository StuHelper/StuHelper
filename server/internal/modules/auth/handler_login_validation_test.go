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

func TestHandleCallback_ValidationAndNativeBranch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newTestHandler(t)
	r := gin.New()
	r.GET("/callback", h.HandleCallback)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/callback?state=s1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing authorization code")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/callback?code=abc&state=s1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid or expired state parameter")

	require.NoError(t, h.storeOIDCState(req.Context(), "native-state", "stuhelper://auth/callback", "verifier-1", true))
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/callback?code=abc123&state=native-state", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "stuhelper://auth/callback")
	assert.Contains(t, w.Header().Get("Location"), "code=abc123")
	assert.Contains(t, w.Header().Get("Location"), "state=native-state")
}

func TestExchangeNative_ValidationBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newTestHandler(t)
	r := gin.New()
	r.POST("/exchange-native", h.ExchangeNative)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/exchange-native", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request body")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/exchange-native", bytes.NewBufferString(`{"code":"abc","state":"missing"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid or expired state parameter")
}
