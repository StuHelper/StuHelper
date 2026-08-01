package oidc

import (
	"context"
	"errors"

	gooidc "github.com/coreos/go-oidc/v3/oidc"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/ctxutil"
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
	keySet := newProviderUnavailableKeySet(ctx, metadata.JWKSURI)
	return gooidc.NewVerifier(cfg.Issuer, keySet, &gooidc.Config{SkipClientIDCheck: true}), nil
}

func newProviderUnavailableKeySet(ctx context.Context, jwksURI string) *providerUnavailableKeySet {
	return &providerUnavailableKeySet{
		inner: gooidc.NewRemoteKeySet(ctxutil.WithoutCancel(ctx), jwksURI),
	}
}

func (k *providerUnavailableKeySet) VerifySignature(ctx context.Context, jwt string) ([]byte, error) {
	payload, err := k.inner.VerifySignature(ctx, jwt)
	if isRemoteKeyFetchError(err) {
		return nil, errors.Join(ErrProviderUnavailable, err)
	}
	return payload, err
}
