package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
)

// Client Redis 客户端封装
type Client struct {
	rdb *redis.Client
}

// NewClient 创建 Redis 客户端
func NewClient(cfg config.RedisConfig) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// 测试连接
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Client{rdb: rdb}, nil
}

// GetClient 获取原始 Redis 客户端
func (c *Client) GetClient() *redis.Client {
	return c.rdb
}

// Close 关闭连接
func (c *Client) Close() error {
	return c.rdb.Close()
}
