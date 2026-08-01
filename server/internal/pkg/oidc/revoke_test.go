package oidc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
)

func TestRevokeTokenFamilyUsesDiscoveredRFC7009Endpoint(t *testing.T) {
	var seenToken string
	var seenHint string
	var seenUser string
	var seenPass string
	client, server := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenUser, seenPass, _ = r.BasicAuth()
		require.NoError(t, r.ParseForm())
		seenToken = r.Form.Get("token")
		seenHint = r.Form.Get("token_type_hint")
		w.WriteHeader(http.StatusOK)
	}), revocationDiscoveryOptions{includeRevocationEndpoint: true})
	defer server.Close()

	err := client.RevokeTokenFamily(context.Background(), "provider-access-token", "provider-refresh-token")

	require.NoError(t, err)
	assert.Equal(t, "provider-refresh-token", seenToken)
	assert.Equal(t, "refresh_token", seenHint)
	assert.Equal(t, "oidc-client", seenUser)
	assert.Equal(t, "oidc-secret", seenPass)
}

func TestRevokeTokenFamilyForApplicationNormalizesRFC7009Inputs(t *testing.T) {
	var seenToken string
	var seenUser string
	var seenPass string
	client, server := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenUser, seenPass, _ = r.BasicAuth()
		require.NoError(t, r.ParseForm())
		seenToken = r.Form.Get("token")
		w.WriteHeader(http.StatusOK)
	}), revocationDiscoveryOptions{includeRevocationEndpoint: true})
	defer server.Close()

	err := client.RevokeTokenFamilyForApplication(
		context.Background(),
		" \t"+ApplicationAdmin+"\n ",
		" \tprovider-access-token\n ",
		" \tprovider-refresh-token\n ",
	)

	require.NoError(t, err)
	assert.Equal(t, "provider-refresh-token", seenToken)
	assert.Equal(t, "admin-client", seenUser)
	assert.Equal(t, "admin-secret", seenPass)
}

func TestRevokeTokenFamilyRejectsBlankRFC7009TokenWithoutProviderCall(t *testing.T) {
	var calls int
	client, server := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}), revocationDiscoveryOptions{includeRevocationEndpoint: true})
	defer server.Close()

	err := client.RevokeTokenFamily(context.Background(), "provider-access-token", " \t\n ")

	require.ErrorIs(t, err, ErrInvalidRefreshToken)
	assert.Equal(t, 0, calls)
}

func TestRevokeTokenFamilyProviderFailureIsUnavailable(t *testing.T) {
	client, server := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "revocation unavailable", http.StatusServiceUnavailable)
	}), revocationDiscoveryOptions{includeRevocationEndpoint: true})
	defer server.Close()

	err := client.RevokeTokenFamily(context.Background(), "provider-access-token", "provider-refresh-token")

	require.ErrorIs(t, err, ErrProviderUnavailable)
}

func TestRevokeTokenFamilyRequiresEndpointMetadata(t *testing.T) {
	client, server := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), revocationDiscoveryOptions{})
	defer server.Close()

	err := client.RevokeTokenFamily(context.Background(), "provider-access-token", "provider-refresh-token")

	require.ErrorIs(t, err, ErrRevocationEndpointUnavailable)
	assert.False(t, errors.Is(err, ErrProviderUnavailable))
}

func TestSupportsTokenFamilyRevocationReflectsMetadata(t *testing.T) {
	supportedClient, supportedServer := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), revocationDiscoveryOptions{includeRevocationEndpoint: true})
	defer supportedServer.Close()

	supported, err := supportedClient.SupportsTokenFamilyRevocation()
	require.NoError(t, err)
	assert.True(t, supported)

	unsupportedClient, unsupportedServer := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), revocationDiscoveryOptions{})
	defer unsupportedServer.Close()

	supported, err = unsupportedClient.SupportsTokenFamilyRevocation()
	require.NoError(t, err)
	assert.False(t, supported)
}

func TestRevokeTokenFamilyUsesCasdoorEndSessionContract(t *testing.T) {
	var seenAccessToken string
	var seenClientID string
	var seenRFC7009Token string
	client, server := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		seenAccessToken = r.Form.Get("id_token_hint")
		seenClientID = r.Form.Get("client_id")
		seenRFC7009Token = r.Form.Get("token")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"status": "ok"}))
	}), revocationDiscoveryOptions{endSessionPath: "/api/logout"})
	defer server.Close()

	supported, err := client.SupportsTokenFamilyRevocation()
	require.NoError(t, err)
	require.True(t, supported)
	require.NoError(t, client.RevokeTokenFamily(
		context.Background(),
		"provider-access-token",
		"provider-refresh-token",
	))
	assert.Equal(t, "provider-access-token", seenAccessToken)
	assert.Equal(t, "oidc-client", seenClientID)
	assert.Empty(t, seenRFC7009Token)
}

