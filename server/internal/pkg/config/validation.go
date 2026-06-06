package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// validate 验证配置是否完整（parseErrs 为 Load 阶段收集的解析错误）
func (c *Config) validate(parseErrs []string) error {
	var errs []string

	productionLike := IsProductionLikeEnv(c.App.Env)
	errs = append(errs, validateAppEnv(c.App.Env)...)
	errs = append(errs, validateAppPort(c.App.Port)...)
	errs = append(errs, validateMaxBodySize(c.App.MaxBodySize)...)
	errs = append(errs, validateHealthCheckTimeout(c.App.HealthCheckTimeout)...)
	errs = append(errs, validateTrustedProxies(c.App.TrustedProxies)...)

	const hmacMinLen = 32
	switch {
	case configStringMissing(c.App.HMACSecret):
		if productionLike {
			errs = append(errs, "HMAC_SECRET is required in production")
		}
	case len(c.App.HMACSecret) < hmacMinLen:
		if productionLike {
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
	if c.Observability.Enabled && configStringMissing(c.Observability.ServiceName) {
		errs = append(errs, "OTEL_SERVICE_NAME is required when OTEL_ENABLED=true")
	}
	if c.Observability.Enabled && configStringMissing(c.Observability.OTLPEndpoint) {
		errs = append(errs, "OTEL_EXPORTER_OTLP_ENDPOINT is required when OTEL_ENABLED=true")
	}
	errs = append(errs, validateOptionalHTTPOrigins(
		"FRONTEND_METRICS_ALLOWED_ORIGINS",
		c.Observability.FrontendMetricsAllowedOrigins,
		productionLike,
	)...)
	errs = append(errs, validateCORSOrigins(c.App.CORSOrigins, productionLike)...)

	if c.SMS.Enabled {
		if configStringMissing(c.SMS.SecretID) {
			errs = append(errs, "SMS_SECRET_ID is required when SMS_ENABLED=true")
		}
		if configStringMissing(c.SMS.SecretKey) {
			errs = append(errs, "SMS_SECRET_KEY is required when SMS_ENABLED=true")
		}
		if configStringMissing(c.SMS.AppID) {
			errs = append(errs, "SMS_APP_ID is required when SMS_ENABLED=true")
		}
		if configStringMissing(c.SMS.SignName) {
			errs = append(errs, "SMS_SIGN_NAME is required when SMS_ENABLED=true")
		}
		if configStringMissing(c.SMS.TemplateID) {
			errs = append(errs, "SMS_TEMPLATE_ID is required when SMS_ENABLED=true")
		}
		if configStringMissing(c.SMS.InternalKey) {
			errs = append(errs, "SMS_INTERNAL_KEY is required when SMS_ENABLED=true")
		}
	}
	if c.Email.Enabled {
		driver := strings.TrimSpace(c.Email.Driver)
		if driver == "" {
			driver = "smtp"
		}
		switch driver {
		case "smtp":
			if configStringMissing(c.Email.SMTPHost) {
				errs = append(errs, "EMAIL_SMTP_HOST is required when EMAIL_ENABLED=true and EMAIL_DRIVER=smtp")
			}
			if c.Email.SMTPPort <= 0 || c.Email.SMTPPort > 65535 {
				errs = append(errs, fmt.Sprintf("EMAIL_SMTP_PORT must be between 1 and 65535 when EMAIL_ENABLED=true and EMAIL_DRIVER=smtp (got %d)", c.Email.SMTPPort))
			}
			if configStringMissing(c.Email.From) {
				errs = append(errs, "EMAIL_FROM is required when EMAIL_ENABLED=true and EMAIL_DRIVER=smtp")
			}
		case "blackhole":
			if c.App.Env == EnvProduction {
				errs = append(errs, "EMAIL_DRIVER=blackhole is only allowed outside production")
			}
		case "tencent_ses":
			if configStringMissing(c.Email.From) {
				errs = append(errs, "EMAIL_FROM is required when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses")
			}
			if configStringMissing(c.Email.TencentSecretID) {
				errs = append(errs, "EMAIL_TENCENT_SECRET_ID is required when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses")
			}
			if configStringMissing(c.Email.TencentSecretKey) {
				errs = append(errs, "EMAIL_TENCENT_SECRET_KEY is required when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses")
			}
			if configStringMissing(c.Email.TencentRegion) {
				errs = append(errs, "EMAIL_TENCENT_REGION is required when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses")
			}
			if configStringMissing(c.Email.TencentEndpoint) {
				errs = append(errs, "EMAIL_TENCENT_ENDPOINT is required when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses")
			}
			if c.Email.TencentTemplateID <= 0 {
				errs = append(errs, "EMAIL_TENCENT_TEMPLATE_ID must be greater than 0 when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses")
			}
			if c.Email.TencentTemplateExpireMinutes <= 0 {
				errs = append(errs, "EMAIL_TENCENT_TEMPLATE_EXPIRE_MINUTES must be greater than 0 when EMAIL_ENABLED=true and EMAIL_DRIVER=tencent_ses")
			}
		case "resend":
			if configStringMissing(c.Email.From) {
				errs = append(errs, "EMAIL_FROM is required when EMAIL_ENABLED=true and EMAIL_DRIVER=resend")
			}
			if configStringMissing(c.Email.ResendAPIKey) {
				errs = append(errs, "EMAIL_RESEND_API_KEY is required when EMAIL_ENABLED=true and EMAIL_DRIVER=resend")
			}
			if configStringMissing(c.Email.ResendEndpoint) {
				errs = append(errs, "EMAIL_RESEND_ENDPOINT is required when EMAIL_ENABLED=true and EMAIL_DRIVER=resend")
			}
		case "multi":
			if configStringMissing(c.Email.From) {
				errs = append(errs, "EMAIL_FROM is required when EMAIL_ENABLED=true and EMAIL_DRIVER=multi")
			}
			if configStringMissing(c.Email.TencentSecretID) {
				errs = append(errs, "EMAIL_TENCENT_SECRET_ID is required when EMAIL_ENABLED=true and EMAIL_DRIVER=multi")
			}
			if configStringMissing(c.Email.TencentSecretKey) {
				errs = append(errs, "EMAIL_TENCENT_SECRET_KEY is required when EMAIL_ENABLED=true and EMAIL_DRIVER=multi")
			}
			if configStringMissing(c.Email.TencentRegion) {
				errs = append(errs, "EMAIL_TENCENT_REGION is required when EMAIL_ENABLED=true and EMAIL_DRIVER=multi")
			}
			if configStringMissing(c.Email.TencentEndpoint) {
				errs = append(errs, "EMAIL_TENCENT_ENDPOINT is required when EMAIL_ENABLED=true and EMAIL_DRIVER=multi")
			}
			if c.Email.TencentTemplateID <= 0 {
				errs = append(errs, "EMAIL_TENCENT_TEMPLATE_ID must be greater than 0 when EMAIL_ENABLED=true and EMAIL_DRIVER=multi")
			}
			if c.Email.TencentTemplateExpireMinutes <= 0 {
				errs = append(errs, "EMAIL_TENCENT_TEMPLATE_EXPIRE_MINUTES must be greater than 0 when EMAIL_ENABLED=true and EMAIL_DRIVER=multi")
			}
			if configStringMissing(c.Email.ResendAPIKey) {
				errs = append(errs, "EMAIL_RESEND_API_KEY is required when EMAIL_ENABLED=true and EMAIL_DRIVER=multi")
			}
			if configStringMissing(c.Email.ResendEndpoint) {
				errs = append(errs, "EMAIL_RESEND_ENDPOINT is required when EMAIL_ENABLED=true and EMAIL_DRIVER=multi")
			}
		default:
			errs = append(errs, "EMAIL_DRIVER must be smtp, blackhole, tencent_ses, resend, or multi")
		}
	}

	if productionLike {
		if configStringMissing(c.Database.URL) {
			errs = append(errs, "DATABASE_URL is required in production")
		}
		if c.App.Env == EnvProduction && !c.Token.CookieSecure {
			errs = append(errs, "TOKEN_COOKIE_SECURE must be true in production")
		}
		if len(c.App.TrustedProxies) == 0 {
			errs = append(errs, "TRUSTED_PROXIES is required in production for secure IP detection")
		}
		if configStringMissing(c.App.MetricsUser) {
			errs = append(errs, "METRICS_USER is required in production")
		}
		if configStringMissing(c.App.MetricsPassword) {
			errs = append(errs, "METRICS_PASSWORD is required in production")
		}
		if configStringMissing(c.Redis.Password) {
			errs = append(errs, "REDIS_PASSWORD is required in production")
		}
		if configStringMissing(c.ObjectStorage.Endpoint) {
			errs = append(errs, "OBJECT_STORAGE_ENDPOINT is required in production")
		}
		if configStringMissing(c.ObjectStorage.Bucket) {
			errs = append(errs, "OBJECT_STORAGE_BUCKET is required in production")
		}
		if configStringMissing(c.ObjectStorage.AccessKeyID) {
			errs = append(errs, "OBJECT_STORAGE_ACCESS_KEY_ID is required in production")
		}
		if configStringMissing(c.ObjectStorage.SecretAccessKey) {
			errs = append(errs, "OBJECT_STORAGE_SECRET_ACCESS_KEY is required in production")
		}
		if configStringMissing(c.Bot.ServiceToken) {
			errs = append(errs, "BOT_SERVICE_TOKEN is required in production")
		}
		if configStringMissing(c.Admission.PublicBaseURL) {
			errs = append(errs, "ADMISSION_PUBLIC_BASE_URL is required in production")
		}
		if !c.SMS.Enabled {
			errs = append(errs, "SMS_ENABLED must be true in production")
		}
		if !c.Observability.Enabled {
			errs = append(errs, "OTEL_ENABLED must be true in production")
		}
		if configStringMissing(c.Observability.OTLPEndpoint) {
			errs = append(errs, "OTEL_EXPORTER_OTLP_ENDPOINT is required in production")
		}

		plaintextPostgresAllowed := c.Database.AllowPlaintext && c.Database.SSLMode == "disable"
		if c.Database.SSLMode != "verify-full" && !plaintextPostgresAllowed {
			errs = append(errs, "DB_SSL_MODE must be 'verify-full' in production")
		}
		if c.Database.SSLMode != "disable" && configStringMissing(c.Database.SSLRootCert) {
			errs = append(errs, "DB_SSL_ROOT_CERT is required in production")
		}
		if !c.Redis.TLSEnabled {
			errs = append(errs, "REDIS_TLS_ENABLED must be true in production")
		}
		if c.Redis.TLSEnabled && configStringMissing(c.Redis.TLSCAFile) {
			errs = append(errs, "REDIS_TLS_CA is required in production")
		}
		if !c.ObjectStorage.UseSSL {
			errs = append(errs, "OBJECT_STORAGE_USE_SSL must be true in production")
		}
	}
	errs = append(errs, validateAdmissionPublicBaseURL(c.Admission.PublicBaseURL, productionLike)...)
	errs = append(errs, validateExternalDataConfig(c.ExternalData)...)

	if len(parseErrs) > 0 {
		errs = append(errs, parseErrs...)
	}

	if !isAllowedAppEnv(c.App.Env) && !c.Token.CookieSecure {
		errs = append(errs, "TOKEN_COOKIE_SECURE can only be false in development or prod-parity")
	}

	// Casdoor OIDC 配置校验
	if configStringMissing(c.Casdoor.Issuer) {
		errs = append(errs, "CASDOOR_ISSUER is required")
	}
	if configStringMissing(c.Casdoor.ClientID) {
		errs = append(errs, "CASDOOR_CLIENT_ID is required")
	}
	if configStringMissing(c.Casdoor.ClientSecret) {
		errs = append(errs, "CASDOOR_CLIENT_SECRET is required")
	}
	if configStringMissing(c.Casdoor.RedirectURI) {
		errs = append(errs, "CASDOOR_REDIRECT_URI is required")
	}
	if configStringMissing(c.Casdoor.IntrospectionClientID) {
		errs = append(errs, "CASDOOR_INTROSPECTION_CLIENT_ID is required")
	}
	if configStringMissing(c.Casdoor.IntrospectionClientSecret) {
		errs = append(errs, "CASDOOR_INTROSPECTION_CLIENT_SECRET is required")
	}
	if configStringMissing(c.Casdoor.Organization) {
		errs = append(errs, "CASDOOR_ORGANIZATION is required")
	}
	errs = append(errs, validateCasdoorAdminCredentials(c.Casdoor, productionLike)...)

	// OpenFGA 是应用运行时必需依赖，所有环境都需要完整配置。
	if configStringMissing(c.OpenFGA.StoreID) {
		errs = append(errs, "OPENFGA_STORE_ID is required")
	}
	if configStringMissing(c.OpenFGA.AuthorizationModelID) {
		errs = append(errs, "OPENFGA_MODEL_ID is required")
	}
	if configStringMissing(c.OpenFGA.APIUrl) {
		errs = append(errs, "OPENFGA_API_URL is required")
	}

	const maxRateLimit = 100000
	if c.App.APIIPRateLimit <= 0 || c.App.APIIPRateLimit > maxRateLimit {
		errs = append(errs, fmt.Sprintf("API_IP_RATE_LIMIT must be between 1 and %d (got %d)", maxRateLimit, c.App.APIIPRateLimit))
	}
	if c.App.APIGlobalLimit <= 0 || c.App.APIGlobalLimit > maxRateLimit {
		errs = append(errs, fmt.Sprintf("API_GLOBAL_RATE_LIMIT must be between 1 and %d (got %d)", maxRateLimit, c.App.APIGlobalLimit))
	}
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
	errs = append(errs, validateOpenPlatformTokenProbe(c.OpenPlatform.TokenProbe, productionLike)...)
	errs = append(errs, validateOpenPlatformBaseURLs(c.OpenPlatform, productionLike)...)

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
	}

	return nil
}

func configStringMissing(value string) bool {
	return strings.TrimSpace(value) == ""
}

func validateAppEnv(env string) []string {
	if configStringMissing(env) {
		return []string{"APP_ENV is required"}
	}
	if strings.TrimSpace(env) != env {
		return []string{"APP_ENV must not include leading or trailing whitespace"}
	}
	if !isAllowedAppEnv(env) {
		return []string{"APP_ENV must be development, production, or prod-parity"}
	}
	return nil
}

func isAllowedAppEnv(env string) bool {
	return env == EnvDevelopment || env == EnvProduction || env == EnvProdParity
}

func validateAppPort(port string) []string {
	if configStringMissing(port) {
		return []string{"APP_PORT is required"}
	}
	if strings.TrimSpace(port) != port {
		return []string{"APP_PORT must not include leading or trailing whitespace"}
	}
	value, err := strconv.Atoi(port)
	if err != nil {
		return []string{"APP_PORT must be an integer between 1 and 65535"}
	}
	if value < 1 || value > 65535 {
		return []string{fmt.Sprintf("APP_PORT must be between 1 and 65535 (got %d)", value)}
	}
	return nil
}

func validateMaxBodySize(maxBodySize int64) []string {
	const maxAllowedBodySize = int64(100 << 20)
	if maxBodySize < 1 || maxBodySize > maxAllowedBodySize {
		return []string{fmt.Sprintf("MAX_BODY_SIZE must be between 1 and %d bytes (got %d)", maxAllowedBodySize, maxBodySize)}
	}
	return nil
}

func validateHealthCheckTimeout(timeout int) []string {
	const maxHealthCheckTimeout = 60
	if timeout < 1 || timeout > maxHealthCheckTimeout {
		return []string{fmt.Sprintf("HEALTH_CHECK_TIMEOUT must be between 1 and %d seconds (got %d)", maxHealthCheckTimeout, timeout)}
	}
	return nil
}

func validateTrustedProxies(proxies []string) []string {
	var errs []string
	for _, proxy := range proxies {
		errs = append(errs, validateTrustedProxy(proxy)...)
	}
	return errs
}

func validateTrustedProxy(proxy string) []string {
	trimmed := strings.TrimSpace(proxy)
	if trimmed == "" {
		return []string{"TRUSTED_PROXIES contains an empty proxy entry"}
	}
	if trimmed != proxy {
		return []string{fmt.Sprintf("TRUSTED_PROXIES entry %q must not include leading or trailing whitespace", proxy)}
	}
	if strings.Contains(proxy, "/") {
		if _, _, err := net.ParseCIDR(proxy); err != nil {
			return []string{fmt.Sprintf("TRUSTED_PROXIES entry %q must be an IPv4/IPv6 address or CIDR", proxy)}
		}
		return nil
	}
	if net.ParseIP(proxy) == nil {
		return []string{fmt.Sprintf("TRUSTED_PROXIES entry %q must be an IPv4/IPv6 address or CIDR", proxy)}
	}
	return nil
}

// ValidateCORSOrigins validates CORS allow-list entries before they are passed
// to gin-contrib/cors. Runtime wiring uses this as a fail-fast guard without
// duplicating Config validation rules.
func ValidateCORSOrigins(origins []string) error {
	errs := validateCORSOrigins(origins, false)
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}

func validateCORSOrigins(origins []string, productionLike bool) []string {
	if len(origins) == 0 {
		return []string{"CORS_ORIGINS is required"}
	}

	var errs []string
	for _, origin := range origins {
		errs = append(errs, validateCORSOrigin(origin, productionLike)...)
	}
	return errs
}

func validateCORSOrigin(origin string, productionLike bool) []string {
	switch httpOriginViolation(origin) {
	case originViolationNone:
		if productionLike && !isHTTPSOrigin(origin) {
			return []string{fmt.Sprintf("CORS configuration error: origin %q must use https in production", origin)}
		}
		return nil
	case originViolationEmpty:
		return []string{"CORS configuration error: empty origin is not allowed"}
	case originViolationWhitespace:
		return []string{fmt.Sprintf("CORS configuration error: origin %q must not include leading or trailing whitespace", origin)}
	case originViolationWildcard:
		return []string{"CORS configuration error: wildcard '*' is not allowed when AllowCredentials is true"}
	case originViolationInvalid:
		return []string{fmt.Sprintf("CORS configuration error: origin %q must be an absolute http(s) origin", origin)}
	case originViolationUserInfo:
		return []string{fmt.Sprintf("CORS configuration error: origin %q must not include user info", origin)}
	case originViolationPort:
		return []string{fmt.Sprintf("CORS configuration error: origin %q must include a valid port when a port is specified", origin)}
	case originViolationTrailingSlash:
		return []string{fmt.Sprintf("CORS configuration error: origin %q must not have a trailing slash", origin)}
	case originViolationPath:
		return []string{fmt.Sprintf("CORS configuration error: origin %q must not include a path", origin)}
	case originViolationQueryFragment:
		return []string{fmt.Sprintf("CORS configuration error: origin %q must not include query or fragment", origin)}
	default:
		return []string{fmt.Sprintf("CORS configuration error: origin %q must be an absolute http(s) origin", origin)}
	}
}

type originViolation string

const (
	originViolationNone          originViolation = ""
	originViolationEmpty         originViolation = "empty"
	originViolationWhitespace    originViolation = "whitespace"
	originViolationWildcard      originViolation = "wildcard"
	originViolationInvalid       originViolation = "invalid"
	originViolationUserInfo      originViolation = "user_info"
	originViolationPort          originViolation = "port"
	originViolationTrailingSlash originViolation = "trailing_slash"
	originViolationPath          originViolation = "path"
	originViolationQueryFragment originViolation = "query_fragment"
)

func httpOriginViolation(origin string) originViolation {
	trimmed := strings.TrimSpace(origin)
	if trimmed == "" {
		return originViolationEmpty
	}
	if trimmed != origin {
		return originViolationWhitespace
	}
	if origin == "*" {
		return originViolationWildcard
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		if strings.Contains(err.Error(), "invalid port") {
			return originViolationPort
		}
		return originViolationInvalid
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return originViolationInvalid
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return originViolationInvalid
	}
	if parsed.User != nil {
		return originViolationUserInfo
	}
	if !hasValidHTTPOriginPort(parsed) {
		return originViolationPort
	}
	if parsed.Path == "/" {
		return originViolationTrailingSlash
	}
	if parsed.Path != "" {
		return originViolationPath
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return originViolationQueryFragment
	}
	return originViolationNone
}

func hasValidHTTPOriginPort(parsed *url.URL) bool {
	port := parsed.Port()
	if port == "" {
		host := parsed.Host
		if strings.HasPrefix(host, "[") {
			end := strings.LastIndex(host, "]")
			return end >= 0 && host[end+1:] == ""
		}
		return !strings.Contains(host, ":")
	}

	portNumber, err := strconv.Atoi(port)
	return err == nil && portNumber >= 1 && portNumber <= 65535
}

func validateOpenPlatformBaseURLs(cfg OpenPlatformConfig, productionLike bool) []string {
	var errs []string
	if productionLike {
		errs = append(errs, validateRequiredHTTPOrigin("OPEN_PLATFORM_CONSENT_BASE_URL", cfg.ConsentBaseURL, productionLike)...)
		errs = append(errs, validateRequiredHTTPOrigin("OPEN_PLATFORM_ACCOUNT_BASE_URL", cfg.AccountBaseURL, productionLike)...)
		return errs
	}
	errs = append(errs, validateOptionalHTTPOrigin("OPEN_PLATFORM_CONSENT_BASE_URL", cfg.ConsentBaseURL, productionLike)...)
	errs = append(errs, validateOptionalHTTPOrigin("OPEN_PLATFORM_ACCOUNT_BASE_URL", cfg.AccountBaseURL, productionLike)...)
	return errs
}

func validateRequiredHTTPOrigin(name, origin string, productionLike bool) []string {
	if configStringMissing(origin) {
		return []string{fmt.Sprintf("%s is required in production", name)}
	}
	return validateOptionalHTTPOrigin(name, origin, productionLike)
}

func validateOptionalHTTPOrigins(name string, origins []string, productionLike bool) []string {
	var errs []string
	for _, origin := range origins {
		errs = append(errs, validateOptionalHTTPOrigin(name, origin, productionLike)...)
	}
	return errs
}

func validateOptionalHTTPOrigin(name, origin string, productionLike bool) []string {
	if origin == "" {
		return nil
	}
	switch httpOriginViolation(origin) {
	case originViolationNone:
		if productionLike && !isHTTPSOrigin(origin) {
			return []string{fmt.Sprintf("%s must use https in production", name)}
		}
		return nil
	case originViolationWhitespace:
		return []string{fmt.Sprintf("%s must not include leading or trailing whitespace", name)}
	case originViolationPort:
		return []string{fmt.Sprintf("%s must include a valid port when a port is specified", name)}
	case originViolationUserInfo:
		return []string{fmt.Sprintf("%s must not include user info", name)}
	case originViolationTrailingSlash:
		return []string{fmt.Sprintf("%s must not have a trailing slash", name)}
	case originViolationPath:
		return []string{fmt.Sprintf("%s must not include a path", name)}
	case originViolationQueryFragment:
		return []string{fmt.Sprintf("%s must not include query or fragment", name)}
	default:
		return []string{fmt.Sprintf("%s must be an absolute http(s) origin", name)}
	}
}

func isHTTPSOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	return err == nil && strings.EqualFold(parsed.Scheme, "https")
}

