package auth

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

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
