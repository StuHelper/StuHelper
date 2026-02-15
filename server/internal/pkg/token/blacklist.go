package token

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/circuitbreaker"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

const (
	// 黑名单 key 前缀
	blacklistPrefix = "token:blacklist:"
	// 用户 token 集合前缀
	userTokensPrefix = "token:user:"
)

// localCacheEntry 本地缓存条目
type localCacheEntry struct {
	blacklisted bool
	expiresAt   time.Time
}

// localCacheTTL 本地缓存 TTL（短时间缓存，仅用于 Redis 不可用时降级）
const localCacheTTL = 30 * time.Second

const (
	// minBlacklistTTL 黑名单 TTL 最小值
	minBlacklistTTL = 1 * time.Second
	// maxBlacklistTTL 黑名单 TTL 最大值（30 天）
	maxBlacklistTTL = 30 * 24 * time.Hour
)

// Blacklist Token 黑名单服务
type Blacklist struct {
	rdb        *redis.Client
	cb         *circuitbreaker.CircuitBreaker
	localCache sync.Map // map[string]localCacheEntry
	stopCh     chan struct{}
}

// NewBlacklist 创建黑名单服务
func NewBlacklist(rdb *redis.Client) *Blacklist {
	b := &Blacklist{
		rdb: rdb,
		cb: circuitbreaker.New(circuitbreaker.Config{
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          30 * time.Second,
		}),
		stopCh: make(chan struct{}),
	}
	// 定期清理过期的本地缓存条目，防止 sync.Map 无限增长
	go b.cleanupLoop()
	return b
}

// Close 优雅关闭黑名单服务，停止后台清理 goroutine
func (b *Blacklist) Close() {
	close(b.stopCh)
}

// cleanupLoop 定期清理过期的本地缓存条目
func (b *Blacklist) cleanupLoop() {
	ticker := time.NewTicker(localCacheTTL * 2)
	defer ticker.Stop()
	for {
		select {
		case <-b.stopCh:
			return
		case <-ticker.C:
			now := time.Now()
			b.localCache.Range(func(key, value any) bool {
				if entry, ok := value.(localCacheEntry); ok && now.After(entry.expiresAt) {
					b.localCache.Delete(key)
				}
				return true
			})
		}
	}
}

// hashToken 使用 HMAC-SHA256 对 token 进行哈希，减少 Redis 内存占用
func hashToken(token string) (string, error) {
	return crypto.HMACHash(token)
}

// Add 将 token 加入黑名单
func (b *Blacklist) Add(ctx context.Context, token string, expiry time.Duration) error {
	if expiry < minBlacklistTTL || expiry > maxBlacklistTTL {
		return fmt.Errorf("blacklist TTL %v out of valid range [%v, %v]", expiry, minBlacklistTTL, maxBlacklistTTL)
	}

	hash, err := hashToken(token)
	if err != nil {
		return fmt.Errorf("failed to hash token: %w", err)
	}

	// 写入本地缓存（无论 Redis 是否可用，确保当前实例立即生效）
	b.localCache.Store(hash, localCacheEntry{
		blacklisted: true,
		expiresAt:   time.Now().Add(localCacheTTL),
	})

	if !b.cb.Allow() {
		return fmt.Errorf("blacklist service unavailable (circuit breaker open)")
	}

	key := blacklistPrefix + hash
	if err := b.rdb.Set(ctx, key, "1", expiry).Err(); err != nil {
		b.cb.RecordFailure()
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}
	b.cb.RecordSuccess()
	return nil
}