func TestRevokeTokenFamilyRejectsCasdoorBusinessFailureAtHTTP200(t *testing.T) {
	client, server := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"status": "error",
			"msg":    "token not found",
		}))
	}), revocationDiscoveryOptions{endSessionPath: "/api/logout"})
	defer server.Close()

	err := client.RevokeTokenFamily(
		context.Background(),
		"provider-access-token",
		"provider-refresh-token",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reported failure")
}

func TestRevokeTokenFamilyRejectsMalformedCasdoorSuccessResponse(t *testing.T) {
	client, server := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}), revocationDiscoveryOptions{endSessionPath: "/api/logout"})
	defer server.Close()

	err := client.RevokeTokenFamily(
		context.Background(),
		"provider-access-token",
		"provider-refresh-token",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode Casdoor logout response")
}

func TestRevokeTokenFamilyRequiresAccessOrRefreshTokenForCasdoorWithoutCallingProvider(t *testing.T) {
	var calls int
	client, server := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"status": "ok"}))
	}), revocationDiscoveryOptions{endSessionPath: "/api/logout"})
	defer server.Close()

	err := client.RevokeTokenFamily(context.Background(), " \t\n ", " \t\n ")

	require.ErrorIs(t, err, ErrInvalidRefreshToken)
	assert.Zero(t, calls)
}

func TestCasdoorLegacyRevocationDoesNotMaskUnexpectedRefreshFailure(t *testing.T) {
	var logoutCalls int
	client, server := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		logoutCalls++
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"status": "ok"}))
	}), revocationDiscoveryOptions{endSessionPath: "/api/logout"})
	defer server.Close()

	// The fixture intentionally has no /token handler, so the legacy refresh
	// bridge receives an unexpected 404 rather than Casdoor's invalid_grant.
	err := client.RevokeTokenFamily(
		context.Background(),
		"",
		"provider-refresh-token",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "rotate legacy token family")
	assert.Zero(t, logoutCalls)
}

func TestSupportsTokenFamilyRevocationRejectsGenericEndSessionEndpoint(t *testing.T) {
	client, server := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), revocationDiscoveryOptions{endSessionPath: "/oidc/logout"})
	defer server.Close()

	supported, err := client.SupportsTokenFamilyRevocation()
	require.NoError(t, err)
	assert.False(t, supported)
}

