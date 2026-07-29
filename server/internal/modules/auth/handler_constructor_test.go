package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto/pii"
	"github.com/StuHelper/StuHelper/server/internal/pkg/oidc"
	"github.com/StuHelper/StuHelper/server/internal/pkg/token"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

func newTokenServiceForHandlerCtor(t *testing.T) (*token.Service, *redisfixture.Fixture) {
	t.Helper()
	fixture := redisfixture.Start(t)
	tokenSvc, err := token.NewService(token.ServiceConfig{RedisClient: fixture.Client, AccessTTL: 300, RefreshTTL: 600})
	require.NoError(t, err)
	t.Cleanup(tokenSvc.Close)
	return tokenSvc, fixture
}

func TestNewHandler_WiresDependencies(t *testing.T) {
	tokenSvc, fixture := newTokenServiceForHandlerCtor(t)
	client := newHandlerConstructorOIDCClient(t, "casdoor-end-session")
	cfg := HandlerConfig{
		CORSOrigins:            []string{"https://web.example.com", "https://admin.example.com"},
		Token:                  config.TokenConfig{AccessTokenTTL: 300, RefreshTokenTTL: 600, CookieSecure: true, CookieDomain: ".example.com"},
		OIDCIssuer:             "https://sso.example.com",
		AccountSettingsBaseURL: "https://account.example.com",
		ProviderTokenCipher:    newHandlerProviderCipher(t),
	}

	h, err := NewHandler(cfg, tokenSvc, fixture.Client, client, &fakeUserSyncRepo{})
	require.NoError(t, err)
	require.NotNil(t, h)
	assert.NotNil(t, h.svc)
	assert.Equal(t, client, h.oidcClient)
	assert.Equal(t, tokenSvc, h.tokenService)
	assert.Equal(t, fixture.Client, h.redisClient)
	assert.Equal(t, "https://web.example.com", h.defaultRedirectURL)
	assert.Contains(t, h.allowedRedirectHosts, "web.example.com")
	assert.Contains(t, h.allowedRedirectHosts, "admin.example.com")
	assert.NotNil(t, h.refreshLimiter)
	assert.NotNil(t, h.authFailureGuard)
	assert.Equal(t, "https://sso.example.com", h.oidcIssuer)
	assert.Equal(t, "https://account.example.com", h.accountSettingsBaseURL)
}

func TestNewHandlerFailsClosedWithoutProviderRefreshTokenRevocation(t *testing.T) {
	tokenSvc, fixture := newTokenServiceForHandlerCtor(t)
	supportedClient := newHandlerConstructorOIDCClient(t, "advertise")
	unsupportedClient := newHandlerConstructorOIDCClient(t, nil)
	malformedMetadataClient := newHandlerConstructorOIDCClient(t, map[string]string{"invalid": "shape"})
	cipher := newHandlerProviderCipher(t)
	baseConfig := HandlerConfig{
		CORSOrigins:         []string{"https://web.example.com"},
		Token:               config.TokenConfig{AccessTokenTTL: 300, RefreshTokenTTL: 600},
		ProviderTokenCipher: cipher,
	}

	tests := []struct {
		name   string
		client *oidc.Client
		cfg    HandlerConfig
	}{
		{
			name:   "missing OIDC client",
			client: nil,
			cfg:    baseConfig,
		},
		{
			name:   "missing provider token cipher",
			client: supportedClient,
			cfg: HandlerConfig{
				CORSOrigins: []string{"https://web.example.com"},
				Token:       baseConfig.Token,
			},
		},
		{
			name:   "revocation endpoint not advertised",
			client: unsupportedClient,
			cfg:    baseConfig,
		},
		{
			name:   "revocation metadata malformed",
			client: malformedMetadataClient,
			cfg:    baseConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := NewHandler(tt.cfg, tokenSvc, fixture.Client, tt.client, &fakeUserSyncRepo{})

			require.ErrorIs(t, err, ErrProviderRefreshTokenRevocationUnavailable)
			assert.Nil(t, h)
		})
	}
}

func newHandlerProviderCipher(t *testing.T) pii.EncryptDecryptor {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	cipher, err := pii.NewCipher(1, map[uint8][]byte{1: key})
	require.NoError(t, err)
	return cipher
}

func newHandlerConstructorOIDCClient(t *testing.T, revocationMetadata any) *oidc.Client {
	t.Helper()
	var issuer string
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	issuer = server.URL

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		metadata := map[string]any{
			"issuer":                 issuer,
			"authorization_endpoint": issuer + "/authorize",
			"token_endpoint":         issuer + "/token",
			"jwks_uri":               issuer + "/keys",
			"introspection_endpoint": issuer + "/introspect",
		}
		if revocationMetadata != nil {
			switch revocationMetadata {
			case "advertise":
				metadata["revocation_endpoint"] = issuer + "/revoke"
			case "casdoor-end-session":
				metadata["end_session_endpoint"] = issuer + "/api/logout"
			default:
				metadata["revocation_endpoint"] = revocationMetadata
			}
		}
		require.NoError(t, json.NewEncoder(w).Encode(metadata))
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"keys": []any{}}))
	})

	client, err := oidc.NewClient(context.Background(), config.CasdoorConfig{
		Issuer:                    issuer,
		ClientID:                  "oidc-client",
		ClientSecret:              "oidc-secret",
		RedirectURI:               "https://web.example.com/api/v1/auth/callback",
		IntrospectionClientID:     "introspection-client",
		IntrospectionClientSecret: "introspection-secret",
	})
	require.NoError(t, err)
	return client
}
