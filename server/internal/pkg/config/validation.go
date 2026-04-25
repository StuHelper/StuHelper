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
		if !c.Observability.Enabled {
			errs = append(errs, "OTEL_ENABLED must be true in production")
		}
		if c.Observability.OTLPEndpoint == "" {
			errs = append(errs, "OTEL_EXPORTER_OTLP_ENDPOINT is required in production")
		}

		if c.Database.SSLMode != "verify-full" {
			errs = append(errs, "DB_SSL_MODE must be 'verify-full' in production")
		}
		if c.Database.SSLRootCert == "" {
			errs = append(errs, "DB_SSL_ROOT_CERT is required in production")
		}
		if !c.Redis.TLSEnabled {
			errs = append(errs, "REDIS_TLS_ENABLED must be true in production")
		}
		if c.Redis.TLSCAFile == "" {
			errs = append(errs, "REDIS_TLS_CA is required in production")
		}
		if !c.ObjectStorage.UseSSL {
			errs = append(errs, "OBJECT_STORAGE_USE_SSL must be true in production")
		}
	}

	if len(parseErrs) > 0 {
		errs = append(errs, parseErrs...)
	}

	if c.App.Env != "production" && c.App.Env != "development" && !c.Token.CookieSecure {
		errs = append(errs, "TOKEN_COOKIE_SECURE can only be false in development")
	}

	// Zitadel OIDC 配置校验
	if c.Zitadel.Issuer == "" {
		errs = append(errs, "ZITADEL_ISSUER is required")
	}
	if c.Zitadel.ClientID == "" {
		errs = append(errs, "ZITADEL_CLIENT_ID is required")
	}
	if c.Zitadel.ClientSecret == "" {
		errs = append(errs, "ZITADEL_CLIENT_SECRET is required")
	}
	if c.Zitadel.RedirectURI == "" {
		errs = append(errs, "ZITADEL_REDIRECT_URI is required")
	}
	if c.Zitadel.ProjectID == "" {
		errs = append(errs, "ZITADEL_PROJECT_ID is required")
	}

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

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
	}

	return nil
}
