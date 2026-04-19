package cache

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/singleflightx"
)

const (
	// DefaultTTL 默认缓存过期时间
	DefaultTTL = 5 * time.Minute
	// VersionKeyTTL 版本号 key 的过期时间
	VersionKeyTTL = 24 * time.Hour
	// versionLocalTTL 版本号本地缓存有效期
	versionLocalTTL = 1 * time.Second
	// defaultMaxVersionEntries 版本号本地缓存默认最大条目数
	defaultMaxVersionEntries = 1000
	// jitterFraction TTL 抖动比例（±15%）
	jitterFraction = 0.15
)

// incrExpireScript 原子执行 INCR + EXPIRE，避免 Expire 失败导致 key 永不过期
var incrExpireScript = redis.NewScript(`
	local v = redis.call('INCR', KEYS[1])
	redis.call('EXPIRE', KEYS[1], ARGV[1])
	return v
`)

// versionEntry 版本号本地缓存条目
type versionEntry struct {
	version   string
	expiresAt time.Time
}

// Helper Redis 缓存辅助工具
type Helper struct {
	client            *redis.Client
	sf                singleflight.Group
	vmu               sync.RWMutex
	versions          map[string]versionEntry
	maxVersionEntries int
}

// NewHelper 创建缓存辅助工具
func NewHelper(client *redis.Client) *Helper {
	return &Helper{
		client:            client,
		versions:          make(map[string]versionEntry),
		maxVersionEntries: defaultMaxVersionEntries,
	}
}

// NewHelperWithMaxVersions 创建缓存辅助工具，可自定义版本号本地缓存上限
func NewHelperWithMaxVersions(client *redis.Client, maxVersions int) *Helper {
	if maxVersions <= 0 {
		maxVersions = defaultMaxVersionEntries
	}
	return &Helper{
		client:            client,
		versions:          make(map[string]versionEntry),
		maxVersionEntries: maxVersions,
	}
}

// JitteredTTL 返回带随机抖动的 TTL，防止缓存雪崩
// 在 base ± jitterFraction 范围内随机浮动
func JitteredTTL(base time.Duration) time.Duration {
	jitter := float64(base) * jitterFraction
	delta := randFloat64()*2*jitter - jitter
	return base + time.Duration(delta)
}

// randFloat64 使用非安全随机源生成 [0, 1) 范围的 float64。
//
//nolint:gosec // TTL jitter 只需要低成本随机性，不参与任何安全决策。
func randFloat64() float64 {
	return rand.Float64()
}

// Client 返回底层 Redis 客户端（用于需要直接访问的场景）
func (h *Helper) Client() *redis.Client {
	return h.client
}

// GetRaw 直接返回缓存中的 JSON 字节。
// 相比 Get，它避免了重复反序列化以及 float64 精度丢失问题。
// 返回的 json.RawMessage 可直接传给 response.Success。
func (h *Helper) GetRaw(ctx context.Context, key string) (json.RawMessage, bool) {
	if h.client == nil {
		return nil, false
	}
	start := time.Now()
	data, err := h.client.Get(ctx, key).Bytes()
	metrics.CacheOperationDuration.WithLabelValues("get", "redis").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.CacheMissesTotal.WithLabelValues("redis").Inc()
		return nil, false
	}
	metrics.CacheHitsTotal.WithLabelValues("redis").Inc()
	return json.RawMessage(data), true
}

// GetAs 获取缓存值并反序列化为指定类型（泛型版本，避免 any 类型丢失问题）
func GetAs[T any](h *Helper, ctx context.Context, key string) (T, bool) {
	var zero T
	if h.client == nil {
		return zero, false
	}
	start := time.Now()
	data, err := h.client.Get(ctx, key).Bytes()
	metrics.CacheOperationDuration.WithLabelValues("get", "redis").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.CacheMissesTotal.WithLabelValues("redis").Inc()
		return zero, false
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return zero, false
	}
	metrics.CacheHitsTotal.WithLabelValues("redis").Inc()
	return v, true
}

