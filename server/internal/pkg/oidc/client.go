// Package oidc 封装标准 OIDC 客户端。
package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
)

// Client 标准 OIDC 客户端
type Client struct {
	provider          *gooidc.Provider
	verifier          *gooidc.IDTokenVerifier
	oauth2Cfg         oauth2.Config
	oauth2Configs     map[string]oauth2.Config
	rolesClaim        string
	metricName        string
	httpClient        *http.Client
	introspectionURL  string
	introspectionCfg  oauth2.Config
	publicAuthBaseURL string
}

const (
	ApplicationWeb    = "web"
	ApplicationAdmin  = "admin"
	ApplicationUniapp = "uniapp"
)

var (
	ErrApplicationNotConfigured  = errors.New("oidc: application is not configured")
	ErrInvalidAudience           = errors.New("oidc: invalid token audience")
	ErrInvalidAccessToken        = errors.New("oidc: access token invalid")
	ErrInvalidIDToken            = errors.New("oidc: id_token invalid")
	ErrAuthorizationCodeRequired = errors.New("oidc: authorization code is required")
	ErrPKCEVerifierRequired      = errors.New("oidc: pkce code_verifier is required")
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
	introspectionURL, err := discoveredIntrospectionEndpoint(provider, cfg.IntrospectionEndpoint)
	if err != nil {
		return nil, err
	}
	introspectionCfg, err := introspectionOAuth2Config(provider.Endpoint(), cfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		provider:          provider,
		verifier:          verifier,
		oauth2Cfg:         oauth2Cfg,
		oauth2Configs:     oauth2Configs,
		rolesClaim:        defaultRolesClaim(cfg.RolesClaim),
		metricName:        "casdoor_oidc",
		httpClient:        httpClient,
		introspectionURL:  introspectionURL,
		introspectionCfg:  introspectionCfg,
		publicAuthBaseURL: cfg.PublicAuthBaseURL,
	}, nil
}
