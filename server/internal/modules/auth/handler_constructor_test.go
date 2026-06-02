package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
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
	cfg := HandlerConfig{
		CORSOrigins:            []string{"https://web.example.com", "https://admin.example.com"},
		Token:                  config.TokenConfig{AccessTokenTTL: 300, RefreshTokenTTL: 600, CookieSecure: true, CookieDomain: ".example.com"},
		OIDCIssuer:             "https://sso.example.com",
		AccountSettingsBaseURL: "https://account.example.com",
	}
	client := oidc.NewStubClient("https://sso.example.com/authorize")

	h := NewHandler(cfg, tokenSvc, fixture.Client, client, &fakeUserSyncRepo{})
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
