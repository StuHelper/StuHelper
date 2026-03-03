package sso

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

var (
	initOnce       sync.Once
	initializedCfg config.CasdoorConfig // 记录首次初始化的配置，用于一致性检查
)

// ssoTransport 为 SSO HTTP 调用提供细粒度超时的 Transport
var ssoTransport = &http.Transport{
	DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
	TLSHandshakeTimeout:   5 * time.Second,
	ResponseHeaderTimeout: 5 * time.Second,
	IdleConnTimeout:       90 * time.Second,
	MaxIdleConns:          10,
	MaxIdleConnsPerHost:   5,
}

// ssoGlobalClient 是 Casdoor SDK 用于所有非 OAuth 调用（GetUser、GetUserByUserId 等）的全局 HTTP 客户端。
// 通过 casdoorsdk.SetHttpClient 设置，确保所有 SDK 内部 HTTP 调用都受超时约束。
var ssoGlobalClient = &http.Client{
	Timeout:   10 * time.Second,
	Transport: ssoTransport,
}

// errConfigMismatch 表示 NewClient 传入的配置与首次初始化不一致
var errConfigMismatch = fmt.Errorf("sso: NewClient called with different config than initial initialization")

// Client Casdoor SSO 客户端
type Client struct {
	organization    string
	endpoint        string
	clientID        string
	applicationName string
	cache           *UserCache
	stateManager    *StateManager
	logger          *zap.Logger
}

// NewClient 创建并初始化 Casdoor 客户端。
// sync.Once 保证 InitConfig 只执行一次，后续调用若传入不同配置会返回错误。
func NewClient(cfg config.CasdoorConfig) (*Client, error) {
	initOnce.Do(func() {
		casdoorsdk.InitConfig(
			cfg.Endpoint,
			cfg.ClientID,
			cfg.ClientSecret,
			cfg.Certificate,
			cfg.Organization,
			cfg.Application,
		)
		casdoorsdk.SetHttpClient(ssoGlobalClient)
		initializedCfg = cfg
	})

	// 检查后续调用的配置是否与首次初始化一致
	if initializedCfg.Endpoint != cfg.Endpoint ||
		initializedCfg.ClientID != cfg.ClientID ||
		initializedCfg.ClientSecret != cfg.ClientSecret ||
		initializedCfg.Organization != cfg.Organization ||
		initializedCfg.Application != cfg.Application {
		return nil, errConfigMismatch
	}

	return &Client{
		organization:    cfg.Organization,
		endpoint:        cfg.Endpoint,
		clientID:        cfg.ClientID,
		applicationName: cfg.Application,
		logger:          zap.L(),
	}, nil
}

// NewClientWithCache 创建带缓存的 Casdoor 客户端
func NewClientWithCache(cfg config.CasdoorConfig, rdb *redis.Client) (*Client, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	client.cache = NewUserCache(rdb)
	client.stateManager = NewStateManager(rdb)
	return client, nil
}

// Close 优雅关闭客户端，释放后台资源（UserCache 清理 goroutine）
func (c *Client) Close() {
	if c.cache != nil {
		c.cache.Close()
	}
}

// GetSigninURL 获取登录 URL（使用随机 state 防止 CSRF）
func (c *Client) GetSigninURL(ctx context.Context, redirectURI string) (string, string, error) {
	if err := validateRedirectURI(redirectURI); err != nil {
		return "", "", err
	}
	state, err := c.stateManager.Generate(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}
	return c.buildOAuthURL("/login/oauth/authorize", redirectURI, state), state, nil
}

// GetSignupURL 获取注册 URL（使用随机 state 防止 CSRF）
func (c *Client) GetSignupURL(ctx context.Context, redirectURI string) (string, string, error) {
	if err := validateRedirectURI(redirectURI); err != nil {
		return "", "", err
	}
	state, err := c.stateManager.Generate(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}
	return c.buildOAuthURL("/signup/oauth/authorize", redirectURI, state), state, nil
}

// validateRedirectURI 验证 redirectURI 格式，必须是合法的绝对 URL（含 scheme 和 host）
func validateRedirectURI(redirectURI string) error {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return fmt.Errorf("invalid redirect URI: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("redirect URI must be an absolute URL with scheme and host")
	}
	return nil
}

// ErrStateManagerRequired 表示 state manager 未初始化的错误
var ErrStateManagerRequired = fmt.Errorf("state manager is required for OAuth state validation")

// ValidateState 验证并消费 OAuth state（一次性使用）
// 必须使用随机 state，不支持固定 state 以防止 CSRF 攻击
func (c *Client) ValidateState(ctx context.Context, state string) (bool, error) {
	if c.stateManager == nil {
		return false, ErrStateManagerRequired
	}
	return c.stateManager.Validate(ctx, state)
}

// buildOAuthURL 构建 OAuth 授权 URL
func (c *Client) buildOAuthURL(path, redirectURI, state string) string {
	return fmt.Sprintf("%s%s?client_id=%s&response_type=code&redirect_uri=%s&scope=read&state=%s",
		c.endpoint, path, c.clientID, url.QueryEscape(redirectURI), state)
}

// contextRoundTripper 将请求 context 注入到 HTTP 请求中，使 Casdoor SDK 调用可被取消
type contextRoundTripper struct {
	ctx  context.Context
	base http.RoundTripper
}

func (t *contextRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.base.RoundTrip(req.WithContext(t.ctx))
}

