package oidc

import (
	"context"
	crand "crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
)

func newTestOIDCClient(t *testing.T) (*Client, *httptest.Server) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(crand.Reader, 2048)
	require.NoError(t, err)
	const clientID = "oidc-client"
	const clientSecret = "oidc-secret"
	const projectID = "project-oidc"

	jwk := jose.JSONWebKey{Key: &privateKey.PublicKey, KeyID: "kid-1", Algorithm: string(jose.RS256), Use: "sig"}
	var issuer string

	issueIDToken := func() string {
		signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: jose.JSONWebKey{Key: privateKey, KeyID: jwk.KeyID}}, nil)
		require.NoError(t, err)
		raw, err := josejwt.Signed(signer).Claims(map[string]any{
			"iss":                issuer,
			"sub":                "user-oidc-1",
			"aud":                clientID,
			"exp":                time.Now().Add(time.Hour).Unix(),
			"iat":                time.Now().Unix(),
			"name":               "OIDC User",
			"preferred_username": "oidc-user",
			"email":              "oidc@example.com",
			"picture":            "https://cdn.example.com/oidc.png",
			"urn:zitadel:iam:org:project:" + projectID + ":roles": map[string]map[string]string{
				"school_admin": {"school-1": "example.org"},
			},
		}).Serialize()
		require.NoError(t, err)
		return raw
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 issuer,
			"authorization_endpoint": issuer + "/authorize",
			"token_endpoint":         issuer + "/token",
			"jwks_uri":               issuer + "/keys",
			"introspection_endpoint": issuer + "/introspect",
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "provider-access-token",
			"token_type":    "Bearer",
			"refresh_token": "provider-refresh-token",
			"expires_in":    3600,
			"id_token":      issueIDToken(),
		})
	})
	mux.HandleFunc("/introspect", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"active":   true,
			"sub":      "user-oidc-1",
			"username": "oidc-user",
			"email":    "oidc@example.com",
			"name":     "OIDC User",
			"urn:zitadel:iam:org:project:" + projectID + ":roles": map[string]map[string]string{
				"school_admin": {"school-1": "example.org"},
			},
		})
	})

	srv := httptest.NewServer(mux)
	issuer = srv.URL
	client, err := NewClient(context.Background(), config.ZitadelConfig{
		Issuer:       issuer,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  "https://web.example.com/api/v1/auth/callback",
		ProjectID:    projectID,
	})
	require.NoError(t, err)
	return client, srv
}

func TestOIDCClient_IntegrationFlows(t *testing.T) {
	client, srv := newTestOIDCClient(t)
	defer srv.Close()

	authURL, verifier := client.GetAuthURL("state-1")
	assert.Contains(t, authURL, srv.URL+"/authorize")
	assert.NotEmpty(t, verifier)

	tok, err := client.ExchangeCode(context.Background(), "code-1", verifier)
	require.NoError(t, err)
	assert.Equal(t, "provider-refresh-token", tok.RefreshToken)

	idToken := ExtractIDToken(tok)
	require.NotEmpty(t, idToken)
	claims, err := client.VerifyIDToken(context.Background(), idToken)
	require.NoError(t, err)
	assert.Equal(t, "user-oidc-1", claims.GetUserID())
	assert.True(t, claims.HasRoleInOrg("school_admin", "school-1"))

	refreshed, err := client.RefreshToken(context.Background(), "provider-refresh-token")
	require.NoError(t, err)
	assert.Equal(t, "provider-refresh-token", refreshed.RefreshToken)

	result, err := client.IntrospectToken(context.Background(), "provider-access-token")
	require.NoError(t, err)
	assert.True(t, result.Active)
	assert.Contains(t, result.Roles, "school_admin")

	raw, err := marshalIDTokenClaims(mustVerifyIDToken(t, client, idToken))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "oidc-user")
}

func mustVerifyIDToken(t *testing.T, client *Client, raw string) *gooidc.IDToken {
	t.Helper()
	idToken, err := client.verifier.Verify(context.Background(), raw)
	require.NoError(t, err)
	return idToken
}
