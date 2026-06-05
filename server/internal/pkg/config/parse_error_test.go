package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validConfigForValidation() *Config {
	return &Config{
		App: AppConfig{
			Env:             "development",
			HMACSecret:      "0123456789abcdef0123456789abcdef",
			CORSOrigins:     []string{"http://localhost:3000"},
			MetricsPassword: "metrics-password",
			APIIPRateLimit:  100,
			APIGlobalLimit:  10000,
		},
		Security: SecurityConfig{
			DocAESActiveKeyID: 1,
			DocAESKeys: map[uint8][]byte{
				1: make([]byte, 32),
			},
		},
		Database: DatabaseConfig{
			QueryTimeout: 5,
			MaxConns:     20,
			MinConns:     2,
		},
		Token: TokenConfig{
			AccessTokenTTL:  300,
			RefreshTokenTTL: 604800,
			CookieSecure:    true,
		},
		ObjectStorage: ObjectStorageConfig{
			PresignTTL: 600,
		},
		Casdoor: CasdoorConfig{
			Issuer:                    "https://sso.example.com",
			ClientID:                  "client-id",
			ClientSecret:              "client-secret",
			RedirectURI:               "https://api.example.com/api/v1/auth/callback",
			IntrospectionClientID:     "introspection-client",
			IntrospectionClientSecret: "introspection-secret",
			Organization:              "stuhelper",
		},
		OpenFGA: OpenFGAConfig{
			StoreID:              "store-id",
			AuthorizationModelID: "model-id",
			APIUrl:               "http://openfga:8080",
		},
		RateLimit: ReviewRateLimitConfig{
			PostLimit:       5,
			VoteLimit:       30,
			ReportLimit:     10,
			ReplyLimit:      10,
			WriteLimit:      10,
			SearchAnonLimit: 5,
			SearchUserLimit: 60,
			BatchAnonLimit:  5,
			BatchUserLimit:  60,
		},
	}
}

func TestValidate_RejectsParseErrorsInDevelopment(t *testing.T) {
	cfg := validConfigForValidation()

	err := cfg.validate([]string{"DB_MAX_CONNS must be an integer"})

	require.Error(t, err)
	require.Contains(t, err.Error(), "DB_MAX_CONNS must be an integer")
}

func TestLoadOpenPlatformConfigFromEnv(t *testing.T) {
	t.Setenv("OPEN_PLATFORM_DISCLOSURE_APP_LIMIT", "700")
	t.Setenv("OPEN_PLATFORM_DISCLOSURE_APP_USER_LIMIT", "140")
	t.Setenv("OPEN_PLATFORM_DISCLOSURE_ENDPOINT_LIMIT", "1400")
	t.Setenv("OPEN_PLATFORM_DISCLOSURE_CONSENT_LIMIT", "30")
	t.Setenv("OPEN_PLATFORM_DISCLOSURE_REPLAY_LIMIT", "9")
	t.Setenv("OPEN_PLATFORM_DISCLOSURE_WINDOW_SECONDS", "90")
	t.Setenv("OPEN_PLATFORM_DISCLOSURE_REPLAY_WINDOW_SECONDS", "600")
	t.Setenv("OPEN_PLATFORM_DISCLOSURE_REPLAY_AUDIT_COOLDOWN_SECONDS", "1200")
	t.Setenv("OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED", "true")
	t.Setenv("OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND", "/usr/local/bin/stuhelper-token-probe")
	t.Setenv("OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_TIMEOUT_SECONDS", "45")

	var parseErrs []string
	cfg := loadOpenPlatformConfig(&parseErrs)

	require.Empty(t, parseErrs)
	require.Equal(t, 700, cfg.DisclosureRateLimit.AppLimit)
	require.Equal(t, 140, cfg.DisclosureRateLimit.AppUserLimit)
	require.Equal(t, 1400, cfg.DisclosureRateLimit.EndpointLimit)
	require.Equal(t, 30, cfg.DisclosureRateLimit.ConsentLimit)
	require.Equal(t, 9, cfg.DisclosureRateLimit.ReplayLimit)
	require.Equal(t, 90, cfg.DisclosureRateLimit.WindowSeconds)
	require.Equal(t, 600, cfg.DisclosureRateLimit.ReplayWindowSeconds)
	require.Equal(t, 1200, cfg.DisclosureRateLimit.ReplayAuditCooldownSeconds)
	require.True(t, cfg.TokenProbe.RuntimeRequired)
	require.Equal(t, "/usr/local/bin/stuhelper-token-probe", cfg.TokenProbe.RuntimeCommand)
	require.Equal(t, 45, cfg.TokenProbe.RuntimeTimeoutSeconds)
}

func TestValidate_OpenPlatformDisclosureRateLimitsRejectInvalidValues(t *testing.T) {
	cfg := validConfigForValidation()
	cfg.OpenPlatform.DisclosureRateLimit.AppLimit = -1
	cfg.OpenPlatform.DisclosureRateLimit.ReplayWindowSeconds = 90000

	err := cfg.validate(nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "OPEN_PLATFORM_DISCLOSURE_APP_LIMIT must be between 1 and 100000 when set")
	require.Contains(t, err.Error(), "OPEN_PLATFORM_DISCLOSURE_REPLAY_WINDOW_SECONDS must be between 1 and 86400 seconds when set")
}

func TestValidate_APIRateLimitsRejectInvalidValues(t *testing.T) {
	cfg := validConfigForValidation()
	cfg.App.APIIPRateLimit = 0
	cfg.App.APIGlobalLimit = 100001

	err := cfg.validate(nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "API_IP_RATE_LIMIT must be between 1 and 100000 (got 0)")
	require.Contains(t, err.Error(), "API_GLOBAL_RATE_LIMIT must be between 1 and 100000 (got 100001)")
}
