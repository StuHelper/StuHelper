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

func TestValidate_ProductionAllowsExplicitExternalPlaintextPostgres(t *testing.T) {
	c := validProductionConfigForTest()
	c.Database.SSLMode = "disable"
	c.Database.SSLRootCert = ""
	c.Database.AllowPlaintext = true

	err := c.validate(nil)

	require.NoError(t, err)
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

func TestLoadExternalDataConfigFromEnv(t *testing.T) {
	t.Setenv("EXTERNAL_STUDENT_SOURCE_ENABLED", "true")
	t.Setenv("EXTERNAL_STUDENT_SOURCE_PROVIDER", "oracle")
	t.Setenv("EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE", "4111010006")
	t.Setenv("EXTERNAL_STUDENT_SOURCE_ORACLE_HOST", "oracle.example.test")
	t.Setenv("EXTERNAL_STUDENT_SOURCE_ORACLE_SERVICE_NAME", "ORCLPDB1")
	t.Setenv("EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME", "SYSTEM")
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
	assert.Equal(t, "ORCLPDB1", source.Oracle.ServiceName)
	assert.Equal(t, "USR_JWBIZ", source.Oracle.Schema)
	assert.Equal(t, "T_XS_JBXX", source.Oracle.Table)
	assert.Equal(t, "XH", source.Oracle.StudentIDColumn)
	assert.Equal(t, "XM", source.Oracle.StudentNameColumn)
}

func TestValidateRejectsIncompleteExternalStudentSource(t *testing.T) {
	c := validProductionConfigForTest()
	c.ExternalData.StudentSources = []ExternalStudentSourceConfig{{
		Name:       "buaa",
		Enabled:    true,
		Provider:   "oracle",
		SchoolCode: "4111010006",
		Oracle: ExternalOracleStudentSourceConfig{
			Host:                  "oracle.example.test",
			Port:                  1521,
			ServiceName:           "ORCLPDB1",
			Username:              "SYSTEM",
			Schema:                "USR_JWBIZ",
			Table:                 "T_XS_JBXX",
			StudentIDColumn:       "XH",
			StudentNameColumn:     "XM",
			ConnectTimeoutSeconds: 5,
			QueryTimeoutSeconds:   3,
			MaxOpenConns:          4,
			MaxIdleConns:          1,
		},
	}}

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD is required")
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
	c.Casdoor.RoleSyncClientSecret = ""
	c.Casdoor.UserLookupApplication = ""

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CASDOOR_APP_PROVISIONING_CLIENT_ID is required")
	assert.Contains(t, err.Error(), "CASDOOR_USER_PROFILE_CLIENT_SECRET is required")
	assert.Contains(t, err.Error(), "CASDOOR_ROLE_SYNC_CLIENT_SECRET is required")
	assert.Contains(t, err.Error(), "CASDOOR_USER_LOOKUP_APPLICATION is required")
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

func TestValidate_DevelopmentRejectsPartialCasdoorRoleSyncCredential(t *testing.T) {
	c := validProductionConfigForTest()
	c.App.Env = "development"
	c.Token.CookieSecure = false
	c.Casdoor.RoleSyncClientID = "role-sync-client"
	c.Casdoor.RoleSyncClientSecret = ""
	c.Casdoor.RoleSyncApplication = ""

	err := c.validate(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CASDOOR_ROLE_SYNC_CLIENT_SECRET is required")
	assert.Contains(t, err.Error(), "CASDOOR_ROLE_SYNC_APPLICATION is required")
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
		Casdoor: CasdoorConfig{
			Issuer:                      "https://sso.example.com",
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
			RoleSyncClientID:            "role-sync-client",
			RoleSyncClientSecret:        "role-sync-secret",
			RoleSyncApplication:         "stuhelper-role-sync",
			UserLookupClientID:          "user-lookup-client",
			UserLookupClientSecret:      "user-lookup-secret",
			UserLookupApplication:       "stuhelper-user-lookup",
		},
		OpenFGA: OpenFGAConfig{StoreID: "store-id", AuthorizationModelID: "model-id", APIUrl: "http://openfga:8080"},
		OpenPlatform: OpenPlatformConfig{TokenProbe: OpenPlatformTokenProbeConfig{
			RuntimeRequired:       true,
			RuntimeCommand:        "/usr/local/bin/stuhelper-token-probe",
			RuntimeTimeoutSeconds: 30,
		}},
		Observability: ObservabilityConfig{
			Enabled: true, ServiceName: "stuhelper-backend", OTLPEndpoint: "http://alloy:4318", TraceSampleRatio: 0.2,
		},
		Admission: AdmissionConfig{PublicBaseURL: "https://join.stuhelper.com"},
		RateLimit: ReviewRateLimitConfig{PostLimit: 5, VoteLimit: 30, ReportLimit: 10, ReplyLimit: 10, WriteLimit: 10, SearchAnonLimit: 5, SearchUserLimit: 60, BatchAnonLimit: 5, BatchUserLimit: 60},
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