// ssoHTTPClient 创建带超时和 context 传播的 HTTP 客户端，用于 OAuth token 操作（GetOAuthToken、RefreshOAuthToken）。
// 复用 ssoTransport 共享连接池，通过 contextRoundTripper 注入请求级 context。
func ssoHTTPClient(ctx context.Context) *http.Client {
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: &contextRoundTripper{ctx: ctx, base: ssoTransport},
	}
}

// GetOAuthToken 通过授权码获取 OAuth Token
func (c *Client) GetOAuthToken(ctx context.Context, code, state string) (*oauth2.Token, error) {
	if code == "" {
		return nil, fmt.Errorf("authorization code is required")
	}

	token, err := casdoorsdk.GetOAuthToken(code, state, casdoorsdk.WithHTTPClient(ssoHTTPClient(ctx)))
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
func (c *Client) RefreshOAuthToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token is required")
	}

	token, err := casdoorsdk.RefreshOAuthToken(refreshToken, casdoorsdk.WithHTTPClient(ssoHTTPClient(ctx)))
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return token, nil
}

// GetUser 通过用户名获取用户信息（支持 context 取消）
// 注意：Casdoor SDK v1.44.0 的 GetUser 不支持 per-request HTTP client 注入，
// 底层使用全局 HTTP client（已配置 10s 超时 + 细粒度 Transport 超时）。
// goroutine+select 模式确保调用方 context 取消时立即返回，泄漏的 goroutine 受全局超时约束。
func (c *Client) GetUser(ctx context.Context, name string) (*casdoorsdk.User, error) {
	type result struct {
		user *casdoorsdk.User
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		user, err := casdoorsdk.GetUser(name)
		ch <- result{user, err}
	}()
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("get user cancelled: %w", ctx.Err())
	case r := <-ch:
		if r.err != nil {
			return nil, fmt.Errorf("failed to get user: %w", r.err)
		}
		return r.user, nil
	}
}

// GetUserByID 通过不可变用户 ID 获取用户信息（支持 context 取消）
// 注意：Casdoor SDK v1.44.0 的 GetUserByUserId 不支持 per-request HTTP client 注入，
// 底层使用全局 HTTP client（已配置 10s 超时 + 细粒度 Transport 超时）。
// goroutine+select 模式确保调用方 context 取消时立即返回，泄漏的 goroutine 受全局超时约束。
func (c *Client) GetUserByID(ctx context.Context, userID string) (*casdoorsdk.User, error) {
	type result struct {
		user *casdoorsdk.User
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		user, err := casdoorsdk.GetUserByUserId(userID)
		ch <- result{user, err}
	}()
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("get user by id cancelled: %w", ctx.Err())
	case r := <-ch:
		if r.err != nil {
			return nil, fmt.Errorf("failed to get user by id: %w", r.err)
		}
		return r.user, nil
	}
}

// GetCachedUser 通过用户名获取缓存的用户信息
// 注意：此方法无法利用缓存（缓存 key 基于 user.ID，而此处只有 username），
// 建议优先使用 GetCachedUserByID。此方法仅在只有 username 时使用。
func (c *Client) GetCachedUser(ctx context.Context, username string) (*CachedUser, error) {
	user, err := c.GetUser(ctx, username)
	if err != nil {
		return nil, err
	}

	if c.cache == nil {
		return FromCasdoorUser(user), nil
	}

	// 获取到 user.Id 后写入缓存供后续 GetCachedUserByID 使用
	cached := FromCasdoorUser(user)
	if cached == nil {
		return nil, fmt.Errorf("casdoor returned nil user for username %s", username)
	}
	if err := c.cache.Set(ctx, cached); err != nil {
		c.logger.Warn("failed to cache user",
			zap.String("username", username),
			zap.Error(err),
		)
	}

	return cached, nil
}

// GetCachedUserByID 通过用户 ID 获取缓存的用户信息（优先从缓存读取）
func (c *Client) GetCachedUserByID(ctx context.Context, userID string) (*CachedUser, error) {
	// 如果没有配置缓存，直接从 Casdoor 获取
	if c.cache == nil {
		user, err := c.GetUserByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		return FromCasdoorUser(user), nil
	}

	// 尝试从缓存获取（L1 本地内存 → L2 Redis，错误降级为未命中）
	cached, _ := c.cache.Get(ctx, userID)
	if cached != nil {
		return cached, nil
	}

	// 缓存未命中（或 Redis 不可用），从 Casdoor 获取
	user, err := c.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	cached = FromCasdoorUser(user)
	if cached == nil {
		return nil, fmt.Errorf("casdoor returned nil user for id %s", userID)
	}
	if err := c.cache.Set(ctx, cached); err != nil {
		c.logger.Warn("failed to cache user",
			zap.String("user_id", userID),
			zap.Error(err),
		)
	}

	return cached, nil
}

// InvalidateUserCacheByID 通过用户 ID 使缓存失效（缓存 key 基于 user.ID）
func (c *Client) InvalidateUserCacheByID(ctx context.Context, userID string) error {
	if c.cache == nil {
		return nil
	}
	return c.cache.Delete(ctx, userID)
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

// DeleteUserSession 删除用户在指定应用的 Casdoor 会话（用于组织不匹配时强制登出）
// org 为用户实际所属组织（如 "built-in"），而非本应用配置的组织
func (c *Client) DeleteUserSession(org, username string) error {
	// 直接构造 session 对象，避免 SDK 的 GetSession 硬编码 c.OrganizationName
	session := &casdoorsdk.Session{
		Owner:       org,
		Name:        username,
		Application: c.applicationName,
	}
	if _, err := casdoorsdk.DeleteSession(session); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}
