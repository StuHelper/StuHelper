package config

import (
	"fmt"
	"log"
	"net/url"
)

// Config 应用配置
type Config struct {
	App       AppConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Casdoor   CasdoorConfig
	Token     TokenConfig
	Log       LogConfig
	RateLimit ReviewRateLimitConfig
	Security  SecurityConfig
}

// SecurityConfig PII 加密安全配置（已验证、可直接消费的强类型结果）
type SecurityConfig struct {
	DocAESActiveKeyID uint8
	DocAESKeys        map[uint8][]byte
}

// ReviewRateLimitConfig 评课模块端点限流配置
type ReviewRateLimitConfig struct {
	PostLimit   int
	VoteLimit   int
	ReportLimit int
	ReplyLimit  int
	WriteLimit  int
}

// LogConfig 日志配置
type LogConfig struct {
	Level           string
	Format          string
	Output          string
	SamplingEnabled bool
	SamplingInitial int
	SamplingAfter   int
	FileEnabled     bool
	FilePath        string
	FileMaxSize     int
	FileMaxBackups  int
	FileMaxAge      int
	FileCompress    bool
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
	Host            string
	Port            string
	Name            string
	User            string
	Password        string
	MaxConns        int
	MinConns        int
	MaxConnLifetime int
	MaxConnIdleTime int
	QueryTimeout    int
	SSLMode         string
	SSLRootCert     string
	SSLCert         string
	SSLKey          string
}

