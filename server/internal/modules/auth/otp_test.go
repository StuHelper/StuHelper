package auth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
)

func TestOTPVerify_AttemptsTTLAndGenerateClearsAttempts(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-auth-otp-secret-32-chars-long!", false))
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	svc := NewOTPService(rdb)
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
	assert.True(t, mr.Exists(attemptsKey))
	assert.Greater(t, mr.TTL(attemptsKey), time.Duration(0))

	mr.FastForward(otpCooldown)
	_, err = svc.Generate(ctx, phone)
	require.NoError(t, err)
	assert.False(t, mr.Exists(attemptsKey))
}

func TestOTPCleanup_RemovesCodeCooldownAndAttempts(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-auth-otp-secret-32-chars-long!", false))
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	svc := NewOTPService(rdb)
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

	assert.True(t, mr.Exists(codeKey))
	assert.True(t, mr.Exists(cooldownKey))
	assert.False(t, mr.Exists(attemptsKey))

	require.NoError(t, svc.Cleanup(ctx, phone))

	assert.False(t, mr.Exists(codeKey))
	assert.False(t, mr.Exists(cooldownKey))
	assert.False(t, mr.Exists(attemptsKey))
}
