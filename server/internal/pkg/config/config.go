package config

import "strings"

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
	EnvProdParity  = "prod-parity"
)

// IsProductionLikeEnv reports whether an environment should use production-grade
// validation and runtime behavior.
func IsProductionLikeEnv(env string) bool {
	env = strings.TrimSpace(env)
	return env == EnvProduction || env == EnvProdParity
}

// Config 应用配置
type Config struct {
	App           AppConfig
	Database      DatabaseConfig
	Redis         RedisConfig
	Casdoor       CasdoorConfig
	OpenFGA       OpenFGAConfig
	ObjectStorage ObjectStorageConfig
	Token         TokenConfig
	Log           LogConfig
	RateLimit     ReviewRateLimitConfig
	Security      SecurityConfig
	SMS           SMSConfig
	Email         EmailConfig
	Bot           BotConfig
	Admission     AdmissionConfig
	ExternalData  ExternalDataConfig
	Observability ObservabilityConfig
	OpenPlatform  OpenPlatformConfig
}

// SecurityConfig PII 加密安全配置（已验证、可直接消费的强类型结果）
type SecurityConfig struct {
	DocAESActiveKeyID uint8
	DocAESKeys        map[uint8][]byte
}

// ReviewRateLimitConfig 评课模块端点限流配置
type ReviewRateLimitConfig struct {
	PostLimit       int
	VoteLimit       int
	ReportLimit     int
	ReplyLimit      int
	WriteLimit      int
	SearchAnonLimit int
	SearchUserLimit int
	BatchAnonLimit  int
	BatchUserLimit  int
}

// OpenPlatformConfig 开放平台配置。
type OpenPlatformConfig struct {
	DisclosureRateLimit OpenPlatformDisclosureRateLimitConfig
	TokenProbe          OpenPlatformTokenProbeConfig
	ConsentBaseURL      string
	AccountBaseURL      string
}

// OpenPlatformDisclosureRateLimitConfig 控制 disclosure 路径限流和异常重放检测阈值。
type OpenPlatformDisclosureRateLimitConfig struct {
	AppLimit                   int
	AppUserLimit               int
	EndpointLimit              int
	ConsentLimit               int
	ReplayLimit                int
	WindowSeconds              int
	ReplayWindowSeconds        int
	ReplayAuditCooldownSeconds int
}

// OpenPlatformTokenProbeConfig 控制第三方 Casdoor app runtime code-flow token 探针。
type OpenPlatformTokenProbeConfig struct {
	RuntimeRequired       bool
	RuntimeCommand        string
	RuntimeTimeoutSeconds int
}

// AdmissionConfig controls the public group admission verification surface.
type AdmissionConfig struct {
	PublicBaseURL string
}

// ExternalDataConfig controls external school/business data sources.
type ExternalDataConfig struct {
	StudentSources []ExternalStudentSourceConfig
}

type ExternalStudentSourceConfig struct {
	Name       string
	Enabled    bool
	Provider   string
	SchoolCode string
	Oracle     ExternalOracleStudentSourceConfig
}

type ExternalOracleStudentSourceConfig struct {
	Host                  string
	Port                  int
	ServiceName           string
	Username              string
	Password              string
	Schema                string
	Table                 string
	StudentIDColumn       string
	StudentNameColumn     string
	ConnectTimeoutSeconds int
	QueryTimeoutSeconds   int
	MaxOpenConns          int
	MaxIdleConns          int
}

// LogConfig 日志配置
type LogConfig struct {
	Level           string
	Format          string
	Output          string
	SamplingEnabled bool
	SamplingInitial int
	SamplingAfter   int
	ServiceName     string
	Environment     string
	ServiceVersion  string
}

// ObservabilityConfig OpenTelemetry / tracing 配置
type ObservabilityConfig struct {
	Enabled                       bool
	ServiceName                   string
	ServiceNamespace              string
	OTLPEndpoint                  string
	OTLPInsecure                  bool
	TraceSampleRatio              float64
	FrontendMetricsAllowedOrigins []string
}

// ObjectStorageConfig 对象存储配置。
type ObjectStorageConfig struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	ForcePathStyle  bool
	PresignTTL      int
	TLSCAFile       string
}

