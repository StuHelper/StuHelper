package config

import (
	"fmt"
	"os"
	"strings"
)

// validate 验证配置是否完整（parseErrs 为 Load 阶段收集的解析错误）
func (c *Config) validate(parseErrs []string) error {
	var errs []string

	const hmacMinLen = 32
	switch {
	case c.App.HMACSecret == "":
		if c.App.Env == "production" {
			errs = append(errs, "HMAC_SECRET is required in production")
		}
	case len(c.App.HMACSecret) < hmacMinLen:
		if c.App.Env == "production" {
			errs = append(errs, fmt.Sprintf("HMAC_SECRET must be at least %d characters in production (got %d)", hmacMinLen, len(c.App.HMACSecret)))
		} else {
			fmt.Fprintf(os.Stderr, "WARNING: HMAC_SECRET is shorter than %d characters (%d), consider using a stronger secret\n", hmacMinLen, len(c.App.HMACSecret))
		}
	}

	if len(c.Security.DocAESKeys) == 0 {
		errs = append(errs, "DOC_AES_KEYS is required (PII encryption keys must be configured in all environments)")
	}
	if c.Security.DocAESActiveKeyID == 0 && len(c.Security.DocAESKeys) > 0 {
		errs = append(errs, "DOC_AES_ACTIVE_KEY_ID is required")
	}
	if len(c.Security.DocAESKeys) > 0 && c.Security.DocAESActiveKeyID > 0 {
		if _, ok := c.Security.DocAESKeys[c.Security.DocAESActiveKeyID]; !ok {
			errs = append(errs, fmt.Sprintf("DOC_AES_ACTIVE_KEY_ID=%d not found in DOC_AES_KEYS", c.Security.DocAESActiveKeyID))
		}
	}

	if c.Database.QueryTimeout < 1 || c.Database.QueryTimeout > 60 {
		errs = append(errs, fmt.Sprintf("DB_QUERY_TIMEOUT must be between 1 and 60 seconds (got %d)", c.Database.QueryTimeout))
	}
	if c.Database.MaxConns <= 0 {
		errs = append(errs, fmt.Sprintf("DB_MAX_CONNS must be greater than 0 (got %d)", c.Database.MaxConns))
	} else if c.Database.MaxConns > 10000 {
		errs = append(errs, fmt.Sprintf("DB_MAX_CONNS must not exceed 10000 (got %d)", c.Database.MaxConns))
	}
	if c.Database.MinConns < 0 {
		errs = append(errs, fmt.Sprintf("DB_MIN_CONNS must be >= 0 (got %d)", c.Database.MinConns))
	} else if c.Database.MinConns > 10000 {
		errs = append(errs, fmt.Sprintf("DB_MIN_CONNS must not exceed 10000 (got %d)", c.Database.MinConns))
	}
	if c.Database.MaxConns > 0 && c.Database.MinConns > c.Database.MaxConns {
		errs = append(errs, fmt.Sprintf("DB_MIN_CONNS (%d) must not exceed DB_MAX_CONNS (%d)", c.Database.MinConns, c.Database.MaxConns))
	}

	if c.Token.AccessTokenTTL < 60 || c.Token.AccessTokenTTL > 86400 {
		errs = append(errs, fmt.Sprintf("TOKEN_ACCESS_TTL must be between 60 and 86400 seconds (got %d)", c.Token.AccessTokenTTL))
	}
	if c.Token.RefreshTokenTTL < 3600 || c.Token.RefreshTokenTTL > 2592000 {
		errs = append(errs, fmt.Sprintf("TOKEN_REFRESH_TTL must be between 3600 and 2592000 seconds (got %d)", c.Token.RefreshTokenTTL))
	}
	if c.Identity.AccessTokenTTL < 60 || c.Identity.AccessTokenTTL > 86400 {
		errs = append(errs, fmt.Sprintf("IDENTITY_ACCESS_TOKEN_TTL must be between 60 and 86400 seconds (got %d)", c.Identity.AccessTokenTTL))
	}
	if c.Identity.RefreshTokenTTL < 3600 || c.Identity.RefreshTokenTTL > 2592000 {
		errs = append(errs, fmt.Sprintf("IDENTITY_REFRESH_TOKEN_TTL must be between 3600 and 2592000 seconds (got %d)", c.Identity.RefreshTokenTTL))
	}
	if c.Identity.AuthorizationCodeTTL < 60 || c.Identity.AuthorizationCodeTTL > 600 {
		errs = append(errs, fmt.Sprintf("IDENTITY_AUTH_CODE_TTL must be between 60 and 600 seconds (got %d)", c.Identity.AuthorizationCodeTTL))
	}

	if c.Observability.TraceSampleRatio < 0 || c.Observability.TraceSampleRatio > 1 {
		errs = append(errs, fmt.Sprintf("OTEL_TRACE_SAMPLE_RATIO must be between 0 and 1 (got %.4f)", c.Observability.TraceSampleRatio))
	}
	if c.ObjectStorage.PresignTTL < 60 || c.ObjectStorage.PresignTTL > 86400 {
		errs = append(errs, fmt.Sprintf("OBJECT_STORAGE_PRESIGN_TTL must be between 60 and 86400 seconds (got %d)", c.ObjectStorage.PresignTTL))
	}
	if c.Observability.Enabled && c.Observability.ServiceName == "" {
		errs = append(errs, "OTEL_SERVICE_NAME is required when OTEL_ENABLED=true")
	}
	if c.Observability.Enabled && c.Observability.OTLPEndpoint == "" {
		errs = append(errs, "OTEL_EXPORTER_OTLP_ENDPOINT is required when OTEL_ENABLED=true")
	}
	if len(c.App.CORSOrigins) == 0 {
		errs = append(errs, "CORS_ORIGINS is required")
	}

	if c.SMS.Enabled {
		if c.SMS.SecretID == "" {
			errs = append(errs, "SMS_SECRET_ID is required when SMS_ENABLED=true")
		}
		if c.SMS.SecretKey == "" {
			errs = append(errs, "SMS_SECRET_KEY is required when SMS_ENABLED=true")
		}
		if c.SMS.AppID == "" {
			errs = append(errs, "SMS_APP_ID is required when SMS_ENABLED=true")
		}
		if c.SMS.SignName == "" {
			errs = append(errs, "SMS_SIGN_NAME is required when SMS_ENABLED=true")
		}
		if c.SMS.TemplateID == "" {
			errs = append(errs, "SMS_TEMPLATE_ID is required when SMS_ENABLED=true")
		}
		if c.SMS.InternalKey == "" {
			errs = append(errs, "SMS_INTERNAL_KEY is required when SMS_ENABLED=true")
		}
	}

	if c.App.Env == "production" {
		if c.Database.URL == "" {
			errs = append(errs, "DATABASE_URL is required in production")
		}
		if !c.Token.CookieSecure {
			errs = append(errs, "TOKEN_COOKIE_SECURE must be true in production")
		}
		if len(c.App.TrustedProxies) == 0 {
			errs = append(errs, "TRUSTED_PROXIES is required in production for secure IP detection")
		}
		if c.App.MetricsPassword == "" {
			errs = append(errs, "METRICS_PASSWORD is required in production")
		}
		if c.Redis.Password == "" {
			errs = append(errs, "REDIS_PASSWORD is required in production")
		}
		if c.ObjectStorage.Endpoint == "" {
			errs = append(errs, "OBJECT_STORAGE_ENDPOINT is required in production")
		}
		if c.ObjectStorage.Bucket == "" {
			errs = append(errs, "OBJECT_STORAGE_BUCKET is required in production")
		}
		if c.ObjectStorage.AccessKeyID == "" {
			errs = append(errs, "OBJECT_STORAGE_ACCESS_KEY_ID is required in production")
		}
		if c.ObjectStorage.SecretAccessKey == "" {
			errs = append(errs, "OBJECT_STORAGE_SECRET_ACCESS_KEY is required in production")
		}
		if c.Bot.ServiceToken == "" {
			errs = append(errs, "BOT_SERVICE_TOKEN is required in production")
		}
		if !c.SMS.Enabled {
			errs = append(errs, "SMS_ENABLED must be true in production")
		}
		if !c.Observability.Enabled {
			errs = append(errs, "OTEL_ENABLED must be true in production")
		}
		if c.Observability.OTLPEndpoint == "" {
			errs = append(errs, "OTEL_EXPORTER_OTLP_ENDPOINT is required in production")
		}

		plaintextPostgresAllowed := c.Database.AllowPlaintext && c.Database.SSLMode == "disable"
		if c.Database.SSLMode != "verify-full" && !plaintextPostgresAllowed {
			errs = append(errs, "DB_SSL_MODE must be 'verify-full' in production")
		}
		if c.Database.SSLMode != "disable" && c.Database.SSLRootCert == "" {
			errs = append(errs, "DB_SSL_ROOT_CERT is required in production")
		}
		if !c.Redis.TLSEnabled {
			errs = append(errs, "REDIS_TLS_ENABLED must be true in production")
		}
		if c.Redis.TLSEnabled && c.Redis.TLSCAFile == "" {
			errs = append(errs, "REDIS_TLS_CA is required in production")
		}
		if !c.ObjectStorage.UseSSL {
			errs = append(errs, "OBJECT_STORAGE_USE_SSL must be true in production")
		}
		if c.Identity.Issuer == "" {
			errs = append(errs, "IDENTITY_ISSUER is required in production")
		}
		if c.Identity.SigningPrivateKeyPEM == "" {
			errs = append(errs, "IDENTITY_SIGNING_PRIVATE_KEY_PEM is required in production")
		}
	}

	if len(parseErrs) > 0 {
		errs = append(errs, parseErrs...)
	}

	if c.App.Env != "production" && c.App.Env != "development" && !c.Token.CookieSecure {
		errs = append(errs, "TOKEN_COOKIE_SECURE can only be false in development")
	}

	// Casdoor OIDC 配置校验
	if c.Casdoor.Issuer == "" {
		errs = append(errs, "CASDOOR_ISSUER is required")
	}
	if c.Casdoor.ClientID == "" {
		errs = append(errs, "CASDOOR_CLIENT_ID is required")
	}
	if c.Casdoor.ClientSecret == "" {
		errs = append(errs, "CASDOOR_CLIENT_SECRET is required")
	}
	if c.Casdoor.RedirectURI == "" {
		errs = append(errs, "CASDOOR_REDIRECT_URI is required")
	}
	if c.Casdoor.IntrospectionClientID == "" {
		errs = append(errs, "CASDOOR_INTROSPECTION_CLIENT_ID is required")
	}
	if c.Casdoor.IntrospectionClientSecret == "" {
		errs = append(errs, "CASDOOR_INTROSPECTION_CLIENT_SECRET is required")
	}
	if c.Casdoor.Organization == "" {
		errs = append(errs, "CASDOOR_ORGANIZATION is required")
	}
	errs = append(errs, validateCasdoorAdminCredentials(c.Casdoor, c.App.Env == "production")...)

	// OpenFGA 是应用运行时必需依赖，所有环境都需要完整配置。
	if c.OpenFGA.StoreID == "" {
		errs = append(errs, "OPENFGA_STORE_ID is required")
	}
	if c.OpenFGA.AuthorizationModelID == "" {
		errs = append(errs, "OPENFGA_MODEL_ID is required")
	}
	if c.OpenFGA.APIUrl == "" {
		errs = append(errs, "OPENFGA_API_URL is required")
	}

	const maxRateLimit = 100000
	if c.RateLimit.PostLimit <= 0 || c.RateLimit.PostLimit > maxRateLimit {
		errs = append(errs, fmt.Sprintf("REVIEW_RATE_POST_LIMIT must be between 1 and %d (got %d)", maxRateLimit, c.RateLimit.PostLimit))
	}
	if c.RateLimit.VoteLimit <= 0 || c.RateLimit.VoteLimit > maxRateLimit {
		errs = append(errs, fmt.Sprintf("REVIEW_RATE_VOTE_LIMIT must be between 1 and %d (got %d)", maxRateLimit, c.RateLimit.VoteLimit))
	}
	if c.RateLimit.ReportLimit <= 0 || c.RateLimit.ReportLimit > maxRateLimit {
		errs = append(errs, fmt.Sprintf("REVIEW_RATE_REPORT_LIMIT must be between 1 and %d (got %d)", maxRateLimit, c.RateLimit.ReportLimit))
	}
	if c.RateLimit.ReplyLimit <= 0 || c.RateLimit.ReplyLimit > maxRateLimit {
		errs = append(errs, fmt.Sprintf("REVIEW_RATE_REPLY_LIMIT must be between 1 and %d (got %d)", maxRateLimit, c.RateLimit.ReplyLimit))
	}
	if c.RateLimit.WriteLimit <= 0 || c.RateLimit.WriteLimit > maxRateLimit {
		errs = append(errs, fmt.Sprintf("REVIEW_RATE_WRITE_LIMIT must be between 1 and %d (got %d)", maxRateLimit, c.RateLimit.WriteLimit))
	}
	if c.RateLimit.SearchAnonLimit <= 0 || c.RateLimit.SearchAnonLimit > maxRateLimit {
		errs = append(errs, fmt.Sprintf("REVIEW_RATE_SEARCH_ANON_LIMIT must be between 1 and %d (got %d)", maxRateLimit, c.RateLimit.SearchAnonLimit))
	}
	if c.RateLimit.SearchUserLimit <= 0 || c.RateLimit.SearchUserLimit > maxRateLimit {
		errs = append(errs, fmt.Sprintf("REVIEW_RATE_SEARCH_USER_LIMIT must be between 1 and %d (got %d)", maxRateLimit, c.RateLimit.SearchUserLimit))
	}
	if c.RateLimit.BatchAnonLimit <= 0 || c.RateLimit.BatchAnonLimit > maxRateLimit {
		errs = append(errs, fmt.Sprintf("REVIEW_RATE_BATCH_ANON_LIMIT must be between 1 and %d (got %d)", maxRateLimit, c.RateLimit.BatchAnonLimit))
	}
	if c.RateLimit.BatchUserLimit <= 0 || c.RateLimit.BatchUserLimit > maxRateLimit {
		errs = append(errs, fmt.Sprintf("REVIEW_RATE_BATCH_USER_LIMIT must be between 1 and %d (got %d)", maxRateLimit, c.RateLimit.BatchUserLimit))
	}
	errs = append(errs, validateOpenPlatformDisclosureRateLimits(c.OpenPlatform.DisclosureRateLimit)...)
	errs = append(errs, validateOpenPlatformTokenProbe(c.OpenPlatform.TokenProbe, c.App.Env == "production")...)

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
	}

	return nil
}

