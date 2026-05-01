package oidc

import (
	"errors"
	"strings"

	"golang.org/x/oauth2"
)

var ErrProviderUnavailable = errors.New("oidc: provider unavailable")

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
