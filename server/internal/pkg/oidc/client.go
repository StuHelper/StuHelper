// Package oidc 封装标准 OIDC 客户端。
package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

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
	var err error
	cfg, err = validateLocalCasdoorConfig(cfg)
	if err != nil {
		return nil, err
	}

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

func validateLocalCasdoorConfig(cfg config.CasdoorConfig) (config.CasdoorConfig, error) {
	issuer, err := validateCasdoorURL("issuer", cfg.Issuer, true)
	if err != nil {
		return cfg, err
	}
	cfg.Issuer = issuer

	publicAuthBaseURL, err := validateCasdoorURL("public auth base URL", cfg.PublicAuthBaseURL, false)
	if err != nil {
		return cfg, err
	}
	cfg.PublicAuthBaseURL = publicAuthBaseURL

	if err := validateOAuth2ApplicationConfig(cfg); err != nil {
		return cfg, err
	}
	if _, err := introspectionOAuth2Config(oauth2.Endpoint{}, cfg); err != nil {
		return cfg, err
	}
	if endpoint := strings.TrimSpace(cfg.IntrospectionEndpoint); endpoint != "" {
		endpoint, err = validateIntrospectionEndpoint(endpoint)
		if err != nil {
			return cfg, err
		}
		cfg.IntrospectionEndpoint = endpoint
	}
	return cfg, nil
}

func validateOAuth2ApplicationConfig(cfg config.CasdoorConfig) error {
	for _, input := range oauth2ApplicationInputs(cfg) {
		if _, _, err := oauth2ConfigFromInput(oauth2.Endpoint{}, input); err != nil {
			return err
		}
	}
	return nil
}

func validateCasdoorURL(name, raw string, required bool) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if !required {
			return "", nil
		}
		return "", fmt.Errorf("%w: %s is required", ErrApplicationNotConfigured, name)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("oidc: invalid %s %q: %w", name, raw, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("oidc: %s %q must be an absolute http(s) URL", name, raw)
	}
	if scheme := strings.ToLower(parsed.Scheme); scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("oidc: %s %q must use http or https", name, raw)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("oidc: %s %q must not include user info", name, raw)
	}
	if parsed.RawQuery != "" {
		return "", fmt.Errorf("oidc: %s %q must not include a query", name, raw)
	}
	if parsed.Fragment != "" {
		return "", fmt.Errorf("oidc: %s %q must not include a fragment", name, raw)
	}
	return raw, nil
}
