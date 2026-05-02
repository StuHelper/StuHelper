// Package oidc 封装标准 OIDC 客户端。
package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

// Client 标准 OIDC 客户端
type Client struct {
	provider         *gooidc.Provider
	verifier         *gooidc.IDTokenVerifier
	oauth2Cfg        oauth2.Config
	oauth2Configs    map[string]oauth2.Config
	allowedClientIDs map[string]struct{}
	rolesClaim       string
	metricName       string
	httpClient       *http.Client
}

const (
	ApplicationWeb    = "web"
	ApplicationAdmin  = "admin"
	ApplicationUniapp = "uniapp"
)

var (
	ErrApplicationNotConfigured = errors.New("oidc: application is not configured")
	ErrInvalidAudience          = errors.New("oidc: invalid token audience")
)

// NewClient 基于 Casdoor 配置创建 OIDC 客户端。
// 自动发现 issuer 的 OIDC 配置（JWKS、授权端点、Token 端点等）。
func NewClient(ctx context.Context, cfg config.CasdoorConfig) (*Client, error) {
	httpClient := newOIDCHTTPClient(cfg, "casdoor_oidc")

	// 将自定义 HTTP 客户端注入到 OIDC Provider 发现过程
	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	provider, err := gooidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc: failed to discover provider at %s: %w", cfg.Issuer, err)
	}

	verifier, err := newProviderVerifier(ctx, provider, cfg)
	if err != nil {
		return nil, fmt.Errorf("oidc: failed to initialize verifier: %w", err)
	}

	oauth2Configs, err := oauth2ConfigsFromCasdoor(provider.Endpoint(), cfg)
	if err != nil {
		return nil, err
	}
	oauth2Cfg := oauth2Configs[ApplicationWeb]

	return &Client{
		provider:         provider,
		verifier:         verifier,
		oauth2Cfg:        oauth2Cfg,
		oauth2Configs:    oauth2Configs,
		allowedClientIDs: allowedClientIDsFromOAuth2Configs(oauth2Configs),
		rolesClaim:       defaultRolesClaim(cfg.RolesClaim),
		metricName:       "casdoor_oidc",
		httpClient:       httpClient,
	}, nil
}

func oauth2ConfigsFromCasdoor(endpoint oauth2.Endpoint, cfg config.CasdoorConfig) (map[string]oauth2.Config, error) {
	apps := map[string]oauth2.Config{}
	for _, input := range oauth2ApplicationInputs(cfg) {
		oauth2Cfg, ok, err := oauth2ConfigFromInput(endpoint, input)
		if err != nil {
			return nil, err
		}
		if ok {
			apps[input.appKey] = oauth2Cfg
		}
	}
	return apps, nil
}

type oauth2ApplicationInput struct {
	appKey       string
	clientID     string
	clientSecret string
	redirectURI  string
	required     bool
}

func oauth2ApplicationInputs(cfg config.CasdoorConfig) []oauth2ApplicationInput {
	return []oauth2ApplicationInput{
		{
			appKey:       ApplicationWeb,
			clientID:     cfg.ClientID,
			clientSecret: cfg.ClientSecret,
			redirectURI:  cfg.RedirectURI,
			required:     true,
		},
		{
			appKey:       ApplicationAdmin,
			clientID:     cfg.AdminClientID,
			clientSecret: cfg.AdminClientSecret,
			redirectURI:  cfg.AdminRedirectURI,
		},
		{
			appKey:       ApplicationUniapp,
			clientID:     cfg.UniappClientID,
			clientSecret: cfg.UniappClientSecret,
			redirectURI:  cfg.UniappRedirectURI,
		},
	}
}

func oauth2ConfigFromInput(endpoint oauth2.Endpoint, input oauth2ApplicationInput) (oauth2.Config, bool, error) {
	if input.clientID == "" && input.clientSecret == "" && input.redirectURI == "" && !input.required {
		return oauth2.Config{}, false, nil
	}
	if input.clientID == "" || input.clientSecret == "" || input.redirectURI == "" {
		return oauth2.Config{}, false, fmt.Errorf("%w: %s requires client id, secret and redirect uri", ErrApplicationNotConfigured, input.appKey)
	}
	return oauth2.Config{
		ClientID:     input.clientID,
		ClientSecret: input.clientSecret,
		RedirectURL:  input.redirectURI,
		Endpoint:     endpoint,
		Scopes:       []string{gooidc.ScopeOpenID, "profile", "email", "offline_access"},
	}, true, nil
}

func allowedClientIDsFromOAuth2Configs(configs map[string]oauth2.Config) map[string]struct{} {
	ids := make(map[string]struct{}, len(configs))
	for _, cfg := range configs {
		ids[cfg.ClientID] = struct{}{}
	}
	return ids
}

