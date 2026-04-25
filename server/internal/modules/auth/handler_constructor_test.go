package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/sms"
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
		CORSOrigins: []string{"https://web.example.com", "https://admin.example.com"},
		Token:       config.TokenConfig{AccessTokenTTL: 300, RefreshTokenTTL: 600, CookieSecure: true, CookieDomain: ".example.com"},
		OIDCIssuer:  "https://sso.example.com",
	}
	client := oidc.NewStubClient("https://sso.example.com/authorize")

	h := NewHandler(cfg, tokenSvc, fixture.Client, client, &fakeUserSyncRepo{}, nil)
	require.NotNil(t, h)
	assert.NotNil(t, h.svc)
	assert.Equal(t, client, h.oidcClient)
	assert.Equal(t, tokenSvc, h.tokenService)
	assert.Equal(t, fixture.Client, h.redisClient)
	assert.Equal(t, "https://web.example.com", h.defaultRedirectURL)
	assert.Contains(t, h.allowedRedirectHosts, "web.example.com")
	assert.Contains(t, h.allowedRedirectHosts, "admin.example.com")
	assert.NotNil(t, h.refreshLimiter)
	assert.NotNil(t, h.phoneLimiter)
	assert.Nil(t, h.otpService)
	assert.Equal(t, "https://sso.example.com", h.oidcIssuer)

	// sanity-check limiter wiring is live
	for i := 0; i < 5; i++ {
		allowed, err := h.phoneLimiter.Allow(t.Context(), "phone:1")
		require.NoError(t, err)
		assert.True(t, allowed)
	}
	blocked, err := h.phoneLimiter.Allow(t.Context(), "phone:1")
	require.NoError(t, err)
	assert.False(t, blocked)
}

func TestNewHandler_WithSMSInitializesOTPService(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-auth-otp-secret-32-chars-long!", false))
	tokenSvc, fixture := newTokenServiceForHandlerCtor(t)
	cfg := HandlerConfig{CORSOrigins: []string{"https://web.example.com"}, Token: config.TokenConfig{AccessTokenTTL: 300, RefreshTokenTTL: 600}}
	h := NewHandler(cfg, tokenSvc, fixture.Client, oidc.NewStubClient("https://sso.example.com/authorize"), &fakeUserSyncRepo{}, &sms.Service{})
	require.NotNil(t, h.otpService)

	// OTP service should use same redis backend and be operational
	ctx := t.Context()
	code, err := h.otpService.Generate(ctx, "13800138000")
	require.NoError(t, err)
	require.Len(t, code, otpLength)
	// avoid unused time import lint drift guard by touching exported timing assumption
	assert.Equal(t, 5*time.Minute, otpTTL)
}
