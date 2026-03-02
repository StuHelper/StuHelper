// Package wire 提供依赖注入的 Provider 定义
package wire

import (
	"log"
	"time"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	redisv9 "github.com/redis/go-redis/v9"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/modules/auth"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/modules/course"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/health"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/redis"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/sso"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

// ConfigSet 配置相关的 Provider
var ConfigSet = wire.NewSet(
	config.Load,
	ProvideIsProduction,
)

// ProvideIsProduction 提供生产环境标志
func ProvideIsProduction(cfg *config.Config) bool {
	return cfg.App.Env == "production"
}

// InfraSet 基础设施相关的 Provider
var InfraSet = wire.NewSet(
	ProvideRedisClient,
	ProvidePGPool,
	ProvideDatabase,
	wire.Bind(new(redisv9.Cmdable), new(*redisv9.Client)),
)

// ProvideRedisClient 提供 Redis 客户端
func ProvideRedisClient(cfg *config.Config) (*redis.Client, func(), error) {
	client, err := redis.NewClient(cfg.Redis)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		if err := client.Close(); err != nil {
			// 关闭阶段无法恢复，仅记录
			log.Printf("WARN: failed to close Redis client: %v", err)
		}
	}
	return client, cleanup, nil
}

// ProvidePGPool 提供 PostgreSQL 连接池
func ProvidePGPool(cfg *config.Config) (*pgxpool.Pool, func(), error) {
	pool, err := db.NewPGPool(cfg.Database)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { pool.Close() }
	return pool, cleanup, nil
}

// ProvideDatabase 提供数据库封装
func ProvideDatabase(pool *pgxpool.Pool, cfg *config.Config) *db.DB {
	timeout := time.Duration(cfg.Database.QueryTimeout) * time.Second
	return db.NewDB(pool, timeout)
}

// ServiceSet 服务层相关的 Provider
var ServiceSet = wire.NewSet(
	ProvideTokenService,
)

// ProvideTokenService 提供 Token 服务
func ProvideTokenService(cfg *config.Config, rc *redis.Client) (*token.Service, error) {
	return token.NewService(token.ServiceConfig{
		RedisClient:    rc.GetClient(),
		AccessTTL:      cfg.Token.AccessTokenTTL,
		RefreshTTL:     cfg.Token.RefreshTokenTTL,
		JWTIssuer:      cfg.Casdoor.Endpoint,
		JWTAudience:    cfg.Casdoor.ClientID,
		JWTCertificate: cfg.Casdoor.Certificate,
	})
}

// HandlerSet Handler 层相关的 Provider
var HandlerSet = wire.NewSet(
	ProvideHealthHandler,
	ProvideAuthHandler,
	ProvideCourseHandler,
)

// ProvideHealthHandler 提供健康检查 Handler
func ProvideHealthHandler(
	pool *pgxpool.Pool,
	rc *redis.Client,
	isProduction bool,
) *health.Handler {
	return health.NewHandler(pool, rc.GetClient(), health.BuildInfo{
		Version:   "1.0.0",
		GitCommit: "unknown",
		BuildTime: "unknown",
	}, isProduction, 0)
}

// ProvideAuthHandler 提供认证 Handler
func ProvideAuthHandler(
	cfg *config.Config,
	ts *token.Service,
	rc *redis.Client,
	ssoClient *sso.Client,
) *auth.Handler {
	return auth.NewHandler(cfg, ts, rc.GetClient(), ssoClient)
}

// ProvideCourseHandler 提供课程 Handler
func ProvideCourseHandler(
	database *db.DB,
	rc *redis.Client,
	ssoClient *sso.Client,
	cfg *config.Config,
) *course.Handler {
	return course.NewHandler(database, rc.GetClient(), ssoClient, cfg)
}
