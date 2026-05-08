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

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	oidcpkg "git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
)

func newUnavailableRefreshOIDCProvider(t *testing.T) *fakeOIDCProvider {
	t.Helper()
	const clientID = "test-client-id"
	const clientSecret = "test-client-secret"
	var issuer string

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 issuer,
			"authorization_endpoint": issuer + "/authorize",
			"token_endpoint":         issuer + "/token",
			"jwks_uri":               issuer + "/keys",
			"introspection_endpoint": issuer + "/introspect",
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "token endpoint unavailable", http.StatusServiceUnavailable)
	})

	srv := httptest.NewServer(mux)
	issuer = srv.URL
	t.Cleanup(srv.Close)

	client, err := oidcpkg.NewClient(context.Background(), config.CasdoorConfig{
		Issuer:                    issuer,
		ClientID:                  clientID,
		ClientSecret:              clientSecret,
		RedirectURI:               "https://web.example.com/api/v1/auth/callback",
		IntrospectionClientID:     "introspection-client",
		IntrospectionClientSecret: "introspection-secret",
	})
	require.NoError(t, err)
	return &fakeOIDCProvider{server: srv, client: client, clientID: clientID}
}

func TestRefreshOIDCToken_ProviderUnavailableDoesNotClearSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := newUnavailableRefreshOIDCProvider(t)
	h, _ := newOIDCTestHandlerWithProvider(t, nil, provider)

	_, err := h.svc.CreateSession(
		t.Context(),
		"sid-oidc-provider-unavailable",
		"oidc-user-1",
		"old-access-token",
		"old-refresh-token",
		"oidc",
		"browser",
	)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid-oidc-provider-unavailable"})
	c.Request = req

	ok := h.refreshOIDCToken(c, "old-refresh-token")

	assert.False(t, ok)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "refresh service temporarily unavailable")
	assert.Empty(t, w.Result().Cookies())

	session, err := h.tokenService.GetSessionStore().Get(t.Context(), "sid-oidc-provider-unavailable")
	require.NoError(t, err)
	assert.Equal(t, "sid-oidc-provider-unavailable", session.SessionID)
}
