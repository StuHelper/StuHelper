package token

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/circuitbreaker"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

const (
	// 黑名单 key 前缀
	blacklistPrefix = "token:blacklist:"
	// 用户 token 集合前缀
	userTokensPrefix = "token:user:"
)

// Blacklist Token 黑名单服务
type Blacklist struct {
	rdb *redis.Client
	cb  *circuitbreaker.CircuitBreaker
}

// NewBlacklist 创建黑名单服务
func NewBlacklist(rdb *redis.Client) *Blacklist {
	return &Blacklist{
		rdb: rdb,
		cb: circuitbreaker.New(circuitbreaker.Config{
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          30 * time.Second,
		}),
	}
}

// hashToken 对 token 进行 SHA256 哈希，减少 Redis 内存占用
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Add 将 token 加入黑名单
func (b *Blacklist) Add(ctx context.Context, token string, expiry time.Duration) error {
	key := blacklistPrefix + hashToken(token)
	return b.rdb.Set(ctx, key, "1", expiry).Err()
}

// IsBlacklisted 检查 token 是否在黑名单中
// 使用熔断器模式：Redis 持续故障时降级运行，避免服务完全不可用
func (b *Blacklist) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	// 检查熔断器状态
	if !b.cb.Allow() {
		// 熔断器打开：安全优先 - 拒绝请求，防止已撤销 token 被放行
		logger.L().Warn("circuit breaker open, denying request (fail-closed)",
			zap.String("operation", "IsBlacklisted"),
		)
		return true, fmt.Errorf("blacklist service unavailable (circuit breaker open)")
	}

	key := blacklistPrefix + hashToken(token)
	exists, err := b.rdb.Exists(ctx, key).Result()
	if err != nil {
		b.cb.RecordFailure()
		logger.L().Warn("redis error checking blacklist",
			zap.Error(err),
			zap.String("operation", "IsBlacklisted"),
			zap.String("circuit_state", b.cb.State().String()),
		)
		// 安全优先：Redis 错误时拒绝请求
		return true, fmt.Errorf("failed to check blacklist: %w", err)
	}

	b.cb.RecordSuccess()
	return exists > 0, nil
}

// RevokeAllUserTokens 撤销用户的所有 token
func (b *Blacklist) RevokeAllUserTokens(ctx context.Context, userID string, expiry time.Duration) error {
	key := userTokensPrefix + userID
	tokenHashes, err := b.rdb.SMembers(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to get user tokens: %w", err)
	}

	if len(tokenHashes) == 0 {
		return nil
	}

	pipe := b.rdb.Pipeline()
	for _, tokenHash := range tokenHashes {
		pipe.Set(ctx, blacklistPrefix+tokenHash, "1", expiry)
	}
	pipe.Del(ctx, key)

	_, err = pipe.Exec(ctx)
	return err
}

// TrackUserToken 记录用户的 token（用于批量撤销）
func (b *Blacklist) TrackUserToken(ctx context.Context, userID, token string, expiry time.Duration) error {
	key := userTokensPrefix + userID
	tokenHash := hashToken(token)
	pipe := b.rdb.Pipeline()
	pipe.SAdd(ctx, key, tokenHash)
	pipe.Expire(ctx, key, expiry)
	_, err := pipe.Exec(ctx)
	return err
}

// CircuitBreakerMetrics 获取熔断器指标（用于监控）
func (b *Blacklist) CircuitBreakerMetrics() map[string]interface{} {
	return b.cb.Metrics()
}
