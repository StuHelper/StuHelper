package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/StuHelper/StuHelper/server/internal/pkg/metrics"
)

var ErrRevocationEndpointUnavailable = errors.New("oidc: revocation endpoint unavailable")

type tokenFamilyRevocationMode uint8

const (
	tokenFamilyRevocationRFC7009 tokenFamilyRevocationMode = iota + 1
	tokenFamilyRevocationCasdoorLogout
)

type tokenFamilyRevocationEndpoint struct {
	url  string
	mode tokenFamilyRevocationMode
}

// RevokeTokenFamily invalidates the provider token family owned by the default
// web application.
func (c *Client) RevokeTokenFamily(ctx context.Context, accessToken, refreshToken string) error {
	return c.RevokeTokenFamilyForApplication(ctx, ApplicationWeb, accessToken, refreshToken)
}

// RevokeTokenFamilyForApplication chooses the provider contract advertised by
// discovery. A real RFC 7009 endpoint receives the refresh token. Casdoor's
// exact /api/logout compatibility endpoint instead requires the provider
// access token as id_token_hint and invalidates the database row that owns both
// the access and refresh credentials.
func (c *Client) RevokeTokenFamilyForApplication(
	ctx context.Context,
	appKey, accessToken, refreshToken string,
) (err error) {
	cfg, err := c.oauth2ConfigForApplication(appKey)
	if err != nil {
		return err
	}

	endpoint, err := c.tokenFamilyRevocationEndpoint()
	if err != nil {
		return err
	}

	start := time.Now()
	defer func() {
		metrics.ObserveExternalRequest(c.metricName, "revoke_token_family", start, err)
	}()

	var req *http.Request
	switch endpoint.mode {
	case tokenFamilyRevocationRFC7009:
		refreshToken, err = normalizeRequiredRefreshToken(refreshToken, "revocation")
		if err != nil {
			return err
		}
		req, err = newRFC7009RevocationRequest(ctx, endpoint.url, refreshToken, cfg)
	case tokenFamilyRevocationCasdoorLogout:
		accessToken = strings.TrimSpace(accessToken)
		if accessToken == "" {
			// Sessions created before ProviderAccessTokenEnc was introduced
			// still contain the encrypted provider refresh token. Casdoor
			// refresh rotation deletes the old row; immediately logging out the
			// returned family then invalidates the replacement row as well.
			refreshToken, err = normalizeRequiredRefreshToken(refreshToken, "legacy revocation")
			if err != nil {
				return fmt.Errorf("oidc: access or refresh token is required for Casdoor revocation: %w", err)
			}
			rotated, refreshErr := c.RefreshTokenForApplication(ctx, appKey, refreshToken)
			if refreshErr != nil {
				if casdoorRefreshFamilyAlreadyInactive(refreshErr) {
					// In Casdoor, invalid/expired refresh means the backing row
					// is missing or has expires_in <= 0, so both credentials in
					// that family are already inactive.
					return nil
				}
				return fmt.Errorf("oidc: rotate legacy token family for revocation: %w", refreshErr)
			}
			accessToken, err = normalizeRequiredAccessToken(rotated.AccessToken, "legacy revocation")
			if err != nil {
				return err
			}
		}
		req, err = newCasdoorLogoutRequest(ctx, endpoint.url, accessToken, cfg)
	default:
		return ErrRevocationEndpointUnavailable
	}
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
	if endpoint.mode == tokenFamilyRevocationCasdoorLogout {
		return handleCasdoorLogoutResponse(resp)
	}
	return handleRFC7009RevocationResponse(resp)
}

func casdoorRefreshFamilyAlreadyInactive(err error) bool {
	var retrieveErr *oauth2.RetrieveError
	return errors.As(err, &retrieveErr) && retrieveErr.ErrorCode == "invalid_grant"
}

func (c *Client) SupportsTokenFamilyRevocation() (bool, error) {
	if c == nil || c.provider == nil {
		return false, nil
	}
	_, err := c.tokenFamilyRevocationEndpoint()
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrRevocationEndpointUnavailable) {
		return false, nil
	}
	return false, err
}