func validateExternalDataConfig(cfg ExternalDataConfig) []string {
	var errs []string
	for i, source := range cfg.StudentSources {
		if !source.Enabled {
			continue
		}
		label := fmt.Sprintf("EXTERNAL_STUDENT_SOURCE[%d]", i)
		if strings.TrimSpace(source.Name) == "" {
			errs = append(errs, label+" name is required")
		}
		provider := strings.TrimSpace(source.Provider)
		if provider == "" {
			errs = append(errs, "EXTERNAL_STUDENT_SOURCE_PROVIDER is required when EXTERNAL_STUDENT_SOURCE_ENABLED=true")
		}
		if !isTenDigitCode(source.SchoolCode) {
			errs = append(errs, "EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE must be a 10-digit school code")
		}
		switch provider {
		case "oracle":
			errs = append(errs, validateExternalOracleStudentSource(source.Oracle)...)
		default:
			errs = append(errs, "EXTERNAL_STUDENT_SOURCE_PROVIDER must be oracle when EXTERNAL_STUDENT_SOURCE_ENABLED=true")
		}
	}
	return errs
}

func validateExternalOracleStudentSource(cfg ExternalOracleStudentSourceConfig) []string {
	var errs []string
	required := map[string]string{
		"EXTERNAL_STUDENT_SOURCE_ORACLE_HOST":                cfg.Host,
		"EXTERNAL_STUDENT_SOURCE_ORACLE_SERVICE_NAME":        cfg.ServiceName,
		"EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME":            cfg.Username,
		"EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD":            cfg.Password,
		"EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA":              cfg.Schema,
		"EXTERNAL_STUDENT_SOURCE_ORACLE_TABLE":               cfg.Table,
		"EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_ID_COLUMN":   cfg.StudentIDColumn,
		"EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_NAME_COLUMN": cfg.StudentNameColumn,
	}
	for key, value := range required {
		if strings.TrimSpace(value) == "" {
			errs = append(errs, key+" is required when EXTERNAL_STUDENT_SOURCE_PROVIDER=oracle")
		}
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		errs = append(errs, fmt.Sprintf("EXTERNAL_STUDENT_SOURCE_ORACLE_PORT must be between 1 and 65535 (got %d)", cfg.Port))
	}
	if cfg.ConnectTimeoutSeconds < 1 || cfg.ConnectTimeoutSeconds > 60 {
		errs = append(errs, fmt.Sprintf("EXTERNAL_STUDENT_SOURCE_ORACLE_CONNECT_TIMEOUT_SECONDS must be between 1 and 60 (got %d)", cfg.ConnectTimeoutSeconds))
	}
	if cfg.QueryTimeoutSeconds < 1 || cfg.QueryTimeoutSeconds > 60 {
		errs = append(errs, fmt.Sprintf("EXTERNAL_STUDENT_SOURCE_ORACLE_QUERY_TIMEOUT_SECONDS must be between 1 and 60 (got %d)", cfg.QueryTimeoutSeconds))
	}
	if cfg.MaxOpenConns < 1 || cfg.MaxOpenConns > 100 {
		errs = append(errs, fmt.Sprintf("EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_OPEN_CONNS must be between 1 and 100 (got %d)", cfg.MaxOpenConns))
	}
	if cfg.MaxIdleConns < 0 || cfg.MaxIdleConns > cfg.MaxOpenConns {
		errs = append(errs, "EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_IDLE_CONNS must be between 0 and EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_OPEN_CONNS")
	}
	return errs
}

