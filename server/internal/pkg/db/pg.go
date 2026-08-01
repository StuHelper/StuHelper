package db

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
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
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = time.Duration(cfg.MaxConnLifetime) * time.Minute // cfg 单位：分钟
	poolCfg.MaxConnIdleTime = time.Duration(cfg.MaxConnIdleTime) * time.Minute // cfg 单位：分钟
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
	if poolCfg == nil || poolCfg.ConnConfig == nil {
		return fmt.Errorf("pg pool configuration is required")
	}

	mode := strings.ToLower(strings.TrimSpace(cfg.SSLMode))
	switch mode {
	case "", "disable":
		poolCfg.ConnConfig.TLSConfig = nil
		return rewritePGFallbackTLS(poolCfg, nil)
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

		// 加载客户端证书（如果提供）。只配置一侧必须失败，不能静默退化为
		// 无客户端证书连接。
		if (cfg.SSLCert == "") != (cfg.SSLKey == "") {
			return fmt.Errorf("both DB_SSL_CERT and DB_SSL_KEY are required for client certificate authentication")
		}
		if cfg.SSLCert != "" {
			cert, err := tls.LoadX509KeyPair(cfg.SSLCert, cfg.SSLKey)
			if err != nil {
				return fmt.Errorf("failed to load client cert: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		// pgx 的 verify-ca 语义只验证证书链，不验证主机名。Go 标准库没有
		// 对应开关，因此跳过默认验证后在 VerifyConnection 中显式验证链。
		if mode == "verify-ca" {
			tlsConfig.InsecureSkipVerify = true // #nosec G402 -- certificate chains are verified below while hostname verification is intentionally omitted.
			tlsConfig.VerifyConnection = pgCertificateChainVerifier(tlsConfig.RootCAs)
		}
		if poolCfg.ConnConfig.SSLNegotiation == "direct" {
			tlsConfig.NextProtos = []string{"postgresql"}
		}

		if err := applyPGTLSConfig(poolCfg.ConnConfig.Host, poolCfg.ConnConfig.Port, tlsConfig, &poolCfg.ConnConfig.TLSConfig); err != nil {
			return err
		}
		if err := rewritePGFallbackTLS(poolCfg, tlsConfig); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid SSL mode: %s", cfg.SSLMode)
	}
	return nil
}

func rewritePGFallbackTLS(poolCfg *pgxpool.Config, template *tls.Config) error {
	seen := map[string]struct{}{
		pgEndpointKey(poolCfg.ConnConfig.Host, poolCfg.ConnConfig.Port): {},
	}
	fallbacks := make([]*pgconn.FallbackConfig, 0, len(poolCfg.ConnConfig.Fallbacks))
	for _, fallback := range poolCfg.ConnConfig.Fallbacks {
		if fallback == nil {
			continue
		}
		key := pgEndpointKey(fallback.Host, fallback.Port)
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}

		configured := &pgconn.FallbackConfig{
			Host: fallback.Host,
			Port: fallback.Port,
		}
		if template != nil {
			network, _ := pgconn.NetworkAddress(fallback.Host, fallback.Port)
			if network == "unix" {
				return fmt.Errorf("verified PostgreSQL TLS cannot use Unix socket fallback host %q", fallback.Host)
			}
			configured.TLSConfig = template.Clone()
			configured.TLSConfig.ServerName = fallback.Host
		}
		fallbacks = append(fallbacks, configured)
	}
	poolCfg.ConnConfig.Fallbacks = fallbacks
	return nil
}

func applyPGTLSConfig(host string, port uint16, template *tls.Config, destination **tls.Config) error {
	network, _ := pgconn.NetworkAddress(host, port)
	if network == "unix" {
		return fmt.Errorf("verified PostgreSQL TLS cannot use Unix socket host %q", host)
	}
	configured := template.Clone()
	configured.ServerName = host
	*destination = configured
	return nil
}

func pgEndpointKey(host string, port uint16) string {
	return host + "\x00" + strconv.FormatUint(uint64(port), 10)
}

func pgCertificateChainVerifier(roots *x509.CertPool) func(tls.ConnectionState) error {
	return func(state tls.ConnectionState) error {
		if len(state.PeerCertificates) == 0 {
			return fmt.Errorf("postgres TLS peer did not provide a certificate")
		}
		intermediates := x509.NewCertPool()
		for _, certificate := range state.PeerCertificates[1:] {
			intermediates.AddCert(certificate)
		}
		_, err := state.PeerCertificates[0].Verify(x509.VerifyOptions{
			Roots:         roots,
			Intermediates: intermediates,
			KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		})
		if err != nil {
			return fmt.Errorf("verify postgres TLS certificate chain: %w", err)
		}
		return nil
	}
}
