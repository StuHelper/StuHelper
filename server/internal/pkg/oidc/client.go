// Package oidc 封装标准 OIDC 客户端，用于 Zitadel SSO 认证。
package oidc

import (
	"context"
	"encoding/json"
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
	provider   *gooidc.Provider
	verifier   *gooidc.IDTokenVerifier
	oauth2Cfg  oauth2.Config
	projectID  string
	httpClient *http.Client
}

// NewClient 基于 Zitadel 配置创建 OIDC 客户端。
// 自动发现 issuer 的 OIDC 配置（JWKS、授权端点、Token 端点等）。
func NewClient(ctx context.Context, cfg config.ZitadelConfig) (*Client, error) {
	httpClient := newOIDCHTTPClient(cfg, "zitadel_oidc")

	// 将自定义 HTTP 客户端注入到 OIDC Provider 发现过程
	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	provider, err := gooidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc: failed to discover provider at %s: %w", cfg.Issuer, err)
	}

	verifier := provider.Verifier(&gooidc.Config{
		ClientID: cfg.ClientID,
	})

	oauth2Cfg := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{gooidc.ScopeOpenID, "profile", "email", "offline_access", "urn:zitadel:iam:org:project:id:zitadel:aud"},
	}

	return &Client{
		provider:   provider,
		verifier:   verifier,
		oauth2Cfg:  oauth2Cfg,
		projectID:  cfg.ProjectID,
		httpClient: httpClient,
	}, nil
}

// GetAuthURL 生成 OIDC 授权 URL（授权码 + PKCE）。
// 返回 (authURL, codeVerifier)。调用方必须将 codeVerifier 持久化（如 Redis），
// 在 callback 时传给 ExchangeCode。
func (c *Client) GetAuthURL(state string) (string, string) {
	verifier := oauth2.GenerateVerifier()
	authURL := c.oauth2Cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	return authURL, verifier
}

// ExchangeCode 用授权码 + PKCE code_verifier 交换 Token。
// codeVerifier 为空时退化为无 PKCE 的交换（兼容测试场景）。
func (c *Client) ExchangeCode(ctx context.Context, code, codeVerifier string) (*oauth2.Token, error) {
	start := time.Now()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	var opts []oauth2.AuthCodeOption
	if codeVerifier != "" {
		opts = append(opts, oauth2.VerifierOption(codeVerifier))
	}
	token, err := c.oauth2Cfg.Exchange(ctx, code, opts...)
	metrics.ObserveExternalRequest("zitadel_oidc", "exchange_code", start, err)
	if err != nil {
		return nil, fmt.Errorf("oidc: code exchange failed: %w", err)
	}
	return token, nil
}

// VerifyIDToken 验证 ID Token 并返回解析后的 Claims
func (c *Client) VerifyIDToken(ctx context.Context, rawIDToken string) (*Claims, error) {
	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("oidc: id_token verification failed: %w", err)
	}

	// 先解析标准字段
	claims := &Claims{}
	if err := idToken.Claims(claims); err != nil {
		return nil, fmt.Errorf("oidc: failed to parse claims: %w", err)
	}

	// 从原始 JSON 中提取 Zitadel 项目角色（动态 claim key 无法用 struct tag 解析）
	var rawJSON []byte
	if rawJSON, err = marshalIDTokenClaims(idToken); err == nil {
		roles, scoped, parseErr := ParseRolesFromRaw(rawJSON, c.projectID)
		if parseErr != nil {
			logger.L().Warn("oidc: failed to parse roles from id_token", zap.Error(parseErr))
		} else {
			claims.Roles = roles
			claims.OrgScopedRoles = scoped
		}
	}

	return claims, nil
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
	metrics.ObserveExternalRequest("zitadel_oidc", "refresh_token", start, err)
	if err != nil {
		return nil, fmt.Errorf("oidc: token refresh failed: %w", err)
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
	Roles          []string            `json:"-"` // 从原始 JSON 解析
	OrgScopedRoles map[string][]string `json:"-"`
}

// IntrospectToken 调用 Zitadel Token 内省端点验证 Bearer token。
// 用于 Bearer token 的即时吊销验证（Zitadel 端吊销后立即生效）。
// Cookie 端仍用本地 JWKS 验证（性能优先），Bearer 端用 introspection（安全优先）。
func (c *Client) IntrospectToken(ctx context.Context, accessToken string) (_ *IntrospectionResult, err error) {
	start := time.Now()
	defer func() {
		metrics.ObserveExternalRequest("zitadel_oidc", "introspect_token", start, err)
	}()
	// 发现 introspection 端点
	var providerCfg struct {
		IntrospectionEndpoint string `json:"introspection_endpoint"`
	}
	if err := c.provider.Claims(&providerCfg); err != nil || providerCfg.IntrospectionEndpoint == "" {
		// Fallback: Zitadel 标准路径
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
		return nil, fmt.Errorf("oidc: introspect request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("oidc: close introspection response body: %w", closeErr)
		}
	}()

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

	// 从原始 JSON 解析 Zitadel 项目角色
	roles, scoped, parseErr := ParseRolesFromRaw(rawJSON, c.projectID)
	if parseErr != nil {
		logger.L().Warn("oidc: failed to parse roles from introspection response", zap.Error(parseErr))
	} else {
		result.Roles = roles
		result.OrgScopedRoles = scoped
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
