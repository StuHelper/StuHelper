package oidc

import (
	"context"
	"errors"

	gooidc "github.com/coreos/go-oidc/v3/oidc"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
)

type providerUnavailableKeySet struct {
	inner gooidc.KeySet
}

func newProviderVerifier(
	ctx context.Context,
	provider *gooidc.Provider,
	cfg config.CasdoorConfig,
) (*gooidc.IDTokenVerifier, error) {
	var metadata struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := provider.Claims(&metadata); err != nil {
		return nil, err
	}
	if metadata.JWKSURI == "" {
		return nil, errors.New("oidc: jwks_uri unavailable")
	}
	keySet := providerUnavailableKeySet{inner: gooidc.NewRemoteKeySet(ctx, metadata.JWKSURI)}
	return gooidc.NewVerifier(cfg.Issuer, keySet, &gooidc.Config{ClientID: cfg.ClientID}), nil
}

func (k providerUnavailableKeySet) VerifySignature(ctx context.Context, jwt string) ([]byte, error) {
	payload, err := k.inner.VerifySignature(ctx, jwt)
	if isRemoteKeyFetchError(err) {
		return nil, errors.Join(ErrProviderUnavailable, err)
	}
	return payload, err
}
