package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDocAESKeys_Valid(t *testing.T) {
	// 有效的 32 字节 hex key
	fixtureHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	keys, errs := parseDocAESKeys("1:" + fixtureHex)
	assert.Empty(t, errs)
	assert.Len(t, keys, 1)
	assert.Len(t, keys[1], 32)
}

func TestParseDocAESKeys_MultipleKeys(t *testing.T) {
	fixtureHexA := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	fixtureHexB := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	keys, errs := parseDocAESKeys("1:" + fixtureHexA + ",2:" + fixtureHexB)
	assert.Empty(t, errs)
	assert.Len(t, keys, 2)
}

func TestParseDocAESKeys_InvalidFormat(t *testing.T) {
	_, errs := parseDocAESKeys("invalid-no-colon")
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "invalid entry")
}

func TestParseDocAESKeys_InvalidKeyID(t *testing.T) {
	fixtureHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tests := []struct {
		name  string
		input string
	}{
		{"零值", "0:" + fixtureHex},
		{"超范围", "256:" + fixtureHex},
		{"非数字", "abc:" + fixtureHex},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errs := parseDocAESKeys(tt.input)
			assert.NotEmpty(t, errs)
			assert.Contains(t, errs[0], "invalid key ID")
		})
	}
}

func TestParseDocAESKeys_DuplicateKeyID(t *testing.T) {
	fixtureHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	_, errs := parseDocAESKeys("1:" + fixtureHex + ",1:" + fixtureHex)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "duplicate key ID")
}

func TestParseDocAESKeys_InvalidHex(t *testing.T) {
	_, errs := parseDocAESKeys("1:not-valid-hex!")
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "invalid hex")
}

func TestParseDocAESKeys_WrongKeyLength(t *testing.T) {
	// 16 字节而不是 32 字节
	shortHex := "0123456789abcdef0123456789abcdef"
	_, errs := parseDocAESKeys("1:" + shortHex)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "must be exactly 32 bytes")
}

func TestParseDocAESKeys_EmptyInput(t *testing.T) {
	_, errs := parseDocAESKeys("")
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "no valid keys found")
}

func TestParseSecurityConfig_MissingActiveKeyID(t *testing.T) {
	t.Setenv("DOC_AES_ACTIVE_KEY_ID", "")
	t.Setenv("DOC_AES_KEYS", "1:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

	_, errs := parseSecurityConfig()
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "DOC_AES_ACTIVE_KEY_ID is required")
}

func TestParseSecurityConfig_MissingKeys(t *testing.T) {
	t.Setenv("DOC_AES_ACTIVE_KEY_ID", "1")
	t.Setenv("DOC_AES_KEYS", "")

	_, errs := parseSecurityConfig()
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "DOC_AES_KEYS is required")
}

func TestParseSecurityConfig_ActiveKeyNotInKeySet(t *testing.T) {
	fixtureHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Setenv("DOC_AES_ACTIVE_KEY_ID", "2")
	t.Setenv("DOC_AES_KEYS", "1:"+fixtureHex)

	cfg, errs := parseSecurityConfig()
	// parseSecurityConfig 本身不校验 activeKeyID 是否在 keys 中，
	// 由 validate() 统一校验
	assert.Empty(t, errs)
	assert.Equal(t, uint8(2), cfg.DocAESActiveKeyID)
}