// IsBlacklisted 检查 token 是否在黑名单中
// 使用熔断器模式：Redis 持续故障时降级到本地缓存，避免服务完全不可用
func (b *Blacklist) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	hash, err := hashToken(token)
	if err != nil {
		// 哈希失败是配置错误，不应将 token 视为已黑名单；返回 false 让调用方区分处理
		return false, fmt.Errorf("failed to hash token: %w", err)
	}

	// 检查熔断器状态
	if !b.cb.Allow() {
		// 熔断器打开：尝试本地缓存降级
		if entry, ok := b.localCache.Load(hash); ok {
			cached, ok := entry.(localCacheEntry)
			if !ok {
				b.localCache.Delete(hash)
				return true, fmt.Errorf("blacklist service unavailable (circuit breaker open)")
			}
			if time.Now().Before(cached.expiresAt) {
				logger.L().Warn("circuit breaker open, using local cache fallback",
					zap.String("operation", "IsBlacklisted"),
				)
				return cached.blacklisted, nil
			}
			// 缓存过期，原子清理（仅当值未被其他 goroutine 更新时才删除）
			b.localCache.CompareAndDelete(hash, entry)
		}
		// 无本地缓存可用：安全优先 - 拒绝请求
		logger.L().Warn("circuit breaker open, no local cache, denying request (fail-closed)",
			zap.String("operation", "IsBlacklisted"),
		)
		return true, fmt.Errorf("blacklist service unavailable (circuit breaker open)")
	}

	key := blacklistPrefix + hash
	exists, err := b.rdb.Exists(ctx, key).Result()
	if err != nil {
		b.cb.RecordFailure()
		logger.L().Warn("redis unavailable for blacklist check",
			zap.String("operation", "IsBlacklisted"),
			zap.String("circuit_state", b.cb.State().String()),
		)

		// Redis 错误时尝试本地缓存降级
		if entry, ok := b.localCache.Load(hash); ok {
			cached := entry.(localCacheEntry)
			if time.Now().Before(cached.expiresAt) {
				return cached.blacklisted, nil
			}
			b.localCache.CompareAndDelete(hash, entry)
		}

		// 安全优先：无缓存时拒绝请求
		return true, fmt.Errorf("blacklist service unavailable")
	}

	b.cb.RecordSuccess()
	blacklisted := exists > 0

	// 更新本地缓存
	b.localCache.Store(hash, localCacheEntry{
		blacklisted: blacklisted,
		expiresAt:   time.Now().Add(localCacheTTL),
	})

	return blacklisted, nil
}

// RevokeAllUserTokens 撤销用户的所有 token
func (b *Blacklist) RevokeAllUserTokens(ctx context.Context, userID string, expiry time.Duration) error {
	if !b.cb.Allow() {
		return fmt.Errorf("RevokeAllUserTokens: blacklist service unavailable (circuit breaker open)")
	}

	key := userTokensPrefix + userID
	tokenHashes, err := b.rdb.SMembers(ctx, key).Result()
	if err != nil {
		b.cb.RecordFailure()
		return fmt.Errorf("RevokeAllUserTokens: failed to get user tokens: %w", err)
	}

	if len(tokenHashes) == 0 {
		b.cb.RecordSuccess()
		return nil
	}

	pipe := b.rdb.Pipeline()
	for _, tokenHash := range tokenHashes {
		pipe.Set(ctx, blacklistPrefix+tokenHash, "1", expiry)
	}
	pipe.Del(ctx, key)

	_, err = pipe.Exec(ctx)
	if err != nil {
		b.cb.RecordFailure()
		return fmt.Errorf("RevokeAllUserTokens: failed to execute pipeline: %w", err)
	}
	b.cb.RecordSuccess()

	// 同步更新本地缓存，避免 IsBlacklisted 在 TTL 窗口内返回旧的 false
	for _, tokenHash := range tokenHashes {
		b.localCache.Store(tokenHash, localCacheEntry{
			blacklisted: true,
			expiresAt:   time.Now().Add(localCacheTTL),
		})
	}

	return nil
}

// TrackUserToken 记录用户的 token（用于批量撤销）
func (b *Blacklist) TrackUserToken(ctx context.Context, userID, token string, expiry time.Duration) error {
	if !b.cb.Allow() {
		return fmt.Errorf("TrackUserToken: blacklist service unavailable (circuit breaker open)")
	}

	key := userTokensPrefix + userID
	tokenHash, err := hashToken(token)
	if err != nil {
		return fmt.Errorf("failed to hash token: %w", err)
	}
	pipe := b.rdb.Pipeline()
	saddCmd := pipe.SAdd(ctx, key, tokenHash)
	expireCmd := pipe.Expire(ctx, key, expiry)
	_, err = pipe.Exec(ctx)
	if err != nil {
		b.cb.RecordFailure()
		return fmt.Errorf("TrackUserToken: pipeline exec failed: %w", err)
	}

	// 检查各命令是否成功
	if saddErr := saddCmd.Err(); saddErr != nil {
		b.cb.RecordFailure()
		return fmt.Errorf("TrackUserToken: SAdd failed: %w", saddErr)
	}
	if expireErr := expireCmd.Err(); expireErr != nil {
		logger.L().Warn("TrackUserToken: Expire failed, token set may not expire",
			zap.String("user_id", userID),
			zap.Error(expireErr),
		)
	}

	b.cb.RecordSuccess()
	return nil
}

// CircuitBreakerMetrics 获取熔断器指标（用于监控）
func (b *Blacklist) CircuitBreakerMetrics() map[string]any {
	return b.cb.Metrics()
}
