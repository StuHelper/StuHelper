package config

import (
	"encoding/pem"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
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
}

// ReviewRateLimitConfig 评课模块端点限流配置
type ReviewRateLimitConfig struct {
	PostLimit   int // 每分钟发布评论上限
	VoteLimit   int // 每分钟投票上限
	ReportLimit int // 每分钟举报上限
	ReplyLimit  int // 每分钟回复上限
	WriteLimit  int // 每分钟更新/删除上限
}

// LogConfig 日志配置
type LogConfig struct {
	Level           string
	Format          string // json, console
	Output          string // stdout, stderr
	SamplingEnabled bool
	SamplingInitial int
	SamplingAfter   int
	FileEnabled     bool
	FilePath        string
	FileMaxSize     int // MB
	FileMaxBackups  int
	FileMaxAge      int // days
	FileCompress    bool
}

// AppConfig 应用配置
type AppConfig struct {
	Env                string
	Port               string
	CORSOrigins        []string
	TrustedProxies     []string // 可信代理 IP 列表，用于正确获取客户端 IP
	HMACSecret         string   // HMAC 密钥，用于用户 ID 哈希等场景
	MaxBodySize        int64    // 请求体最大大小（字节）
	MetricsUser        string   // Prometheus metrics BasicAuth 用户名
	MetricsPassword    string   // Prometheus metrics BasicAuth 密码
	APIIPRateLimit     int      // 每 IP 每分钟 API 请求上限
	APIGlobalLimit     int      // 全局每分钟 API 请求上限
	HealthCheckTimeout int      // 健康检查探测超时（秒），默认 3
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	URL             string
	Host            string
	Port            string
	Name            string
	User            string
	Password        string
	MaxConns        int    // 最大连接数
	MinConns        int    // 最小连接数
	MaxConnLifetime int    // 连接最大生命周期（分钟）
	MaxConnIdleTime int    // 连接最大空闲时间（分钟）
	QueryTimeout    int    // 查询超时时间（秒）
	SSLMode         string // TLS 模式: disable, require, verify-ca, verify-full
	SSLRootCert     string // CA 证书路径
	SSLCert         string // 客户端证书路径
	SSLKey          string // 客户端私钥路径
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
	PoolSize     int    // 连接池大小
	MinIdleConns int    // 最小空闲连接数
	TLSEnabled   bool   // 是否启用 TLS
	TLSCertFile  string // TLS 证书文件路径
	TLSKeyFile   string // TLS 私钥文件路径
	TLSCAFile    string // TLS CA 证书文件路径
	TLSInsecure  bool   // 是否跳过证书验证（仅用于测试）
}

// TokenConfig Token 配置
type TokenConfig struct {
	AccessTokenTTL  int  // Access Token 过期时间（秒）
	RefreshTokenTTL int  // Refresh Token 过期时间（秒）
	CookieSecure    bool // Cookie 是否仅 HTTPS
	CookieDomain    string
}

// Load 从环境变量加载配置
func Load() (*Config, error) {
	// parseErrs 为局部变量，收集解析阶段的错误，传递给 helper 函数
	var parseErrs []string

	cert, certErr := loadCertificate()
	if certErr != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", certErr)
	}

	cfg := &Config{
		App: AppConfig{
			Env:            getEnv("APP_ENV", "development"),
			Port:           getEnv("APP_PORT", "8080"),
			CORSOrigins:    getEnvSlice("CORS_ORIGINS", []string{}),
			TrustedProxies: getEnvSlice("TRUSTED_PROXIES", []string{}),
			HMACSecret:      getEnv("HMAC_SECRET", ""),
			MaxBodySize:     getEnvInt64("MAX_BODY_SIZE", 10<<20, &parseErrs), // 默认 10MB
			MetricsUser:     getEnv("METRICS_USER", "prometheus"),
			MetricsPassword: getEnv("METRICS_PASSWORD", ""),
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

	// DATABASE_URL 优先；未设置时从 DB_HOST/DB_PORT 等字段构建
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

// validate 验证配置是否完整（parseErrs 为 Load 阶段收集的解析错误）
func (c *Config) validate(parseErrs []string) error {
	var errs []string

	// ---- HMAC 密钥校验（统一检查：先判空，再判长度，按环境决定报错或警告） ----
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
			log.Printf("WARNING: HMAC_SECRET is shorter than %d characters (%d), consider using a stronger secret", hmacMinLen, len(c.App.HMACSecret))
		}
	}

	// ---- 数据库连接参数范围校验 ----
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

	// ---- Token TTL 范围校验 ----
	if c.Token.AccessTokenTTL < 60 || c.Token.AccessTokenTTL > 86400 {
		errs = append(errs, fmt.Sprintf("TOKEN_ACCESS_TTL must be between 60 and 86400 seconds (got %d)", c.Token.AccessTokenTTL))
	}
	if c.Token.RefreshTokenTTL < 3600 || c.Token.RefreshTokenTTL > 2592000 {
		errs = append(errs, fmt.Sprintf("TOKEN_REFRESH_TTL must be between 3600 and 2592000 seconds (got %d)", c.Token.RefreshTokenTTL))
	}

	// 生产环境额外验证
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
		// 生产环境强制 TLS
		if c.Database.SSLMode == "disable" || c.Database.SSLMode == "" {
			errs = append(errs, "DB_SSL_MODE must be 'require', 'verify-ca', or 'verify-full' in production")
		} else if c.Database.SSLMode == "require" {
			log.Println("WARNING: DB_SSL_MODE=require skips certificate verification (MITM risk). Consider 'verify-ca' or 'verify-full' for production.")
		}
		if !c.Redis.TLSEnabled {
			log.Println("WARNING: Redis TLS is disabled in production. Ensure Redis is in a secure internal network.")
		}
		if c.Redis.TLSEnabled && c.Redis.TLSInsecure {
			errs = append(errs, "REDIS_TLS_INSECURE must not be true in production (certificate verification required)")
		}
		// 生产环境 fail-fast: 配置解析错误时直接退出
		if len(parseErrs) > 0 {
			errs = append(errs, parseErrs...)
		}
	} else if len(parseErrs) > 0 {
		// 非生产环境：不阻断启动，但显著警告，避免开发者忽略配置错误
		log.Printf("WARNING: %d config parse error(s) detected (using defaults): %s",
			len(parseErrs), strings.Join(parseErrs, "; "))
	}

	// 验证 Casdoor 配置
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
	} else {
		// 验证证书格式
		if err := validatePEMCertificate(c.Casdoor.Certificate); err != nil {
			errs = append(errs, fmt.Sprintf("CASDOOR_CERTIFICATE format invalid: %v", err))
		}
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

	// 验证限流配置（必须 > 0 且 <= 100000，防止配置错误导致限流失效）
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

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvSlice 获取逗号分隔的环境变量值
func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return defaultValue
}