// AppConfig 应用配置
type AppConfig struct {
	Env                string
	Port               string
	CORSOrigins        []string
	TrustedProxies     []string
	HMACSecret         string
	MaxBodySize        int64
	MetricsUser        string
	MetricsPassword    string
	APIIPRateLimit     int
	APIGlobalLimit     int
	HealthCheckTimeout int
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime int
	MaxConnIdleTime int
	QueryTimeout    int
	SSLMode         string
	SSLRootCert     string
	SSLCert         string
	SSLKey          string
	AllowPlaintext  bool
}

// CasdoorConfig OIDC 认证配置。
type CasdoorConfig struct {
	Issuer                      string // OIDC 对外签发者地址，如 https://sso.stuhelper.com
	InternalAddress             string // 可选：容器内访问 IDP 的拨号地址，如 host.docker.internal:8085
	PublicAuthBaseURL           string // 可选：浏览器可见的 Casdoor 授权入口 origin，如 https://sso.stuhelper.com
	ClientID                    string
	ClientSecret                string
	RedirectURI                 string
	WebScopes                   []string
	AdminClientID               string
	AdminClientSecret           string
	AdminRedirectURI            string
	AdminScopes                 []string
	UniappClientID              string
	UniappClientSecret          string
	UniappRedirectURI           string
	UniappScopes                []string
	IntrospectionEndpoint       string
	IntrospectionClientID       string
	IntrospectionClientSecret   string
	Organization                string // Casdoor organization 名称
	RolesClaim                  string // 角色 claim 名称，默认 roles
	AppProvisioningClientID     string
	AppProvisioningClientSecret string
	AppProvisioningApplication  string
	AppProvisioningCertificate  string
	UserProfileClientID         string
	UserProfileClientSecret     string
	UserProfileApplication      string
	UserProfileCertificate      string
	RoleSyncClientID            string
	RoleSyncClientSecret        string
	RoleSyncApplication         string
	RoleSyncCertificate         string
	UserLookupClientID          string
	UserLookupClientSecret      string
	UserLookupApplication       string
	UserLookupCertificate       string
}

// OpenFGAConfig OpenFGA 关系型授权引擎配置
type OpenFGAConfig struct {
	APIUrl               string // OpenFGA HTTP API 地址
	StoreID              string // 授权 Store ID
	AuthorizationModelID string // 授权模型版本 ID
}

// SMSConfig 腾讯云短信配置
type SMSConfig struct {
	Enabled     bool
	SecretID    string
	SecretKey   string
	AppID       string
	SignName    string
	TemplateID  string
	Region      string
	InternalKey string // 内部调用鉴权密钥
}

type EmailConfig struct {
	Enabled                      bool
	Driver                       string
	ProviderPolicy               string
	StudentVerificationSubject   string
	SMTPHost                     string
	SMTPPort                     int
	SMTPUsername                 string
	SMTPPassword                 string
	From                         string
	FromName                     string
	UseTLS                       bool
	StartTLS                     bool
	TencentSecretID              string
	TencentSecretKey             string
	TencentRegion                string
	TencentEndpoint              string
	TencentTemplateID            int64
	TencentReplyTo               string
	TencentTemplatePurpose       string
	TencentTemplateSchoolName    string
	TencentTemplateExpireMinutes int
	ResendAPIKey                 string
	ResendEndpoint               string
	ResendReplyTo                string
}

// BotConfig 机器人内部调用配置。
type BotConfig struct {
	ServiceToken string // 机器人访问后端内部接口的服务令牌
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host         string
	Port         string
	Username     string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	TLSEnabled   bool
	TLSCertFile  string
	TLSKeyFile   string
	TLSCAFile    string
}

// TokenConfig Token 配置
type TokenConfig struct {
	AccessTokenTTL  int
	RefreshTokenTTL int
	CookieSecure    bool
	CookieDomain    string
}