// Set 设置缓存值
func (h *Helper) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if h.client == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		logger.L().Warn("failed to marshal cache value",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}
	start := time.Now()
	defer func() {
		metrics.CacheOperationDuration.WithLabelValues("set", "redis").Observe(time.Since(start).Seconds())
	}()
	if err := h.client.Set(ctx, key, data, ttl).Err(); err != nil {
		logger.L().Warn("failed to set cache",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// Invalidate 批量删除匹配前缀的缓存
// 注意：优先使用 InvalidateByVersion 代替此方法，避免 SCAN 的性能问题
func (h *Helper) Invalidate(ctx context.Context, prefix string) error {
	if h.client == nil {
		return nil
	}

	// 添加超时保护，防止 SCAN 长时间阻塞（30s 以应对大规模前缀场景）
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	pattern := prefix + "*"
	var cursor uint64
	var deletedTotal int
	// 持续扫描直到 cursor 归零，确保遍历所有匹配 key
	for {
		var batch []string
		var err error
		batch, cursor, err = h.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			// 超时或取消时已删除部分 key，缓存处于不一致状态
			if ctx.Err() != nil && deletedTotal > 0 {
				logger.L().Warn("cache invalidation incomplete due to timeout, partial keys deleted",
					zap.String("prefix", prefix),
					zap.String("pattern", pattern),
					zap.Int("deleted_so_far", deletedTotal),
					zap.Error(err),
				)
			} else {
				logger.L().Warn("failed to scan cache keys",
					zap.String("prefix", prefix),
					zap.String("pattern", pattern),
					zap.Error(err),
				)
			}
			return err
		}

		// 每批立即删除，避免内存中积累大量 key
		if len(batch) > 0 {
			pipe := h.client.Pipeline()
			for _, key := range batch {
				pipe.Del(ctx, key)
			}
			if _, err := pipe.Exec(ctx); err != nil {
				logger.L().Warn("failed to invalidate cache batch",
					zap.String("prefix", prefix),
					zap.Int("batch_size", len(batch)),
					zap.Error(err),
				)
				return err
			}
			deletedTotal += len(batch)
		}

		if cursor == 0 {
			break
		}
	}

	if deletedTotal > 0 {
		logger.L().Debug("cache invalidation completed",
			zap.String("prefix", prefix),
			zap.Int("deleted_total", deletedTotal),
		)
	}

	return nil
}

// GetInt 获取整数缓存值
func (h *Helper) GetInt(ctx context.Context, key string) (int, bool) {
	if h.client == nil {
		return 0, false
	}
	val, err := h.client.Get(ctx, key).Int()
	if err != nil {
		return 0, false
	}
	return val, true
}

// SetInt 设置整数缓存值
func (h *Helper) SetInt(ctx context.Context, key string, value int, ttl time.Duration) error {
	if h.client == nil {
		return nil
	}
	if err := h.client.Set(ctx, key, value, ttl).Err(); err != nil {
		logger.L().Warn("failed to set int cache",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// VersionKey 返回版本号的 Redis key
func VersionKey(prefix string) string {
	return "cache:version:" + prefix
}

// GetVersion 获取缓存版本号（带本地短时缓存，减少 Redis 往返）
// 使用 singleflight 去重并发 Redis 查询
func (h *Helper) GetVersion(ctx context.Context, prefix string) string {
	if h.client == nil {
		return "0"
	}

	// 快速检查 context 是否已取消，避免向 Redis 发送无意义请求
	select {
	case <-ctx.Done():
		return "0"
	default:
	}

	vk := VersionKey(prefix)

	// 先查本地缓存（读锁）
	h.vmu.RLock()
	if entry, ok := h.versions[vk]; ok && time.Now().Before(entry.expiresAt) {
		h.vmu.RUnlock()
		return entry.version
	}
	h.vmu.RUnlock()

	// 本地缓存未命中，通过 singleflight 去重并发 Redis 查询
	version, err := singleflightx.DoValue(&h.sf, "version:"+vk, func() (string, error) {
		// 二次检查本地缓存（可能在等待期间被其他请求填充）
		h.vmu.RLock()
		if entry, ok := h.versions[vk]; ok && time.Now().Before(entry.expiresAt) {
			h.vmu.RUnlock()
			return entry.version, nil
		}
		h.vmu.RUnlock()

		version, err := h.client.Get(ctx, vk).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				return "0", nil
			}
			// 非 key-not-found 错误，记录日志并返回默认值
			logger.L().Warn("failed to get cache version from redis",
				zap.String("key", vk),
				zap.Error(err),
			)
			return "0", nil
		}
		return version, nil
	})
	if err != nil {
		return "0"
	}
	if version == "" {
		version = "0"
	}

	// 写入本地缓存
	h.vmu.Lock()
	if _, exists := h.versions[vk]; !exists && len(h.versions) >= h.maxVersionEntries {
		// 本地版本缓存只是 Redis 命中的短时优化，达到上限时直接重置即可。
		// 这样避免为了淘汰一个条目而按 map 大小 O(N) 扫描整张表。
		h.versions = make(map[string]versionEntry, h.maxVersionEntries)
	}
	h.versions[vk] = versionEntry{
		version:   version,
		expiresAt: time.Now().Add(versionLocalTTL),
	}
	h.vmu.Unlock()

	return version
}

// BuildVersionedKey 构建带版本号的缓存 key
func (h *Helper) BuildVersionedKey(ctx context.Context, prefix, key string) string {
	version := h.GetVersion(ctx, prefix)
	return prefix + ":v" + version + ":" + key
}

// InvalidateByVersion 通过递增版本号使缓存失效
// 旧的缓存 key 会根据 TTL 自然过期，避免 SCAN 操作的性能问题
func (h *Helper) InvalidateByVersion(ctx context.Context, prefix string) error {
	if h.client == nil {
		return nil
	}

	versionKey := VersionKey(prefix)

	// 使用 Lua 脚本原子执行 INCR + EXPIRE，避免 Expire 失败导致 key 永不过期
	newVersion, err := incrExpireScript.Run(ctx, h.client, []string{versionKey}, int(VersionKeyTTL.Seconds())).Int64()
	if err != nil {
		logger.L().Warn("failed to increment cache version",
			zap.String("prefix", prefix),
			zap.Error(err),
		)
		return err
	}

	// 清除本地版本缓存，确保下次读取拿到最新版本
	h.vmu.Lock()
	delete(h.versions, versionKey)
	h.vmu.Unlock()

	logger.L().Debug("cache invalidated by version increment",
		zap.String("prefix", prefix),
		zap.Int64("new_version", newVersion),
	)
	return nil
}

// GetOrSet 获取缓存值，缓存未命中时通过 loader 加载并写入缓存
// 内部使用 singleflight 去重，同一 key 的并发请求只会执行一次 loader
func GetOrSet[T any](h *Helper, ctx context.Context, key string, ttl time.Duration, loader func(ctx context.Context) (T, error)) (T, error) {
	var zero T

	// 先尝试从缓存获取
	if val, ok := GetAs[T](h, ctx, key); ok {
		return val, nil
	}

	// 缓存未命中，通过 singleflight 去重并发加载
	val, err := singleflightx.DoValue(&h.sf, key, func() (T, error) {
		// 再次检查缓存（可能在等待期间被其他请求填充）
		if val, ok := GetAs[T](h, ctx, key); ok {
			return val, nil
		}

		// 执行 loader 从数据源加载
		val, err := loader(ctx)
		if err != nil {
			// 加载失败时移除 singleflight 缓存，允许后续请求重试
			h.sf.Forget(key)
			return zero, err
		}

		// 写入缓存（使用带抖动的 TTL）
		if setErr := h.Set(ctx, key, val, JitteredTTL(ttl)); setErr != nil {
			logger.L().Warn("GetOrSet: failed to set cache after load",
				zap.String("key", key),
				zap.Error(setErr),
			)
		}

		return val, nil
	})
	if err != nil {
		return zero, err
	}
	return val, nil
}
