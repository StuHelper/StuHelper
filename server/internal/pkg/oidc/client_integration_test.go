package oidc

import (
	"context"
	crand "crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
)

func newTestOIDCClient(t *testing.T) (*Client, *httptest.Server) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(crand.Reader, 2048)
	require.NoError(t, err)
	const clientID = "oidc-client"
	const clientSecret = "oidc-secret"

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
			"amr":                []string{"pwd", "totp"},
			"auth_time":          time.Now().Add(-time.Minute).Unix(),
			"roles":              []string{"school_admin"},
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
		user, pass, ok := r.BasicAuth()
		if !ok || user != "introspection-client" || pass != "introspection-secret" {
			http.Error(w, "invalid introspection credentials", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"active":    true,
			"client_id": "oidc-client",
			"sub":       "user-oidc-1",
			"username":  "oidc-user",
			"email":     "oidc@example.com",
			"name":      "OIDC User",
			"amr":       []string{"pwd", "totp"},
			"auth_time": time.Now().Add(-time.Minute).Unix(),
			"roles":     []string{"school_admin"},
		})
	})

	srv := httptest.NewServer(mux)
	issuer = srv.URL
	client, err := NewClient(context.Background(), config.CasdoorConfig{
		Issuer:                    issuer,
		ClientID:                  clientID,
		ClientSecret:              clientSecret,
		RedirectURI:               "https://web.example.com/api/v1/auth/callback",
		IntrospectionClientID:     "introspection-client",
		IntrospectionClientSecret: "introspection-secret",
	})
	require.NoError(t, err)
	return client, srv
}

func TestOIDCClient_IntegrationFlows(t *testing.T) {
	client, srv := newTestOIDCClient(t)
	defer srv.Close()

	authURL, verifier, err := client.GetAuthURLForApplication(ApplicationWeb, "state-1")
	require.NoError(t, err)
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
	assert.Equal(t, "oidc-client", claims.GetAppID())
	assert.Contains(t, claims.Roles, "school_admin")
	assert.Contains(t, claims.AMR, "totp")
	assert.False(t, claims.MFAProofVerifiedAt().IsZero())
	assert.Nil(t, claims.OrgScopedRoles)

	refreshed, err := client.RefreshToken(context.Background(), "provider-refresh-token")
	require.NoError(t, err)
	assert.Equal(t, "provider-refresh-token", refreshed.RefreshToken)

	result, err := client.IntrospectToken(context.Background(), "provider-access-token")
	require.NoError(t, err)
	assert.True(t, result.Active)
	assert.Equal(t, "oidc-client", result.GetAppID())
	assert.Contains(t, result.Roles, "school_admin")
	assert.False(t, result.MFAProofVerifiedAt().IsZero())
	assert.Nil(t, result.OrgScopedRoles)

	raw, err := marshalIDTokenClaims(mustVerifyIDToken(t, client, idToken))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "oidc-user")
}

func TestOIDCClient_GetAuthURLForApplicationUsesSelectedClient(t *testing.T) {
	client, srv := newTestOIDCClient(t)
	defer srv.Close()

	client.oauth2Configs["admin"] = oauth2.Config{
		ClientID:     "admin-client",
		ClientSecret: "admin-secret",
		RedirectURL:  "https://admin.example.com/api/v1/auth/callback",
		Endpoint:     client.oauth2Cfg.Endpoint,
		Scopes:       client.oauth2Cfg.Scopes,
	}

	authURL, verifier, err := client.GetAuthURLForApplication("admin", "state-admin")
	require.NoError(t, err)
	assert.NotEmpty(t, verifier)
	assert.Contains(t, authURL, "client_id=admin-client")
	assert.Contains(t, authURL, "redirect_uri=https%3A%2F%2Fadmin.example.com%2Fapi%2Fv1%2Fauth%2Fcallback")
	assert.Contains(t, authURL, "state=state-admin")
}

