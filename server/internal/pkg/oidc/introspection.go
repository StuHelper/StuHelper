package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

// IntrospectionResult Token 内省结果
type IntrospectionResult struct {
	Active         bool                `json:"active"`
	Sub            string              `json:"sub"`
	Username       string              `json:"username"`
	Email          string              `json:"email"`
	Name           string              `json:"name"`
	Scope          string              `json:"scope"`
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

func (r *IntrospectionResult) Scopes() []string {
	if r == nil {
		return nil
	}
	return strings.Fields(r.Scope)
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
	accessToken, err = normalizeRequiredAccessToken(accessToken, "introspection")
	if err != nil {
		return nil, err
	}

	start := time.Now()
	defer func() {
		metrics.ObserveExternalRequest(c.metricName, "introspect_token", start, err)
	}()

	resp, err := c.sendIntrospectionRequest(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("oidc: close introspection response body: %w", closeErr)
		}
	}()

	if err := verifyIntrospectionStatus(resp.StatusCode); err != nil {
		return nil, err
	}
	rawJSON, err := decodeIntrospectionBody(resp.Body)
	if err != nil {
		return nil, err
	}
	result, err := parseIntrospectionResult(rawJSON)
	if err != nil {
		return nil, err
	}
	if result.Active {
		c.decorateIntrospectionResult(&result, rawJSON)
	}
	return &result, nil
}

func (c *Client) sendIntrospectionRequest(ctx context.Context, accessToken string) (*http.Response, error) {
	body := url.Values{"token": []string{accessToken}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.introspectionURL, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("oidc: introspect request build failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.introspectionCfg.ClientID, c.introspectionCfg.ClientSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oidc: introspect request failed: %w", errors.Join(ErrProviderUnavailable, err))
	}
	return resp, nil
}

func verifyIntrospectionStatus(statusCode int) error {
	if statusCode >= http.StatusInternalServerError {
		return fmt.Errorf("%w: oidc: introspect returned status %d", ErrProviderUnavailable, statusCode)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("oidc: introspect returned status %d", statusCode)
	}
	return nil
}

func decodeIntrospectionBody(body io.Reader) (json.RawMessage, error) {
	var rawJSON json.RawMessage
	if err := json.NewDecoder(io.LimitReader(body, 1<<20)).Decode(&rawJSON); err != nil {
		return nil, fmt.Errorf("oidc: introspect decode failed: %w", err)
	}
	return rawJSON, nil
}

func parseIntrospectionResult(rawJSON json.RawMessage) (IntrospectionResult, error) {
	var result IntrospectionResult
	if err := json.Unmarshal(rawJSON, &result); err != nil {
		return result, fmt.Errorf("oidc: introspect unmarshal failed: %w", err)
	}
	return result, nil
}

func (c *Client) decorateIntrospectionResult(result *IntrospectionResult, rawJSON json.RawMessage) {
	result.AppID = appIDFromRawClaims(rawJSON)
	roles, parseErr := ParseProviderRolesFromRaw(rawJSON, c.rolesClaim)
	if parseErr != nil {
		logger.L().Warn("oidc: failed to parse roles from introspection response", zap.Error(parseErr))
		return
	}
	result.Roles = roles
}

func discoveredIntrospectionEndpoint(provider *gooidc.Provider, configured string) (string, error) {
	if endpoint := strings.TrimSpace(configured); endpoint != "" {
		return validateIntrospectionEndpoint(endpoint)
	}
	var providerCfg struct {
		IntrospectionEndpoint string `json:"introspection_endpoint"`
	}
	if err := provider.Claims(&providerCfg); err != nil {
		return "", fmt.Errorf("oidc: load introspection metadata: %w", err)
	}
	endpoint := strings.TrimSpace(providerCfg.IntrospectionEndpoint)
	if endpoint == "" {
		return "", fmt.Errorf("oidc: introspection endpoint unavailable")
	}
	return validateIntrospectionEndpoint(endpoint)
}

func validateIntrospectionEndpoint(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", fmt.Errorf("oidc: introspection endpoint unavailable")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("oidc: invalid introspection endpoint %q: %w", endpoint, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("oidc: introspection endpoint %q must be an absolute http(s) URL", endpoint)
	}
	if scheme := strings.ToLower(parsed.Scheme); scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("oidc: introspection endpoint %q must use http or https", endpoint)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("oidc: introspection endpoint %q must not include user info", endpoint)
	}
	if parsed.Fragment != "" {
		return "", fmt.Errorf("oidc: introspection endpoint %q must not include a fragment", endpoint)
	}
	return endpoint, nil
}

func introspectionOAuth2Config(endpoint oauth2.Endpoint, cfg config.CasdoorConfig) (oauth2.Config, error) {
	clientID := strings.TrimSpace(cfg.IntrospectionClientID)
	clientSecret := strings.TrimSpace(cfg.IntrospectionClientSecret)
	if clientID == "" || clientSecret == "" {
		return oauth2.Config{}, fmt.Errorf("%w: introspection requires client id and secret", ErrApplicationNotConfigured)
	}
	return oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     endpoint,
	}, nil
}