func isTenDigitCode(value string) bool {
	if len(value) != 10 {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func validateAdmissionPublicBaseURL(raw string, productionLike bool) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return []string{"ADMISSION_PUBLIC_BASE_URL must be an absolute http(s) URL"}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return []string{"ADMISSION_PUBLIC_BASE_URL must be an absolute http(s) URL"}
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return []string{"ADMISSION_PUBLIC_BASE_URL must not include query or fragment"}
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return []string{"ADMISSION_PUBLIC_BASE_URL must not include a path"}
	}
	if productionLike && parsed.Scheme != "https" {
		return []string{"ADMISSION_PUBLIC_BASE_URL must use https in production"}
	}
	if productionLike && strings.TrimRight(raw, "/") != "https://join.stuhelper.com" {
		return []string{"ADMISSION_PUBLIC_BASE_URL must be https://join.stuhelper.com in production"}
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
	return !configStringMissing(cfg.AppProvisioningClientID) ||
		!configStringMissing(cfg.AppProvisioningClientSecret) ||
		!configStringMissing(cfg.AppProvisioningApplication)
}

func userProfileCredentialConfigured(cfg CasdoorConfig) bool {
	return !configStringMissing(cfg.UserProfileClientID) ||
		!configStringMissing(cfg.UserProfileClientSecret) ||
		!configStringMissing(cfg.UserProfileApplication)
}

func roleSyncCredentialConfigured(cfg CasdoorConfig) bool {
	return !configStringMissing(cfg.RoleSyncClientID) ||
		!configStringMissing(cfg.RoleSyncClientSecret) ||
		!configStringMissing(cfg.RoleSyncApplication)
}

func userLookupCredentialConfigured(cfg CasdoorConfig) bool {
	return !configStringMissing(cfg.UserLookupClientID) ||
		!configStringMissing(cfg.UserLookupClientSecret) ||
		!configStringMissing(cfg.UserLookupApplication)
}

func validateCasdoorCredentialSet(required bool, prefix, clientID, clientSecret, application string) []string {
	if !required {
		return nil
	}
	var errs []string
	if configStringMissing(clientID) {
		errs = append(errs, "CASDOOR_"+prefix+"_CLIENT_ID is required")
	}
	if configStringMissing(clientSecret) {
		errs = append(errs, "CASDOOR_"+prefix+"_CLIENT_SECRET is required")
	}
	if configStringMissing(application) {
		errs = append(errs, "CASDOOR_"+prefix+"_APPLICATION is required")
	}
	return errs
}
