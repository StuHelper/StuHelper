package redis

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

// Client Redis 客户端封装
type Client struct {
	rdb       *redis.Client
	stopCh    chan struct{}
	closeOnce sync.Once
}

// NewClient 创建 Redis 客户端
func NewClient(cfg config.RedisConfig) (*Client, error) {
	opts := &redis.Options{
		Addr:            fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		MaxRetries:      2,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
	}

	// 配置 TLS
	if cfg.TLSEnabled {
		tlsConfig, err := configureRedisTLS(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
		opts.TLSConfig = tlsConfig
	}

	rdb := redis.NewClient(opts)

	// 测试连接（带超时，3s 足够检测连通性）
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	client := &Client{rdb: rdb, stopCh: make(chan struct{})}
	go client.collectPoolMetrics()
	return client, nil
}

// GetClient 获取原始 Redis 客户端
func (c *Client) GetClient() *redis.Client {
	return c.rdb
}

// Close 关闭连接（通过 sync.Once 保证幂等，避免重复 close(stopCh) panic）
func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.stopCh)
		err = c.rdb.Close()
	})
	return err
}

// collectPoolMetrics 定期采集 Redis 连接池指标
func (c *Client) collectPoolMetrics() {
	// H-34: panic recovery，防止后台 goroutine panic 导致进程崩溃
	defer func() {
		if r := recover(); r != nil {
			// 使用 fmt 而非 logger，因为 logger 可能依赖 Redis
			fmt.Fprintf(os.Stderr, "redis: collectPoolMetrics goroutine panicked: %v\n", r)
		}
	}()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			stats := c.rdb.PoolStats()
			metrics.RedisPoolHits.Set(float64(stats.Hits))
			metrics.RedisPoolMisses.Set(float64(stats.Misses))
			metrics.RedisPoolTimeouts.Set(float64(stats.Timeouts))
			metrics.RedisPoolTotalConns.Set(float64(stats.TotalConns))
			metrics.RedisPoolIdleConns.Set(float64(stats.IdleConns))
			metrics.RedisPoolStaleConns.Set(float64(stats.StaleConns))
		}
	}
}

// configureRedisTLS 配置 Redis TLS
func configureRedisTLS(cfg config.RedisConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: cfg.TLSInsecure, //nolint:gosec // configurable for testing
	}

	// 加载 CA 证书
	if cfg.TLSCAFile != "" {
		caCert, err := os.ReadFile(cfg.TLSCAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA cert")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// 加载客户端证书（如果提供）
	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client cert: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}
