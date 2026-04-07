package token

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/circuitbreaker"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

const (
	// 黑名单 key 前缀
	blacklistPrefix = "token:blacklist:"
	// 已消费的一次性 refresh token 标记前缀
	refreshConsumedPrefix = "token:refresh:consumed:"
	// 用户 token 集合前缀（ZSET: member = type:hash, score = expiry unix timestamp）
	userTokensPrefix = "token:user:"
)

// TokenType 标识 token 类型（用于 ZSET member 前缀）
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
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
	// maxTrackingKeyTTL ZSET key 的安全网 TTL，防止孤立 key 永不过期。
	// 实际过期成员由 ZREMRANGEBYSCORE 清理。
	maxTrackingKeyTTL = 30 * 24 * time.Hour
)

// Blacklist Token 黑名单服务
type Blacklist struct {
	rdb        *redis.Client
	cb         *circuitbreaker.CircuitBreaker
	localCache sync.Map // map[string]localCacheEntry
	stopCh     chan struct{}
	closeOnce  sync.Once
}

// NewBlacklist 创建黑名单服务
func NewBlacklist(rdb *redis.Client) *Blacklist {
	b := &Blacklist{
		rdb: rdb,
		cb: circuitbreaker.NewNamed("token_blacklist", circuitbreaker.Config{
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

// Close 优雅关闭黑名单服务，停止后台清理 goroutine（安全支持多次调用）
func (b *Blacklist) Close() {
	b.closeOnce.Do(func() {
		close(b.stopCh)
	})
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

// TryConsumeRefreshToken 原子标记 refresh token 已被使用（一次性使用）。
// 返回 true 表示本次成功占用；false 表示该 token 已被并发请求或历史请求消费。
func (b *Blacklist) TryConsumeRefreshToken(ctx context.Context, token string, expiry time.Duration) (bool, error) {
	if expiry < minBlacklistTTL || expiry > maxBlacklistTTL {
		return false, fmt.Errorf("refresh consume TTL %v out of valid range [%v, %v]", expiry, minBlacklistTTL, maxBlacklistTTL)
	}

	hash, err := hashToken(token)
	if err != nil {
		return false, fmt.Errorf("failed to hash token: %w", err)
	}

	b.localCache.Store(hash, localCacheEntry{
		blacklisted: true,
		expiresAt:   time.Now().Add(localCacheTTL),
	})

	if !b.cb.Allow() {
		return false, fmt.Errorf("blacklist service unavailable (circuit breaker open)")
	}

	ok, err := b.rdb.SetNX(ctx, refreshConsumedPrefix+hash, "1", expiry).Result()
	if err != nil {
		b.cb.RecordFailure()
		return false, fmt.Errorf("failed to mark refresh token consumed: %w", err)
	}
	b.cb.RecordSuccess()
	if !ok {
		return false, nil
	}
	return true, nil
}

// ReleaseConsumedRefreshToken 释放已占用但未真正完成轮换的 refresh token 标记，允许客户端重试。
func (b *Blacklist) ReleaseConsumedRefreshToken(ctx context.Context, token string) error {
	hash, err := hashToken(token)
	if err != nil {
		return fmt.Errorf("failed to hash token: %w", err)
	}

	b.localCache.Delete(hash)

	if !b.cb.Allow() {
		return fmt.Errorf("blacklist service unavailable (circuit breaker open)")
	}

	if err := b.rdb.Del(ctx, refreshConsumedPrefix+hash).Err(); err != nil {
		b.cb.RecordFailure()
		return fmt.Errorf("failed to release refresh token consume mark: %w", err)
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

	exists, err := b.rdb.Exists(ctx, blacklistPrefix+hash, refreshConsumedPrefix+hash).Result()
	if err != nil {
		b.cb.RecordFailure()
		logger.L().Warn("redis unavailable for blacklist check",
			zap.String("operation", "IsBlacklisted"),
			zap.String("circuit_state", b.cb.State().String()),
		)

		// Redis 错误时尝试本地缓存降级
		if entry, ok := b.localCache.Load(hash); ok {
			cached, ok := entry.(localCacheEntry)
			if !ok {
				b.localCache.Delete(hash)
				return true, fmt.Errorf("blacklist service unavailable")
			}
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

// RevokeAllUserTokens 撤销用户的所有活跃 token。
// 先 prune 已过期成员，再将剩余活跃 token 加入黑名单。
func (b *Blacklist) RevokeAllUserTokens(ctx context.Context, userID string, blacklistExpiry time.Duration) error {
	if !b.cb.Allow() {
		return fmt.Errorf("RevokeAllUserTokens: blacklist service unavailable (circuit breaker open)")
	}

	key := userTokensPrefix + userID

	// 先清理已过期成员
	nowStr := fmt.Sprintf("%d", time.Now().Unix())
	if err := b.rdb.ZRemRangeByScore(ctx, key, "-inf", nowStr).Err(); err != nil {
		logger.L().Warn("RevokeAllUserTokens: failed to prune expired members",
			zap.String("user_id", userID),
			zap.Error(err),
		)
	}

	// 获取剩余活跃成员
	members, err := b.rdb.ZRange(ctx, key, 0, -1).Result()
	if err != nil {
		b.cb.RecordFailure()
		return fmt.Errorf("RevokeAllUserTokens: failed to get user tokens: %w", err)
	}

	if len(members) == 0 {
		b.cb.RecordSuccess()
		return nil
	}

	pipe := b.rdb.Pipeline()
	for _, member := range members {
		tokenHash := extractHash(member)
		pipe.Set(ctx, blacklistPrefix+tokenHash, "1", blacklistExpiry)
	}
	pipe.Del(ctx, key)

	_, err = pipe.Exec(ctx)
	if err != nil {
		b.cb.RecordFailure()
		return fmt.Errorf("RevokeAllUserTokens: failed to execute pipeline: %w", err)
	}
	b.cb.RecordSuccess()

	// 同步更新本地缓存，避免 IsBlacklisted 在 TTL 窗口内返回旧的 false
	for _, member := range members {
		tokenHash := extractHash(member)
		b.localCache.Store(tokenHash, localCacheEntry{
			blacklisted: true,
			expiresAt:   time.Now().Add(localCacheTTL),
		})
	}

	return nil
}

// TrackUserToken 记录用户 token 到 ZSET（按过期时间戳排序）。
// member 格式: "access:<hash>" 或 "refresh:<hash>"
// score: token 过期的 Unix 时间戳
// 每次写入时顺带清理已过期成员，防止集合无限增长。
func (b *Blacklist) TrackUserToken(ctx context.Context, userID, token string, tokenType TokenType, expiresAt time.Time) error {
	if !b.cb.Allow() {
		return fmt.Errorf("TrackUserToken: blacklist service unavailable (circuit breaker open)")
	}

	tokenHash, err := hashToken(token)
	if err != nil {
		return fmt.Errorf("failed to hash token: %w", err)
	}

	key := userTokensPrefix + userID
	member := string(tokenType) + ":" + tokenHash
	score := float64(expiresAt.Unix())

	pipe := b.rdb.Pipeline()
	// 清理已过期成员
	nowStr := fmt.Sprintf("%d", time.Now().Unix())
	pipe.ZRemRangeByScore(ctx, key, "-inf", nowStr)
	// 添加新成员
	pipe.ZAdd(ctx, key, redis.Z{Score: score, Member: member})
	// 设置安全网 TTL，防止 key 孤立
	pipe.Expire(ctx, key, maxTrackingKeyTTL)

	_, err = pipe.Exec(ctx)
	if err != nil {
		b.cb.RecordFailure()
		return fmt.Errorf("TrackUserToken: pipeline exec failed: %w", err)
	}

	b.cb.RecordSuccess()
	return nil
}

// UntrackUserToken 从用户 token 集合中移除指定 token
func (b *Blacklist) UntrackUserToken(ctx context.Context, userID, token string, tokenType TokenType) error {
	if !b.cb.Allow() {
		return fmt.Errorf("UntrackUserToken: blacklist service unavailable (circuit breaker open)")
	}

	tokenHash, err := hashToken(token)
	if err != nil {
		return fmt.Errorf("failed to hash token: %w", err)
	}

	member := string(tokenType) + ":" + tokenHash
	if err := b.rdb.ZRem(ctx, userTokensPrefix+userID, member).Err(); err != nil {
		b.cb.RecordFailure()
		return fmt.Errorf("UntrackUserToken: ZRem failed: %w", err)
	}
	b.cb.RecordSuccess()
	return nil
}

// CircuitBreakerMetrics 获取熔断器指标（用于监控）
func (b *Blacklist) CircuitBreakerMetrics() map[string]any {
	return b.cb.Metrics()
}

// extractHash 从 ZSET member 中提取 token hash。
// member 格式: "type:hash"。如果没有前缀（兼容旧数据），直接返回原值。
func extractHash(member string) string {
	if idx := strings.IndexByte(member, ':'); idx >= 0 {
		return member[idx+1:]
	}
	return member
}
