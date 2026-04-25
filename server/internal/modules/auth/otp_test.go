package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

func TestOTPVerify_AttemptsTTLAndGenerateClearsAttempts(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-auth-otp-secret-32-chars-long!", false))
	fixture := redisfixture.Start(t)

	svc := NewOTPService(fixture.Client)
	ctx := context.Background()
	phone := "+8613800000000"

	code, err := svc.Generate(ctx, phone)
	require.NoError(t, err)

	wrongCode := "000000"
	if code == wrongCode {
		wrongCode = "000001"
	}
	err = svc.Verify(ctx, phone, wrongCode)
	require.ErrorIs(t, err, ErrOTPInvalidCode)

	phoneKey, err := phoneutil.HashLookup(phone)
	require.NoError(t, err)
	attemptsKey := otpAttemptsPrefix + phoneKey
	assert.True(t, fixture.Server.Exists(attemptsKey))
	assert.Greater(t, fixture.Server.TTL(attemptsKey), time.Duration(0))

	fixture.Server.FastForward(otpCooldown)
	_, err = svc.Generate(ctx, phone)
	require.NoError(t, err)
	assert.False(t, fixture.Server.Exists(attemptsKey))
}

func TestOTPCleanup_RemovesCodeCooldownAndAttempts(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-auth-otp-secret-32-chars-long!", false))
	fixture := redisfixture.Start(t)

	svc := NewOTPService(fixture.Client)
	ctx := context.Background()
	phone := "13800138000"

	code, err := svc.Generate(ctx, phone)
	require.NoError(t, err)
	require.NotEmpty(t, code)

	phoneKey, err := phoneutil.HashLookup(phone)
	require.NoError(t, err)
	codeKey := otpCodePrefix + phoneKey
	cooldownKey := otpCooldownPrefix + phoneKey
	attemptsKey := otpAttemptsPrefix + phoneKey

	assert.True(t, fixture.Server.Exists(codeKey))
	assert.True(t, fixture.Server.Exists(cooldownKey))
	assert.False(t, fixture.Server.Exists(attemptsKey))

	require.NoError(t, svc.Cleanup(ctx, phone))

	assert.False(t, fixture.Server.Exists(codeKey))
	assert.False(t, fixture.Server.Exists(cooldownKey))
	assert.False(t, fixture.Server.Exists(attemptsKey))
}