func validateOpenPlatformTokenProbe(cfg OpenPlatformTokenProbeConfig, production bool) []string {
	var errs []string
	if production && !cfg.RuntimeRequired {
		errs = append(errs, "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED must be true in production")
	}
	if cfg.RuntimeRequired && strings.TrimSpace(cfg.RuntimeCommand) == "" {
		errs = append(errs, "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND is required when OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED=true")
	}
	if cfg.RuntimeTimeoutSeconds != 0 && (cfg.RuntimeTimeoutSeconds < 1 || cfg.RuntimeTimeoutSeconds > 600) {
		errs = append(errs, fmt.Sprintf("OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_TIMEOUT_SECONDS must be between 1 and 600 seconds (got %d)", cfg.RuntimeTimeoutSeconds))
	}
	return errs
}

func validateOpenPlatformDisclosureRateLimits(cfg OpenPlatformDisclosureRateLimitConfig) []string {
	var errs []string
	const maxRateLimit = 100000
	errs = append(errs,
		validateOptionalLimit("OPEN_PLATFORM_DISCLOSURE_APP_LIMIT", cfg.AppLimit, maxRateLimit)...)
	errs = append(errs,
		validateOptionalLimit("OPEN_PLATFORM_DISCLOSURE_APP_USER_LIMIT", cfg.AppUserLimit, maxRateLimit)...)
	errs = append(errs,
		validateOptionalLimit("OPEN_PLATFORM_DISCLOSURE_ENDPOINT_LIMIT", cfg.EndpointLimit, maxRateLimit)...)
	errs = append(errs,
		validateOptionalLimit("OPEN_PLATFORM_DISCLOSURE_CONSENT_LIMIT", cfg.ConsentLimit, maxRateLimit)...)
	errs = append(errs,
		validateOptionalLimit("OPEN_PLATFORM_DISCLOSURE_REPLAY_LIMIT", cfg.ReplayLimit, maxRateLimit)...)

	const maxWindowSeconds = 86400
	errs = append(errs,
		validateOptionalDurationSeconds("OPEN_PLATFORM_DISCLOSURE_WINDOW_SECONDS", cfg.WindowSeconds, maxWindowSeconds)...)
	errs = append(errs,
		validateOptionalDurationSeconds("OPEN_PLATFORM_DISCLOSURE_REPLAY_WINDOW_SECONDS", cfg.ReplayWindowSeconds, maxWindowSeconds)...)
	errs = append(errs,
		validateOptionalDurationSeconds("OPEN_PLATFORM_DISCLOSURE_REPLAY_AUDIT_COOLDOWN_SECONDS", cfg.ReplayAuditCooldownSeconds, maxWindowSeconds)...)
	return errs
}

