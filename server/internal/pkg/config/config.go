package config

// Config 应用配置
type Config struct {
	App           AppConfig
	Database      DatabaseConfig
	Redis         RedisConfig
	Zitadel       ZitadelConfig
	OpenFGA       OpenFGAConfig
	ObjectStorage ObjectStorageConfig
	Token         TokenConfig
	Log           LogConfig
	RateLimit     ReviewRateLimitConfig
	Security      SecurityConfig
	SMS           SMSConfig
	Bot           BotConfig
	Observability ObservabilityConfig
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
	Enabled          bool
	ServiceName      string
	ServiceNamespace string
	OTLPEndpoint     string
	OTLPInsecure     bool
	TraceSampleRatio float64
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
}

// ZitadelConfig Zitadel OIDC 认证配置
type ZitadelConfig struct {
	Issuer          string // OIDC 对外签发者地址，如 https://sso.stuhelper.com
	InternalAddress string // 可选：容器内访问 Zitadel 的拨号地址，如 host.docker.internal:8085
	ClientID        string
	ClientSecret    string
	RedirectURI     string
	ProjectID       string // 用于解析 Token 中的角色 claim
	OrgID           string // 默认组织 ID
	ManagementPAT   string // Service Account PAT，用于 Management API（角色同步等）
}

// OpenFGAConfig OpenFGA 关系型授权引擎配置
type OpenFGAConfig struct {
	APIUrl               string // OpenFGA HTTP API 地址
	StoreID              string // 授权 Store ID
	AuthorizationModelID string // 授权模型版本 ID
}

// SMSConfig 腾讯云短信配置
type SMSConfig struct {
	Enabled      bool
	SecretID     string
	SecretKey    string
	AppID        string
	SignName     string
	TemplateID   string
	Region       string
	InternalKey  string // 内部调用鉴权密钥
	InternalPort string // 内部 HTTP 端口（SMS 转发服务）
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
		Zitadel:       loadZitadelConfig(),
		OpenFGA:       loadOpenFGAConfig(),
		ObjectStorage: loadObjectStorageConfig(&parseErrs),
		Redis:         loadRedisConfig(&parseErrs),
		Token:         loadTokenConfig(&parseErrs),
		Log:           loadLogConfig(&parseErrs),
		RateLimit:     loadReviewRateLimitConfig(&parseErrs),
		SMS:           loadSMSConfig(&parseErrs),
		Bot:           loadBotConfig(),
		Observability: loadObservabilityConfig(&parseErrs),
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
	}
}

func loadZitadelConfig() ZitadelConfig {
	return ZitadelConfig{
		Issuer:          getEnv("ZITADEL_ISSUER", ""),
		InternalAddress: getEnv("ZITADEL_INTERNAL_ADDRESS", ""),
		ClientID:        getEnv("ZITADEL_CLIENT_ID", ""),
		ClientSecret:    getEnv("ZITADEL_CLIENT_SECRET", ""),
		RedirectURI:     getEnv("ZITADEL_REDIRECT_URI", ""),
		ProjectID:       getEnv("ZITADEL_PROJECT_ID", ""),
		OrgID:           getEnv("ZITADEL_ORG_ID", ""),
		ManagementPAT:   getEnv("ZITADEL_MANAGEMENT_PAT", ""),
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
	return ObjectStorageConfig{
		Endpoint:        getEnv("OBJECT_STORAGE_ENDPOINT", ""),
		Region:          getEnv("OBJECT_STORAGE_REGION", "us-east-1"),
		Bucket:          getEnv("OBJECT_STORAGE_BUCKET", ""),
		AccessKeyID:     getEnv("OBJECT_STORAGE_ACCESS_KEY_ID", ""),
		SecretAccessKey: getEnv("OBJECT_STORAGE_SECRET_ACCESS_KEY", ""),
		UseSSL:          getEnvBool("OBJECT_STORAGE_USE_SSL", false, parseErrs),
		ForcePathStyle:  getEnvBool("OBJECT_STORAGE_FORCE_PATH_STYLE", true, parseErrs),
		PresignTTL:      getEnvInt("OBJECT_STORAGE_PRESIGN_TTL", 600, parseErrs),
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

func loadSMSConfig(parseErrs *[]string) SMSConfig {
	return SMSConfig{
		Enabled:      getEnvBool("SMS_ENABLED", false, parseErrs),
		SecretID:     getEnv("SMS_SECRET_ID", ""),
		SecretKey:    getEnv("SMS_SECRET_KEY", ""),
		AppID:        getEnv("SMS_APP_ID", ""),
		SignName:     getEnv("SMS_SIGN_NAME", ""),
		TemplateID:   getEnv("SMS_TEMPLATE_ID", ""),
		Region:       getEnv("SMS_REGION", "ap-beijing"),
		InternalKey:  getEnv("SMS_INTERNAL_KEY", ""),
		InternalPort: getEnv("SMS_INTERNAL_PORT", "9090"),
	}
}

func loadBotConfig() BotConfig {
	return BotConfig{
		ServiceToken: getEnv("BOT_SERVICE_TOKEN", ""),
	}
}

func loadObservabilityConfig(parseErrs *[]string) ObservabilityConfig {
	return ObservabilityConfig{
		Enabled:          getEnvBool("OTEL_ENABLED", false, parseErrs),
		ServiceName:      getEnv("OTEL_SERVICE_NAME", "stuhelper-backend"),
		ServiceNamespace: getEnv("OTEL_SERVICE_NAMESPACE", "stuhelper"),
		OTLPEndpoint:     getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		OTLPInsecure:     getEnvBool("OTEL_EXPORTER_OTLP_INSECURE", true, parseErrs),
		TraceSampleRatio: getEnvFloat64("OTEL_TRACE_SAMPLE_RATIO", 0.2, parseErrs),
	}
}
