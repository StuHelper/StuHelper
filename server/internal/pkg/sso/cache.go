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
	// 默认缓存过期时间
	defaultCacheTTL = 5 * time.Minute
)

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

// UserCache 用户信息缓存服务
type UserCache struct {
	rdb *redis.Client
	mu  sync.RWMutex
	ttl time.Duration
}

// NewUserCache 创建用户缓存服务
func NewUserCache(rdb *redis.Client) *UserCache {
	return &UserCache{
		rdb: rdb,
		ttl: defaultCacheTTL,
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

// Get 从缓存获取用户信息
func (c *UserCache) Get(ctx context.Context, identifier string) (*CachedUser, error) {
	key := userCachePrefix + hashKey(identifier)
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // 缓存未命中
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user cache: %w", err)
	}

	var user CachedUser
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user cache: %w", err)
	}
	return &user, nil
}

// Set 设置用户信息缓存（以 user.ID 为 key）
func (c *UserCache) Set(ctx context.Context, user *CachedUser) error {
	key := userCachePrefix + hashKey(user.ID)
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user cache: %w", err)
	}
	c.mu.RLock()
	ttl := c.ttl
	c.mu.RUnlock()
	return c.rdb.Set(ctx, key, data, ttl).Err()
}

// Delete 删除用户缓存
func (c *UserCache) Delete(ctx context.Context, identifier string) error {
	key := userCachePrefix + hashKey(identifier)
	return c.rdb.Del(ctx, key).Err()
}

// FromCasdoorUser 从 Casdoor 用户转换为缓存用户
func FromCasdoorUser(user *casdoorsdk.User) *CachedUser {
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