// getEnvInt 获取整数类型的环境变量
func getEnvInt(key string, defaultValue int, parseErrs *[]string) int {
	val := os.Getenv(key)
	if val == "" {
		// 环境变量显式设置为空字符串时警告
		if _, exists := os.LookupEnv(key); exists {
			log.Printf("WARNING: %s is set but empty, using default %d", key, defaultValue)
		}
		return defaultValue
	}
	if intValue, err := strconv.Atoi(val); err == nil {
		return intValue
	}
	// 记录解析错误，在 Validate 中检查
	errMsg := fmt.Sprintf("invalid integer value for %s: %s", key, val)
	*parseErrs = append(*parseErrs, errMsg)
	log.Printf("warning: %s, using default: %d", errMsg, defaultValue)
	return defaultValue
}

// getEnvBool 获取布尔类型的环境变量
func getEnvBool(key string, defaultValue bool, parseErrs *[]string) bool {
	val := os.Getenv(key)
	if val == "" {
		// 环境变量显式设置为空字符串时警告
		if _, exists := os.LookupEnv(key); exists {
			log.Printf("WARNING: %s is set but empty, using default %v", key, defaultValue)
		}
		return defaultValue
	}
	if boolValue, err := strconv.ParseBool(val); err == nil {
		return boolValue
	}
	// 记录解析错误，在 Validate 中检查
	errMsg := fmt.Sprintf("invalid boolean value for %s: %s", key, val)
	*parseErrs = append(*parseErrs, errMsg)
	log.Printf("warning: %s, using default: %v", errMsg, defaultValue)
	return defaultValue
}

// getEnvInt64 获取 int64 类型的环境变量
func getEnvInt64(key string, defaultValue int64, parseErrs *[]string) int64 {
	val := os.Getenv(key)
	if val == "" {
		// 环境变量显式设置为空字符串时警告
		if _, exists := os.LookupEnv(key); exists {
			log.Printf("WARNING: %s is set but empty, using default %d", key, defaultValue)
		}
		return defaultValue
	}
	if intValue, err := strconv.ParseInt(val, 10, 64); err == nil {
		return intValue
	}
	// 记录解析错误，在 Validate 中检查
	errMsg := fmt.Sprintf("invalid int64 value for %s: %s", key, val)
	*parseErrs = append(*parseErrs, errMsg)
	log.Printf("warning: %s, using default: %d", errMsg, defaultValue)
	return defaultValue
}

// loadCertificate 加载 Casdoor 证书，优先从环境变量读取，其次从文件读取
func loadCertificate() (string, error) {
	if cert := os.Getenv("CASDOOR_CERTIFICATE"); cert != "" {
		return cert, nil
	}
	if path := os.Getenv("CASDOOR_CERTIFICATE_FILE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read CASDOOR_CERTIFICATE_FILE %s: %w", path, err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	return "", nil
}

// validatePEMCertificate 验证 PEM 格式的证书
func validatePEMCertificate(cert string) error {
	// 检查基本的 PEM 格式标记
	if !strings.Contains(cert, "-----BEGIN") {
		return fmt.Errorf("missing PEM header (-----BEGIN)")
	}
	if !strings.Contains(cert, "-----END") {
		return fmt.Errorf("missing PEM footer (-----END)")
	}
	// 尝试解码 PEM 块
	block, _ := pem.Decode([]byte(cert))
	if block == nil {
		return fmt.Errorf("failed to decode PEM block")
	}
	// 验证是否为证书或公钥
	validTypes := []string{"CERTIFICATE", "PUBLIC KEY", "RSA PUBLIC KEY"}
	isValid := false
	for _, t := range validTypes {
		if block.Type == t {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("unexpected PEM block type: %s (expected CERTIFICATE or PUBLIC KEY)", block.Type)
	}
	return nil
}
