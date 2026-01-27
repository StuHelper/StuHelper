package db

import (
	"context"
	"fmt"
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
