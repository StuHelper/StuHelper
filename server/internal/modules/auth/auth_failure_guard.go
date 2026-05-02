package auth

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	authFailureIPLimit  = 20
	authFailureIPWindow = time.Hour
	authFailureIPLock   = time.Hour
)

const (
	authFailureIPPrefix = "auth:fail:ip:"
	authLockIPPrefix    = "auth:lock:ip:"
)

var ErrAuthIPLocked = errors.New("auth: too many failed attempts from this IP")

var authFailureIPScript = redis.NewScript(`
if redis.call("EXISTS", KEYS[2]) == 1 then
  return 1
end
local attempts = redis.call("INCR", KEYS[1])
if attempts == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
if attempts >= tonumber(ARGV[2]) then
  redis.call("SET", KEYS[2], "1", "PX", ARGV[3])
  redis.call("DEL", KEYS[1])
  return 1
end
return 0
`)

type AuthFailureGuard struct {
	rdb *redis.Client
}

func NewAuthFailureGuard(rdb *redis.Client) *AuthFailureGuard {
	return &AuthFailureGuard{rdb: rdb}
}

func (g *AuthFailureGuard) EnsureAllowed(ctx context.Context, ip string) error {
	if g == nil || g.rdb == nil {
		return errors.New("auth failure guard is not configured")
	}
	key, err := authLockIPKey(ip)
	if err != nil {
		return err
	}
	locked, err := g.rdb.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("auth failure guard: check ip lock: %w", err)
	}
	if locked > 0 {
		return ErrAuthIPLocked
	}
	return nil
}

func (g *AuthFailureGuard) RecordFailure(ctx context.Context, ip string) error {
	if g == nil || g.rdb == nil {
		return errors.New("auth failure guard is not configured")
	}
	failKey, lockKey, err := authFailureIPKeys(ip)
	if err != nil {
		return err
	}
	locked, err := authFailureIPScript.Run(
		ctx,
		g.rdb,
		[]string{failKey, lockKey},
		authFailureIPWindow.Milliseconds(),
		authFailureIPLimit,
		authFailureIPLock.Milliseconds(),
	).Int()
	if err != nil {
		return fmt.Errorf("auth failure guard: record ip failure: %w", err)
	}
	if locked == 1 {
		return ErrAuthIPLocked
	}
	return nil
}

func (g *AuthFailureGuard) ClearFailures(ctx context.Context, ip string) error {
	if g == nil || g.rdb == nil {
		return errors.New("auth failure guard is not configured")
	}
	key, err := authFailureIPKey(ip)
	if err != nil {
		return err
	}
	if err := g.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("auth failure guard: clear ip failures: %w", err)
	}
	return nil
}

func authFailureIPKeys(ip string) (string, string, error) {
	failKey, err := authFailureIPKey(ip)
	if err != nil {
		return "", "", err
	}
	lockKey, err := authLockIPKey(ip)
	if err != nil {
		return "", "", err
	}
	return failKey, lockKey, nil
}

func authFailureIPKey(ip string) (string, error) {
	normalized, err := normalizeAuthIP(ip)
	if err != nil {
		return "", err
	}
	return authFailureIPPrefix + normalized, nil
}

func authLockIPKey(ip string) (string, error) {
	normalized, err := normalizeAuthIP(ip)
	if err != nil {
		return "", err
	}
	return authLockIPPrefix + normalized, nil
}

func normalizeAuthIP(ip string) (string, error) {
	raw := strings.TrimSpace(ip)
	if raw == "" {
		return "", errors.New("auth failure guard: IP is required")
	}
	addr, err := netip.ParseAddr(raw)
	if err != nil {
		return "", fmt.Errorf("auth failure guard: invalid IP: %w", err)
	}
	return addr.String(), nil
}
