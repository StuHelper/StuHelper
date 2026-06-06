package app

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

func TestDevBindPhoneSMSSenderStoresOTPForLocalVerification(t *testing.T) {
	fixture := redisfixture.Start(t)
	sender := newDevBindPhoneSMSSender(fixture.Client)

	require.NoError(t, sender.Send(context.Background(), "+8613800138000", "123456"))

	code, err := fixture.Client.Get(context.Background(), "dev:bind_phone_otp:13800138000").Result()
	require.NoError(t, err)
	require.Equal(t, "123456", code)

	ttl, err := fixture.Client.TTL(context.Background(), "dev:bind_phone_otp:13800138000").Result()
	require.NoError(t, err)
	require.Greater(t, ttl, 4*time.Minute)
}

func TestNormalizeDevBindPhoneOTPPhoneKeepsUnexpectedValueVisible(t *testing.T) {
	require.Equal(t, "13800138000", normalizeDevBindPhoneOTPPhone("+8613800138000"))
	require.Equal(t, "not-a-phone", normalizeDevBindPhoneOTPPhone(" not-a-phone "))
}