func (c *Client) tokenFamilyRevocationEndpoint() (tokenFamilyRevocationEndpoint, error) {
	var metadata struct {
		RevocationEndpoint string `json:"revocation_endpoint"`
		EndSessionEndpoint string `json:"end_session_endpoint"`
		Issuer             string `json:"issuer"`
	}
	if err := c.provider.Claims(&metadata); err != nil {
		return tokenFamilyRevocationEndpoint{}, fmt.Errorf("oidc: load revocation metadata: %w", err)
	}
	endpoint := strings.TrimSpace(metadata.RevocationEndpoint)
	if endpoint != "" {
		validated, err := validateDiscoveredRevocationEndpoint(endpoint)
		if err != nil {
			return tokenFamilyRevocationEndpoint{}, err
		}
		return tokenFamilyRevocationEndpoint{
			url:  validated,
			mode: tokenFamilyRevocationRFC7009,
		}, nil
	}

	// The repository-pinned Casdoor release advertises /api/logout only as the
	// OIDC end_session_endpoint. It is not RFC 7009: its server implementation
	// consumes id_token_hint and returns a JSON business status. Do not treat an
	// arbitrary provider logout URL as this compatibility contract.
	validated, err := casdoorEndSessionRevocationEndpoint(metadata.Issuer, metadata.EndSessionEndpoint)
	if err != nil {
		return tokenFamilyRevocationEndpoint{}, err
	}
	return tokenFamilyRevocationEndpoint{
		url:  validated,
		mode: tokenFamilyRevocationCasdoorLogout,
	}, nil
}

func validateDiscoveredRevocationEndpoint(endpoint string) (string, error) {
	validated, err := validateCasdoorURL("revocation endpoint", endpoint, true)
	if err != nil {
		return "", fmt.Errorf("oidc: invalid revocation endpoint: %w", err)
	}
	return validated, nil
}

func casdoorEndSessionRevocationEndpoint(issuer, endpoint string) (string, error) {
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
	validatedIssuer, err := validateCasdoorURL("issuer", issuer, true)
	if err != nil {
		return "", fmt.Errorf("oidc: invalid discovery issuer: %w", err)
	}
	parsedIssuer, err := url.Parse(validatedIssuer)
	if err != nil {
		return "", fmt.Errorf("oidc: parse discovery issuer: %w", err)
	}
	if !sameHTTPOrigin(parsedIssuer, parsed) {
		return "", errors.New("oidc: Casdoor end session endpoint must share the issuer origin")
	}
	return validated, nil
}

func sameHTTPOrigin(left, right *url.URL) bool {
	if left == nil || right == nil || !strings.EqualFold(left.Scheme, right.Scheme) {
		return false
	}
	return canonicalHTTPOriginHost(left) == canonicalHTTPOriginHost(right)
}

func canonicalHTTPOriginHost(parsed *url.URL) string {
	port := parsed.Port()
	if port == "" {
		switch strings.ToLower(parsed.Scheme) {
		case "http":
			port = "80"
		case "https":
			port = "443"
		}
	}
	return net.JoinHostPort(strings.ToLower(parsed.Hostname()), port)
}

func newRFC7009RevocationRequest(
	ctx context.Context,
	endpoint,
	refreshToken string,
	cfg oauth2.Config,
) (*http.Request, error) {
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

func newCasdoorLogoutRequest(
	ctx context.Context,
	endpoint,
	accessToken string,
	cfg oauth2.Config,
) (*http.Request, error) {
	body := url.Values{
		"id_token_hint": []string{accessToken},
		"client_id":     []string{cfg.ClientID},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oidc: Casdoor logout request build failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

func handleRFC7009RevocationResponse(resp *http.Response) error {
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

func handleCasdoorLogoutResponse(resp *http.Response) error {
	if resp.StatusCode >= http.StatusInternalServerError {
		return fmt.Errorf("%w: oidc: Casdoor logout returned status %d", ErrProviderUnavailable, resp.StatusCode)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("oidc: Casdoor logout returned status %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4097))
	if err != nil {
		return fmt.Errorf("oidc: read Casdoor logout response: %w", err)
	}
	if len(raw) > 4096 {
		return errors.New("oidc: Casdoor logout response exceeds 4096 bytes")
	}
	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return fmt.Errorf("oidc: decode Casdoor logout response: %w", err)
	}
	if result.Status != "ok" {
		return errors.New("oidc: Casdoor logout reported failure")
	}
	return nil
}