// GetAuthURL 生成 OIDC 授权 URL（授权码 + PKCE）。
// 返回 (authURL, codeVerifier)。调用方必须将 codeVerifier 持久化（如 Redis），
// 在 callback 时传给 ExchangeCode。
func (c *Client) GetAuthURL(state string) (string, string) {
	authURL, verifier, err := c.GetAuthURLForApplication(ApplicationWeb, state)
	if err != nil {
		panic(err)
	}
	return authURL, verifier
}

func (c *Client) GetAuthURLForApplication(appKey, state string) (string, string, error) {
	cfg, err := c.oauth2ConfigForApplication(appKey)
	if err != nil {
		return "", "", err
	}
	verifier := oauth2.GenerateVerifier()
	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	return authURL, verifier, nil
}

// GetStepUpAuthURL 生成 OIDC step-up 重认证 URL。
// 仅请求 provider 重新认证并要求 MFA；是否产生可用 mfa proof 仍取决于 provider 是否签发 amr/auth_time。
func (c *Client) GetStepUpAuthURL(state string) (string, string) {
	authURL, verifier, err := c.GetStepUpAuthURLForApplication(ApplicationWeb, state)
	if err != nil {
		panic(err)
	}
	return authURL, verifier
}

func (c *Client) GetStepUpAuthURLForApplication(appKey, state string) (string, string, error) {
	cfg, err := c.oauth2ConfigForApplication(appKey)
	if err != nil {
		return "", "", err
	}
	verifier := oauth2.GenerateVerifier()
	authURL := cfg.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("prompt", "login"),
		oauth2.SetAuthURLParam("max_age", "0"),
		oauth2.SetAuthURLParam("acr_values", "mfa"),
	)
	return authURL, verifier, nil
}

// ExchangeCode 用授权码 + PKCE code_verifier 交换 Token。
// codeVerifier 为空时退化为无 PKCE 的交换（兼容测试场景）。
func (c *Client) ExchangeCode(ctx context.Context, code, codeVerifier string) (*oauth2.Token, error) {
	return c.ExchangeCodeForApplication(ctx, ApplicationWeb, code, codeVerifier)
}

func (c *Client) ExchangeCodeForApplication(ctx context.Context, appKey, code, codeVerifier string) (*oauth2.Token, error) {
	cfg, err := c.oauth2ConfigForApplication(appKey)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	var opts []oauth2.AuthCodeOption
	if codeVerifier != "" {
		opts = append(opts, oauth2.VerifierOption(codeVerifier))
	}
	token, err := cfg.Exchange(ctx, code, opts...)
	metrics.ObserveExternalRequest(c.metricName, "exchange_code", start, err)
	if err != nil {
		return nil, fmt.Errorf("oidc: code exchange failed: %w", err)
	}
	return token, nil
}

func (c *Client) oauth2ConfigForApplication(appKey string) (oauth2.Config, error) {
	if appKey == "" {
		appKey = ApplicationWeb
	}
	cfg, ok := c.oauth2Configs[appKey]
	if !ok {
		return oauth2.Config{}, fmt.Errorf("%w: %s", ErrApplicationNotConfigured, appKey)
	}
	return cfg, nil
}

func (c *Client) ApplicationKeyForClientID(clientID string) string {
	for appKey, cfg := range c.oauth2Configs {
		if cfg.ClientID == clientID {
			return appKey
		}
	}
	return ""
}

// VerifyIDToken 验证 ID Token 并返回解析后的 Claims
func (c *Client) VerifyIDToken(ctx context.Context, rawIDToken string) (*Claims, error) {
	if err := validateJWTSigningAlgorithm(rawIDToken); err != nil {
		return nil, err
	}
	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("oidc: id_token verification failed: %w", classifyVerifierError(err))
	}
	if err := c.verifyIDTokenAudience(idToken.Audience); err != nil {
		return nil, err
	}

	// 先解析标准字段
	claims := &Claims{}
	if err := idToken.Claims(claims); err != nil {
		return nil, fmt.Errorf("oidc: failed to parse claims: %w", err)
	}

	// 从原始 JSON 中提取 provider-specific 角色 claim
	var rawJSON []byte
	if rawJSON, err = marshalIDTokenClaims(idToken); err == nil {
		claims.AppID = appIDFromRawClaims(rawJSON)
		roles, parseErr := ParseProviderRolesFromRaw(rawJSON, c.rolesClaim)
		if parseErr != nil {
			logger.L().Warn("oidc: failed to parse roles from id_token", zap.Error(parseErr))
		} else {
			claims.Roles = roles
		}
	}

	return claims, nil
}

func (c *Client) verifyIDTokenAudience(audience []string) error {
	for _, aud := range audience {
		if _, ok := c.allowedClientIDs[aud]; ok {
			return nil
		}
	}
	return fmt.Errorf("%w: %v", ErrInvalidAudience, audience)
}

