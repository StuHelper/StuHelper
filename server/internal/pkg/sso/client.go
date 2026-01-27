package sso

import (
	"context"
	"fmt"
	"sync"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
)

var initOnce sync.Once

// Client Casdoor SSO 客户端
type Client struct {
	organization string
	cache        *UserCache
}

// NewClient 创建并初始化 Casdoor 客户端
func NewClient(cfg config.CasdoorConfig) *Client {
	// 使用 sync.Once 确保 InitConfig 只调用一次，避免并发问题
	initOnce.Do(func() {
		casdoorsdk.InitConfig(
			cfg.Endpoint,
			cfg.ClientID,
			cfg.ClientSecret,
			cfg.Certificate,
			cfg.Organization,
			cfg.Application,
		)
	})
	return &Client{
		organization: cfg.Organization,
	}
}

// NewClientWithCache 创建带缓存的 Casdoor 客户端
func NewClientWithCache(cfg config.CasdoorConfig, rdb *redis.Client) *Client {
	client := NewClient(cfg)
	client.cache = NewUserCache(rdb)
	return client
}

// GetSigninURL 获取登录 URL
func (c *Client) GetSigninURL(redirectURI string) string {
	return casdoorsdk.GetSigninUrl(redirectURI)
}

// GetSignupURL 获取注册 URL
func (c *Client) GetSignupURL(redirectURI string) string {
	return casdoorsdk.GetSignupUrl(true, redirectURI)
}

// GetOAuthToken 通过授权码获取 OAuth Token
func (c *Client) GetOAuthToken(code, state string) (*oauth2.Token, error) {
	if code == "" {
		return nil, fmt.Errorf("authorization code is required")
	}

	token, err := casdoorsdk.GetOAuthToken(code, state)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth token: %w", err)
	}

	return token, nil
}

// ParseJwtToken 解析并验证 JWT Token
func (c *Client) ParseJwtToken(token string) (*casdoorsdk.Claims, error) {
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}

	claims, err := casdoorsdk.ParseJwtToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired token: %w", err)
	}

	return claims, nil
}

// RefreshOAuthToken 使用 Refresh Token 刷新 Access Token
func (c *Client) RefreshOAuthToken(refreshToken string) (*oauth2.Token, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token is required")
	}

	token, err := casdoorsdk.RefreshOAuthToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return token, nil
}

// GetUser 获取用户信息
func (c *Client) GetUser(name string) (*casdoorsdk.User, error) {
	user, err := casdoorsdk.GetUser(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get user %s: %w", name, err)
	}
	return user, nil
}

// GetCachedUser 获取缓存的用户信息（优先从缓存读取）
func (c *Client) GetCachedUser(ctx context.Context, username string) (*CachedUser, error) {
	// 如果没有配置缓存，直接从 Casdoor 获取
	if c.cache == nil {
		user, err := c.GetUser(username)
		if err != nil {
			return nil, err
		}
		return FromCasdoorUser(user), nil
	}

	// 尝试从缓存获取
	cached, err := c.cache.Get(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user from cache: %w", err)
	}
	if cached != nil {
		return cached, nil
	}

	// 缓存未命中，从 Casdoor 获取
	user, err := c.GetUser(username)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	cached = FromCasdoorUser(user)
	if err := c.cache.Set(ctx, cached); err != nil {
		// 缓存写入失败不影响返回结果，只记录警告日志
		fmt.Printf("warning: failed to cache user %s: %v\n", username, err)
	}

	return cached, nil
}

// InvalidateUserCache 使用户缓存失效
func (c *Client) InvalidateUserCache(ctx context.Context, username string) error {
	if c.cache == nil {
		return nil
	}
	return c.cache.Delete(ctx, username)
}

// IsUserAdmin 检查用户是否是管理员
func (c *Client) IsUserAdmin(username string) (bool, error) {
	user, err := c.GetUser(username)
	if err != nil {
		return false, err
	}
	return user.IsAdmin, nil
}

// Enforce 检查用户是否有权限执行某个操作
// permissionId: 权限ID（在Casdoor中配置）
// owner: 组织名
// sub: 用户标识, obj: 资源, act: 操作
func (c *Client) Enforce(permissionId, owner, sub, obj, act string) (bool, error) {
	request := casdoorsdk.CasbinRequest{sub, obj, act}
	return casdoorsdk.Enforce(permissionId, "", "", "", owner, request)
}

// GetOrganization 获取组织名
func (c *Client) GetOrganization() string {
	return c.organization
}

// GetUserRoles 获取用户的角色列表
func (c *Client) GetUserRoles(username string) ([]*casdoorsdk.Role, error) {
	user, err := c.GetUser(username)
	if err != nil {
		return nil, err
	}
	return user.Roles, nil
}

// GetUserPermissions 获取用户的权限列表
func (c *Client) GetUserPermissions(username string) ([]*casdoorsdk.Permission, error) {
	user, err := c.GetUser(username)
	if err != nil {
		return nil, err
	}
	return user.Permissions, nil
}

// GetUserGroups 获取用户的组列表
func (c *Client) GetUserGroups(username string) ([]string, error) {
	user, err := c.GetUser(username)
	if err != nil {
		return nil, err
	}
	return user.Groups, nil
}

// HasRole 检查用户是否拥有指定角色
func (c *Client) HasRole(username, roleName string) (bool, error) {
	roles, err := c.GetUserRoles(username)
	if err != nil {
		return false, err
	}
	for _, role := range roles {
		if role.Name == roleName {
			return true, nil
		}
	}
	return false, nil
}

// HasAnyRole 检查用户是否拥有任一指定角色
func (c *Client) HasAnyRole(username string, roleNames ...string) (bool, error) {
	roles, err := c.GetUserRoles(username)
	if err != nil {
		return false, err
	}
	roleSet := make(map[string]bool)
	for _, role := range roles {
		roleSet[role.Name] = true
	}
	for _, name := range roleNames {
		if roleSet[name] {
			return true, nil
		}
	}
	return false, nil
}

// HasPermission 检查用户是否拥有指定权限
func (c *Client) HasPermission(username, permissionName string) (bool, error) {
	permissions, err := c.GetUserPermissions(username)
	if err != nil {
		return false, err
	}
	for _, perm := range permissions {
		if perm.Name == permissionName {
			return true, nil
		}
	}
	return false, nil
}