func TestValidate_SecurityConfig_ActiveKeyMissing(t *testing.T) {
	fixtureHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Setenv("DOC_AES_ACTIVE_KEY_ID", "2")
	t.Setenv("DOC_AES_KEYS", "1:"+fixtureHex)

	cfg, securityErrs := parseSecurityConfig()
	require.Empty(t, securityErrs)

	c := &Config{
		Security: cfg,
	}
	// 只验证安全配置相关校验
	err := c.validate(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DOC_AES_ACTIVE_KEY_ID=2 not found in DOC_AES_KEYS")
}

func TestParseSecurityConfig_ValidConfig(t *testing.T) {
	fixtureHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Setenv("DOC_AES_ACTIVE_KEY_ID", "1")
	t.Setenv("DOC_AES_KEYS", "1:"+fixtureHex)

	cfg, errs := parseSecurityConfig()
	assert.Empty(t, errs)
	assert.Equal(t, uint8(1), cfg.DocAESActiveKeyID)
	assert.Len(t, cfg.DocAESKeys, 1)
	assert.Len(t, cfg.DocAESKeys[1], 32)
}

func TestValidate_ProductionRequiresObservability(t *testing.T) {
	c := &Config{
		App: AppConfig{
			Env:                "production",
			Port:               "8080",
			HMACSecret:         "0123456789abcdef0123456789abcdef",
			CORSOrigins:        []string{"https://stuhelper.example.com"},
			TrustedProxies:     []string{"10.0.0.0/8"},
			MetricsUser:        "prometheus",
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
			URL:          "postgres://user:pass@db:5432/stuhelper?sslmode=verify-full",
			QueryTimeout: 5,
			MaxConns:     20,
			MinConns:     2,
			SSLMode:      "verify-full",
		},
		Token: TokenConfig{
			AccessTokenTTL:  300,
			RefreshTokenTTL: 604800,
			CookieSecure:    true,
		},
		Redis: RedisConfig{
			Password: "redis-password",
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

	err := c.validate(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OTEL_ENABLED must be true in production")
}

func TestIsProductionLikeEnvTrimsWhitespace(t *testing.T) {
	assert.True(t, IsProductionLikeEnv(" production "))
	assert.True(t, IsProductionLikeEnv("\tprod-parity\n"))
	assert.False(t, IsProductionLikeEnv(" development "))
}

func TestLoadAppConfigDefaultBodyLimitAccommodatesTenMiBBase64Payload(t *testing.T) {
	t.Setenv("MAX_BODY_SIZE", "")
	var parseErrs []string

	cfg := loadAppConfig(&parseErrs)

	require.Empty(t, parseErrs)
	assert.Equal(t, int64(16<<20), cfg.MaxBodySize)
}

func TestValidate_RejectsAppEnvWhitespace(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.Env = " production "

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "APP_ENV must not include leading or trailing whitespace")
}

func TestValidate_RejectsUnknownAppEnvEvenWithSecureCookie(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.Env = "staging"
	c.Token.CookieSecure = true

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "APP_ENV must be development, production, or prod-parity")
}

func TestValidate_RejectsInvalidAppPort(t *testing.T) {
	tests := []struct {
		name string
		port string
		want string
	}{
		{name: "empty", port: "", want: "APP_PORT is required"},
		{name: "blank", port: " \t\n ", want: "APP_PORT is required"},
		{name: "surrounding whitespace", port: " 8080 ", want: "APP_PORT must not include leading or trailing whitespace"},
		{name: "non numeric", port: "abc", want: "APP_PORT must be an integer between 1 and 65535"},
		{name: "zero", port: "0", want: "APP_PORT must be between 1 and 65535 (got 0)"},
		{name: "negative", port: "-1", want: "APP_PORT must be between 1 and 65535 (got -1)"},
		{name: "too large", port: "65536", want: "APP_PORT must be between 1 and 65535 (got 65536)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validProductionConfigForTest()
			c.App.Port = tt.port

			err := c.validate(nil)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestValidate_AllowsValidAppPort(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.Port = "65535"

	require.NoError(t, c.validate(nil))
}

func TestValidate_RejectsInvalidMaxBodySize(t *testing.T) {
	const maxAllowedBodySize = int64(100 << 20)
	tests := []struct {
		name string
		size int64
		want string
	}{
		{name: "zero", size: 0, want: "MAX_BODY_SIZE must be between 1 and 104857600 bytes (got 0)"},
		{name: "negative", size: -1, want: "MAX_BODY_SIZE must be between 1 and 104857600 bytes (got -1)"},
		{name: "too large", size: maxAllowedBodySize + 1, want: "MAX_BODY_SIZE must be between 1 and 104857600 bytes (got 104857601)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validProductionConfigForTest()
			c.App.MaxBodySize = tt.size

			err := c.validate(nil)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestValidate_AllowsValidMaxBodySize(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.MaxBodySize = 100 << 20

	require.NoError(t, c.validate(nil))
}

func TestValidate_RejectsInvalidHealthCheckTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout int
		want    string
	}{
		{name: "zero", timeout: 0, want: "HEALTH_CHECK_TIMEOUT must be between 1 and 60 seconds (got 0)"},
		{name: "negative", timeout: -1, want: "HEALTH_CHECK_TIMEOUT must be between 1 and 60 seconds (got -1)"},
		{name: "too large", timeout: 61, want: "HEALTH_CHECK_TIMEOUT must be between 1 and 60 seconds (got 61)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validProductionConfigForTest()
			c.App.HealthCheckTimeout = tt.timeout

			err := c.validate(nil)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestValidate_AllowsValidHealthCheckTimeout(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.HealthCheckTimeout = 60

	require.NoError(t, c.validate(nil))
}

func TestValidate_RejectsInvalidTrustedProxies(t *testing.T) {
	tests := []struct {
		name    string
		proxies []string
		want    string
	}{
		{
			name:    "empty entry",
			proxies: []string{""},
			want:    "TRUSTED_PROXIES contains an empty proxy entry",
		},
		{
			name:    "surrounding whitespace",
			proxies: []string{" 10.0.0.1 "},
			want:    `TRUSTED_PROXIES entry " 10.0.0.1 " must not include leading or trailing whitespace`,
		},
		{
			name:    "invalid ip",
			proxies: []string{"192.168.1.256"},
			want:    `TRUSTED_PROXIES entry "192.168.1.256" must be an IPv4/IPv6 address or CIDR`,
		},
		{
			name:    "invalid cidr",
			proxies: []string{"10.0.0.0/33"},
			want:    `TRUSTED_PROXIES entry "10.0.0.0/33" must be an IPv4/IPv6 address or CIDR`,
		},
		{
			name:    "hostname",
			proxies: []string{"proxy.internal"},
			want:    `TRUSTED_PROXIES entry "proxy.internal" must be an IPv4/IPv6 address or CIDR`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validProductionConfigForTest()
			c.App.TrustedProxies = tt.proxies

			err := c.validate(nil)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestValidate_AllowsValidTrustedProxies(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.TrustedProxies = []string{
		"10.0.0.1",
		"10.0.0.0/8",
		"2001:db8::1",
		"2001:db8::/32",
	}

	require.NoError(t, c.validate(nil))
}

func TestValidate_ProductionRejectsBlankCoreRequiredConfig(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.HMACSecret = "  "
	c.App.MetricsUser = "  "
	c.App.MetricsPassword = "  "
	c.Database.URL = "  "
	c.Database.SSLRootCert = "  "
	c.Redis.Password = "  "
	c.Redis.TLSCAFile = "  "
	c.ObjectStorage.Endpoint = "  "
	c.ObjectStorage.Bucket = "  "
	c.ObjectStorage.AccessKeyID = "  "
	c.ObjectStorage.SecretAccessKey = "  "
	c.Bot.ServiceToken = "  "
	c.Admission.PublicBaseURL = "  "
	c.StudentVerification.PublicBaseURL = "  "

	err := c.validate(nil)

	require.Error(t, err)
	for _, message := range []string{
		"HMAC_SECRET is required in production",
		"DATABASE_URL is required in production",
		"METRICS_USER is required in production",
		"METRICS_PASSWORD is required in production",
		"REDIS_PASSWORD is required in production",
		"OBJECT_STORAGE_ENDPOINT is required in production",
		"OBJECT_STORAGE_BUCKET is required in production",
		"OBJECT_STORAGE_ACCESS_KEY_ID is required in production",
		"OBJECT_STORAGE_SECRET_ACCESS_KEY is required in production",
		"BOT_SERVICE_TOKEN is required in production",
		"ADMISSION_PUBLIC_BASE_URL is required in production",
		"STUDENT_VERIFICATION_PUBLIC_BASE_URL is required in production",
		"DB_SSL_ROOT_CERT is required in production",
		"REDIS_TLS_CA is required in production",
	} {
		assert.Contains(t, err.Error(), message)
	}
}

func TestValidate_ProductionRequiresBotServiceToken(t *testing.T) {
	c := validProductionConfigForTest()
	c.Bot.ServiceToken = ""

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "BOT_SERVICE_TOKEN is required in production")
}

func TestValidate_ProductionRequiresAdmissionPublicBaseURL(t *testing.T) {
	c := validProductionConfigForTest()
	c.Admission.PublicBaseURL = ""

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ADMISSION_PUBLIC_BASE_URL is required in production")
}

func TestValidate_ProductionRequiresStudentVerificationPublicBaseURL(t *testing.T) {
	c := validProductionConfigForTest()
	c.StudentVerification.PublicBaseURL = ""

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "STUDENT_VERIFICATION_PUBLIC_BASE_URL is required in production")
}

func TestValidate_StudentVerificationPublicBaseURL(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		wantError string
	}{
		{name: "relative", baseURL: "stuhelper.com", wantError: "must be an absolute http(s) URL"},
		{name: "path", baseURL: "https://stuhelper.com/verify", wantError: "must not include a path"},
		{name: "query", baseURL: "https://stuhelper.com?from=env", wantError: "must not include query or fragment"},
		{name: "production http", baseURL: "http://stuhelper.com", wantError: "must use https in production"},
		{name: "wrong production origin", baseURL: "https://verify.stuhelper.com", wantError: "must be https://stuhelper.com in production"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validProductionConfigForTest()
			c.StudentVerification.PublicBaseURL = tt.baseURL
			err := c.validate(nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestValidate_ProductionRequiresCanonicalAdmissionPublicBaseURL(t *testing.T) {
	c := validProductionConfigForTest()
	c.Admission.PublicBaseURL = "https://admission.example.com"

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ADMISSION_PUBLIC_BASE_URL must be https://join.stuhelper.com in production")
}

func TestValidate_RejectsInvalidAdmissionPublicBaseURL(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		wantError string
	}{
		{
			name:      "relative URL",
			baseURL:   "join.stuhelper.com",
			wantError: "ADMISSION_PUBLIC_BASE_URL must be an absolute http(s) URL",
		},
		{
			name:      "unsupported scheme",
			baseURL:   "ftp://join.stuhelper.com",
			wantError: "ADMISSION_PUBLIC_BASE_URL must be an absolute http(s) URL",
		},
		{
			name:      "path",
			baseURL:   "https://join.stuhelper.com/verify",
			wantError: "ADMISSION_PUBLIC_BASE_URL must not include a path",
		},
		{
			name:      "query",
			baseURL:   "https://join.stuhelper.com?from=env",
			wantError: "ADMISSION_PUBLIC_BASE_URL must not include query or fragment",
		},
		{
			name:      "production http",
			baseURL:   "http://join.stuhelper.com",
			wantError: "ADMISSION_PUBLIC_BASE_URL must use https in production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validProductionConfigForTest()
			c.Admission.PublicBaseURL = tt.baseURL

			err := c.validate(nil)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestValidate_ProdParityAllowsInsecureCookiesButKeepsProductionRequirements(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.Env = EnvProdParity
	c.Token.CookieSecure = false

	require.NoError(t, c.validate(nil))

	c.Bot.ServiceToken = ""
	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "BOT_SERVICE_TOKEN is required in production")
}

func TestValidate_ProductionRejectsInsecureCookies(t *testing.T) {
	c := validProductionConfigForTest()
	c.Token.CookieSecure = false

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "TOKEN_COOKIE_SECURE must be true in production")
}

func TestValidate_AllowsValidTokenCookieDomain(t *testing.T) {
	tests := []struct {
		name   string
		env    string
		domain string
	}{
		{name: "production root domain", env: EnvProduction, domain: "stuhelper.com"},
		{name: "production legacy leading dot", env: EnvProduction, domain: ".stuhelper.com"},
		{name: "production subdomain", env: EnvProduction, domain: "auth.stuhelper.com"},
		{name: "prod parity", env: EnvProdParity, domain: "stuhelper.com"},
		{name: "development empty", env: EnvDevelopment, domain: ""},
		{name: "development hostname", env: EnvDevelopment, domain: "dev.stuhelper.test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validProductionConfigForTest()
			c.App.Env = tt.env
			c.Token.CookieDomain = tt.domain

			require.NoError(t, c.validate(nil))
		})
	}
}

func TestValidate_RejectsInvalidTokenCookieDomain(t *testing.T) {
	tests := []struct {
		name   string
		domain string
	}{
		{name: "scheme", domain: "https://stuhelper.com"},
		{name: "port", domain: "stuhelper.com:443"},
		{name: "path", domain: "stuhelper.com/auth"},
		{name: "wildcard", domain: "*.stuhelper.com"},
		{name: "ipv4", domain: "192.0.2.10"},
		{name: "ipv6", domain: "2001:db8::1"},
		{name: "localhost", domain: "localhost"},
		{name: "multiple leading empty labels", domain: "..stuhelper.com"},
		{name: "trailing empty label", domain: "stuhelper.com."},
		{name: "middle empty label", domain: "stuhelper..com"},
		{name: "single label", domain: "stuhelper"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validProductionConfigForTest()
			c.Token.CookieDomain = tt.domain

			err := c.validate(nil)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "TOKEN_COOKIE_DOMAIN must be a hostname usable as a cookie Domain in production")
		})
	}
}

func TestValidate_RejectsTokenCookieDomainWhitespaceWhenNotTrimmed(t *testing.T) {
	c := validProductionConfigForTest()
	c.Token.CookieDomain = " stuhelper.com "

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "TOKEN_COOKIE_DOMAIN must not include leading or trailing whitespace")
}

func TestLoad_TrimsTokenCookieDomain(t *testing.T) {
	t.Setenv("APP_ENV", EnvDevelopment)
	t.Setenv("APP_PORT", "8080")
	t.Setenv("CORS_ORIGINS", "http://localhost:5173")
	t.Setenv("DOC_AES_ACTIVE_KEY_ID", "1")
	t.Setenv("DOC_AES_KEYS", "1:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("CASDOOR_ISSUER", "http://localhost:8000")
	t.Setenv("CASDOOR_CLIENT_ID", "client-id")
	t.Setenv("CASDOOR_CLIENT_SECRET", "client-secret")
	t.Setenv("CASDOOR_REDIRECT_URI", "http://localhost:8080/api/v1/auth/callback")
	t.Setenv("CASDOOR_INTROSPECTION_CLIENT_ID", "introspection-client")
	t.Setenv("CASDOOR_INTROSPECTION_CLIENT_SECRET", "introspection-secret")
	t.Setenv("CASDOOR_ORGANIZATION", "stuhelper")
	t.Setenv("OPENFGA_STORE_ID", "store-id")
	t.Setenv("OPENFGA_MODEL_ID", "model-id")
	t.Setenv("TOKEN_COOKIE_DOMAIN", "  stuhelper.test  ")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "stuhelper.test", cfg.Token.CookieDomain)
}

func TestLoad_NormalizesLegacyLeadingDotTokenCookieDomain(t *testing.T) {
	t.Setenv("APP_ENV", EnvDevelopment)
	t.Setenv("APP_PORT", "8080")
	t.Setenv("CORS_ORIGINS", "http://localhost:5173")
	t.Setenv("DOC_AES_ACTIVE_KEY_ID", "1")
	t.Setenv("DOC_AES_KEYS", "1:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("CASDOOR_ISSUER", "http://localhost:8000")
	t.Setenv("CASDOOR_CLIENT_ID", "client-id")
	t.Setenv("CASDOOR_CLIENT_SECRET", "client-secret")
	t.Setenv("CASDOOR_REDIRECT_URI", "http://localhost:8080/api/v1/auth/callback")
	t.Setenv("CASDOOR_INTROSPECTION_CLIENT_ID", "introspection-client")
	t.Setenv("CASDOOR_INTROSPECTION_CLIENT_SECRET", "introspection-secret")
	t.Setenv("CASDOOR_ORGANIZATION", "stuhelper")
	t.Setenv("OPENFGA_STORE_ID", "store-id")
	t.Setenv("OPENFGA_MODEL_ID", "model-id")
	t.Setenv("TOKEN_COOKIE_DOMAIN", ".stuhelper.test")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "stuhelper.test", cfg.Token.CookieDomain)
}

func TestLoad_TokenCookieDomainWhitespaceOnlyKeepsEmptyDevelopmentSemantics(t *testing.T) {
	t.Setenv("APP_ENV", EnvDevelopment)
	t.Setenv("APP_PORT", "8080")
	t.Setenv("CORS_ORIGINS", "http://localhost:5173")
	t.Setenv("DOC_AES_ACTIVE_KEY_ID", "1")
	t.Setenv("DOC_AES_KEYS", "1:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("CASDOOR_ISSUER", "http://localhost:8000")
	t.Setenv("CASDOOR_CLIENT_ID", "client-id")
	t.Setenv("CASDOOR_CLIENT_SECRET", "client-secret")
	t.Setenv("CASDOOR_REDIRECT_URI", "http://localhost:8080/api/v1/auth/callback")
	t.Setenv("CASDOOR_INTROSPECTION_CLIENT_ID", "introspection-client")
	t.Setenv("CASDOOR_INTROSPECTION_CLIENT_SECRET", "introspection-secret")
	t.Setenv("CASDOOR_ORGANIZATION", "stuhelper")
	t.Setenv("OPENFGA_STORE_ID", "store-id")
	t.Setenv("OPENFGA_MODEL_ID", "model-id")
	t.Setenv("TOKEN_COOKIE_DOMAIN", "   ")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Empty(t, cfg.Token.CookieDomain)
}

func TestLoad_EmptyOptionalTypedEnvUsesDefaults(t *testing.T) {
	t.Setenv("APP_ENV", EnvDevelopment)
	t.Setenv("APP_PORT", "8080")
	t.Setenv("CORS_ORIGINS", "http://localhost:5173")
	t.Setenv("DOC_AES_ACTIVE_KEY_ID", "1")
	t.Setenv("DOC_AES_KEYS", "1:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("CASDOOR_ISSUER", "http://localhost:8000")
	t.Setenv("CASDOOR_CLIENT_ID", "client-id")
	t.Setenv("CASDOOR_CLIENT_SECRET", "client-secret")
	t.Setenv("CASDOOR_REDIRECT_URI", "http://localhost:8080/api/v1/auth/callback")
	t.Setenv("CASDOOR_INTROSPECTION_CLIENT_ID", "introspection-client")
	t.Setenv("CASDOOR_INTROSPECTION_CLIENT_SECRET", "introspection-secret")
	t.Setenv("CASDOOR_ORGANIZATION", "stuhelper")
	t.Setenv("OPENFGA_STORE_ID", "store-id")
	t.Setenv("OPENFGA_MODEL_ID", "model-id")
	t.Setenv("EMAIL_ENABLED", "")
	t.Setenv("EMAIL_SMTP_PORT", "")
	t.Setenv("EMAIL_TENCENT_TEMPLATE_ID", "")
	t.Setenv("EMAIL_TENCENT_TEMPLATE_EXPIRE_MINUTES", "")
	t.Setenv("OTEL_TRACE_SAMPLE_RATIO", "")
	t.Setenv("REVIEW_TEACHER_STATS_REFRESH_TIMEOUT_SECONDS", "")

	cfg, err := Load()

	require.NoError(t, err)
	assert.False(t, cfg.Email.Enabled)
	assert.Equal(t, 587, cfg.Email.SMTPPort)
	assert.Equal(t, int64(0), cfg.Email.TencentTemplateID)
	assert.Equal(t, 5, cfg.Email.TencentTemplateExpireMinutes)
	assert.False(t, cfg.Email.InboundEnabled)
	assert.Equal(t, 300, cfg.Email.InboundWebhookMaxSkewSeconds)
	assert.InDelta(t, 0.2, cfg.Observability.TraceSampleRatio, 0.0001)
	assert.Equal(t, 60, cfg.Review.TeacherStatsRefreshTimeoutSeconds)
}

func TestValidateReviewConfigRejectsTimeoutOutsideProjectionLeaseBudget(t *testing.T) {
	for _, timeout := range []string{"0", "4", "91"} {
		t.Run(timeout, func(t *testing.T) {
			t.Setenv("REVIEW_TEACHER_STATS_REFRESH_TIMEOUT_SECONDS", timeout)
			var parseErrs []string

			reviewCfg := loadReviewConfig(&parseErrs)
			require.Empty(t, parseErrs)
			cfg := validProductionConfigForTest()
			cfg.Review = reviewCfg

			err := cfg.validate(parseErrs)
			require.Error(t, err)
			assert.Contains(t, err.Error(),
				"REVIEW_TEACHER_STATS_REFRESH_TIMEOUT_SECONDS must be between 5 and 90 seconds")
		})
	}
}

func TestValidateReviewConfigAcceptsLeaseSafeBoundaries(t *testing.T) {
	for _, timeout := range []int{5, 60, 90} {
		t.Run(fmt.Sprint(timeout), func(t *testing.T) {
			cfg := validProductionConfigForTest()
			cfg.Review.TeacherStatsRefreshTimeoutSeconds = timeout

			require.NoError(t, cfg.validate(nil))
		})
	}
}

func TestValidate_ProductionRequiresSMSEnabled(t *testing.T) {
	c := validProductionConfigForTest()
	c.SMS.Enabled = false

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "SMS_ENABLED must be true in production")
}

func TestValidate_EmailSMTPRequiresHostAndFrom(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.Env = "development"
	c.Token.CookieSecure = false
	c.Email = EmailConfig{
		Enabled: true,
		Driver:  "smtp",
	}

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "EMAIL_SMTP_HOST is required when EMAIL_ENABLED=true and EMAIL_DRIVER=smtp")
	assert.Contains(t, err.Error(), "EMAIL_FROM is required when EMAIL_ENABLED=true and EMAIL_DRIVER=smtp")
}

func TestValidate_InboundEmailRequiresTargetStrongWebhookSecretAndBoundedSkew(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.Env = "development"
	c.Token.CookieSecure = false
	c.Email.InboundEnabled = true
	c.Email.InboundTargetAddress = "invalid"
	c.Email.InboundWebhookSecret = "short"
	c.Email.InboundWebhookMaxSkewSeconds = 30

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "EMAIL_INBOUND_TARGET_ADDRESS must be a mailbox")
	assert.Contains(t, err.Error(), "EMAIL_INBOUND_WEBHOOK_SECRET must be at least 32 bytes")
	assert.Contains(t, err.Error(), "EMAIL_INBOUND_WEBHOOK_MAX_SKEW_SECONDS must be between 60 and 1800")
}

func TestValidate_ProdParityAllowsBlackholeEmailDriver(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.Env = EnvProdParity
	c.Token.CookieSecure = false
	c.Email = EmailConfig{
		Enabled: true,
		Driver:  "blackhole",
	}

	require.NoError(t, c.validate(nil))
}

func TestValidate_ProductionRejectsBlackholeEmailDriver(t *testing.T) {
	c := validProductionConfigForTest()
	c.Email = EmailConfig{
		Enabled: true,
		Driver:  "blackhole",
	}

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "EMAIL_DRIVER=blackhole is only allowed outside production")
}

func TestValidate_EmailTencentSESRequiresProviderConfig(t *testing.T) {
	c := validProductionConfigForTest()
	c.Email = EmailConfig{
		Enabled: true,
		Driver:  "tencent_ses",
	}

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "EMAIL_FROM is required when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses")
	assert.Contains(t, err.Error(), "EMAIL_TENCENT_SECRET_ID is required when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses")
	assert.Contains(t, err.Error(), "EMAIL_TENCENT_SECRET_KEY is required when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses")
	assert.Contains(t, err.Error(), "EMAIL_TENCENT_REGION is required when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses")
	assert.Contains(t, err.Error(), "EMAIL_TENCENT_ENDPOINT is required when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses")
	assert.Contains(t, err.Error(), "EMAIL_TENCENT_TEMPLATE_ID must be greater than 0 when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses")
	assert.Contains(t, err.Error(), "EMAIL_TENCENT_TEMPLATE_EXPIRE_MINUTES must be greater than 0 when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses")
}

func TestValidate_EmailTencentSESAllowsProduction(t *testing.T) {
	c := validProductionConfigForTest()
	c.Email = EmailConfig{
		Enabled:                      true,
		Driver:                       "tencent_ses",
		From:                         "noreply@notify.stuhelper.com",
		TencentSecretID:              "AKIDEXAMPLE",
		TencentSecretKey:             "secret-example",
		TencentRegion:                "ap-guangzhou",
		TencentEndpoint:              "ses.tencentcloudapi.com",
		TencentTemplateID:            49779,
		TencentTemplateExpireMinutes: 5,
	}

	require.NoError(t, c.validate(nil))
}

func TestValidate_EmailResendRequiresProviderConfig(t *testing.T) {
	c := validProductionConfigForTest()
	c.Email = EmailConfig{
		Enabled: true,
		Driver:  "resend",
	}

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "EMAIL_FROM is required when EMAIL_ENABLED=true and EMAIL_DRIVER=resend")
	assert.Contains(t, err.Error(), "EMAIL_RESEND_API_KEY is required when EMAIL_ENABLED=true and EMAIL_DRIVER=resend")
	assert.Contains(t, err.Error(), "EMAIL_RESEND_ENDPOINT is required when EMAIL_ENABLED=true and EMAIL_DRIVER=resend")
}

func TestValidate_EmailRejectsBlankRequiredProviderConfig(t *testing.T) {
	tests := []struct {
		name     string
		email    EmailConfig
		expected []string
	}{
		{
			name: "smtp",
			email: EmailConfig{
				Enabled:  true,
				Driver:   "smtp",
				SMTPHost: "  ",
				SMTPPort: 587,
				From:     "  ",
			},
			expected: []string{
				"EMAIL_SMTP_HOST is required when EMAIL_ENABLED=true and EMAIL_DRIVER=smtp",
				"EMAIL_FROM is required when EMAIL_ENABLED=true and EMAIL_DRIVER=smtp",
			},
		},
		{
			name: "tencent_ses",
			email: EmailConfig{
				Enabled:                      true,
				Driver:                       "tencent_ses",
				From:                         "  ",
				TencentSecretID:              "  ",
				TencentSecretKey:             "  ",
				TencentRegion:                "  ",
				TencentEndpoint:              "  ",
				TencentTemplateID:            49779,
				TencentTemplateExpireMinutes: 5,
			},
			expected: []string{
				"EMAIL_FROM is required when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses",
				"EMAIL_TENCENT_SECRET_ID is required when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses",
				"EMAIL_TENCENT_SECRET_KEY is required when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses",
				"EMAIL_TENCENT_REGION is required when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses",
				"EMAIL_TENCENT_ENDPOINT is required when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses",
			},
		},
		{
			name: "resend",
			email: EmailConfig{
				Enabled:        true,
				Driver:         "resend",
				From:           "  ",
				ResendAPIKey:   "  ",
				ResendEndpoint: "  ",
			},
			expected: []string{
				"EMAIL_FROM is required when EMAIL_ENABLED=true and EMAIL_DRIVER=resend",
				"EMAIL_RESEND_API_KEY is required when EMAIL_ENABLED=true and EMAIL_DRIVER=resend",
				"EMAIL_RESEND_ENDPOINT is required when EMAIL_ENABLED=true and EMAIL_DRIVER=resend",
			},
		},
		{
			name: "multi",
			email: EmailConfig{
				Enabled:                      true,
				Driver:                       "multi",
				From:                         "  ",
				TencentSecretID:              "  ",
				TencentSecretKey:             "  ",
				TencentRegion:                "  ",
				TencentEndpoint:              "  ",
				TencentTemplateID:            49779,
				TencentTemplateExpireMinutes: 5,
				ResendAPIKey:                 "  ",
				ResendEndpoint:               "  ",
			},
			expected: []string{
				"EMAIL_FROM is required when EMAIL_ENABLED=true and EMAIL_DRIVER=multi",
				"EMAIL_TENCENT_SECRET_ID is required when EMAIL_ENABLED=true and EMAIL_DRIVER=multi",
				"EMAIL_TENCENT_SECRET_KEY is required when EMAIL_ENABLED=true and EMAIL_DRIVER=multi",
				"EMAIL_TENCENT_REGION is required when EMAIL_ENABLED=true and EMAIL_DRIVER=multi",
				"EMAIL_TENCENT_ENDPOINT is required when EMAIL_ENABLED=true and EMAIL_DRIVER=multi",
				"EMAIL_RESEND_API_KEY is required when EMAIL_ENABLED=true and EMAIL_DRIVER=multi",
				"EMAIL_RESEND_ENDPOINT is required when EMAIL_ENABLED=true and EMAIL_DRIVER=multi",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validProductionConfigForTest()
			c.Email = tt.email

			err := c.validate(nil)

			require.Error(t, err)
			for _, message := range tt.expected {
				assert.Contains(t, err.Error(), message)
			}
		})
	}
}

func TestValidate_EmailMultiAllowsProduction(t *testing.T) {
	c := validProductionConfigForTest()
	c.Email = EmailConfig{
		Enabled:                      true,
		Driver:                       "multi",
		From:                         "noreply@notify.stuhelper.com",
		TencentSecretID:              "AKIDEXAMPLE",
		TencentSecretKey:             "secret-example",
		TencentRegion:                "ap-guangzhou",
		TencentEndpoint:              "ses.tencentcloudapi.com",
		TencentTemplateID:            49779,
		TencentTemplateExpireMinutes: 5,
		ResendAPIKey:                 "re_test",
		ResendEndpoint:               "https://api.resend.com/emails",
	}

	require.NoError(t, c.validate(nil))
}

func TestValidate_ProductionRejectsExplicitExternalPlaintextPostgres(t *testing.T) {
	c := validProductionConfigForTest()
	c.Database.SSLMode = "disable"
	c.Database.SSLRootCert = ""
	c.Database.AllowPlaintext = true

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "EXTERNAL_POSTGRES_ALLOW_PLAINTEXT is only allowed in prod-parity")
	assert.Contains(t, err.Error(), "DB_SSL_MODE must be 'verify-full' in production")
}

func TestValidate_ProdParityAllowsExplicitPlaintextPostgres(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.Env = EnvProdParity
	c.Database.SSLMode = "disable"
	c.Database.SSLRootCert = ""
	c.Database.AllowPlaintext = true

	require.NoError(t, c.validate(nil))
}

func TestValidate_ProductionRejectsImplicitPlaintextDatastores(t *testing.T) {
	c := validProductionConfigForTest()
	c.Database.SSLMode = "disable"
	c.Database.SSLRootCert = ""
	c.Redis.TLSEnabled = false
	c.Redis.TLSCAFile = ""

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_SSL_MODE must be 'verify-full' in production")
	assert.Contains(t, err.Error(), "REDIS_TLS_ENABLED must be true in production")
}

func TestLoadAdmissionConfigFromEnv(t *testing.T) {
	t.Setenv("ADMISSION_PUBLIC_BASE_URL", "https://join.stuhelper.com")

	cfg := loadAdmissionConfig()

	assert.Equal(t, "https://join.stuhelper.com", cfg.PublicBaseURL)
}

func TestLoadStudentVerificationConfigFromEnv(t *testing.T) {
	t.Setenv("STUDENT_VERIFICATION_PUBLIC_BASE_URL", "https://stuhelper.com")

	cfg := loadStudentVerificationConfig()

	assert.Equal(t, "https://stuhelper.com", cfg.PublicBaseURL)
}

func TestLoadExternalDataConfigFromEnv(t *testing.T) {
	t.Setenv("EXTERNAL_STUDENT_SOURCE_ENABLED", "true")
	t.Setenv("EXTERNAL_STUDENT_SOURCE_PROVIDER", "oracle")
	t.Setenv("EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE", "4111010006")
	t.Setenv("EXTERNAL_STUDENT_SOURCE_ORACLE_HOST", "oracle.example.test")
	t.Setenv("EXTERNAL_STUDENT_SOURCE_ORACLE_SERVICE_NAME", "ORCLPDB1")
	t.Setenv("EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME", "stuhelper_academic_ro")
	t.Setenv("EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME", "stuhelper_academic_ro")
	t.Setenv("EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD", "secret")
	t.Setenv("EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA", "USR_JWBIZ")
	t.Setenv("EXTERNAL_STUDENT_SOURCE_ORACLE_TABLE", "T_XS_JBXX")

	var parseErrs []string
	cfg := loadExternalDataConfig(&parseErrs)

	require.Empty(t, parseErrs)
	require.Len(t, cfg.StudentSources, 1)
	source := cfg.StudentSources[0]
	assert.True(t, source.Enabled)
	assert.Equal(t, "oracle", source.Provider)
	assert.Equal(t, "4111010006", source.SchoolCode)
	assert.Equal(t, "oracle.example.test", source.Oracle.Host)
	assert.Equal(t, 2484, source.Oracle.Port)
	assert.Equal(t, "ORCLPDB1", source.Oracle.ServiceName)
	assert.Equal(t, "stuhelper_academic_ro", source.Oracle.ExpectedUsername)
	assert.Equal(t, "verify-full", source.Oracle.TLSMode)
	assert.Equal(t, "/external-student-source-tls/ca.crt", source.Oracle.TLSCAFile)
	assert.Equal(t, "USR_JWBIZ", source.Oracle.Schema)
	assert.Equal(t, "T_XS_JBXX", source.Oracle.Table)
	assert.Equal(t, "XH", source.Oracle.StudentIDColumn)
	assert.Equal(t, "XM", source.Oracle.StudentNameColumn)
	assert.Equal(t, 5, source.Oracle.BreakerFailureThreshold)
	assert.Equal(t, 2, source.Oracle.BreakerSuccessThreshold)
	assert.Equal(t, 30, source.Oracle.BreakerOpenSeconds)
}

func TestValidateRejectsIncompleteExternalStudentSource(t *testing.T) {
	c := validProductionConfigForTest()
	c.ExternalData.StudentSources = []ExternalStudentSourceConfig{{
		Name:       "buaa",
		Enabled:    true,
		Provider:   "oracle",
		SchoolCode: "4111010006",
		Oracle: ExternalOracleStudentSourceConfig{
			Host:                    "oracle.example.test",
			Port:                    1521,
			ServiceName:             "ORCLPDB1",
			Username:                "stuhelper_academic_ro",
			TLSMode:                 "verify-full",
			TLSCAFile:               "/external-student-source-tls/ca.crt",
			Schema:                  "USR_JWBIZ",
			Table:                   "T_XS_JBXX",
			StudentIDColumn:         "XH",
			StudentNameColumn:       "XM",
			ConnectTimeoutSeconds:   5,
			QueryTimeoutSeconds:     3,
			MaxOpenConns:            4,
			MaxIdleConns:            1,
			ConnMaxLifetimeSeconds:  300,
			ConnMaxIdleTimeSeconds:  60,
			BreakerFailureThreshold: 5,
			BreakerSuccessThreshold: 2,
			BreakerOpenSeconds:      30,
		},
	}}

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD is required")
}

func TestValidateRejectsPlaintextOracleStudentSourceInProduction(t *testing.T) {
	c := validProductionConfigForTest()
	c.ExternalData.StudentSources = []ExternalStudentSourceConfig{{
		Name:       "buaa",
		Enabled:    true,
		Provider:   "oracle",
		SchoolCode: "4111010006",
		Oracle: ExternalOracleStudentSourceConfig{
			Host:                    "oracle.example.test",
			Port:                    1521,
			ServiceName:             "ORCLPDB1",
			Username:                "stuhelper_ro",
			Password:                "secret",
			TLSMode:                 "disable",
			Schema:                  "USR_JWBIZ",
			Table:                   "T_XS_JBXX",
			StudentIDColumn:         "XH",
			StudentNameColumn:       "XM",
			ConnectTimeoutSeconds:   5,
			QueryTimeoutSeconds:     3,
			MaxOpenConns:            4,
			MaxIdleConns:            1,
			ConnMaxLifetimeSeconds:  300,
			ConnMaxIdleTimeSeconds:  60,
			BreakerFailureThreshold: 5,
			BreakerSuccessThreshold: 2,
			BreakerOpenSeconds:      30,
		},
	}}

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE must be verify-full in production")
}

func TestValidateRejectsAdministrativeOracleStudentSourceAccount(t *testing.T) {
	for _, username := range []string{"SYS", "system", "SYSBACKUP", "SYSDG", "SYSKM", "SYSRAC"} {
		t.Run(username, func(t *testing.T) {
			errs := validateExternalOracleStudentSource(ExternalOracleStudentSourceConfig{
				Host:                    "oracle.example.test",
				Port:                    2484,
				ServiceName:             "ORCLPDB1",
				Username:                username,
				Password:                "secret",
				TLSMode:                 "verify-full",
				TLSCAFile:               "/external-student-source-tls/ca.crt",
				Schema:                  "USR_JWBIZ",
				Table:                   "T_XS_JBXX",
				StudentIDColumn:         "XH",
				StudentNameColumn:       "XM",
				ConnectTimeoutSeconds:   5,
				QueryTimeoutSeconds:     3,
				MaxOpenConns:            4,
				MaxIdleConns:            1,
				ConnMaxLifetimeSeconds:  300,
				ConnMaxIdleTimeSeconds:  60,
				BreakerFailureThreshold: 5,
				BreakerSuccessThreshold: 2,
				BreakerOpenSeconds:      30,
			}, true)

			assert.Contains(
				t,
				errs,
				"EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME must not be a built-in administrative account",
			)
		})
	}
}

func TestValidateAllowsExplicitExistingOracleStudentSourceSchemaOwnerAccount(t *testing.T) {
	errs := validateExternalOracleStudentSource(ExternalOracleStudentSourceConfig{
		Host:                    "oracle.example.test",
		Port:                    2484,
		ServiceName:             "ORCLPDB1",
		Username:                "usr_jwbiz",
		ExpectedUsername:        "usr_jwbiz",
		Password:                "secret",
		TLSMode:                 "verify-full",
		TLSCAFile:               "/external-student-source-tls/ca.crt",
		Schema:                  "USR_JWBIZ",
		Table:                   "T_XS_JBXX",
		StudentIDColumn:         "XH",
		StudentNameColumn:       "XM",
		ConnectTimeoutSeconds:   5,
		QueryTimeoutSeconds:     3,
		MaxOpenConns:            4,
		MaxIdleConns:            1,
		ConnMaxLifetimeSeconds:  300,
		ConnMaxIdleTimeSeconds:  60,
		BreakerFailureThreshold: 5,
		BreakerSuccessThreshold: 2,
		BreakerOpenSeconds:      30,
	}, true)

	assert.Empty(t, errs)
}

func TestValidateRejectsOracleStudentSourceRuntimeIdentityDrift(t *testing.T) {
	errs := validateExternalOracleStudentSource(ExternalOracleStudentSourceConfig{
		Host:                    "oracle.example.test",
		Port:                    2484,
		ServiceName:             "ORCLPDB1",
		Username:                "unexpected_ro",
		ExpectedUsername:        "stuhelper_academic_ro",
		Password:                "secret",
		TLSMode:                 "verify-full",
		TLSCAFile:               "/external-student-source-tls/ca.crt",
		Schema:                  "USR_JWBIZ",
		Table:                   "T_XS_JBXX",
		StudentIDColumn:         "XH",
		StudentNameColumn:       "XM",
		ConnectTimeoutSeconds:   5,
		QueryTimeoutSeconds:     3,
		MaxOpenConns:            4,
		MaxIdleConns:            1,
		ConnMaxLifetimeSeconds:  300,
		ConnMaxIdleTimeSeconds:  60,
		BreakerFailureThreshold: 5,
		BreakerSuccessThreshold: 2,
		BreakerOpenSeconds:      30,
	}, true)

	assert.Contains(
		t,
		errs,
		"EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME must match the explicitly configured existing account",
	)
}

func TestValidate_DevelopmentAllowsMissingBotServiceToken(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.Env = "development"
	c.Token.CookieSecure = false
	c.Bot.ServiceToken = ""

	err := c.validate(nil)

	require.NoError(t, err)
}

func TestValidate_ProductionRequiresCasdoorAdminCredentials(t *testing.T) {
	c := validProductionConfigForTest()
	c.Casdoor.AppProvisioningClientID = ""
	c.Casdoor.UserProfileClientSecret = ""
	c.Casdoor.UserLookupApplication = ""

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CASDOOR_APP_PROVISIONING_CLIENT_ID is required")
	assert.Contains(t, err.Error(), "CASDOOR_USER_PROFILE_CLIENT_SECRET is required")
	assert.Contains(t, err.Error(), "CASDOOR_USER_LOOKUP_APPLICATION is required")
}

func TestValidate_ProductionRejectsBlankCasdoorAdminCredentials(t *testing.T) {
	c := validProductionConfigForTest()
	c.Casdoor.AppProvisioningClientID = "  "
	c.Casdoor.AppProvisioningClientSecret = "  "
	c.Casdoor.AppProvisioningApplication = "  "
	c.Casdoor.UserProfileClientID = "  "
	c.Casdoor.UserProfileClientSecret = "  "
	c.Casdoor.UserProfileApplication = "  "
	c.Casdoor.UserLookupClientID = "  "
	c.Casdoor.UserLookupClientSecret = "  "
	c.Casdoor.UserLookupApplication = "  "

	err := c.validate(nil)

	require.Error(t, err)
	for _, message := range []string{
		"CASDOOR_APP_PROVISIONING_CLIENT_ID is required",
		"CASDOOR_APP_PROVISIONING_CLIENT_SECRET is required",
		"CASDOOR_APP_PROVISIONING_APPLICATION is required",
		"CASDOOR_USER_PROFILE_CLIENT_ID is required",
		"CASDOOR_USER_PROFILE_CLIENT_SECRET is required",
		"CASDOOR_USER_PROFILE_APPLICATION is required",
		"CASDOOR_USER_LOOKUP_CLIENT_ID is required",
		"CASDOOR_USER_LOOKUP_CLIENT_SECRET is required",
		"CASDOOR_USER_LOOKUP_APPLICATION is required",
	} {
		assert.Contains(t, err.Error(), message)
	}
}

func TestValidate_CasdoorEndpointConfig(t *testing.T) {
	t.Run("requires explicit public auth base URL in production-like environments", func(t *testing.T) {
		for _, env := range []string{EnvProduction, EnvProdParity} {
			t.Run(env, func(t *testing.T) {
				c := validProductionConfigForTest()
				c.App.Env = env
				c.Casdoor.PublicAuthBaseURL = ""

				err := c.validate(nil)

				require.Error(t, err)
				assert.Contains(t, err.Error(), "CASDOOR_PUBLIC_AUTH_BASE_URL is required in production")
			})
		}
	})

	t.Run("requires https public auth origin in production-like environments", func(t *testing.T) {
		for _, env := range []string{EnvProduction, EnvProdParity} {
			t.Run(env, func(t *testing.T) {
				c := validProductionConfigForTest()
				c.App.Env = env
				c.Casdoor.PublicAuthBaseURL = "http://sso.example.com"

				err := c.validate(nil)

				require.Error(t, err)
				assert.Contains(t, err.Error(), "CASDOOR_PUBLIC_AUTH_BASE_URL must use https in production")
			})
		}
	})

	t.Run("requires public auth base URL to be an origin", func(t *testing.T) {
		tests := []struct {
			name    string
			value   string
			message string
		}{
			{name: "trailing slash", value: "https://sso.example.com/", message: "CASDOOR_PUBLIC_AUTH_BASE_URL must not have a trailing slash"},
			{name: "path", value: "https://sso.example.com/login", message: "CASDOOR_PUBLIC_AUTH_BASE_URL must not include a path"},
			{name: "query", value: "https://sso.example.com?next=1", message: "CASDOOR_PUBLIC_AUTH_BASE_URL must not include query or fragment"},
			{name: "fragment", value: "https://sso.example.com#login", message: "CASDOOR_PUBLIC_AUTH_BASE_URL must not include query or fragment"},
			{name: "userinfo", value: "https://user@sso.example.com", message: "CASDOOR_PUBLIC_AUTH_BASE_URL must not include user info"},
			{name: "invalid port", value: "https://sso.example.com:bad", message: "CASDOOR_PUBLIC_AUTH_BASE_URL must include a valid port when a port is specified"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := validProductionConfigForTest()
				c.Casdoor.PublicAuthBaseURL = tt.value

				err := c.validate(nil)

				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.message)
			})
		}
	})

	t.Run("allows empty public auth base URL in development", func(t *testing.T) {
		c := validConfigForValidation()
		c.Casdoor.PublicAuthBaseURL = ""

		require.NoError(t, c.validate(nil))
	})

	t.Run("allows local http public auth origin in development", func(t *testing.T) {
		c := validConfigForValidation()
		c.Casdoor.PublicAuthBaseURL = "http://localhost:8000"

		require.NoError(t, c.validate(nil))
	})

	t.Run("rejects URL internal address", func(t *testing.T) {
		c := validConfigForValidation()
		c.Casdoor.InternalAddress = "http://casdoor:8000"

		err := c.validate(nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "CASDOOR_INTERNAL_ADDRESS must be a host or host:port dial address, not a URL")
	})

	t.Run("rejects invalid internal address port", func(t *testing.T) {
		for _, address := range []string{"casdoor:", "casdoor:bad", "casdoor:0", "casdoor:65536", "[::1]:bad"} {
			t.Run(address, func(t *testing.T) {
				c := validConfigForValidation()
				c.Casdoor.InternalAddress = address

				err := c.validate(nil)

				require.Error(t, err)
				assert.Contains(t, err.Error(), "CASDOOR_INTERNAL_ADDRESS must include a valid port when a port is specified")
			})
		}
	})

	t.Run("rejects malformed internal address", func(t *testing.T) {
		for _, address := range []string{":8000", "[::1", "cas door:8000"} {
			t.Run(address, func(t *testing.T) {
				c := validConfigForValidation()
				c.Casdoor.InternalAddress = address

				err := c.validate(nil)

				require.Error(t, err)
				assert.Contains(t, err.Error(), "CASDOOR_INTERNAL_ADDRESS must be a host or host:port dial address")
			})
		}
	})

	t.Run("allows dial internal addresses", func(t *testing.T) {
		for _, address := range []string{
			"casdoor",
			"casdoor:8000",
			"127.0.0.1",
			"127.0.0.1:8000",
			"::1",
			"[::1]",
			"[::1]:8000",
		} {
			t.Run(address, func(t *testing.T) {
				c := validConfigForValidation()
				c.Casdoor.InternalAddress = address

				require.NoError(t, c.validate(nil))
			})
		}
	})
}

func TestValidate_ProductionRequiresOpenPlatformRuntimeTokenProbe(t *testing.T) {
	c := validProductionConfigForTest()
	c.OpenPlatform.TokenProbe.RuntimeRequired = false

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED must be true in production")
}

func TestValidate_OpenPlatformRuntimeTokenProbeRequiresCommandWhenRequired(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.Env = "development"
	c.Token.CookieSecure = false
	c.OpenPlatform.TokenProbe.RuntimeRequired = true
	c.OpenPlatform.TokenProbe.RuntimeCommand = ""

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND is required")
}

func TestValidate_OpenPlatformRuntimeTokenProbeRejectsInvalidTimeout(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.Env = "development"
	c.Token.CookieSecure = false
	c.OpenPlatform.TokenProbe.RuntimeTimeoutSeconds = 601

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_TIMEOUT_SECONDS must be between 1 and 600 seconds")
}

func TestValidate_DevelopmentRejectsPartialCasdoorAppProvisioningCredential(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.Env = "development"
	c.Token.CookieSecure = false
	c.Casdoor.AppProvisioningClientID = "app-provisioning-client"
	c.Casdoor.AppProvisioningClientSecret = ""
	c.Casdoor.AppProvisioningApplication = ""

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CASDOOR_APP_PROVISIONING_CLIENT_SECRET is required")
	assert.Contains(t, err.Error(), "CASDOOR_APP_PROVISIONING_APPLICATION is required")
}

func TestValidate_DevelopmentRejectsPartialCasdoorUserProfileCredential(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.Env = "development"
	c.Token.CookieSecure = false
	c.Casdoor.UserProfileClientID = "user-profile-client"
	c.Casdoor.UserProfileClientSecret = ""
	c.Casdoor.UserProfileApplication = ""

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CASDOOR_USER_PROFILE_CLIENT_SECRET is required")
	assert.Contains(t, err.Error(), "CASDOOR_USER_PROFILE_APPLICATION is required")
}

func TestValidate_RejectsInvalidTraceSampleRatio(t *testing.T) {
	c := &Config{
		Observability: ObservabilityConfig{
			Enabled:          true,
			ServiceName:      "stuhelper-backend",
			OTLPEndpoint:     "http://alloy:4318",
			TraceSampleRatio: 1.5,
		},
	}

	err := c.validate(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OTEL_TRACE_SAMPLE_RATIO must be between 0 and 1")
}

func TestValidate_ObservabilityRejectsBlankRequiredFieldsWhenEnabled(t *testing.T) {
	c := &Config{
		Observability: ObservabilityConfig{
			Enabled:      true,
			ServiceName:  "  ",
			OTLPEndpoint: "  ",
		},
	}

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "OTEL_SERVICE_NAME is required when OTEL_ENABLED=true")
	assert.Contains(t, err.Error(), "OTEL_EXPORTER_OTLP_ENDPOINT is required when OTEL_ENABLED=true")
}

func TestValidate_SMSRequiresFullConfigWhenEnabled(t *testing.T) {
	c := &Config{
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
		SMS: SMSConfig{
			Enabled: true,
		},
	}

	err := c.validate(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SMS_SECRET_ID is required when SMS_ENABLED=true")
	assert.Contains(t, err.Error(), "SMS_INTERNAL_KEY is required when SMS_ENABLED=true")
}

func TestValidate_SMSRejectsBlankRequiredConfigWhenEnabled(t *testing.T) {
	c := validProductionConfigForTest()
	c.SMS = SMSConfig{
		Enabled:     true,
		SecretID:    "  ",
		SecretKey:   "  ",
		AppID:       "  ",
		SignName:    "  ",
		TemplateID:  "  ",
		InternalKey: "  ",
	}

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "SMS_SECRET_ID is required when SMS_ENABLED=true")
	assert.Contains(t, err.Error(), "SMS_SECRET_KEY is required when SMS_ENABLED=true")
	assert.Contains(t, err.Error(), "SMS_APP_ID is required when SMS_ENABLED=true")
	assert.Contains(t, err.Error(), "SMS_SIGN_NAME is required when SMS_ENABLED=true")
	assert.Contains(t, err.Error(), "SMS_TEMPLATE_ID is required when SMS_ENABLED=true")
	assert.Contains(t, err.Error(), "SMS_INTERNAL_KEY is required when SMS_ENABLED=true")
}

func TestValidate_SMSDisabledAllowsEmptyConfig(t *testing.T) {
	c := &Config{
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
		SMS: SMSConfig{
			Enabled: false,
		},
	}

	require.NoError(t, c.validate(nil))
}

func TestLoadObjectStorageConfig_TLSCAFilePrefersObjectStorageEnv(t *testing.T) {
	t.Setenv("OBJECT_STORAGE_TLS_CA", "/run/secrets/object-storage-ca.crt")
	t.Setenv("AWS_CA_BUNDLE", "/run/secrets/aws-ca-bundle.crt")

	var parseErrs []string
	cfg := loadObjectStorageConfig(&parseErrs)

	require.Empty(t, parseErrs)
	assert.Equal(t, "/run/secrets/object-storage-ca.crt", cfg.TLSCAFile)
}

func TestLoadObjectStorageConfig_TLSCAFileFallsBackToAWSBundle(t *testing.T) {
	t.Setenv("OBJECT_STORAGE_TLS_CA", "")
	t.Setenv("AWS_CA_BUNDLE", "/run/secrets/aws-ca-bundle.crt")

	var parseErrs []string
	cfg := loadObjectStorageConfig(&parseErrs)

	require.Empty(t, parseErrs)
	assert.Equal(t, "/run/secrets/aws-ca-bundle.crt", cfg.TLSCAFile)
}

func TestValidate_RequiresCORSOriginsInDevelopment(t *testing.T) {
	c := &Config{
		App: AppConfig{
			Env:             "development",
			HMACSecret:      "0123456789abcdef0123456789abcdef",
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
		SMS: SMSConfig{
			Enabled: false,
		},
	}

	err := c.validate(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CORS_ORIGINS is required")
}

func TestValidate_RejectsInvalidCORSOrigins(t *testing.T) {
	tests := []struct {
		name    string
		origins []string
		want    string
	}{
		{
			name:    "empty entry",
			origins: []string{""},
			want:    "empty origin is not allowed",
		},
		{
			name:    "surrounding whitespace",
			origins: []string{" https://stuhelper.example.com"},
			want:    "must not include leading or trailing whitespace",
		},
		{
			name:    "wildcard",
			origins: []string{"*"},
			want:    "wildcard '*' is not allowed",
		},
		{
			name:    "relative URL",
			origins: []string{"localhost:3000"},
			want:    "must be an absolute http(s) origin",
		},
		{
			name:    "unsupported scheme",
			origins: []string{"ftp://files.example.com"},
			want:    "must be an absolute http(s) origin",
		},
		{
			name:    "userinfo",
			origins: []string{"https://user:pass@stuhelper.example.com"},
			want:    "must not include user info",
		},
		{
			name:    "non-numeric port",
			origins: []string{"https://stuhelper.example.com:bad"},
			want:    "must include a valid port",
		},
		{
			name:    "port out of range",
			origins: []string{"https://stuhelper.example.com:65536"},
			want:    "must include a valid port",
		},
		{
			name:    "trailing slash",
			origins: []string{"https://stuhelper.example.com/"},
			want:    "must not have a trailing slash",
		},
		{
			name:    "path",
			origins: []string{"https://stuhelper.example.com/app"},
			want:    "must not include a path",
		},
		{
			name:    "query",
			origins: []string{"https://stuhelper.example.com?next=/app"},
			want:    "must not include query or fragment",
		},
		{
			name:    "fragment",
			origins: []string{"https://stuhelper.example.com#app"},
			want:    "must not include query or fragment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfigForValidation()
			cfg.App.CORSOrigins = tt.origins

			err := cfg.validate(nil)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestValidate_AllowsValidCORSOrigins(t *testing.T) {
	cfg := validConfigForValidation()
	cfg.App.CORSOrigins = []string{
		"http://localhost:3000",
		"http://127.0.0.1:3001",
		"http://[::1]:3002",
		"https://stuhelper.example.com",
	}

	require.NoError(t, cfg.validate(nil))
}

func TestValidate_ProductionRejectsHTTPCORSOrigins(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{name: "production", env: EnvProduction},
		{name: "prod parity", env: EnvProdParity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validProductionConfigForTest()
			cfg.App.Env = tt.env
			cfg.App.CORSOrigins = []string{"http://web.example.com"}

			err := cfg.validate(nil)

			require.Error(t, err)
			assert.Contains(t, err.Error(), `CORS configuration error: origin "http://web.example.com" must use https in production`)
		})
	}
}

func TestValidate_DevelopmentAllowsLocalHTTPCORSOrigins(t *testing.T) {
	cfg := validConfigForValidation()
	cfg.App.CORSOrigins = []string{
		"http://localhost:3000",
		"http://127.0.0.1:3001",
		"http://[::1]:3002",
	}

	require.NoError(t, cfg.validate(nil))
}

func TestValidate_RequiresOpenFGAInDevelopment(t *testing.T) {
	c := &Config{
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
		SMS: SMSConfig{
			Enabled: false,
		},
	}

	err := c.validate(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OPENFGA_STORE_ID is required")
	assert.Contains(t, err.Error(), "OPENFGA_MODEL_ID is required")
	assert.Contains(t, err.Error(), "OPENFGA_API_URL is required")
}

func TestValidate_RejectsBlankIdentityAndAuthorizationConfig(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.Env = "development"
	c.Token.CookieSecure = false
	c.Casdoor.Issuer = "  "
	c.Casdoor.ClientID = "  "
	c.Casdoor.ClientSecret = "  "
	c.Casdoor.RedirectURI = "  "
	c.Casdoor.IntrospectionClientID = "  "
	c.Casdoor.IntrospectionClientSecret = "  "
	c.Casdoor.Organization = "  "
	c.OpenFGA.StoreID = "  "
	c.OpenFGA.AuthorizationModelID = "  "
	c.OpenFGA.APIUrl = "  "

	err := c.validate(nil)

	require.Error(t, err)
	for _, message := range []string{
		"CASDOOR_ISSUER is required",
		"CASDOOR_CLIENT_ID is required",
		"CASDOOR_CLIENT_SECRET is required",
		"CASDOOR_REDIRECT_URI is required",
		"CASDOOR_INTROSPECTION_CLIENT_ID is required",
		"CASDOOR_INTROSPECTION_CLIENT_SECRET is required",
		"CASDOOR_ORGANIZATION is required",
		"OPENFGA_STORE_ID is required",
		"OPENFGA_MODEL_ID is required",
		"OPENFGA_API_URL is required",
	} {
		assert.Contains(t, err.Error(), message)
	}
}

func validProductionConfigForTest() *Config {
	return &Config{
		Review: ReviewConfig{TeacherStatsRefreshTimeoutSeconds: 60},
		App: AppConfig{
			Env:                "production",
			Port:               "8080",
			HMACSecret:         "0123456789abcdef0123456789abcdef",
			CORSOrigins:        []string{"https://stuhelper.example.com"},
			TrustedProxies:     []string{"10.0.0.0/8"},
			MetricsUser:        "prometheus",
			MetricsPassword:    "metrics-password",
			MaxBodySize:        10 << 20,
			APIIPRateLimit:     100,
			APIGlobalLimit:     10000,
			HealthCheckTimeout: 3,
		},
		Security: SecurityConfig{DocAESActiveKeyID: 1, DocAESKeys: map[uint8][]byte{1: make([]byte, 32)}},
		Database: DatabaseConfig{URL: "postgres://user:pass@db:5432/stuhelper?sslmode=verify-full", QueryTimeout: 5, MaxConns: 20, MinConns: 2, SSLMode: "verify-full", SSLRootCert: "/run/secrets/postgres-ca.crt"},
		Token:    TokenConfig{AccessTokenTTL: 300, RefreshTokenTTL: 604800, CookieSecure: true},
		Redis:    RedisConfig{Password: "redis-password", TLSEnabled: true, TLSCAFile: "/run/secrets/redis-ca.crt"},
		ObjectStorage: ObjectStorageConfig{
			Endpoint: "https://s3.example.com", Bucket: "stuhelper", AccessKeyID: "access", SecretAccessKey: "secret", UseSSL: true, PresignTTL: 600,
		},
		Casdoor: CasdoorConfig{
			Issuer:                      "https://sso.example.com",
			PublicAuthBaseURL:           "https://sso.example.com",
			ClientID:                    "client-id",
			ClientSecret:                "client-secret",
			RedirectURI:                 "https://api.example.com/api/v1/auth/callback",
			IntrospectionClientID:       "introspection-client",
			IntrospectionClientSecret:   "introspection-secret",
			Organization:                "stuhelper",
			AppProvisioningClientID:     "app-provisioning-client",
			AppProvisioningClientSecret: "app-provisioning-secret",
			AppProvisioningApplication:  "stuhelper-app-provisioning",
			UserProfileClientID:         "user-profile-client",
			UserProfileClientSecret:     "user-profile-secret",
			UserProfileApplication:      "stuhelper-user-profile",
			UserLookupClientID:          "user-lookup-client",
			UserLookupClientSecret:      "user-lookup-secret",
			UserLookupApplication:       "stuhelper-user-lookup",
		},
		OpenFGA: OpenFGAConfig{StoreID: "store-id", AuthorizationModelID: "model-id", APIUrl: "http://openfga:8080"},
		OpenPlatform: OpenPlatformConfig{
			ConsentBaseURL: "https://stuhelper.example.com",
			AccountBaseURL: "https://stuhelper.example.com",
			TokenProbe: OpenPlatformTokenProbeConfig{
				RuntimeRequired:       true,
				RuntimeCommand:        "/usr/local/bin/stuhelper-token-probe",
				RuntimeTimeoutSeconds: 30,
			},
		},
		Observability: ObservabilityConfig{
			Enabled: true, ServiceName: "stuhelper-backend", OTLPEndpoint: "http://alloy:4318", TraceSampleRatio: 0.2,
		},
		Admission:           AdmissionConfig{PublicBaseURL: "https://join.stuhelper.com"},
		StudentVerification: StudentVerificationConfig{PublicBaseURL: "https://stuhelper.com"},
		RateLimit:           ReviewRateLimitConfig{PostLimit: 5, VoteLimit: 30, ReportLimit: 10, ReplyLimit: 10, WriteLimit: 10, SearchAnonLimit: 5, SearchUserLimit: 60, BatchAnonLimit: 5, BatchUserLimit: 60},
		SMS: SMSConfig{
			Enabled:     true,
			SecretID:    "sms-secret-id",
			SecretKey:   "sms-secret-key",
			AppID:       "sms-app-id",
			SignName:    "StuHelper",
			TemplateID:  "sms-template-id",
			Region:      "ap-beijing",
			InternalKey: "sms-internal-key",
		},
		Bot: BotConfig{ServiceToken: "bot-service-token"},
	}
}
