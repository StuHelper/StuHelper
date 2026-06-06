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

func TestOIDCClientIntrospectTokenNormalizesInputs(t *testing.T) {
	var seenToken string
	var seenUser string
	var seenPass string
	client, srv := newIntrospectionOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenUser, seenPass, _ = r.BasicAuth()
		require.NoError(t, r.ParseForm())
		seenToken = r.Form.Get("token")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"active":    true,
			"client_id": "oidc-client",
			"sub":       "user-oidc-1",
		})
	}), func(cfg *config.CasdoorConfig) {
		cfg.IntrospectionClientID = " \tintrospection-client\n "
		cfg.IntrospectionClientSecret = " \tintrospection-secret\n "
	})
	defer srv.Close()

	result, err := client.IntrospectToken(context.Background(), " \tprovider-access-token\n ")

	require.NoError(t, err)
	assert.True(t, result.Active)
	assert.Equal(t, "oidc-client", result.GetAppID())
	assert.Equal(t, "provider-access-token", seenToken)
	assert.Equal(t, "introspection-client", seenUser)
	assert.Equal(t, "introspection-secret", seenPass)
}

func TestOIDCClientIntrospectTokenRejectsBlankTokenWithoutProviderCall(t *testing.T) {
	var introspectionRequests atomic.Int32
	client, srv := newIntrospectionOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		introspectionRequests.Add(1)
		w.WriteHeader(http.StatusOK)
	}), nil)
	defer srv.Close()

	_, err := client.IntrospectToken(context.Background(), " \t\n ")

	require.ErrorIs(t, err, ErrInvalidAccessToken)
	assert.Equal(t, int32(0), introspectionRequests.Load())
}

func TestIntrospectionOAuth2ConfigNormalizesCredentials(t *testing.T) {
	cfg, err := introspectionOAuth2Config(oauth2.Endpoint{}, config.CasdoorConfig{
		IntrospectionClientID:     " \tclient-id\n ",
		IntrospectionClientSecret: " \tclient-secret\n ",
	})
	require.NoError(t, err)
	assert.Equal(t, "client-id", cfg.ClientID)
	assert.Equal(t, "client-secret", cfg.ClientSecret)

	_, err = introspectionOAuth2Config(oauth2.Endpoint{}, config.CasdoorConfig{
		IntrospectionClientID:     " \t\n ",
		IntrospectionClientSecret: "client-secret",
	})
	require.ErrorIs(t, err, ErrApplicationNotConfigured)

	_, err = introspectionOAuth2Config(oauth2.Endpoint{}, config.CasdoorConfig{
		IntrospectionClientID:     "client-id",
		IntrospectionClientSecret: " \t\n ",
	})
	require.ErrorIs(t, err, ErrApplicationNotConfigured)
}

func TestOIDCClientExchangeCodeForApplicationNormalizesInputs(t *testing.T) {
	var seenCode string
	var seenCodeVerifier string
	var seenClientID string
	var seenGrantType string
	client, srv := newTokenEndpointOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		seenCode = r.Form.Get("code")
		seenCodeVerifier = r.Form.Get("code_verifier")
		seenGrantType = r.Form.Get("grant_type")
		if user, _, ok := r.BasicAuth(); ok {
			seenClientID = user
		} else {
			seenClientID = r.Form.Get("client_id")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "provider-access-token",
			"token_type":    "Bearer",
			"refresh_token": "provider-refresh-token",
			"expires_in":    3600,
		})
	}))
	defer srv.Close()
	adminCfg := client.oauth2Cfg
	adminCfg.ClientID = "admin-client"
	adminCfg.ClientSecret = "admin-secret"
	client.oauth2Configs[ApplicationAdmin] = adminCfg

	tok, err := client.ExchangeCodeForApplication(
		context.Background(),
		" \t"+ApplicationAdmin+"\n ",
		" \tauthorization-code\n ",
		" \tpkce-verifier\n ",
	)

	require.NoError(t, err)
	assert.Equal(t, "provider-refresh-token", tok.RefreshToken)
	assert.Equal(t, "authorization_code", seenGrantType)
	assert.Equal(t, "authorization-code", seenCode)
	assert.Equal(t, "pkce-verifier", seenCodeVerifier)
	assert.Equal(t, "admin-client", seenClientID)
}

func TestOIDCClientExchangeCodeRejectsBlankInputsWithoutProviderCall(t *testing.T) {
	var tokenRequests atomic.Int32
	client, srv := newTokenEndpointOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		tokenRequests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := client.ExchangeCodeForApplication(context.Background(), ApplicationWeb, " \t\n ", "pkce-verifier")
	require.ErrorIs(t, err, ErrAuthorizationCodeRequired)

	_, err = client.ExchangeCodeForApplication(context.Background(), ApplicationWeb, "authorization-code", " \t\n ")
	require.ErrorIs(t, err, ErrPKCEVerifierRequired)

	assert.Equal(t, int32(0), tokenRequests.Load())
}

