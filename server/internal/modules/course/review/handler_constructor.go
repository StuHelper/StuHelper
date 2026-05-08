package review

import (
	"time"

	"github.com/redis/go-redis/v9"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

type HandlerConfig struct {
	CacheHelper            *cache.Helper
	Service                *Service
	Redis                  *redis.Client
	RateLimit              config.ReviewRateLimitConfig
	Authorizer             AuthorizationProvider
	InternalUserIDResolver middleware.InternalUserIDResolver
}

// NewHandler 创建评课处理器。Admin mutation 授权需要 Authorizer 与
// InternalUserIDResolver 同时配置，缺一即启动失败。
func NewHandler(cfg HandlerConfig) *Handler {
	validateHandlerConfig(cfg)
	return &Handler{
		cache:                  cfg.CacheHelper,
		service:                cfg.Service,
		fga:                    cfg.Authorizer,
		internalUserIDResolver: cfg.InternalUserIDResolver,
		postLimiter:            middleware.NewRedisRateLimiter(cfg.Redis, cfg.RateLimit.PostLimit, time.Minute),
		voteLimiter:            middleware.NewRedisRateLimiter(cfg.Redis, cfg.RateLimit.VoteLimit, time.Minute),
		reportLimiter:          middleware.NewRedisRateLimiter(cfg.Redis, cfg.RateLimit.ReportLimit, time.Minute),
		replyLimiter:           middleware.NewRedisRateLimiter(cfg.Redis, cfg.RateLimit.ReplyLimit, time.Minute),
		writeLimiter:           middleware.NewRedisRateLimiter(cfg.Redis, cfg.RateLimit.WriteLimit, time.Minute),
		searchAnonLimiter:      middleware.NewRedisRateLimiter(cfg.Redis, cfg.RateLimit.SearchAnonLimit, time.Minute),
		searchUserLimiter:      middleware.NewRedisRateLimiter(cfg.Redis, cfg.RateLimit.SearchUserLimit, time.Minute),
		batchAnonLimiter:       middleware.NewRedisRateLimiter(cfg.Redis, cfg.RateLimit.BatchAnonLimit, time.Minute),
		batchUserLimiter:       middleware.NewRedisRateLimiter(cfg.Redis, cfg.RateLimit.BatchUserLimit, time.Minute),
	}
}

func validateHandlerConfig(cfg HandlerConfig) {
	switch {
	case cfg.CacheHelper == nil:
		panic("review.NewHandler: cacheHelper must not be nil")
	case cfg.Service == nil:
		panic("review.NewHandler: service must not be nil")
	case cfg.Redis == nil:
		panic("review.NewHandler: redis client must not be nil")
	case cfg.Authorizer == nil:
		panic("review.NewHandler: authorizer must not be nil")
	case cfg.InternalUserIDResolver == nil:
		panic("review.NewHandler: internal user id resolver must not be nil")
	}
}
