package sso

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/redis/go-redis/v9"
)

const (
	// 用户信息缓存前缀
	userCachePrefix = "sso:user:"
	// 默认 Redis 缓存过期时间
	defaultCacheTTL = 5 * time.Minute
	// 本地内存缓存过期时间（短于 Redis，作为热点数据加速层和 Redis 宕机降级）
	localCacheTTL = 1 * time.Minute
)

// localEntry 本地内存缓存条目
type localEntry struct {
	user      *CachedUser
	expiresAt time.Time
}

// CachedUser 缓存的用户信息
type CachedUser struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Email       string   `json:"email"`
	IsAdmin     bool     `json:"isAdmin"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	Groups      []string `json:"groups"`
}

// UserCache 用户信息缓存服务（L1 本地内存 + L2 Redis 二级缓存）
type UserCache struct {
	rdb       *redis.Client
	local     sync.Map // L1: 本地内存缓存，key → *localEntry
	mu        sync.RWMutex
	ttl       time.Duration
	stopCh    chan struct{}
	closeOnce sync.Once
}

// NewUserCache 创建用户缓存服务
func NewUserCache(rdb *redis.Client) *UserCache {
	uc := &UserCache{
		rdb:    rdb,
		ttl:    defaultCacheTTL,
		stopCh: make(chan struct{}),
	}
	go uc.cleanupLoop()
	return uc
}

// Close 优雅关闭缓存服务，停止后台清理 goroutine（安全支持多次调用）
func (c *UserCache) Close() {
	c.closeOnce.Do(func() {
		close(c.stopCh)
	})
}

// cleanupLoop 定期清理过期的本地缓存条目，防止 sync.Map 无限增长
func (c *UserCache) cleanupLoop() {
	ticker := time.NewTicker(localCacheTTL * 2)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			now := time.Now()
			c.local.Range(func(key, value any) bool {
				if e, ok := value.(*localEntry); ok && now.After(e.expiresAt) {
					c.local.Delete(key)
				}
				return true
			})
		}
	}
}

// hashKey 对标识符进行哈希处理，防止缓存 key 注入
func hashKey(identifier string) string {
	h := sha256.Sum256([]byte(identifier))
	return hex.EncodeToString(h[:])
}

// SetTTL 设置缓存过期时间
func (c *UserCache) SetTTL(ttl time.Duration) {
	c.mu.Lock()
	c.ttl = ttl
	c.mu.Unlock()
}

// Get 从缓存获取用户信息（L1 本地 → L2 Redis，Redis 错误降级为未命中）
func (c *UserCache) Get(ctx context.Context, identifier string) (*CachedUser, error) {
	key := userCachePrefix + hashKey(identifier)

	// L1: 本地内存缓存
	if entry, ok := c.local.Load(key); ok {
		if e, ok := entry.(*localEntry); ok && time.Now().Before(e.expiresAt) {
			return e.user, nil
		}
		c.local.Delete(key) // 过期条目清理
	}

	// L2: Redis 缓存（错误降级为未命中，不阻断请求）
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		// redis.Nil 或 Redis 不可用均视为未命中
		return nil, nil
	}

	var user CachedUser
	if err := json.Unmarshal(data, &user); err != nil {
		// 数据损坏：删除坏条目，降级为未命中
		_ = c.rdb.Del(ctx, key).Err()
		return nil, nil
	}

	// 回填本地缓存
	c.local.Store(key, &localEntry{user: &user, expiresAt: time.Now().Add(localCacheTTL)})

	return &user, nil
}

// Set 设置用户信息缓存（以 user.ID 为 key，同时写入 L1 和 L2）
func (c *UserCache) Set(ctx context.Context, user *CachedUser) error {
	key := userCachePrefix + hashKey(user.ID)

	// L1: 本地内存缓存
	c.local.Store(key, &localEntry{user: user, expiresAt: time.Now().Add(localCacheTTL)})

	// L2: Redis 缓存
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user cache: %w", err)
	}
	c.mu.RLock()
	ttl := c.ttl
	c.mu.RUnlock()
	return c.rdb.Set(ctx, key, data, ttl).Err()
}

// Delete 删除用户缓存（同时清除 L1 和 L2）
func (c *UserCache) Delete(ctx context.Context, identifier string) error {
	key := userCachePrefix + hashKey(identifier)
	c.local.Delete(key)
	return c.rdb.Del(ctx, key).Err()
}

// FromCasdoorUser 从 Casdoor 用户转换为缓存用户
func FromCasdoorUser(user *casdoorsdk.User) *CachedUser {
	if user == nil {
		return nil
	}

	cached := &CachedUser{
		ID:          user.Id,
		Name:        user.Name,
		DisplayName: user.DisplayName,
		Email:       user.Email,
		IsAdmin:     user.IsAdmin,
		Groups:      user.Groups,
	}

	// 提取角色名称
	for _, role := range user.Roles {
		cached.Roles = append(cached.Roles, role.Name)
	}

	// 提取权限名称
	for _, perm := range user.Permissions {
		cached.Permissions = append(cached.Permissions, perm.Name)
	}

	return cached
}

// HasRole 检查缓存用户是否有指定角色
func (u *CachedUser) HasRole(roleName string) bool {
	for _, role := range u.Roles {
		if role == roleName {
			return true
		}
	}
	return false
}

// HasAnyRole 检查缓存用户是否有任一指定角色
func (u *CachedUser) HasAnyRole(roleNames ...string) bool {
	roleSet := make(map[string]bool)
	for _, role := range u.Roles {
		roleSet[role] = true
	}
	for _, name := range roleNames {
		if roleSet[name] {
			return true
		}
	}
	return false
}

// HasPermission 检查缓存用户是否有指定权限
func (u *CachedUser) HasPermission(permName string) bool {
	for _, perm := range u.Permissions {
		if perm == permName {
			return true
		}
	}
	return false
}
