package oidc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

var ErrRevocationEndpointUnavailable = errors.New("oidc: revocation endpoint unavailable")

// RevokeRefreshToken calls the provider revocation endpoint for a refresh token.
func (c *Client) RevokeRefreshToken(ctx context.Context, refreshToken string) (err error) {
	return c.RevokeRefreshTokenForApplication(ctx, ApplicationWeb, refreshToken)
}

func (c *Client) RevokeRefreshTokenForApplication(ctx context.Context, appKey, refreshToken string) (err error) {
	refreshToken, err = normalizeRequiredRefreshToken(refreshToken, "revocation")
	if err != nil {
		return err
	}
	cfg, err := c.oauth2ConfigForApplication(appKey)
	if err != nil {
		return err
	}

	endpoint, err := c.revocationEndpoint()
	if err != nil {
		return err
	}

	start := time.Now()
	defer func() {
		metrics.ObserveExternalRequest(c.metricName, "revoke_refresh_token", start, err)
	}()

	req, err := newRevocationRequest(ctx, endpoint, refreshToken, cfg)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("oidc: revoke request failed: %w", errors.Join(ErrProviderUnavailable, err))
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("oidc: close revocation response body: %w", closeErr)
		}
	}()
	return handleRevocationResponse(resp)
}

func (c *Client) SupportsRefreshTokenRevocation() (bool, error) {
	if c == nil || c.provider == nil {
		return false, nil
	}
	_, err := c.revocationEndpoint()
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrRevocationEndpointUnavailable) {
		return false, nil
	}
	return false, err
}

func (c *Client) revocationEndpoint() (string, error) {
	var metadata struct {
		RevocationEndpoint string `json:"revocation_endpoint"`
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}
	if err := c.provider.Claims(&metadata); err != nil {
		return "", fmt.Errorf("oidc: load revocation metadata: %w", err)
	}
	endpoint := strings.TrimSpace(metadata.RevocationEndpoint)
	if endpoint != "" {
		return validateDiscoveredRevocationEndpoint(endpoint)
	}

	// Casdoor exposes its POST token-revocation API at /api/logout, but releases
	// including the repository-pinned 3.31.1 advertise that URL only as the
	// OIDC end_session_endpoint. Do not treat arbitrary provider logout URLs as
	// RFC 7009 endpoints: the exact Casdoor path is the compatibility boundary.
	return casdoorEndSessionRevocationEndpoint(metadata.EndSessionEndpoint)
}

func validateDiscoveredRevocationEndpoint(endpoint string) (string, error) {
	validated, err := validateCasdoorURL("revocation endpoint", endpoint, true)
	if err != nil {
		return "", fmt.Errorf("oidc: invalid revocation endpoint: %w", err)
	}
	return validated, nil
}

func casdoorEndSessionRevocationEndpoint(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", ErrRevocationEndpointUnavailable
	}
	validated, err := validateCasdoorURL("end session endpoint", endpoint, true)
	if err != nil {
		return "", fmt.Errorf("oidc: invalid end session endpoint: %w", err)
	}
	parsed, err := url.Parse(validated)
	if err != nil {
		return "", fmt.Errorf("oidc: parse end session endpoint: %w", err)
	}
	if parsed.EscapedPath() != "/api/logout" {
		return "", ErrRevocationEndpointUnavailable
	}
	return validated, nil
}

func newRevocationRequest(ctx context.Context, endpoint, refreshToken string, cfg oauth2.Config) (*http.Request, error) {
	body := url.Values{
		"token":           []string{refreshToken},
		"token_type_hint": []string{"refresh_token"},
	}
	if cfg.ClientSecret == "" {
		body.Set("client_id", cfg.ClientID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oidc: revoke request build failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if cfg.ClientSecret != "" {
		req.SetBasicAuth(cfg.ClientID, cfg.ClientSecret)
	}
	return req, nil
}

func handleRevocationResponse(resp *http.Response) error {
	if resp.StatusCode >= http.StatusInternalServerError {
		return fmt.Errorf("%w: oidc: revoke returned status %d", ErrProviderUnavailable, resp.StatusCode)
	}
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("oidc: read revocation error response: %w", err)
	}
	return fmt.Errorf("oidc: revoke returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
}
