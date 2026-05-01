package middleware

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
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
)

func newBearerOIDCClient(t *testing.T) (*oidc.Client, *httptest.Server) {
	t.Helper()
	return newBearerOIDCClientWithIntrospection(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"active":   true,
			"sub":      "user-1",
			"username": "school-admin",
			"email":    "school-admin@example.com",
			"name":     "School Admin",
			"roles":    []string{"school_admin"},
		})
	})
}

func newBearerOIDCClientWithIntrospection(
	t *testing.T,
	introspectHandler http.HandlerFunc,
) (*oidc.Client, *httptest.Server) {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 server.URL,
			"authorization_endpoint": server.URL + "/authorize",
			"token_endpoint":         server.URL + "/token",
			"jwks_uri":               server.URL + "/keys",
			"introspection_endpoint": server.URL + "/introspect",
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{}})
	})
	mux.HandleFunc("/introspect", introspectHandler)

	client, err := oidc.NewClient(context.Background(), config.CasdoorConfig{
		Issuer:       server.URL,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "https://web.example.com/callback",
	})
	require.NoError(t, err)
	return client, server
}

func TestAuthMiddleware_BearerUsesFlatCasdoorRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokenSvc := newTokenServiceForMiddlewareTest(t)
	oidcClient, server := newBearerOIDCClient(t)
	defer server.Close()

	r := gin.New()
	r.Use(AuthMiddleware(oidcClient, tokenSvc))
	r.GET("/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"roles":       GetRoles(c),
			"hasOrgScope": HasRoleInOrg(c, "school_admin", "school-1"),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer provider-access-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"school_admin"`)
	assert.Contains(t, w.Body.String(), `"hasOrgScope":false`)
}

func TestAuthMiddleware_BearerProviderUnavailableReturns503(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokenSvc := newTokenServiceForMiddlewareTest(t)
	oidcClient, server := newBearerOIDCClientWithIntrospection(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "introspection unavailable", http.StatusServiceUnavailable)
	})
	defer server.Close()

	r := gin.New()
	r.Use(AuthMiddleware(oidcClient, tokenSvc))
	r.GET("/me", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer provider-access-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), string(errs.ErrServiceUnavailable))
}

func TestOptionalAuthMiddleware_BearerProviderUnavailableMarksDiagnostic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokenSvc := newTokenServiceForMiddlewareTest(t)
	oidcClient, server := newBearerOIDCClientWithIntrospection(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "introspection unavailable", http.StatusServiceUnavailable)
	})
	defer server.Close()

	r := gin.New()
	r.Use(OptionalAuthMiddleware(oidcClient, tokenSvc, OptionalAuthConfig{}))
	r.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"backendFailure": c.GetBool(CtxKeyAuthBackendFailure),
			"userID":         GetUserID(c),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/public", nil)
	req.Header.Set("Authorization", "Bearer provider-access-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"backendFailure":true`)
	assert.Contains(t, w.Body.String(), `"userID":""`)
}
