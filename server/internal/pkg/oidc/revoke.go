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

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

var ErrRevocationEndpointUnavailable = errors.New("oidc: revocation endpoint unavailable")

// RevokeRefreshToken calls the provider revocation endpoint for a refresh token.
func (c *Client) RevokeRefreshToken(ctx context.Context, refreshToken string) (err error) {
	if strings.TrimSpace(refreshToken) == "" {
		return errors.New("oidc: refresh token is required for revocation")
	}

	endpoint, err := c.revocationEndpoint()
	if err != nil {
		return err
	}

	start := time.Now()
	defer func() {
		metrics.ObserveExternalRequest(c.metricName, "revoke_refresh_token", start, err)
	}()

	req, err := c.newRevocationRequest(ctx, endpoint, refreshToken)
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

func (c *Client) revocationEndpoint() (string, error) {
	var metadata struct {
		RevocationEndpoint string `json:"revocation_endpoint"`
	}
	if err := c.provider.Claims(&metadata); err != nil {
		return "", fmt.Errorf("oidc: load revocation metadata: %w", err)
	}
	endpoint := strings.TrimSpace(metadata.RevocationEndpoint)
	if endpoint == "" {
		return "", ErrRevocationEndpointUnavailable
	}
	return endpoint, nil
}

func (c *Client) newRevocationRequest(ctx context.Context, endpoint, refreshToken string) (*http.Request, error) {
	body := url.Values{
		"token":           []string{refreshToken},
		"token_type_hint": []string{"refresh_token"},
	}
	if c.oauth2Cfg.ClientSecret == "" {
		body.Set("client_id", c.oauth2Cfg.ClientID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oidc: revoke request build failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if c.oauth2Cfg.ClientSecret != "" {
		req.SetBasicAuth(c.oauth2Cfg.ClientID, c.oauth2Cfg.ClientSecret)
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
