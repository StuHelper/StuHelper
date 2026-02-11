package config

import (
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Config 应用配置
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Casdoor  CasdoorConfig
	Token    TokenConfig
	Log      LogConfig
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
	Env             string
	Port            string
	CORSOrigins     []string
	TrustedProxies  []string // 可信代理 IP 列表，用于正确获取客户端 IP
	HMACSecret      string   // HMAC 密钥，用于用户 ID 哈希等场景
	MaxBodySize     int64    // 请求体最大大小（字节）
	MetricsUser     string   // Prometheus metrics BasicAuth 用户名
	MetricsPassword string   // Prometheus metrics BasicAuth 密码
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
	configMu.Lock()
	defer configMu.Unlock()
	// 重置解析错误，防止多次调用时累积
	configParseErrors = nil

	cfg := &Config{
		App: AppConfig{
			Env:            getEnv("APP_ENV", "development"),
			Port:           getEnv("APP_PORT", "8080"),
			CORSOrigins:    getEnvSlice("CORS_ORIGINS", []string{}),
			TrustedProxies: getEnvSlice("TRUSTED_PROXIES", []string{}),
			HMACSecret:      getEnv("HMAC_SECRET", ""),
			MaxBodySize:     getEnvInt64("MAX_BODY_SIZE", 10<<20), // 默认 10MB
			MetricsUser:     getEnv("METRICS_USER", "prometheus"),
			MetricsPassword: getEnv("METRICS_PASSWORD", ""),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", ""),
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			Name:            getEnv("DB_NAME", "stuhelper"),
			User:            getEnv("DB_USER", "stuhelper"),
			Password:        getEnv("DB_PASSWORD", ""),
			MaxConns:        getEnvInt("DB_MAX_CONNS", 20),
			MinConns:        getEnvInt("DB_MIN_CONNS", 2),
			MaxConnLifetime: getEnvInt("DB_MAX_CONN_LIFETIME", 30),
			MaxConnIdleTime: getEnvInt("DB_MAX_CONN_IDLE_TIME", 5),
			QueryTimeout:    getEnvInt("DB_QUERY_TIMEOUT", 5),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			SSLRootCert:     getEnv("DB_SSL_ROOT_CERT", ""),
			SSLCert:         getEnv("DB_SSL_CERT", ""),
			SSLKey:          getEnv("DB_SSL_KEY", ""),
		},
		Casdoor: CasdoorConfig{
			Endpoint:     getEnv("CASDOOR_ENDPOINT", ""),
			ClientID:     getEnv("CASDOOR_CLIENT_ID", ""),
			ClientSecret: getEnv("CASDOOR_CLIENT_SECRET", ""),
			Certificate:  getEnv("CASDOOR_CERTIFICATE", ""),
			Organization: getEnv("CASDOOR_ORGANIZATION", ""),
			Application:  getEnv("CASDOOR_APPLICATION", ""),
			RedirectURI:  getEnv("CASDOOR_REDIRECT_URI", ""),
		},
		Redis: RedisConfig{
			Host:         getEnv("REDIS_HOST", "localhost"),
			Port:         getEnv("REDIS_PORT", "6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvInt("REDIS_DB", 0),
			PoolSize:     getEnvInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", 5),
			TLSEnabled:   getEnvBool("REDIS_TLS_ENABLED", false),
			TLSCertFile:  getEnv("REDIS_TLS_CERT", ""),
			TLSKeyFile:   getEnv("REDIS_TLS_KEY", ""),
			TLSCAFile:    getEnv("REDIS_TLS_CA", ""),
			TLSInsecure:  getEnvBool("REDIS_TLS_INSECURE", false),
		},
		Token: TokenConfig{
			AccessTokenTTL:  getEnvInt("TOKEN_ACCESS_TTL", 900),
			RefreshTokenTTL: getEnvInt("TOKEN_REFRESH_TTL", 604800),
			CookieSecure:    getEnvBool("TOKEN_COOKIE_SECURE", false),
			CookieDomain:    getEnv("TOKEN_COOKIE_DOMAIN", ""),
		},
		Log: LogConfig{
			Level:           getEnv("LOG_LEVEL", "info"),
			Format:          getEnv("LOG_FORMAT", "json"),
			Output:          getEnv("LOG_OUTPUT", "stdout"),
			SamplingEnabled: getEnvBool("LOG_SAMPLING_ENABLED", false),
			SamplingInitial: getEnvInt("LOG_SAMPLING_INITIAL", 100),
			SamplingAfter:   getEnvInt("LOG_SAMPLING_AFTER", 100),
			FileEnabled:     getEnvBool("LOG_FILE_ENABLED", false),
			FilePath:        getEnv("LOG_FILE_PATH", "logs/app.log"),
			FileMaxSize:     getEnvInt("LOG_FILE_MAX_SIZE", 100),
			FileMaxBackups:  getEnvInt("LOG_FILE_MAX_BACKUPS", 3),
			FileMaxAge:      getEnvInt("LOG_FILE_MAX_AGE", 7),
			FileCompress:    getEnvBool("LOG_FILE_COMPRESS", true),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate 验证配置是否完整
func (c *Config) Validate() error {
	var errs []string

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
		if c.App.HMACSecret == "" {
			errs = append(errs, "HMAC_SECRET is required in production")
		}
		if c.App.MetricsPassword == "" {
			errs = append(errs, "METRICS_PASSWORD is required in production")
		}
		// 生产环境强制 TLS
		if c.Database.SSLMode == "disable" || c.Database.SSLMode == "" {
			errs = append(errs, "DB_SSL_MODE must be 'require', 'verify-ca', or 'verify-full' in production")
		}
		if !c.Redis.TLSEnabled {
			log.Println("WARNING: Redis TLS is disabled in production. Ensure Redis is in a secure internal network.")
		}
		if c.Redis.TLSEnabled && c.Redis.TLSInsecure {
			errs = append(errs, "REDIS_TLS_INSECURE must not be true in production (certificate verification required)")
		}
		// 生产环境 fail-fast: 配置解析错误时直接退出
		if len(configParseErrors) > 0 {
			errs = append(errs, configParseErrors...)
		}
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
		errs = append(errs, "CASDOOR_CERTIFICATE is required")
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

// configParseErrors 收集配置解析错误
var (
	configParseErrors []string
	configMu          sync.Mutex
)

// getEnvInt 获取整数类型的环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		// 记录解析错误，在 Validate 中检查
		errMsg := fmt.Sprintf("invalid integer value for %s: %s", key, value)
		configParseErrors = append(configParseErrors, errMsg)
		log.Printf("warning: %s, using default: %d", errMsg, defaultValue)
	}
	return defaultValue
}

// getEnvBool 获取布尔类型的环境变量
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
		// 记录解析错误，在 Validate 中检查
		errMsg := fmt.Sprintf("invalid boolean value for %s: %s", key, value)
		configParseErrors = append(configParseErrors, errMsg)
		log.Printf("warning: %s, using default: %v", errMsg, defaultValue)
	}
	return defaultValue
}

// getEnvInt64 获取 int64 类型的环境变量
func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
		// 记录解析错误，在 Validate 中检查
		errMsg := fmt.Sprintf("invalid int64 value for %s: %s", key, value)
		configParseErrors = append(configParseErrors, errMsg)
		log.Printf("warning: %s, using default: %d", errMsg, defaultValue)
	}
	return defaultValue
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
