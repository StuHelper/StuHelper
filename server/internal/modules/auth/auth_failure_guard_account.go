package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/StuHelper/StuHelper/server/internal/pkg/phoneutil"
)

const (
	authFailureAccountSoftLimit  = 5
	authFailureAccountSoftWindow = 5 * time.Minute
	authFailureAccountSoftLock   = 5 * time.Minute
	authFailureAccountHardLimit  = 20
	authFailureAccountHardWindow = time.Hour
)

const (
	authFailureAccountSoftPrefix     = "auth:fail:account:soft:"
	authFailureAccountHardPrefix     = "auth:fail:account:hard:"
	authLockAccountSoftPrefix        = "auth:lock:account:soft:"
	authLockAccountHardPrefix        = "auth:lock:account:hard:"
	authFailureNoLock            int = 0
	authFailureSoftLock          int = 1
	authFailureHardLock          int = 2
)

var (
	ErrAuthAccountSoftLocked = errors.New("auth: account is temporarily locked")
	ErrAuthAccountHardLocked = errors.New("auth: account is locked")
)

var authFailureAccountScript = redis.NewScript(`
if redis.call("EXISTS", KEYS[4]) == 1 then
  return 2
end
if redis.call("EXISTS", KEYS[2]) == 1 then
  return 1
end
local soft_attempts = redis.call("INCR", KEYS[1])
if soft_attempts == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
local hard_attempts = redis.call("INCR", KEYS[3])
if hard_attempts == 1 then
  redis.call("PEXPIRE", KEYS[3], ARGV[4])
end
if hard_attempts >= tonumber(ARGV[5]) then
  redis.call("SET", KEYS[4], "1")
  redis.call("DEL", KEYS[1], KEYS[2], KEYS[3])
  return 2
end
if soft_attempts >= tonumber(ARGV[2]) then
  redis.call("SET", KEYS[2], "1", "PX", ARGV[3])
  redis.call("DEL", KEYS[1])
  return 1
end
return 0
`)

func (g *AuthFailureGuard) EnsureAccountAllowed(ctx context.Context, account string) error {
	if g == nil || g.rdb == nil {
		return errors.New("auth failure guard is not configured")
	}
	_, softLockKey, _, hardLockKey, err := authFailureAccountKeys(account)
	if err != nil {
		return err
	}
	locked, err := g.rdb.Exists(ctx, hardLockKey, softLockKey).Result()
	if err != nil {
		return fmt.Errorf("auth failure guard: check account lock: %w", err)
	}
	if locked == 0 {
		return nil
	}
	return g.resolveAccountLock(ctx, hardLockKey)
}

func (g *AuthFailureGuard) RecordAccountFailure(ctx context.Context, account string) error {
	if g == nil || g.rdb == nil {
		return errors.New("auth failure guard is not configured")
	}
	keys, err := authFailureAccountKeySet(account)
	if err != nil {
		return err
	}
	lockState, err := authFailureAccountScript.Run(
		ctx,
		g.rdb,
		keys,
		authFailureAccountSoftWindow.Milliseconds(),
		authFailureAccountSoftLimit,
		authFailureAccountSoftLock.Milliseconds(),
		authFailureAccountHardWindow.Milliseconds(),
		authFailureAccountHardLimit,
	).Int()
	if err != nil {
		return fmt.Errorf("auth failure guard: record account failure: %w", err)
	}
	return accountLockError(lockState)
}

func (g *AuthFailureGuard) ClearAccountFailures(ctx context.Context, account string) error {
	if g == nil || g.rdb == nil {
		return errors.New("auth failure guard is not configured")
	}
	softFailKey, softLockKey, hardFailKey, _, err := authFailureAccountKeys(account)
	if err != nil {
		return err
	}
	if err := g.rdb.Del(ctx, softFailKey, softLockKey, hardFailKey).Err(); err != nil {
		return fmt.Errorf("auth failure guard: clear account failures: %w", err)
	}
	return nil
}

func (g *AuthFailureGuard) ClearAccountLock(ctx context.Context, account string) error {
	if g == nil || g.rdb == nil {
		return errors.New("auth failure guard is not configured")
	}
	softFailKey, softLockKey, hardFailKey, hardLockKey, err := authFailureAccountKeys(account)
	if err != nil {
		return err
	}
	if err := g.rdb.Del(ctx, softFailKey, softLockKey, hardFailKey, hardLockKey).Err(); err != nil {
		return fmt.Errorf("auth failure guard: clear account lock: %w", err)
	}
	return nil
}

func (g *AuthFailureGuard) resolveAccountLock(ctx context.Context, hardLockKey string) error {
	hardLocked, err := g.rdb.Exists(ctx, hardLockKey).Result()
	if err != nil {
		return fmt.Errorf("auth failure guard: check hard account lock: %w", err)
	}
	if hardLocked > 0 {
		return ErrAuthAccountHardLocked
	}
	return ErrAuthAccountSoftLocked
}

func authFailureAccountKeySet(account string) ([]string, error) {
	softFailKey, softLockKey, hardFailKey, hardLockKey, err := authFailureAccountKeys(account)
	if err != nil {
		return nil, err
	}
	return []string{softFailKey, softLockKey, hardFailKey, hardLockKey}, nil
}

func authFailureAccountKeys(account string) (string, string, string, string, error) {
	normalized, err := normalizeAuthAccount(account)
	if err != nil {
		return "", "", "", "", err
	}
	return authFailureAccountSoftPrefix + normalized,
		authLockAccountSoftPrefix + normalized,
		authFailureAccountHardPrefix + normalized,
		authLockAccountHardPrefix + normalized,
		nil
}

func normalizeAuthAccount(account string) (string, error) {
	normalized := strings.TrimSpace(account)
	if normalized == "" {
		return "", errors.New("auth failure guard: account is required")
	}
	hash, err := phoneutil.HashLookup(normalized)
	if err != nil {
		return "", fmt.Errorf("auth failure guard: hash account: %w", err)
	}
	return hash, nil
}

func accountLockError(lockState int) error {
	switch lockState {
	case authFailureNoLock:
		return nil
	case authFailureSoftLock:
		return ErrAuthAccountSoftLocked
	case authFailureHardLock:
		return ErrAuthAccountHardLocked
	default:
		return fmt.Errorf("auth failure guard: unknown account lock state %d", lockState)
	}
}
