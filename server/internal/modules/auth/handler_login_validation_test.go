package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/oidc"
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
	req = httptest.NewRequest(http.MethodGet, "/callback?code=%20%09%0A&state=s1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing authorization code")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/callback?code=abc&code=def&state=s1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), string(errs.ErrInvalidParam))

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/callback?code=abc&state=s1&state=s2", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), string(errs.ErrInvalidParam))

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/callback?code=abc&state=s1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid or expired state parameter")

	require.NoError(t, h.storeOIDCState(req.Context(), oidcStateInput{
		state:        "native-state",
		redirect:     "stuhelper://auth/callback",
		codeVerifier: "verifier-1",
		application:  oidc.ApplicationUniapp,
		native:       true,
	}))
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/callback?code=abc123&state=native-state", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "stuhelper://auth/callback")
	assert.Contains(t, w.Header().Get("Location"), "code=abc123")
	assert.Contains(t, w.Header().Get("Location"), "state=native-state")

	require.NoError(t, h.storeOIDCState(req.Context(), oidcStateInput{
		state:        "native-spaced-state",
		redirect:     "stuhelper://auth/callback",
		codeVerifier: "verifier-1",
		application:  oidc.ApplicationUniapp,
		native:       true,
	}))
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/callback?code=%20abc123%20&state=%20native-spaced-state%20", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "code=abc123")
	assert.Contains(t, w.Header().Get("Location"), "state=native-spaced-state")
	assert.NotContains(t, w.Header().Get("Location"), "%20")
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
	req = httptest.NewRequest(http.MethodPost, "/exchange-native", bytes.NewBufferString(`{"code":" \t\n ","state":"missing"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request body")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/exchange-native", bytes.NewBufferString(`{"code":"abc","state":" \t\n "}`))
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
