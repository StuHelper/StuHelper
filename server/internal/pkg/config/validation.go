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

	if c.App.Env == "production" {
		if c.Database.URL == "" {
			errs = append(errs, "DATABASE_URL is required in production")
		}
		if !c.Token.CookieSecure {
			errs = append(errs, "TOKEN_COOKIE_SECURE must be true in production")
		}
		if len(c.App.CORSOrigins) == 0 {
			errs = append(errs, "CORS_ORIGINS is required in production")
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

		switch c.Database.SSLMode {
		case "", "disable":
			errs = append(errs, "DB_SSL_MODE must be 'require', 'verify-ca', or 'verify-full' in production")
		case "require":
			fmt.Fprintf(os.Stderr, "WARNING: DB_SSL_MODE=require skips certificate verification (MITM risk). Consider 'verify-ca' or 'verify-full' for production.\n")
		}
		if !c.Redis.TLSEnabled {
			fmt.Fprintf(os.Stderr, "WARNING: Redis TLS is disabled in production. Ensure Redis is in a secure internal network.\n")
		}
		if c.Redis.TLSEnabled && c.Redis.TLSInsecure {
			errs = append(errs, "REDIS_TLS_INSECURE must not be true in production (certificate verification required)")
		}
		if len(parseErrs) > 0 {
			errs = append(errs, parseErrs...)
		}
	} else if len(parseErrs) > 0 {
		fmt.Fprintf(os.Stderr, "WARNING: %d config parse error(s) detected (using defaults): %s\n",
			len(parseErrs), strings.Join(parseErrs, "; "))
	}

	if c.Casdoor.Endpoint == "" {
		errs = append(errs, "CASDOOR_ENDPOINT is required")
	}
	if c.Casdoor.ClientID == "" {
		errs = append(errs, "CASDOOR_CLIENT_ID is required")
	}
	if c.Casdoor.ClientSecret == "" {
		errs = append(errs, "CASDOOR_CLIENT_SECRET is required")
	}
	if c.Casdoor.Certificate == "" {
		errs = append(errs, "CASDOOR_CERTIFICATE is required (set CASDOOR_CERTIFICATE or CASDOOR_CERTIFICATE_FILE)")
	} else if err := validatePEMCertificate(c.Casdoor.Certificate); err != nil {
		errs = append(errs, fmt.Sprintf("CASDOOR_CERTIFICATE format invalid: %v", err))
	}
	if c.Casdoor.Organization == "" {
		errs = append(errs, "CASDOOR_ORGANIZATION is required")
	}
	if c.Casdoor.Application == "" {
		errs = append(errs, "CASDOOR_APPLICATION is required")
	}
	if c.Casdoor.RedirectURI == "" {
		errs = append(errs, "CASDOOR_REDIRECT_URI is required")
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

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
	}

	return nil
}
