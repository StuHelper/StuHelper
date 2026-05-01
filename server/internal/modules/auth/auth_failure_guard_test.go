package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

func TestAuthFailureGuardLocksIPAfterThreshold(t *testing.T) {
	fixture := redisfixture.Start(t)
	guard := NewAuthFailureGuard(fixture.Client)
	ctx := t.Context()
	ip := "203.0.113.10"

	for i := 0; i < authFailureIPLimit-1; i++ {
		require.NoError(t, guard.RecordFailure(ctx, ip))
		require.NoError(t, guard.EnsureAllowed(ctx, ip))
	}

	err := guard.RecordFailure(ctx, ip)
	require.ErrorIs(t, err, ErrAuthIPLocked)
	require.ErrorIs(t, guard.EnsureAllowed(ctx, ip), ErrAuthIPLocked)

	fixture.Server.FastForward(authFailureIPLock)
	require.NoError(t, guard.EnsureAllowed(ctx, ip))
}

func TestAuthFailureGuardClearFailuresKeepsLockState(t *testing.T) {
	fixture := redisfixture.Start(t)
	guard := NewAuthFailureGuard(fixture.Client)
	ctx := t.Context()
	ip := "203.0.113.11"

	for i := 0; i < authFailureIPLimit-1; i++ {
		require.NoError(t, guard.RecordFailure(ctx, ip))
	}
	require.NoError(t, guard.ClearFailures(ctx, ip))

	for i := 0; i < authFailureIPLimit-1; i++ {
		require.NoError(t, guard.RecordFailure(ctx, ip))
	}
	require.NoError(t, guard.EnsureAllowed(ctx, ip))

	err := guard.RecordFailure(ctx, ip)
	require.True(t, errors.Is(err, ErrAuthIPLocked))
	require.NoError(t, guard.ClearFailures(ctx, ip))
	require.ErrorIs(t, guard.EnsureAllowed(ctx, ip), ErrAuthIPLocked)
}

func TestAuthFailureGuardSoftLocksAccountAfterThreshold(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-auth-failure-account-secret!", false))
	fixture := redisfixture.Start(t)
	guard := NewAuthFailureGuard(fixture.Client)
	ctx := t.Context()
	account := "13800139000"

	for i := 0; i < authFailureAccountSoftLimit-1; i++ {
		require.NoError(t, guard.RecordAccountFailure(ctx, account))
		require.NoError(t, guard.EnsureAccountAllowed(ctx, account))
	}

	err := guard.RecordAccountFailure(ctx, account)
	require.ErrorIs(t, err, ErrAuthAccountSoftLocked)
	require.ErrorIs(t, guard.EnsureAccountAllowed(ctx, account), ErrAuthAccountSoftLocked)

	fixture.Server.FastForward(authFailureAccountSoftLock + time.Second)
	require.NoError(t, guard.EnsureAccountAllowed(ctx, account))
}

func TestAuthFailureGuardHardLocksAccountAfterCumulativeThreshold(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-auth-failure-account-secret!", false))
	fixture := redisfixture.Start(t)
	guard := NewAuthFailureGuard(fixture.Client)
	ctx := t.Context()
	account := "13800139001"

	for attempts := 1; attempts <= authFailureAccountHardLimit; attempts++ {
		err := guard.RecordAccountFailure(ctx, account)
		if attempts == authFailureAccountHardLimit {
			require.ErrorIs(t, err, ErrAuthAccountHardLocked)
			break
		}
		if errors.Is(err, ErrAuthAccountSoftLocked) {
			fixture.Server.FastForward(authFailureAccountSoftLock + time.Second)
			continue
		}
		require.NoError(t, err)
	}

	require.ErrorIs(t, guard.EnsureAccountAllowed(ctx, account), ErrAuthAccountHardLocked)
	require.NoError(t, guard.ClearAccountFailures(ctx, account))
	require.ErrorIs(t, guard.EnsureAccountAllowed(ctx, account), ErrAuthAccountHardLocked)
}
