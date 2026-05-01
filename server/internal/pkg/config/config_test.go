package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDocAESKeys_Valid(t *testing.T) {
	// 有效的 32 字节 hex key
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	keys, errs := parseDocAESKeys("1:" + hexKey)
	assert.Empty(t, errs)
	assert.Len(t, keys, 1)
	assert.Len(t, keys[1], 32)
}

func TestParseDocAESKeys_MultipleKeys(t *testing.T) {
	hexKey1 := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	hexKey2 := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	keys, errs := parseDocAESKeys("1:" + hexKey1 + ",2:" + hexKey2)
	assert.Empty(t, errs)
	assert.Len(t, keys, 2)
}

func TestParseDocAESKeys_InvalidFormat(t *testing.T) {
	_, errs := parseDocAESKeys("invalid-no-colon")
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "invalid entry")
}

func TestParseDocAESKeys_InvalidKeyID(t *testing.T) {
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tests := []struct {
		name  string
		input string
	}{
		{"零值", "0:" + hexKey},
		{"超范围", "256:" + hexKey},
		{"非数字", "abc:" + hexKey},
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
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	_, errs := parseDocAESKeys("1:" + hexKey + ",1:" + hexKey)
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
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Setenv("DOC_AES_ACTIVE_KEY_ID", "2")
	t.Setenv("DOC_AES_KEYS", "1:"+hexKey)

	cfg, errs := parseSecurityConfig()
	// parseSecurityConfig 本身不校验 activeKeyID 是否在 keys 中，
	// 由 validate() 统一校验
	assert.Empty(t, errs)
	assert.Equal(t, uint8(2), cfg.DocAESActiveKeyID)
}

func TestValidate_SecurityConfig_ActiveKeyMissing(t *testing.T) {
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Setenv("DOC_AES_ACTIVE_KEY_ID", "2")
	t.Setenv("DOC_AES_KEYS", "1:"+hexKey)

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
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Setenv("DOC_AES_ACTIVE_KEY_ID", "1")
	t.Setenv("DOC_AES_KEYS", "1:"+hexKey)

	cfg, errs := parseSecurityConfig()
	assert.Empty(t, errs)
	assert.Equal(t, uint8(1), cfg.DocAESActiveKeyID)
	assert.Len(t, cfg.DocAESKeys, 1)
	assert.Len(t, cfg.DocAESKeys[1], 32)
}

func TestValidate_ProductionRequiresObservability(t *testing.T) {
	c := &Config{
		App: AppConfig{
			Env:             "production",
			HMACSecret:      "0123456789abcdef0123456789abcdef",
			CORSOrigins:     []string{"https://stuhelper.example.com"},
			TrustedProxies:  []string{"10.0.0.0/8"},
			MetricsPassword: "metrics-password",
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
			Issuer:       "https://sso.example.com",
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RedirectURI:  "https://api.example.com/api/v1/auth/callback",
			Organization: "stuhelper",
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

func TestValidate_ProductionRequiresBotServiceToken(t *testing.T) {
	c := validProductionConfigForTest()
	c.Bot.ServiceToken = ""

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "BOT_SERVICE_TOKEN is required in production")
}

func TestValidate_DevelopmentAllowsMissingBotServiceToken(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.Env = "development"
	c.Token.CookieSecure = false
	c.Bot.ServiceToken = ""

	err := c.validate(nil)

	require.NoError(t, err)
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

func TestValidate_SMSRequiresFullConfigWhenEnabled(t *testing.T) {
	c := &Config{
		App: AppConfig{
			Env:             "development",
			HMACSecret:      "0123456789abcdef0123456789abcdef",
			CORSOrigins:     []string{"http://localhost:3000"},
			MetricsPassword: "metrics-password",
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
			Issuer:       "https://sso.example.com",
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RedirectURI:  "https://api.example.com/api/v1/auth/callback",
			Organization: "stuhelper",
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

func TestValidate_SMSDisabledAllowsEmptyConfig(t *testing.T) {
	c := &Config{
		App: AppConfig{
			Env:             "development",
			HMACSecret:      "0123456789abcdef0123456789abcdef",
			CORSOrigins:     []string{"http://localhost:3000"},
			MetricsPassword: "metrics-password",
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
			Issuer:       "https://sso.example.com",
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RedirectURI:  "https://api.example.com/api/v1/auth/callback",
			Organization: "stuhelper",
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

func TestValidate_RequiresCORSOriginsInDevelopment(t *testing.T) {
	c := &Config{
		App: AppConfig{
			Env:             "development",
			HMACSecret:      "0123456789abcdef0123456789abcdef",
			MetricsPassword: "metrics-password",
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
			Issuer:       "https://sso.example.com",
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RedirectURI:  "https://api.example.com/api/v1/auth/callback",
			Organization: "stuhelper",
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

func TestValidate_RequiresOpenFGAInDevelopment(t *testing.T) {
	c := &Config{
		App: AppConfig{
			Env:             "development",
			HMACSecret:      "0123456789abcdef0123456789abcdef",
			CORSOrigins:     []string{"http://localhost:3000"},
			MetricsPassword: "metrics-password",
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
			Issuer:       "https://sso.example.com",
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RedirectURI:  "https://api.example.com/api/v1/auth/callback",
			Organization: "stuhelper",
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

func validProductionConfigForTest() *Config {
	return &Config{
		App: AppConfig{
			Env:             "production",
			HMACSecret:      "0123456789abcdef0123456789abcdef",
			CORSOrigins:     []string{"https://stuhelper.example.com"},
			TrustedProxies:  []string{"10.0.0.0/8"},
			MetricsPassword: "metrics-password",
		},
		Security: SecurityConfig{DocAESActiveKeyID: 1, DocAESKeys: map[uint8][]byte{1: make([]byte, 32)}},
		Database: DatabaseConfig{URL: "postgres://user:pass@db:5432/stuhelper?sslmode=verify-full", QueryTimeout: 5, MaxConns: 20, MinConns: 2, SSLMode: "verify-full", SSLRootCert: "/run/secrets/postgres-ca.crt"},
		Token:    TokenConfig{AccessTokenTTL: 300, RefreshTokenTTL: 604800, CookieSecure: true},
		Redis:    RedisConfig{Password: "redis-password", TLSEnabled: true, TLSCAFile: "/run/secrets/redis-ca.crt"},
		ObjectStorage: ObjectStorageConfig{
			Endpoint: "https://s3.example.com", Bucket: "stuhelper", AccessKeyID: "access", SecretAccessKey: "secret", UseSSL: true, PresignTTL: 600,
		},
		Casdoor: CasdoorConfig{Issuer: "https://sso.example.com", ClientID: "client-id", ClientSecret: "client-secret", RedirectURI: "https://api.example.com/api/v1/auth/callback", Organization: "stuhelper"},
		OpenFGA: OpenFGAConfig{StoreID: "store-id", AuthorizationModelID: "model-id", APIUrl: "http://openfga:8080"},
		Observability: ObservabilityConfig{
			Enabled: true, ServiceName: "stuhelper-backend", OTLPEndpoint: "http://alloy:4318", TraceSampleRatio: 0.2,
		},
		RateLimit: ReviewRateLimitConfig{PostLimit: 5, VoteLimit: 30, ReportLimit: 10, ReplyLimit: 10, WriteLimit: 10, SearchAnonLimit: 5, SearchUserLimit: 60, BatchAnonLimit: 5, BatchUserLimit: 60},
		SMS:       SMSConfig{Enabled: false},
		Bot:       BotConfig{ServiceToken: "bot-service-token"},
	}
}