func TestOIDCClientRefreshTokenForApplicationNormalizesInputs(t *testing.T) {
	var seenRefreshToken string
	var seenClientID string
	var seenGrantType string
	client, srv := newTokenEndpointOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		seenRefreshToken = r.Form.Get("refresh_token")
		seenGrantType = r.Form.Get("grant_type")
		if user, _, ok := r.BasicAuth(); ok {
			seenClientID = user
		} else {
			seenClientID = r.Form.Get("client_id")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "provider-access-token",
			"token_type":    "Bearer",
			"refresh_token": "next-refresh-token",
			"expires_in":    3600,
		})
	}))
	defer srv.Close()
	adminCfg := client.oauth2Cfg
	adminCfg.ClientID = "admin-client"
	adminCfg.ClientSecret = "admin-secret"
	client.oauth2Configs[ApplicationAdmin] = adminCfg

	tok, err := client.RefreshTokenForApplication(
		context.Background(),
		" \t"+ApplicationAdmin+"\n ",
		" \tprovider-refresh-token\n ",
	)

	require.NoError(t, err)
	assert.Equal(t, "next-refresh-token", tok.RefreshToken)
	assert.Equal(t, "refresh_token", seenGrantType)
	assert.Equal(t, "provider-refresh-token", seenRefreshToken)
	assert.Equal(t, "admin-client", seenClientID)
}

func TestOIDCClientRefreshTokenRejectsBlankTokenWithoutProviderCall(t *testing.T) {
	var tokenRequests atomic.Int32
	client, srv := newTokenEndpointOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		tokenRequests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := client.RefreshTokenForApplication(context.Background(), " \t"+ApplicationWeb+"\n ", " \t\n ")

	require.ErrorIs(t, err, ErrInvalidRefreshToken)
	assert.Equal(t, int32(0), tokenRequests.Load())
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

func newTokenEndpointOIDCClient(t *testing.T, tokenHandler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	const clientID = "oidc-client"
	const clientSecret = "oidc-secret"
	var issuer string

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 issuer,
			"authorization_endpoint": issuer + "/authorize",
			"token_endpoint":         issuer + "/token",
			"jwks_uri":               issuer + "/keys",
			"introspection_endpoint": issuer + "/introspect",
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{}})
	})
	mux.HandleFunc("/token", tokenHandler)

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

func newIntrospectionOIDCClient(
	t *testing.T,
	introspectionHandler http.HandlerFunc,
	configure func(*config.CasdoorConfig),
) (*Client, *httptest.Server) {
	t.Helper()
	const clientID = "oidc-client"
	const clientSecret = "oidc-secret"
	var issuer string

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 issuer,
			"authorization_endpoint": issuer + "/authorize",
			"token_endpoint":         issuer + "/token",
			"jwks_uri":               issuer + "/keys",
			"introspection_endpoint": issuer + "/introspect",
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{}})
	})
	mux.HandleFunc("/introspect", introspectionHandler)

	srv := httptest.NewServer(mux)
	issuer = srv.URL
	cfg := config.CasdoorConfig{
		Issuer:                    issuer,
		ClientID:                  clientID,
		ClientSecret:              clientSecret,
		RedirectURI:               "https://web.example.com/api/v1/auth/callback",
		IntrospectionClientID:     "introspection-client",
		IntrospectionClientSecret: "introspection-secret",
	}
	if configure != nil {
		configure(&cfg)
	}
	client, err := NewClient(context.Background(), cfg)
	require.NoError(t, err)
	return client, srv
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

func TestVerifyIDTokenNormalizesInputs(t *testing.T) {
	privateKey, err := rsa.GenerateKey(crand.Reader, 2048)
	require.NoError(t, err)

	const clientID = "oidc-client"
	jwk := jose.JSONWebKey{Key: &privateKey.PublicKey, KeyID: "kid-normalize", Algorithm: string(jose.RS256), Use: "sig"}
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
	token := issueSignedJWT(t, privateKey, jwk.KeyID, map[string]any{
		"iss": issuer,
		"sub": "user-oidc-1",
		"aud": clientID,
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})

	claims, err := client.VerifyIDTokenForApplication(
		context.Background(),
		" \t"+ApplicationWeb+"\n ",
		" \t"+token+"\n ",
	)

	require.NoError(t, err)
	assert.Equal(t, "user-oidc-1", claims.GetUserID())
	assert.Equal(t, clientID, claims.GetAppID())
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

func TestVerifyIDTokenRejectsInvalidInputsBeforeJWKSFetch(t *testing.T) {
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

	_, err = client.VerifyIDToken(context.Background(), " \t\n ")
	require.ErrorIs(t, err, ErrInvalidIDToken)
	assert.Equal(t, int64(0), atomic.LoadInt64(&keyRequests))

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

func TestClassifyOAuthRefreshErrorMarksMissingAccessTokenAsInvalidRefresh(t *testing.T) {
	err := classifyOAuthRefreshError(errors.New("oauth2: server response missing access_token"))

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidRefreshToken)
	assert.False(t, errors.Is(err, ErrProviderUnavailable))
}
