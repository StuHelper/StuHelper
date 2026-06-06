package oidc

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
)

var ErrProviderUnavailable = errors.New("oidc: provider unavailable")
var ErrInvalidRefreshToken = errors.New("oidc: refresh token invalid")

const remoteKeyFetchErrorPrefix = "fetching keys "

func classifyOAuthError(err error) error {
	if err == nil {
		return nil
	}
	var retrieveErr *oauth2.RetrieveError
	if errors.As(err, &retrieveErr) && retrieveErr.Response != nil && retrieveErr.Response.StatusCode < 500 {
		return err
	}
	return errors.Join(ErrProviderUnavailable, err)
}

func classifyOAuthRefreshError(err error) error {
	if err == nil {
		return nil
	}
	var retrieveErr *oauth2.RetrieveError
	if errors.As(err, &retrieveErr) && retrieveErr.Response != nil && retrieveErr.Response.StatusCode < 500 {
		return errors.Join(ErrInvalidRefreshToken, err)
	}
	if strings.Contains(err.Error(), "server response missing access_token") {
		return errors.Join(ErrInvalidRefreshToken, err)
	}
	return errors.Join(ErrProviderUnavailable, err)
}

func normalizeRequiredRefreshToken(refreshToken, operation string) (string, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return "", fmt.Errorf("oidc: refresh token is required for %s: %w", operation, ErrInvalidRefreshToken)
	}
	return refreshToken, nil
}

func normalizeRequiredAuthorizationCode(code string) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", ErrAuthorizationCodeRequired
	}
	return code, nil
}

func normalizeRequiredPKCEVerifier(codeVerifier string) (string, error) {
	codeVerifier = strings.TrimSpace(codeVerifier)
	if codeVerifier == "" {
		return "", ErrPKCEVerifierRequired
	}
	return codeVerifier, nil
}

func classifyVerifierError(err error) error {
	if err == nil {
		return nil
	}
	if isVerifierProviderUnavailable(err) {
		return errors.Join(ErrProviderUnavailable, err)
	}
	return err
}

func isVerifierProviderUnavailable(err error) bool {
	if errors.Is(err, ErrProviderUnavailable) || isRemoteKeyFetchError(err) {
		return true
	}
	return strings.Contains(err.Error(), ErrProviderUnavailable.Error()+"\n"+remoteKeyFetchErrorPrefix)
}

func isRemoteKeyFetchError(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), remoteKeyFetchErrorPrefix)
}
