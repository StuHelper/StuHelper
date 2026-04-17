package token

import (
	"context"
	"errors"
	"fmt"
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
)

// TokenType 标识 token 类型（保留给 audit/metric 使用）
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

// AddByHash 将已知 token hash 直接加入黑名单（用于 Session 撤销场景，token 原文不可用）。
func (b *Blacklist) AddByHash(ctx context.Context, tokenHash string, expiry time.Duration) error {
	if expiry < minBlacklistTTL || expiry > maxBlacklistTTL {
		return fmt.Errorf("blacklist TTL %v out of valid range [%v, %v]", expiry, minBlacklistTTL, maxBlacklistTTL)
	}

	b.localCache.Store(tokenHash, localCacheEntry{
		blacklisted: true,
		expiresAt:   time.Now().Add(localCacheTTL),
	})

	if !b.cb.Allow() {
		return fmt.Errorf("blacklist service unavailable (circuit breaker open)")
	}

	key := blacklistPrefix + tokenHash
	if err := b.rdb.Set(ctx, key, "1", expiry).Err(); err != nil {
		b.cb.RecordFailure()
		return fmt.Errorf("failed to add token hash to blacklist: %w", err)
	}
	b.cb.RecordSuccess()
	return nil
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

	status, err := b.rdb.SetArgs(ctx, refreshConsumedPrefix+hash, "1", redis.SetArgs{
		Mode: "NX",
		TTL:  expiry,
	}).Result()
	if errors.Is(err, redis.Nil) {
		b.cb.RecordSuccess()
		return false, nil
	}
	if err != nil {
		b.cb.RecordFailure()
		return false, fmt.Errorf("failed to mark refresh token consumed: %w", err)
	}
	b.cb.RecordSuccess()
	if status != "OK" {
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

// RevokeAllUserTokens / TrackUserToken / UntrackUserToken 已随双轨 session 清理一并移除。
// 全设备撤销通过 SessionStore.RevokeAll 完成，token 原文撤销通过 Add / AddByHash。

// CircuitBreakerMetrics 获取熔断器指标（用于监控）
func (b *Blacklist) CircuitBreakerMetrics() map[string]any {
	return b.cb.Metrics()
}
