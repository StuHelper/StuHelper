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

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
)

func newBearerOIDCClient(t *testing.T) (*oidc.Client, *httptest.Server) {
	t.Helper()

	const projectID = "project-middleware"
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
	mux.HandleFunc("/introspect", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"active":   true,
			"sub":      "user-1",
			"username": "school-admin",
			"email":    "school-admin@example.com",
			"name":     "School Admin",
			"urn:zitadel:iam:org:project:" + projectID + ":roles": map[string]map[string]string{
				"school_admin": {"school-1": "school.example.com"},
			},
		})
	})

	client, err := oidc.NewClient(context.Background(), config.ZitadelConfig{
		Issuer:       server.URL,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "https://web.example.com/callback",
		ProjectID:    projectID,
	})
	require.NoError(t, err)
	return client, server
}

func TestAuthMiddleware_BearerPreservesOrgScopedRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokenSvc := newTokenServiceForMiddlewareTest(t)
	oidcClient, server := newBearerOIDCClient(t)
	defer server.Close()

	r := gin.New()
	r.Use(AuthMiddleware(oidcClient, tokenSvc))
	r.GET("/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"hasRole":     HasRoleInOrg(c, "school_admin", "school-1"),
			"schoolGrant": HasCapabilityInSchool(c, capability.UserSchoolRead, "school-1"),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer provider-access-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"hasRole":true`)
	assert.Contains(t, w.Body.String(), `"schoolGrant":true`)
}
