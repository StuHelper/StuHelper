package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validConfigForValidation() *Config {
	return &Config{
		Review: ReviewConfig{TeacherStatsRefreshTimeoutSeconds: 60},
		App: AppConfig{
			Env:                "development",
			Port:               "8080",
			HMACSecret:         "0123456789abcdef0123456789abcdef",
			CORSOrigins:        []string{"http://localhost:3000"},
			MetricsPassword:    "metrics-password",
			MaxBodySize:        10 << 20,
			APIIPRateLimit:     100,
			APIGlobalLimit:     10000,
			HealthCheckTimeout: 3,
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
	t.Setenv("OPEN_PLATFORM_CONSENT_BASE_URL", "https://stuhelper.example.com")
	t.Setenv("OPEN_PLATFORM_ACCOUNT_BASE_URL", "https://account.example.com")
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
	require.Equal(t, "https://stuhelper.example.com", cfg.ConsentBaseURL)
	require.Equal(t, "https://account.example.com", cfg.AccountBaseURL)
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

func TestLoadObservabilityConfigFromEnv(t *testing.T) {
	t.Setenv("OTEL_ENABLED", "true")
	t.Setenv("OTEL_SERVICE_NAME", "stuhelper-test")
	t.Setenv("OTEL_SERVICE_NAMESPACE", "stuhelper")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://alloy:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "false")
	t.Setenv("OTEL_TRACE_SAMPLE_RATIO", "0.5")
	t.Setenv("FRONTEND_METRICS_ALLOWED_ORIGINS", "https://web.example.com, https://admin.example.com")

	var parseErrs []string
	cfg := loadObservabilityConfig(&parseErrs)

	require.Empty(t, parseErrs)
	require.True(t, cfg.Enabled)
	require.Equal(t, "stuhelper-test", cfg.ServiceName)
	require.Equal(t, "stuhelper", cfg.ServiceNamespace)
	require.Equal(t, "http://alloy:4318", cfg.OTLPEndpoint)
	require.False(t, cfg.OTLPInsecure)
	require.Equal(t, 0.5, cfg.TraceSampleRatio)
	require.Equal(t, []string{"https://web.example.com", "https://admin.example.com"}, cfg.FrontendMetricsAllowedOrigins)
}

func TestValidate_FrontendMetricsAllowedOrigins(t *testing.T) {
	t.Run("allows empty optional origins", func(t *testing.T) {
		cfg := validConfigForValidation()

		require.NoError(t, cfg.validate(nil))
	})

	t.Run("allows explicit origins", func(t *testing.T) {
		cfg := validConfigForValidation()
		cfg.Observability.FrontendMetricsAllowedOrigins = []string{
			"http://localhost:3000",
			"https://web.example.com",
		}

		require.NoError(t, cfg.validate(nil))
	})

	t.Run("rejects invalid origin", func(t *testing.T) {
		cfg := validConfigForValidation()
		cfg.Observability.FrontendMetricsAllowedOrigins = []string{"https://web.example.com/metrics"}

		err := cfg.validate(nil)

		require.Error(t, err)
		require.Contains(t, err.Error(), "FRONTEND_METRICS_ALLOWED_ORIGINS must not include a path")
	})

	t.Run("requires https in production", func(t *testing.T) {
		cfg := validProductionConfigForTest()
		cfg.Observability.FrontendMetricsAllowedOrigins = []string{"http://web.example.com"}

		err := cfg.validate(nil)

		require.Error(t, err)
		require.Contains(t, err.Error(), "FRONTEND_METRICS_ALLOWED_ORIGINS must use https in production")
	})
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

func TestValidate_OpenPlatformBaseURLs(t *testing.T) {
	t.Run("allows empty development base URLs", func(t *testing.T) {
		cfg := validConfigForValidation()

		require.NoError(t, cfg.validate(nil))
	})

	t.Run("requires explicit base URLs in production-like environments", func(t *testing.T) {
		for _, env := range []string{EnvProduction, EnvProdParity} {
			t.Run(env, func(t *testing.T) {
				cfg := validProductionConfigForTest()
				cfg.App.Env = env
				cfg.OpenPlatform.ConsentBaseURL = ""
				cfg.OpenPlatform.AccountBaseURL = ""

				err := cfg.validate(nil)

				require.Error(t, err)
				require.Contains(t, err.Error(), "OPEN_PLATFORM_CONSENT_BASE_URL is required in production")
				require.Contains(t, err.Error(), "OPEN_PLATFORM_ACCOUNT_BASE_URL is required in production")
			})
		}
	})

	t.Run("allows explicit http origins", func(t *testing.T) {
		cfg := validConfigForValidation()
		cfg.OpenPlatform.ConsentBaseURL = "https://stuhelper.example.com"
		cfg.OpenPlatform.AccountBaseURL = "http://localhost:3000"

		require.NoError(t, cfg.validate(nil))
	})

	t.Run("rejects invalid consent base URL", func(t *testing.T) {
		cfg := validConfigForValidation()
		cfg.OpenPlatform.ConsentBaseURL = "https://stuhelper.example.com/consent"

		err := cfg.validate(nil)

		require.Error(t, err)
		require.Contains(t, err.Error(), "OPEN_PLATFORM_CONSENT_BASE_URL must not include a path")
	})

	t.Run("rejects invalid account base URL", func(t *testing.T) {
		cfg := validConfigForValidation()
		cfg.OpenPlatform.AccountBaseURL = "https://account.example.com:bad"

		err := cfg.validate(nil)

		require.Error(t, err)
		require.Contains(t, err.Error(), "OPEN_PLATFORM_ACCOUNT_BASE_URL must include a valid port")
	})

	t.Run("requires https in production", func(t *testing.T) {
		cfg := validProductionConfigForTest()
		cfg.OpenPlatform.ConsentBaseURL = "http://stuhelper.example.com"
		cfg.OpenPlatform.AccountBaseURL = "http://account.example.com"

		err := cfg.validate(nil)

		require.Error(t, err)
		require.Contains(t, err.Error(), "OPEN_PLATFORM_CONSENT_BASE_URL must use https in production")
		require.Contains(t, err.Error(), "OPEN_PLATFORM_ACCOUNT_BASE_URL must use https in production")
	})
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