// Load 从环境变量加载配置
func Load() (*Config, error) {
	var parseErrs []string

	cfg := &Config{
		App:           loadAppConfig(&parseErrs),
		Database:      loadDatabaseConfig(&parseErrs),
		Casdoor:       loadCasdoorConfig(),
		OpenFGA:       loadOpenFGAConfig(),
		ObjectStorage: loadObjectStorageConfig(&parseErrs),
		Redis:         loadRedisConfig(&parseErrs),
		Token:         loadTokenConfig(&parseErrs),
		Log:           loadLogConfig(&parseErrs),
		RateLimit:     loadReviewRateLimitConfig(&parseErrs),
		SMS:           loadSMSConfig(&parseErrs),
		Email:         loadEmailConfig(&parseErrs),
		Bot:           loadBotConfig(),
		Admission:     loadAdmissionConfig(),
		ExternalData:  loadExternalDataConfig(&parseErrs),
		Observability: loadObservabilityConfig(&parseErrs),
		OpenPlatform:  loadOpenPlatformConfig(&parseErrs),
	}

	securityCfg, securityErrs := parseSecurityConfig()
	cfg.Security = securityCfg
	parseErrs = append(parseErrs, securityErrs...)

	if err := cfg.validate(parseErrs); err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadAppConfig(parseErrs *[]string) AppConfig {
	return AppConfig{
		Env:                getEnv("APP_ENV", "development"),
		Port:               getEnv("APP_PORT", "8080"),
		CORSOrigins:        getEnvSlice("CORS_ORIGINS", []string{}),
		TrustedProxies:     getEnvSlice("TRUSTED_PROXIES", []string{}),
		HMACSecret:         getEnv("HMAC_SECRET", ""),
		MaxBodySize:        getEnvInt64("MAX_BODY_SIZE", 10<<20, parseErrs),
		MetricsUser:        getEnv("METRICS_USER", "prometheus"),
		MetricsPassword:    getEnv("METRICS_PASSWORD", ""),
		APIIPRateLimit:     getEnvInt("API_IP_RATE_LIMIT", 100, parseErrs),
		APIGlobalLimit:     getEnvInt("API_GLOBAL_RATE_LIMIT", 10000, parseErrs),
		HealthCheckTimeout: getEnvInt("HEALTH_CHECK_TIMEOUT", 3, parseErrs),
	}
}

func loadDatabaseConfig(parseErrs *[]string) DatabaseConfig {
	return DatabaseConfig{
		URL:             getEnv("DATABASE_URL", ""),
		MaxConns:        getEnvInt32("DB_MAX_CONNS", 20, parseErrs),
		MinConns:        getEnvInt32("DB_MIN_CONNS", 2, parseErrs),
		MaxConnLifetime: getEnvInt("DB_MAX_CONN_LIFETIME", 30, parseErrs),
		MaxConnIdleTime: getEnvInt("DB_MAX_CONN_IDLE_TIME", 5, parseErrs),
		QueryTimeout:    getEnvInt("DB_QUERY_TIMEOUT", 5, parseErrs),
		SSLMode:         getEnv("DB_SSL_MODE", "disable"),
		SSLRootCert:     getEnv("DB_SSL_ROOT_CERT", ""),
		SSLCert:         getEnv("DB_SSL_CERT", ""),
		SSLKey:          getEnv("DB_SSL_KEY", ""),
		AllowPlaintext:  getEnvBool("EXTERNAL_POSTGRES_ALLOW_PLAINTEXT", false, parseErrs),
	}
}

func loadCasdoorConfig() CasdoorConfig {
	return CasdoorConfig{
		Issuer:                      getEnv("CASDOOR_ISSUER", ""),
		InternalAddress:             getEnv("CASDOOR_INTERNAL_ADDRESS", ""),
		PublicAuthBaseURL:           getEnv("CASDOOR_PUBLIC_AUTH_BASE_URL", ""),
		ClientID:                    getEnv("CASDOOR_CLIENT_ID", ""),
		ClientSecret:                getEnv("CASDOOR_CLIENT_SECRET", ""),
		RedirectURI:                 getEnv("CASDOOR_REDIRECT_URI", ""),
		WebScopes:                   getEnvSlice("CASDOOR_WEB_SCOPES", nil),
		AdminClientID:               getEnv("CASDOOR_ADMIN_CLIENT_ID", ""),
		AdminClientSecret:           getEnv("CASDOOR_ADMIN_CLIENT_SECRET", ""),
		AdminRedirectURI:            getEnv("CASDOOR_ADMIN_REDIRECT_URI", ""),
		AdminScopes:                 getEnvSlice("CASDOOR_ADMIN_SCOPES", nil),
		UniappClientID:              getEnv("CASDOOR_UNIAPP_CLIENT_ID", ""),
		UniappClientSecret:          getEnv("CASDOOR_UNIAPP_CLIENT_SECRET", ""),
		UniappRedirectURI:           getEnv("CASDOOR_UNIAPP_REDIRECT_URI", ""),
		UniappScopes:                getEnvSlice("CASDOOR_UNIAPP_SCOPES", nil),
		IntrospectionEndpoint:       getEnv("CASDOOR_INTROSPECTION_ENDPOINT", ""),
		IntrospectionClientID:       getEnv("CASDOOR_INTROSPECTION_CLIENT_ID", ""),
		IntrospectionClientSecret:   getEnv("CASDOOR_INTROSPECTION_CLIENT_SECRET", ""),
		Organization:                getEnv("CASDOOR_ORGANIZATION", ""),
		RolesClaim:                  getEnv("CASDOOR_ROLES_CLAIM", "roles"),
		AppProvisioningClientID:     getEnv("CASDOOR_APP_PROVISIONING_CLIENT_ID", ""),
		AppProvisioningClientSecret: getEnv("CASDOOR_APP_PROVISIONING_CLIENT_SECRET", ""),
		AppProvisioningApplication:  getEnv("CASDOOR_APP_PROVISIONING_APPLICATION", ""),
		AppProvisioningCertificate:  getEnv("CASDOOR_APP_PROVISIONING_CERTIFICATE", ""),
		UserProfileClientID:         getEnv("CASDOOR_USER_PROFILE_CLIENT_ID", ""),
		UserProfileClientSecret:     getEnv("CASDOOR_USER_PROFILE_CLIENT_SECRET", ""),
		UserProfileApplication:      getEnv("CASDOOR_USER_PROFILE_APPLICATION", ""),
		UserProfileCertificate:      getEnv("CASDOOR_USER_PROFILE_CERTIFICATE", ""),
		RoleSyncClientID:            getEnv("CASDOOR_ROLE_SYNC_CLIENT_ID", ""),
		RoleSyncClientSecret:        getEnv("CASDOOR_ROLE_SYNC_CLIENT_SECRET", ""),
		RoleSyncApplication:         getEnv("CASDOOR_ROLE_SYNC_APPLICATION", ""),
		RoleSyncCertificate:         getEnv("CASDOOR_ROLE_SYNC_CERTIFICATE", ""),
		UserLookupClientID:          getEnv("CASDOOR_USER_LOOKUP_CLIENT_ID", ""),
		UserLookupClientSecret:      getEnv("CASDOOR_USER_LOOKUP_CLIENT_SECRET", ""),
		UserLookupApplication:       getEnv("CASDOOR_USER_LOOKUP_APPLICATION", ""),
		UserLookupCertificate:       getEnv("CASDOOR_USER_LOOKUP_CERTIFICATE", ""),
	}
}

func loadOpenFGAConfig() OpenFGAConfig {
	return OpenFGAConfig{
		APIUrl:               getEnv("OPENFGA_API_URL", "http://localhost:8081"),
		StoreID:              getEnv("OPENFGA_STORE_ID", ""),
		AuthorizationModelID: getEnv("OPENFGA_MODEL_ID", ""),
	}
}

func loadObjectStorageConfig(parseErrs *[]string) ObjectStorageConfig {
	tlsCAFile := getEnv("OBJECT_STORAGE_TLS_CA", "")
	if tlsCAFile == "" {
		tlsCAFile = getEnv("AWS_CA_BUNDLE", "")
	}
	return ObjectStorageConfig{
		Endpoint:        getEnv("OBJECT_STORAGE_ENDPOINT", ""),
		Region:          getEnv("OBJECT_STORAGE_REGION", "us-east-1"),
		Bucket:          getEnv("OBJECT_STORAGE_BUCKET", ""),
		AccessKeyID:     getEnv("OBJECT_STORAGE_ACCESS_KEY_ID", ""),
		SecretAccessKey: getEnv("OBJECT_STORAGE_SECRET_ACCESS_KEY", ""),
		UseSSL:          getEnvBool("OBJECT_STORAGE_USE_SSL", false, parseErrs),
		ForcePathStyle:  getEnvBool("OBJECT_STORAGE_FORCE_PATH_STYLE", true, parseErrs),
		PresignTTL:      getEnvInt("OBJECT_STORAGE_PRESIGN_TTL", 600, parseErrs),
		TLSCAFile:       tlsCAFile,
	}
}

func loadRedisConfig(parseErrs *[]string) RedisConfig {
	return RedisConfig{
		Host:         getEnv("REDIS_HOST", "localhost"),
		Port:         getEnv("REDIS_PORT", "6379"),
		Username:     getEnv("REDIS_USERNAME", "stuhelper_app"),
		Password:     getEnv("REDIS_PASSWORD", ""),
		DB:           getEnvInt("REDIS_DB", 0, parseErrs),
		PoolSize:     getEnvInt("REDIS_POOL_SIZE", 10, parseErrs),
		MinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", 5, parseErrs),
		TLSEnabled:   getEnvBool("REDIS_TLS_ENABLED", false, parseErrs),
		TLSCertFile:  getEnv("REDIS_TLS_CERT", ""),
		TLSKeyFile:   getEnv("REDIS_TLS_KEY", ""),
		TLSCAFile:    getEnv("REDIS_TLS_CA", ""),
	}
}

func loadTokenConfig(parseErrs *[]string) TokenConfig {
	return TokenConfig{
		AccessTokenTTL:  getEnvInt("TOKEN_ACCESS_TTL", 300, parseErrs),
		RefreshTokenTTL: getEnvInt("TOKEN_REFRESH_TTL", 604800, parseErrs),
		CookieSecure:    getEnvBool("TOKEN_COOKIE_SECURE", true, parseErrs),
		CookieDomain:    getEnv("TOKEN_COOKIE_DOMAIN", ""),
	}
}

func loadLogConfig(parseErrs *[]string) LogConfig {
	return LogConfig{
		Level:           getEnv("LOG_LEVEL", "info"),
		Format:          getEnv("LOG_FORMAT", "json"),
		Output:          getEnv("LOG_OUTPUT", "stdout"),
		SamplingEnabled: getEnvBool("LOG_SAMPLING_ENABLED", false, parseErrs),
		SamplingInitial: getEnvInt("LOG_SAMPLING_INITIAL", 100, parseErrs),
		SamplingAfter:   getEnvInt("LOG_SAMPLING_AFTER", 100, parseErrs),
		ServiceName:     getEnv("LOG_SERVICE_NAME", getEnv("OTEL_SERVICE_NAME", "stuhelper-backend")),
		Environment:     getEnv("LOG_ENVIRONMENT", getEnv("APP_ENV", "development")),
		ServiceVersion:  getEnv("LOG_SERVICE_VERSION", ""),
	}
}

func loadReviewRateLimitConfig(parseErrs *[]string) ReviewRateLimitConfig {
	return ReviewRateLimitConfig{
		PostLimit:       getEnvInt("REVIEW_RATE_POST_LIMIT", 5, parseErrs),
		VoteLimit:       getEnvInt("REVIEW_RATE_VOTE_LIMIT", 30, parseErrs),
		ReportLimit:     getEnvInt("REVIEW_RATE_REPORT_LIMIT", 10, parseErrs),
		ReplyLimit:      getEnvInt("REVIEW_RATE_REPLY_LIMIT", 10, parseErrs),
		WriteLimit:      getEnvInt("REVIEW_RATE_WRITE_LIMIT", 10, parseErrs),
		SearchAnonLimit: getEnvInt("REVIEW_RATE_SEARCH_ANON_LIMIT", 5, parseErrs),
		SearchUserLimit: getEnvInt("REVIEW_RATE_SEARCH_USER_LIMIT", 60, parseErrs),
		BatchAnonLimit:  getEnvInt("REVIEW_RATE_BATCH_ANON_LIMIT", 5, parseErrs),
		BatchUserLimit:  getEnvInt("REVIEW_RATE_BATCH_USER_LIMIT", 60, parseErrs),
	}
}

func loadOpenPlatformConfig(parseErrs *[]string) OpenPlatformConfig {
	return OpenPlatformConfig{
		ConsentBaseURL: getEnv("OPEN_PLATFORM_CONSENT_BASE_URL", ""),
		AccountBaseURL: getEnv("OPEN_PLATFORM_ACCOUNT_BASE_URL", ""),
		DisclosureRateLimit: OpenPlatformDisclosureRateLimitConfig{
			AppLimit:                   getEnvInt("OPEN_PLATFORM_DISCLOSURE_APP_LIMIT", 600, parseErrs),
			AppUserLimit:               getEnvInt("OPEN_PLATFORM_DISCLOSURE_APP_USER_LIMIT", 120, parseErrs),
			EndpointLimit:              getEnvInt("OPEN_PLATFORM_DISCLOSURE_ENDPOINT_LIMIT", 1200, parseErrs),
			ConsentLimit:               getEnvInt("OPEN_PLATFORM_DISCLOSURE_CONSENT_LIMIT", 20, parseErrs),
			ReplayLimit:                getEnvInt("OPEN_PLATFORM_DISCLOSURE_REPLAY_LIMIT", 8, parseErrs),
			WindowSeconds:              getEnvInt("OPEN_PLATFORM_DISCLOSURE_WINDOW_SECONDS", 60, parseErrs),
			ReplayWindowSeconds:        getEnvInt("OPEN_PLATFORM_DISCLOSURE_REPLAY_WINDOW_SECONDS", 300, parseErrs),
			ReplayAuditCooldownSeconds: getEnvInt("OPEN_PLATFORM_DISCLOSURE_REPLAY_AUDIT_COOLDOWN_SECONDS", 600, parseErrs),
		},
		TokenProbe: OpenPlatformTokenProbeConfig{
			RuntimeRequired:       getEnvBool("OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED", false, parseErrs),
			RuntimeCommand:        getEnv("OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND", ""),
			RuntimeTimeoutSeconds: getEnvInt("OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_TIMEOUT_SECONDS", 30, parseErrs),
		},
	}
}

func loadSMSConfig(parseErrs *[]string) SMSConfig {
	return SMSConfig{
		Enabled:     getEnvBool("SMS_ENABLED", false, parseErrs),
		SecretID:    getEnv("SMS_SECRET_ID", ""),
		SecretKey:   getEnv("SMS_SECRET_KEY", ""),
		AppID:       getEnv("SMS_APP_ID", ""),
		SignName:    getEnv("SMS_SIGN_NAME", ""),
		TemplateID:  getEnv("SMS_TEMPLATE_ID", ""),
		Region:      getEnv("SMS_REGION", "ap-beijing"),
		InternalKey: getEnv("SMS_INTERNAL_KEY", ""),
	}
}

func loadEmailConfig(parseErrs *[]string) EmailConfig {
	return EmailConfig{
		Enabled:                      getEnvBool("EMAIL_ENABLED", false, parseErrs),
		Driver:                       getEnv("EMAIL_DRIVER", "smtp"),
		ProviderPolicy:               getEnv("EMAIL_PROVIDER_POLICY", ""),
		StudentVerificationSubject:   getEnv("EMAIL_STUDENT_VERIFICATION_SUBJECT", "学生认证验证码"),
		SMTPHost:                     getEnv("EMAIL_SMTP_HOST", ""),
		SMTPPort:                     getEnvInt("EMAIL_SMTP_PORT", 587, parseErrs),
		SMTPUsername:                 getEnv("EMAIL_SMTP_USERNAME", ""),
		SMTPPassword:                 getEnv("EMAIL_SMTP_PASSWORD", ""),
		From:                         getEnv("EMAIL_FROM", ""),
		FromName:                     getEnv("EMAIL_FROM_NAME", "StuHelper 系统邮件"),
		UseTLS:                       getEnvBool("EMAIL_SMTP_USE_TLS", false, parseErrs),
		StartTLS:                     getEnvBool("EMAIL_SMTP_STARTTLS", true, parseErrs),
		TencentSecretID:              getEnv("EMAIL_TENCENT_SECRET_ID", ""),
		TencentSecretKey:             getEnv("EMAIL_TENCENT_SECRET_KEY", ""),
		TencentRegion:                getEnv("EMAIL_TENCENT_REGION", "ap-guangzhou"),
		TencentEndpoint:              getEnv("EMAIL_TENCENT_ENDPOINT", "ses.tencentcloudapi.com"),
		TencentTemplateID:            getEnvInt64("EMAIL_TENCENT_TEMPLATE_ID", 0, parseErrs),
		TencentReplyTo:               getEnv("EMAIL_TENCENT_REPLY_TO", ""),
		TencentTemplatePurpose:       getEnv("EMAIL_TENCENT_TEMPLATE_PURPOSE", "学校邮箱认证"),
		TencentTemplateSchoolName:    getEnv("EMAIL_TENCENT_TEMPLATE_SCHOOL_NAME", "北京航空航天大学"),
		TencentTemplateExpireMinutes: getEnvInt("EMAIL_TENCENT_TEMPLATE_EXPIRE_MINUTES", 5, parseErrs),
		ResendAPIKey:                 getEnv("EMAIL_RESEND_API_KEY", ""),
		ResendEndpoint:               getEnv("EMAIL_RESEND_ENDPOINT", "https://api.resend.com/emails"),
		ResendReplyTo:                getEnv("EMAIL_RESEND_REPLY_TO", ""),
	}
}

func loadBotConfig() BotConfig {
	return BotConfig{
		ServiceToken: getEnv("BOT_SERVICE_TOKEN", ""),
	}
}

func loadAdmissionConfig() AdmissionConfig {
	return AdmissionConfig{
		PublicBaseURL: getEnv("ADMISSION_PUBLIC_BASE_URL", ""),
	}
}

func loadExternalDataConfig(parseErrs *[]string) ExternalDataConfig {
	if !getEnvBool("EXTERNAL_STUDENT_SOURCE_ENABLED", false, parseErrs) {
		return ExternalDataConfig{}
	}
	source := ExternalStudentSourceConfig{
		Name:       getEnv("EXTERNAL_STUDENT_SOURCE_NAME", "buaa-academic-oracle"),
		Enabled:    true,
		Provider:   getEnv("EXTERNAL_STUDENT_SOURCE_PROVIDER", "oracle"),
		SchoolCode: getEnv("EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE", ""),
		Oracle: ExternalOracleStudentSourceConfig{
			Host:                  getEnv("EXTERNAL_STUDENT_SOURCE_ORACLE_HOST", ""),
			Port:                  getEnvInt("EXTERNAL_STUDENT_SOURCE_ORACLE_PORT", 1521, parseErrs),
			ServiceName:           getEnv("EXTERNAL_STUDENT_SOURCE_ORACLE_SERVICE_NAME", ""),
			Username:              getEnv("EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME", ""),
			Password:              getEnv("EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD", ""),
			Schema:                getEnv("EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA", ""),
			Table:                 getEnv("EXTERNAL_STUDENT_SOURCE_ORACLE_TABLE", ""),
			StudentIDColumn:       getEnv("EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_ID_COLUMN", "XH"),
			StudentNameColumn:     getEnv("EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_NAME_COLUMN", "XM"),
			ConnectTimeoutSeconds: getEnvInt("EXTERNAL_STUDENT_SOURCE_ORACLE_CONNECT_TIMEOUT_SECONDS", 5, parseErrs),
			QueryTimeoutSeconds:   getEnvInt("EXTERNAL_STUDENT_SOURCE_ORACLE_QUERY_TIMEOUT_SECONDS", 3, parseErrs),
			MaxOpenConns:          getEnvInt("EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_OPEN_CONNS", 4, parseErrs),
			MaxIdleConns:          getEnvInt("EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_IDLE_CONNS", 1, parseErrs),
		},
	}
	return ExternalDataConfig{StudentSources: []ExternalStudentSourceConfig{source}}
}

func loadObservabilityConfig(parseErrs *[]string) ObservabilityConfig {
	return ObservabilityConfig{
		Enabled:                       getEnvBool("OTEL_ENABLED", false, parseErrs),
		ServiceName:                   getEnv("OTEL_SERVICE_NAME", "stuhelper-backend"),
		ServiceNamespace:              getEnv("OTEL_SERVICE_NAMESPACE", "stuhelper"),
		OTLPEndpoint:                  getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		OTLPInsecure:                  getEnvBool("OTEL_EXPORTER_OTLP_INSECURE", true, parseErrs),
		TraceSampleRatio:              getEnvFloat64("OTEL_TRACE_SAMPLE_RATIO", 0.2, parseErrs),
		FrontendMetricsAllowedOrigins: getEnvSlice("FRONTEND_METRICS_ALLOWED_ORIGINS", nil),
	}
}
