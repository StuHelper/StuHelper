package auth

import (
	"context"
	"encoding/json"
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
		" https://WEB.example.com ",
		"https://admin.example.com/app",
		"https://join.stuhelper.com",
		"ftp://files.example.com",
		"not-a-url",
		"",
	})

	assert.Contains(t, hosts, "web.example.com")
	assert.Contains(t, hosts, "admin.example.com")
	assert.Contains(t, hosts, "join.stuhelper.com")
	assert.Len(t, hosts, 3)
}

func TestBuildDefaultRedirectURL(t *testing.T) {
	assert.Equal(t, "https://web.example.com", buildDefaultRedirectURL([]string{
		"not-a-url",
		"ftp://files.example.com",
		"  HTTPS://WEB.example.com/app/  ",
		"https://admin.example.com",
	}))
	assert.Panics(t, func() { buildDefaultRedirectURL([]string{"not-a-url", "ftp://files.example.com", "  ", ""}) })
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

func TestStoreNativeCodeVerifierNormalizesPayload(t *testing.T) {
	h, fixture := newTestHandler(t)

	require.NoError(t, h.storeNativeCodeVerifier(context.Background(), " \tstate-1\n ", nativeCodeVerifierPayload{
		CodeVerifier: " \tverifier-1\n ",
		Application:  " \t\n ",
	}))

	raw, err := fixture.Client.Get(context.Background(), nativeCodeVerifierPrefix+"state-1").Result()
	require.NoError(t, err)
	var payload nativeCodeVerifierPayload
	require.NoError(t, json.Unmarshal([]byte(raw), &payload))
	assert.Equal(t, "verifier-1", payload.CodeVerifier)
	assert.Equal(t, oidc.ApplicationUniapp, payload.Application)
}

func TestStoreNativeCodeVerifierRejectsInvalidPayload(t *testing.T) {
	h, _ := newTestHandler(t)

	require.ErrorContains(t, h.storeNativeCodeVerifier(context.Background(), " \t\n ", nativeCodeVerifierPayload{
		CodeVerifier: "verifier-1",
		Application:  oidc.ApplicationUniapp,
	}), "empty state")
	require.ErrorContains(t, h.storeNativeCodeVerifier(context.Background(), "state-missing-verifier", nativeCodeVerifierPayload{
		CodeVerifier: " \t\n ",
		Application:  oidc.ApplicationUniapp,
	}), "native code_verifier missing")
	require.ErrorContains(t, h.storeNativeCodeVerifier(context.Background(), "state-web-app", nativeCodeVerifierPayload{
		CodeVerifier: "verifier-1",
		Application:  oidc.ApplicationWeb,
	}), "native oidc state application mismatch")
}

func TestConsumeNativeCodeVerifierRejectsInvalidPayload(t *testing.T) {
	h, fixture := newTestHandler(t)

	require.NoError(t, fixture.Client.Set(
		context.Background(),
		nativeCodeVerifierPrefix+"state-missing-verifier",
		`{"codeVerifier":" \t\n ","application":"uniapp"}`,
		stateMaxAge,
	).Err())
	_, _, err := h.consumeNativeCodeVerifier(context.Background(), "state-missing-verifier")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "native code_verifier missing")

	require.NoError(t, fixture.Client.Set(
		context.Background(),
		nativeCodeVerifierPrefix+"state-web-app",
		`{"codeVerifier":"verifier-1","application":"web"}`,
		stateMaxAge,
	).Err())
	_, _, err = h.consumeNativeCodeVerifier(context.Background(), "state-web-app")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "native oidc state application mismatch")
}