// marshalIDTokenClaims 将 ID Token 的 claims 序列化为 JSON 以提取动态字段
func marshalIDTokenClaims(idToken *gooidc.IDToken) ([]byte, error) {
	var raw json.RawMessage
	if err := idToken.Claims(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// RefreshToken 用 refresh token 获取新的 token 对
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	start := time.Now()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	tokenSource := c.oauth2Cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	token, err := tokenSource.Token()
	metrics.ObserveExternalRequest(c.metricName, "refresh_token", start, err)
	if err != nil {
		return nil, fmt.Errorf("oidc: token refresh failed: %w", classifyOAuthError(err))
	}
	return token, nil
}

// IntrospectionResult Token 内省结果
type IntrospectionResult struct {
	Active         bool                `json:"active"`
	Sub            string              `json:"sub"`
	Username       string              `json:"username"`
	Email          string              `json:"email"`
	Name           string              `json:"name"`
	AMR            []string            `json:"amr,omitempty"`
	AuthTime       int64               `json:"auth_time,omitempty"`
	AppID          string              `json:"-"`
	Roles          []string            `json:"-"` // 从原始 JSON 解析
	OrgScopedRoles map[string][]string `json:"-"`
}

func (r *IntrospectionResult) GetAppID() string {
	if r == nil {
		return ""
	}
	return r.AppID
}

func (r *IntrospectionResult) MFAProofVerifiedAt() time.Time {
	if r == nil {
		return time.Time{}
	}
	return MFAProofVerifiedAt(r.AMR, r.AuthTime)
}

// IntrospectToken 调用 OIDC Token 内省端点验证 Bearer token。
// 用于 Bearer token 的即时吊销验证（IDP 端吊销后立即生效）。
// Cookie 端仍用本地 JWKS 验证（性能优先），Bearer 端用 introspection（安全优先）。
func (c *Client) IntrospectToken(ctx context.Context, accessToken string) (_ *IntrospectionResult, err error) {
	start := time.Now()
	defer func() {
		metrics.ObserveExternalRequest(c.metricName, "introspect_token", start, err)
	}()
	// 发现 introspection 端点
	var providerCfg struct {
		IntrospectionEndpoint string `json:"introspection_endpoint"`
	}
	if err := c.provider.Claims(&providerCfg); err != nil || providerCfg.IntrospectionEndpoint == "" {
		// Fallback: common OIDC introspection path
		providerCfg.IntrospectionEndpoint = fallbackIntrospectionEndpoint(c.oauth2Cfg.Endpoint.TokenURL)
		if providerCfg.IntrospectionEndpoint == "" {
			return nil, fmt.Errorf("oidc: introspection endpoint unavailable")
		}
	}

	// 构建 introspection 请求（Basic Auth = client_id:client_secret）
	body := url.Values{"token": []string{accessToken}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, providerCfg.IntrospectionEndpoint, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("oidc: introspect request build failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.oauth2Cfg.ClientID, c.oauth2Cfg.ClientSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oidc: introspect request failed: %w", errors.Join(ErrProviderUnavailable, err))
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("oidc: close introspection response body: %w", closeErr)
		}
	}()

	if resp.StatusCode >= http.StatusInternalServerError {
		return nil, fmt.Errorf("%w: oidc: introspect returned status %d", ErrProviderUnavailable, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oidc: introspect returned status %d", resp.StatusCode)
	}

	var rawJSON json.RawMessage
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rawJSON); err != nil {
		return nil, fmt.Errorf("oidc: introspect decode failed: %w", err)
	}

	var result IntrospectionResult
	if err := json.Unmarshal(rawJSON, &result); err != nil {
		return nil, fmt.Errorf("oidc: introspect unmarshal failed: %w", err)
	}

	if !result.Active {
		return &result, nil
	}

	result.AppID = appIDFromRawClaims(rawJSON)

	// 从原始 JSON 解析 provider-specific 角色 claim
	roles, parseErr := ParseProviderRolesFromRaw(rawJSON, c.rolesClaim)
	if parseErr != nil {
		logger.L().Warn("oidc: failed to parse roles from introspection response", zap.Error(parseErr))
	} else {
		result.Roles = roles
	}

	return &result, nil
}

func fallbackIntrospectionEndpoint(tokenURL string) string {
	if strings.TrimSpace(tokenURL) == "" {
		return ""
	}

	parsed, err := url.Parse(tokenURL)
	if err != nil {
		return ""
	}

	cleanPath := strings.TrimSpace(parsed.Path)
	if cleanPath == "" || cleanPath == "/" {
		parsed.Path = "/introspect"
		return parsed.String()
	}

	dir := path.Dir(cleanPath)
	if dir == "." || dir == "" {
		dir = "/"
	}
	parsed.Path = strings.TrimRight(dir, "/") + "/introspect"
	return parsed.String()
}

// ExtractIDToken 从 OAuth2 Token 的 extra 字段中提取 id_token
func ExtractIDToken(token *oauth2.Token) string {
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		return ""
	}
	return raw
}
