package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
)

func TestBuildAllowedRedirectHosts(t *testing.T) {
	hosts := buildAllowedRedirectHosts([]string{
		" https://web.example.com ",
		"https://admin.example.com/app",
		"https://join.stuhelper.com",
		"not-a-url",
		"",
	})

	assert.Contains(t, hosts, "web.example.com")
	assert.Contains(t, hosts, "admin.example.com")
	assert.Contains(t, hosts, "join.stuhelper.com")
	assert.Len(t, hosts, 3)
}

func TestBuildDefaultRedirectURL(t *testing.T) {
	assert.Equal(t, "https://web.example.com", buildDefaultRedirectURL([]string{"  https://web.example.com/  ", "https://admin.example.com"}))
	assert.Panics(t, func() { buildDefaultRedirectURL([]string{"  ", ""}) })
}

func TestHandleNativeCallbackRedirect(t *testing.T) {
	h, _ := newTestHandler(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback", nil)

	h.handleNativeCallbackRedirect(c, "code+123", "state/456")

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "stuhelper://auth/callback?code=code%2B123&state=state%2F456", w.Header().Get("Location"))
}

func TestNativeCodeVerifierRoundTrip(t *testing.T) {
	h, _ := newTestHandler(t)

	require.NoError(t, h.storeNativeCodeVerifier(context.Background(), "state-1", nativeCodeVerifierPayload{
		CodeVerifier: "verifier-1",
		Application:  oidc.ApplicationUniapp,
	}))
	verifier, appKey, err := h.consumeNativeCodeVerifier(context.Background(), "state-1")
	require.NoError(t, err)
	assert.Equal(t, "verifier-1", verifier)
	assert.Equal(t, oidc.ApplicationUniapp, appKey)

	_, _, err = h.consumeNativeCodeVerifier(context.Background(), "state-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired or already used")
}
