package systemconfig

import (
	"strings"
	"sync"
	"time"
)

const (
	// #nosec G101 -- this is a configuration key name, not a secret.
	AuthAccessTokenTTLSecondsKey = "auth_access_token_ttl_seconds"
)

type AuthTokenPolicySnapshot struct {
	AccessTokenTTLSeconds int
	LoadedAt              time.Time
}

var authTokenPolicyCache = struct {
	mu       sync.RWMutex
	snapshot AuthTokenPolicySnapshot
}{}

func GetAuthTokenPolicySnapshot() AuthTokenPolicySnapshot {
	authTokenPolicyCache.mu.RLock()
	defer authTokenPolicyCache.mu.RUnlock()

	return authTokenPolicyCache.snapshot
}

func SetAuthTokenPolicySnapshot(snapshot AuthTokenPolicySnapshot) {
	authTokenPolicyCache.mu.Lock()
	defer authTokenPolicyCache.mu.Unlock()

	authTokenPolicyCache.snapshot = snapshot
}

func InvalidateAuthTokenPolicySnapshot() {
	SetAuthTokenPolicySnapshot(AuthTokenPolicySnapshot{})
}

func AffectsAuthTokenPolicy(key string) bool {
	return strings.TrimSpace(key) == AuthAccessTokenTTLSecondsKey
}

func ParseAuthAccessTokenTTL(value string) (int, error) {
	return ParseBoundedInt(value, 60, 86400)
}

func EffectiveAuthAccessTokenTTL(defaultSeconds int) int {
	snapshot := GetAuthTokenPolicySnapshot()
	if snapshot.AccessTokenTTLSeconds > 0 {
		return snapshot.AccessTokenTTLSeconds
	}
	return defaultSeconds
}
