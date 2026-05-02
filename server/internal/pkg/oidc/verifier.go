package oidc

import (
	"context"
	"errors"
	"sync"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
)

const defaultJWKSCacheTTL = 5 * time.Minute

type providerUnavailableKeySet struct {
	ctx       context.Context
	jwksURI   string
	cacheTTL  time.Duration
	now       func() time.Time
	mu        sync.Mutex
	inner     gooidc.KeySet
	expiresAt time.Time
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
	keySet := newProviderUnavailableKeySet(ctx, metadata.JWKSURI, defaultJWKSCacheTTL)
	return gooidc.NewVerifier(cfg.Issuer, keySet, &gooidc.Config{ClientID: cfg.ClientID}), nil
}

func newProviderUnavailableKeySet(ctx context.Context, jwksURI string, ttl time.Duration) *providerUnavailableKeySet {
	return &providerUnavailableKeySet{
		ctx:      context.WithoutCancel(ctx),
		jwksURI:  jwksURI,
		cacheTTL: ttl,
		now:      time.Now,
	}
}

func (k *providerUnavailableKeySet) VerifySignature(ctx context.Context, jwt string) ([]byte, error) {
	payload, err := k.current().VerifySignature(ctx, jwt)
	if isRemoteKeyFetchError(err) {
		return nil, errors.Join(ErrProviderUnavailable, err)
	}
	return payload, err
}

func (k *providerUnavailableKeySet) current() gooidc.KeySet {
	k.mu.Lock()
	defer k.mu.Unlock()

	now := k.now()
	if k.inner == nil || !now.Before(k.expiresAt) {
		k.inner = gooidc.NewRemoteKeySet(k.ctx, k.jwksURI)
		k.expiresAt = now.Add(k.cacheTTL)
	}
	return k.inner
}