func TestVerifyIDTokenUnknownKidFetchFailureIsProviderUnavailable(t *testing.T) {
	privateKey, err := rsa.GenerateKey(crand.Reader, 2048)
	require.NoError(t, err)
	rotatedKey, err := rsa.GenerateKey(crand.Reader, 2048)
	require.NoError(t, err)

	const clientID = "oidc-client"
	jwk := jose.JSONWebKey{Key: &privateKey.PublicKey, KeyID: "kid-1", Algorithm: string(jose.RS256), Use: "sig"}
	var issuer string
	keysAvailable := true
	issueIDToken := func(key *rsa.PrivateKey, kid string) string {
		signer, err := jose.NewSigner(jose.SigningKey{
			Algorithm: jose.RS256,
			Key:       jose.JSONWebKey{Key: key, KeyID: kid},
		}, nil)
		require.NoError(t, err)
		raw, err := josejwt.Signed(signer).Claims(map[string]any{
			"iss": issuer,
			"sub": "user-oidc-1",
			"aud": clientID,
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
		}).Serialize()
		require.NoError(t, err)
		return raw
	}

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
		if !keysAvailable {
			http.Error(w, "jwks unavailable", http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	issuer = srv.URL

	client, err := NewClient(context.Background(), config.CasdoorConfig{
		Issuer:                    issuer,
		ClientID:                  clientID,
		ClientSecret:              "oidc-secret",
		RedirectURI:               "https://web.example.com/api/v1/auth/callback",
		IntrospectionClientID:     "introspection-client",
		IntrospectionClientSecret: "introspection-secret",
	})
	require.NoError(t, err)
	_, err = client.VerifyIDToken(context.Background(), issueIDToken(privateKey, "kid-1"))
	require.NoError(t, err)

	keysAvailable = false
	_, err = client.VerifyIDToken(context.Background(), issueIDToken(rotatedKey, "kid-2"))
	require.ErrorIs(t, err, ErrProviderUnavailable)
}

func TestProviderUnavailableKeySetExpiresCachedJWKS(t *testing.T) {
	privateKey, err := rsa.GenerateKey(crand.Reader, 2048)
	require.NoError(t, err)
	jwk := jose.JSONWebKey{Key: &privateKey.PublicKey, KeyID: "kid-ttl", Algorithm: string(jose.RS256), Use: "sig"}
	var keyRequests int64
	mux := http.NewServeMux()
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&keyRequests, 1)
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	now := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	keySet := newProviderUnavailableKeySet(context.Background(), srv.URL+"/keys", time.Minute)
	keySet.now = func() time.Time { return now }
	token := issueSignedJWT(t, privateKey, jwk.KeyID, map[string]any{"sub": "user-1"})

	_, err = keySet.VerifySignature(context.Background(), token)
	require.NoError(t, err)
	_, err = keySet.VerifySignature(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, int64(1), atomic.LoadInt64(&keyRequests))

	now = now.Add(time.Minute + time.Second)
	_, err = keySet.VerifySignature(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, int64(2), atomic.LoadInt64(&keyRequests))
}

func TestVerifyIDTokenRejectsDisallowedAlgorithmBeforeJWKSFetch(t *testing.T) {
	const clientID = "oidc-client"
	var keyRequests int64
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
		atomic.AddInt64(&keyRequests, 1)
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	issuer = srv.URL

	client, err := NewClient(context.Background(), config.CasdoorConfig{
		Issuer:                    issuer,
		ClientID:                  clientID,
		ClientSecret:              "oidc-secret",
		RedirectURI:               "https://web.example.com/api/v1/auth/callback",
		IntrospectionClientID:     "introspection-client",
		IntrospectionClientSecret: "introspection-secret",
	})
	require.NoError(t, err)
	token := issueHS256IDToken(t, issuer, clientID)

	_, err = client.VerifyIDToken(context.Background(), token)
	require.ErrorIs(t, err, errDisallowedJWTAlgorithm)
	assert.Equal(t, int64(0), atomic.LoadInt64(&keyRequests))
}

func issueSignedJWT(t *testing.T, privateKey *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.RS256,
		Key:       jose.JSONWebKey{Key: privateKey, KeyID: kid},
	}, nil)
	require.NoError(t, err)
	raw, err := josejwt.Signed(signer).Claims(claims).Serialize()
	require.NoError(t, err)
	return raw
}

func issueHS256IDToken(t *testing.T, issuer, clientID string) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.HS256,
		Key:       []byte("test-hs256-secret-with-enough-entropy"),
	}, (&jose.SignerOptions{}).WithHeader("kid", "hs-1"))
	require.NoError(t, err)
	raw, err := josejwt.Signed(signer).Claims(map[string]any{
		"iss": issuer,
		"sub": "user-oidc-1",
		"aud": clientID,
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}).Serialize()
	require.NoError(t, err)
	return raw
}

func TestOIDCClientRemoteEndpointFailuresAreProviderUnavailable(t *testing.T) {
	const clientID = "oidc-client"
	const clientSecret = "oidc-secret"
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
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "token endpoint unavailable", http.StatusServiceUnavailable)
	})
	mux.HandleFunc("/introspect", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "introspection unavailable", http.StatusServiceUnavailable)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	issuer = srv.URL

	client, err := NewClient(context.Background(), config.CasdoorConfig{
		Issuer:                    issuer,
		ClientID:                  clientID,
		ClientSecret:              clientSecret,
		RedirectURI:               "https://web.example.com/api/v1/auth/callback",
		IntrospectionClientID:     "introspection-client",
		IntrospectionClientSecret: "introspection-secret",
	})
	require.NoError(t, err)
	_, err = client.RefreshToken(context.Background(), "old-refresh-token")
	require.ErrorIs(t, err, ErrProviderUnavailable)
	_, err = client.IntrospectToken(context.Background(), "provider-access-token")
	require.ErrorIs(t, err, ErrProviderUnavailable)
}

func mustVerifyIDToken(t *testing.T, client *Client, raw string) *gooidc.IDToken {
	t.Helper()
	idToken, err := client.verifier.Verify(context.Background(), raw)
	require.NoError(t, err)
	return idToken
}

func TestClassifyOAuthErrorLeavesClientErrorsAsCredentialFailures(t *testing.T) {
	err := classifyOAuthError(&oauth2.RetrieveError{Response: &http.Response{StatusCode: http.StatusUnauthorized}})

	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrProviderUnavailable))
}