func validateOptionalLimit(name string, value int, limit int) []string {
	if value == 0 {
		return nil
	}
	if value < 0 || value > limit {
		return []string{fmt.Sprintf("%s must be between 1 and %d when set (got %d)", name, limit, value)}
	}
	return nil
}

func validateOptionalDurationSeconds(name string, value int, limit int) []string {
	if value == 0 {
		return nil
	}
	if value < 0 || value > limit {
		return []string{fmt.Sprintf("%s must be between 1 and %d seconds when set (got %d)", name, limit, value)}
	}
	return nil
}

func validateCasdoorAdminCredentials(cfg CasdoorConfig, required bool) []string {
	var errs []string
	errs = append(errs, validateCasdoorCredentialSet(required || appProvisioningCredentialConfigured(cfg),
		"APP_PROVISIONING", cfg.AppProvisioningClientID, cfg.AppProvisioningClientSecret, cfg.AppProvisioningApplication)...)
	errs = append(errs, validateCasdoorCredentialSet(required || userProfileCredentialConfigured(cfg),
		"USER_PROFILE", cfg.UserProfileClientID, cfg.UserProfileClientSecret, cfg.UserProfileApplication)...)
	errs = append(errs, validateCasdoorCredentialSet(required || roleSyncCredentialConfigured(cfg),
		"ROLE_SYNC", cfg.RoleSyncClientID, cfg.RoleSyncClientSecret, cfg.RoleSyncApplication)...)
	errs = append(errs, validateCasdoorCredentialSet(required || userLookupCredentialConfigured(cfg),
		"USER_LOOKUP", cfg.UserLookupClientID, cfg.UserLookupClientSecret, cfg.UserLookupApplication)...)
	return errs
}