// CasdoorConfig Casdoor SSO 配置
type CasdoorConfig struct {
	Endpoint     string
	ClientID     string
	ClientSecret string
	Certificate  string
	Organization string
	Application  string
	RedirectURI  string
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host         string
	Port         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	TLSEnabled   bool
	TLSCertFile  string
	TLSKeyFile   string
	TLSCAFile    string
	TLSInsecure  bool
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

	cert, certErr := loadCertificate()
	if certErr != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", certErr)
	}

	cfg := &Config{
		App: AppConfig{
			Env:                getEnv("APP_ENV", "development"),
			Port:               getEnv("APP_PORT", "8080"),
			CORSOrigins:        getEnvSlice("CORS_ORIGINS", []string{}),
			TrustedProxies:     getEnvSlice("TRUSTED_PROXIES", []string{}),
			HMACSecret:         getEnv("HMAC_SECRET", ""),
			MaxBodySize:        getEnvInt64("MAX_BODY_SIZE", 10<<20, &parseErrs),
			MetricsUser:        getEnv("METRICS_USER", "prometheus"),
			MetricsPassword:    getEnv("METRICS_PASSWORD", ""),
			APIIPRateLimit:     getEnvInt("API_IP_RATE_LIMIT", 100, &parseErrs),
			APIGlobalLimit:     getEnvInt("API_GLOBAL_RATE_LIMIT", 10000, &parseErrs),
			HealthCheckTimeout: getEnvInt("HEALTH_CHECK_TIMEOUT", 3, &parseErrs),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", ""),
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			Name:            getEnv("DB_NAME", "stuhelper"),
			User:            getEnv("DB_USER", "stuhelper"),
			Password:        getEnv("DB_PASSWORD", ""),
			MaxConns:        getEnvInt("DB_MAX_CONNS", 20, &parseErrs),
			MinConns:        getEnvInt("DB_MIN_CONNS", 2, &parseErrs),
			MaxConnLifetime: getEnvInt("DB_MAX_CONN_LIFETIME", 30, &parseErrs),
			MaxConnIdleTime: getEnvInt("DB_MAX_CONN_IDLE_TIME", 5, &parseErrs),
			QueryTimeout:    getEnvInt("DB_QUERY_TIMEOUT", 5, &parseErrs),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			SSLRootCert:     getEnv("DB_SSL_ROOT_CERT", ""),
			SSLCert:         getEnv("DB_SSL_CERT", ""),
			SSLKey:          getEnv("DB_SSL_KEY", ""),
		},
		Casdoor: CasdoorConfig{
			Endpoint:     getEnv("CASDOOR_ENDPOINT", ""),
			ClientID:     getEnv("CASDOOR_CLIENT_ID", ""),
			ClientSecret: getEnv("CASDOOR_CLIENT_SECRET", ""),
			Certificate:  cert,
			Organization: getEnv("CASDOOR_ORGANIZATION", ""),
			Application:  getEnv("CASDOOR_APPLICATION", ""),
			RedirectURI:  getEnv("CASDOOR_REDIRECT_URI", ""),
		},
		Redis: RedisConfig{
			Host:         getEnv("REDIS_HOST", "localhost"),
			Port:         getEnv("REDIS_PORT", "6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvInt("REDIS_DB", 0, &parseErrs),
			PoolSize:     getEnvInt("REDIS_POOL_SIZE", 10, &parseErrs),
			MinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", 5, &parseErrs),
			TLSEnabled:   getEnvBool("REDIS_TLS_ENABLED", false, &parseErrs),
			TLSCertFile:  getEnv("REDIS_TLS_CERT", ""),
			TLSKeyFile:   getEnv("REDIS_TLS_KEY", ""),
			TLSCAFile:    getEnv("REDIS_TLS_CA", ""),
			TLSInsecure:  getEnvBool("REDIS_TLS_INSECURE", false, &parseErrs),
		},
		Token: TokenConfig{
			AccessTokenTTL:  getEnvInt("TOKEN_ACCESS_TTL", 900, &parseErrs),
			RefreshTokenTTL: getEnvInt("TOKEN_REFRESH_TTL", 604800, &parseErrs),
			CookieSecure:    getEnvBool("TOKEN_COOKIE_SECURE", false, &parseErrs),
			CookieDomain:    getEnv("TOKEN_COOKIE_DOMAIN", ""),
		},
		Log: LogConfig{
			Level:           getEnv("LOG_LEVEL", "info"),
			Format:          getEnv("LOG_FORMAT", "json"),
			Output:          getEnv("LOG_OUTPUT", "stdout"),
			SamplingEnabled: getEnvBool("LOG_SAMPLING_ENABLED", false, &parseErrs),
			SamplingInitial: getEnvInt("LOG_SAMPLING_INITIAL", 100, &parseErrs),
			SamplingAfter:   getEnvInt("LOG_SAMPLING_AFTER", 100, &parseErrs),
			FileEnabled:     getEnvBool("LOG_FILE_ENABLED", false, &parseErrs),
			FilePath:        getEnv("LOG_FILE_PATH", "logs/app.log"),
			FileMaxSize:     getEnvInt("LOG_FILE_MAX_SIZE", 100, &parseErrs),
			FileMaxBackups:  getEnvInt("LOG_FILE_MAX_BACKUPS", 3, &parseErrs),
			FileMaxAge:      getEnvInt("LOG_FILE_MAX_AGE", 7, &parseErrs),
			FileCompress:    getEnvBool("LOG_FILE_COMPRESS", true, &parseErrs),
		},
		RateLimit: ReviewRateLimitConfig{
			PostLimit:   getEnvInt("REVIEW_RATE_POST_LIMIT", 5, &parseErrs),
			VoteLimit:   getEnvInt("REVIEW_RATE_VOTE_LIMIT", 30, &parseErrs),
			ReportLimit: getEnvInt("REVIEW_RATE_REPORT_LIMIT", 10, &parseErrs),
			ReplyLimit:  getEnvInt("REVIEW_RATE_REPLY_LIMIT", 10, &parseErrs),
			WriteLimit:  getEnvInt("REVIEW_RATE_WRITE_LIMIT", 10, &parseErrs),
		},
	}

	securityCfg, securityErrs := parseSecurityConfig()
	cfg.Security = securityCfg
	parseErrs = append(parseErrs, securityErrs...)

	if cfg.Database.URL == "" && cfg.Database.Host != "" {
		cfg.Database.URL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			cfg.Database.User, url.QueryEscape(cfg.Database.Password),
			cfg.Database.Host, cfg.Database.Port,
			cfg.Database.Name, cfg.Database.SSLMode)
	} else if cfg.Database.URL != "" && cfg.Database.Host != "localhost" {
		log.Println("WARNING: Both DATABASE_URL and DB_HOST are set; DATABASE_URL takes priority. Individual DB_* fields will be ignored.")
	}

	if err := cfg.validate(parseErrs); err != nil {
		return nil, err
	}

	return cfg, nil
}
