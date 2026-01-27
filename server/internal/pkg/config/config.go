package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
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
	Env         string
	Port        string
	CORSOrigins []string
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	URL             string
	Host            string
	Port            string
	Name            string
	User            string
	Password        string
	MaxConns        int // 最大连接数
	MinConns        int // 最小连接数
	MaxConnLifetime int // 连接最大生命周期（分钟）
	MaxConnIdleTime int // 连接最大空闲时间（分钟）
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
	PoolSize     int // 连接池大小
	MinIdleConns int // 最小空闲连接数
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
	cfg := &Config{
		App: AppConfig{
			Env:         getEnv("APP_ENV", "development"),
			Port:        getEnv("APP_PORT", "8080"),
			CORSOrigins: getEnvSlice("CORS_ORIGINS", []string{}),
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

// getEnvInt 获取整数类型的环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		log.Printf("warning: invalid integer value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}

// getEnvBool 获取布尔类型的环境变量
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
		log.Printf("warning: invalid boolean value for %s: %s, using default: %v", key, value, defaultValue)
	}
	return defaultValue
}
