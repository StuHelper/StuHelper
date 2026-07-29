package oidc

import (
	"context"
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

func TestRevokeRefreshTokenUsesDiscoveredRevocationEndpoint(t *testing.T) {
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

	err := client.RevokeRefreshToken(context.Background(), "provider-refresh-token")

	require.NoError(t, err)
	assert.Equal(t, "provider-refresh-token", seenToken)
	assert.Equal(t, "refresh_token", seenHint)
	assert.Equal(t, "oidc-client", seenUser)
	assert.Equal(t, "oidc-secret", seenPass)
}

func TestRevokeRefreshTokenForApplicationNormalizesInputs(t *testing.T) {
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

	err := client.RevokeRefreshTokenForApplication(
		context.Background(),
		" \t"+ApplicationAdmin+"\n ",
		" \tprovider-refresh-token\n ",
	)

	require.NoError(t, err)
	assert.Equal(t, "provider-refresh-token", seenToken)
	assert.Equal(t, "admin-client", seenUser)
	assert.Equal(t, "admin-secret", seenPass)
}

func TestRevokeRefreshTokenRejectsBlankTokenWithoutProviderCall(t *testing.T) {
	var calls int
	client, server := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}), revocationDiscoveryOptions{includeRevocationEndpoint: true})
	defer server.Close()

	err := client.RevokeRefreshToken(context.Background(), " \t\n ")

	require.ErrorIs(t, err, ErrInvalidRefreshToken)
	assert.Equal(t, 0, calls)
}

func TestRevokeRefreshTokenProviderFailureIsUnavailable(t *testing.T) {
	client, server := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "revocation unavailable", http.StatusServiceUnavailable)
	}), revocationDiscoveryOptions{includeRevocationEndpoint: true})
	defer server.Close()

	err := client.RevokeRefreshToken(context.Background(), "provider-refresh-token")

	require.ErrorIs(t, err, ErrProviderUnavailable)
}

func TestRevokeRefreshTokenRequiresEndpointMetadata(t *testing.T) {
	client, server := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), revocationDiscoveryOptions{})
	defer server.Close()

	err := client.RevokeRefreshToken(context.Background(), "provider-refresh-token")

	require.ErrorIs(t, err, ErrRevocationEndpointUnavailable)
	assert.False(t, errors.Is(err, ErrProviderUnavailable))
}

func TestSupportsRefreshTokenRevocationReflectsMetadata(t *testing.T) {
	supportedClient, supportedServer := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), revocationDiscoveryOptions{includeRevocationEndpoint: true})
	defer supportedServer.Close()

	supported, err := supportedClient.SupportsRefreshTokenRevocation()
	require.NoError(t, err)
	assert.True(t, supported)

	unsupportedClient, unsupportedServer := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), revocationDiscoveryOptions{})
	defer unsupportedServer.Close()

	supported, err = unsupportedClient.SupportsRefreshTokenRevocation()
	require.NoError(t, err)
	assert.False(t, supported)
}

func TestRevokeRefreshTokenUsesCasdoorEndSessionEndpointFallback(t *testing.T) {
	var seenToken string
	client, server := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		seenToken = r.Form.Get("token")
		w.WriteHeader(http.StatusOK)
	}), revocationDiscoveryOptions{endSessionPath: "/api/logout"})
	defer server.Close()

	supported, err := client.SupportsRefreshTokenRevocation()
	require.NoError(t, err)
	require.True(t, supported)
	require.NoError(t, client.RevokeRefreshToken(context.Background(), "provider-refresh-token"))
	assert.Equal(t, "provider-refresh-token", seenToken)
}

func TestSupportsRefreshTokenRevocationRejectsGenericEndSessionEndpoint(t *testing.T) {
	client, server := newRevocationOIDCClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), revocationDiscoveryOptions{endSessionPath: "/oidc/logout"})
	defer server.Close()

	supported, err := client.SupportsRefreshTokenRevocation()
	require.NoError(t, err)
	assert.False(t, supported)
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