func TestCasdoorEndSessionRevocationEndpointRejectsCrossOrigin(t *testing.T) {
	_, err := casdoorEndSessionRevocationEndpoint(
		"https://sso.example.com",
		"https://logout.example.net/api/logout",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must share the issuer origin")
}

func TestCasdoorEndSessionRevocationEndpointAcceptsEquivalentDefaultPort(t *testing.T) {
	endpoint := "https://sso.example.com:443/api/logout"

	got, err := casdoorEndSessionRevocationEndpoint("https://sso.example.com", endpoint)

	require.NoError(t, err)
	assert.Equal(t, endpoint, got)
}

// TestCasdoorTokenFamilyRevocationContract mirrors the repository-pinned
// Casdoor contract: /api/logout resolves id_token_hint through the access-token
// row and sets that row's expires_in to zero. Both introspection and the refresh
// grant consult the same row, so a successful logout must disable both.
func TestCasdoorTokenFamilyRevocationContract(t *testing.T) {
	accessToken := "header." +
		base64.RawURLEncoding.EncodeToString([]byte(`{"tokenType":"access-token"}`)) +
		".signature"
	refreshToken := "provider-refresh-token"
	familyActive := true
	currentAccessToken := accessToken
	currentRefreshToken := refreshToken
	refreshCalls := 0
	logoutCalls := 0

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 server.URL,
			"authorization_endpoint": server.URL + "/authorize",
			"token_endpoint":         server.URL + "/token",
			"jwks_uri":               server.URL + "/keys",
			"introspection_endpoint": server.URL + "/introspect",
			"end_session_endpoint":   server.URL + "/api/logout",
		}))
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"keys": []any{}}))
	})
	mux.HandleFunc("/introspect", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "access_token", r.Form.Get("token_type_hint"))
		active := familyActive && r.Form.Get("token") == currentAccessToken
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"active":    active,
			"sub":       "built-in/test-user",
			"client_id": "oidc-client",
		}))
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		refreshCalls++
		require.NoError(t, r.ParseForm())
		w.Header().Set("Content-Type", "application/json")
		if !familyActive || r.Form.Get("refresh_token") != currentRefreshToken {
			w.WriteHeader(http.StatusBadRequest)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"error":             "invalid_grant",
				"error_description": "refresh token is invalid or revoked",
			}))
			return
		}
		currentAccessToken = "rotated-provider-access-token"
		currentRefreshToken = "rotated-provider-refresh-token"
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"access_token":  currentAccessToken,
			"id_token":      currentAccessToken,
			"refresh_token": currentRefreshToken,
			"token_type":    "Bearer",
			"expires_in":    3600,
		}))
	})
	mux.HandleFunc("/api/logout", func(w http.ResponseWriter, r *http.Request) {
		logoutCalls++
		require.NoError(t, r.ParseForm())
		assert.Empty(t, r.Form.Get("token"))
		assert.Empty(t, r.Form.Get("token_type_hint"))
		if !familyActive || r.Form.Get("id_token_hint") != currentAccessToken {
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"status": "error",
				"msg":    "token not found",
			}))
			return
		}
		familyActive = false
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"status": "ok"}))
	})

	client, err := NewClient(context.Background(), config.CasdoorConfig{
		Issuer:                    server.URL,
		ClientID:                  "oidc-client",
		ClientSecret:              "oidc-secret",
		RedirectURI:               "https://web.example.com/api/v1/auth/callback",
		IntrospectionClientID:     "introspection-client",
		IntrospectionClientSecret: "introspection-secret",
	})
	require.NoError(t, err)

	before, err := client.IntrospectToken(context.Background(), accessToken)
	require.NoError(t, err)
	require.True(t, before.Active)

	require.NoError(t, client.RevokeTokenFamilyForApplication(
		context.Background(),
		ApplicationWeb,
		accessToken,
		refreshToken,
	))
	assert.False(t, familyActive)

	after, err := client.IntrospectToken(context.Background(), accessToken)
	require.NoError(t, err)
	assert.False(t, after.Active)

	_, err = client.RefreshTokenForApplication(context.Background(), ApplicationWeb, refreshToken)
	require.Error(t, err)
	require.NoError(t, client.RevokeTokenFamilyForApplication(
		context.Background(),
		ApplicationWeb,
		"",
		refreshToken,
	))

	// Rolling-upgrade path: a legacy session has the encrypted refresh token
	// but predates ProviderAccessTokenEnc. The adapter rotates once and then
	// logs out the replacement family.
	familyActive = true
	currentAccessToken = accessToken
	currentRefreshToken = refreshToken
	require.NoError(t, client.RevokeTokenFamilyForApplication(
		context.Background(),
		ApplicationWeb,
		"",
		refreshToken,
	))
	assert.False(t, familyActive)
	assert.GreaterOrEqual(t, refreshCalls, 3)
	assert.Equal(t, 2, logoutCalls)
}

type revocationDiscoveryOptions struct {
	includeRevocationEndpoint bool
	endSessionPath            string
}

func newRevocationOIDCClient(
	t *testing.T,
	revocationHandler http.HandlerFunc,
	options revocationDiscoveryOptions,
) (*Client, *httptest.Server) {
	t.Helper()
	var issuer string
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	issuer = server.URL

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		metadata := map[string]any{
			"issuer":                 issuer,
			"authorization_endpoint": issuer + "/authorize",
			"token_endpoint":         issuer + "/token",
			"jwks_uri":               issuer + "/keys",
			"introspection_endpoint": issuer + "/introspect",
		}
		if options.includeRevocationEndpoint {
			metadata["revocation_endpoint"] = issuer + "/revoke"
		}
		if options.endSessionPath != "" {
			metadata["end_session_endpoint"] = issuer + options.endSessionPath
		}
		_ = json.NewEncoder(w).Encode(metadata)
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{}})
	})
	mux.HandleFunc("/revoke", revocationHandler)
	if options.endSessionPath != "" {
		mux.HandleFunc(options.endSessionPath, revocationHandler)
	}

	client, err := NewClient(context.Background(), config.CasdoorConfig{
		Issuer:                    issuer,
		ClientID:                  "oidc-client",
		ClientSecret:              "oidc-secret",
		RedirectURI:               "https://web.example.com/api/v1/auth/callback",
		IntrospectionClientID:     "introspection-client",
		IntrospectionClientSecret: "introspection-secret",
	})
	require.NoError(t, err)
	client.oauth2Configs[ApplicationAdmin] = oauth2.Config{
		ClientID:     "admin-client",
		ClientSecret: "admin-secret",
		Endpoint:     client.oauth2Cfg.Endpoint,
		Scopes:       client.oauth2Cfg.Scopes,
	}
	return client, server
}
