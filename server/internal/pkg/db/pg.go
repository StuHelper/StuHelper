package db

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPGPool 创建 PostgreSQL 连接池
func NewPGPool(cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid DATABASE_URL: %w", err)
	}

	// 使用配置值设置连接池参数
	poolCfg.MaxConns = int32(cfg.MaxConns) //nolint:gosec // G115: config values are validated
	poolCfg.MinConns = int32(cfg.MinConns) //nolint:gosec // G115: config values are validated
	poolCfg.MaxConnLifetime = time.Duration(cfg.MaxConnLifetime) * time.Minute
	poolCfg.MaxConnIdleTime = time.Duration(cfg.MaxConnIdleTime) * time.Minute
	poolCfg.HealthCheckPeriod = 1 * time.Minute

	// 配置 TLS
	if err := configurePGTLS(poolCfg, cfg); err != nil {
		return nil, fmt.Errorf("failed to configure TLS: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create pg pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	return pool, nil
}

// configurePGTLS 配置 PostgreSQL TLS
func configurePGTLS(poolCfg *pgxpool.Config, cfg config.DatabaseConfig) error {
	switch cfg.SSLMode {
	case "", "disable":
		// 不使用 TLS
		return nil
	case "require":
		// 要求 TLS，但不验证证书
		poolCfg.ConnConfig.TLSConfig = &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // require mode skips verification
		}
	case "verify-ca", "verify-full":
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		// 加载 CA 证书
		if cfg.SSLRootCert != "" {
			caCert, err := os.ReadFile(cfg.SSLRootCert)
			if err != nil {
				return fmt.Errorf("failed to read CA cert: %w", err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return fmt.Errorf("failed to parse CA cert")
			}
			tlsConfig.RootCAs = caCertPool
		}

		// 加载客户端证书（如果提供）
		if cfg.SSLCert != "" && cfg.SSLKey != "" {
			cert, err := tls.LoadX509KeyPair(cfg.SSLCert, cfg.SSLKey)
			if err != nil {
				return fmt.Errorf("failed to load client cert: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		// verify-full 需要验证服务器主机名
		if cfg.SSLMode == "verify-full" {
			tlsConfig.ServerName = poolCfg.ConnConfig.Host
		}

		poolCfg.ConnConfig.TLSConfig = tlsConfig
	default:
		return fmt.Errorf("invalid SSL mode: %s", cfg.SSLMode)
	}
	return nil
}