func appProvisioningCredentialConfigured(cfg CasdoorConfig) bool {
	return cfg.AppProvisioningClientID != "" || cfg.AppProvisioningClientSecret != "" || cfg.AppProvisioningApplication != ""
}

func userProfileCredentialConfigured(cfg CasdoorConfig) bool {
	return cfg.UserProfileClientID != "" || cfg.UserProfileClientSecret != "" || cfg.UserProfileApplication != ""
}

func roleSyncCredentialConfigured(cfg CasdoorConfig) bool {
	return cfg.RoleSyncClientID != "" || cfg.RoleSyncClientSecret != "" || cfg.RoleSyncApplication != ""
}

func userLookupCredentialConfigured(cfg CasdoorConfig) bool {
	return cfg.UserLookupClientID != "" || cfg.UserLookupClientSecret != "" || cfg.UserLookupApplication != ""
}

func validateCasdoorCredentialSet(required bool, prefix, clientID, clientSecret, application string) []string {
	if !required {
		return nil
	}
	var errs []string
	if clientID == "" {
		errs = append(errs, "CASDOOR_"+prefix+"_CLIENT_ID is required")
	}
	if clientSecret == "" {
		errs = append(errs, "CASDOOR_"+prefix+"_CLIENT_SECRET is required")
	}
	if application == "" {
		errs = append(errs, "CASDOOR_"+prefix+"_APPLICATION is required")
	}
	return errs
}
