package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/modules/user"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
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

func TestDevBindPhoneOTPGeneratorRemovesCapturedOTPAfterConsume(t *testing.T) {
	fixture := redisfixture.Start(t)
	sender := newDevBindPhoneSMSSender(fixture.Client)
	base := &stubDevBindPhoneOTPGenerator{}
	generator := newDevBindPhoneOTPGenerator(base, fixture.Client)

	require.NoError(t, sender.Send(context.Background(), "+8613800138000", "123456"))

	require.NoError(t, generator.Consume(context.Background(), "+8613800138000", "123456"))

	require.Equal(t, "+8613800138000", base.consumedPhone)
	require.Equal(t, "123456", base.consumedCode)
	_, err := fixture.Client.Get(context.Background(), "dev:bind_phone_otp:13800138000").Result()
	require.ErrorIs(t, err, redis.Nil)
}

func TestDevBindPhoneOTPGeneratorKeepsCapturedOTPWhenConsumeFails(t *testing.T) {
	fixture := redisfixture.Start(t)
	sender := newDevBindPhoneSMSSender(fixture.Client)
	consumeErr := errors.New("consume failed")
	generator := newDevBindPhoneOTPGenerator(
		&stubDevBindPhoneOTPGenerator{consumeErr: consumeErr},
		fixture.Client,
	)

	require.NoError(t, sender.Send(context.Background(), "13800138000", "123456"))

	err := generator.Consume(context.Background(), "13800138000", "123456")
	require.ErrorIs(t, err, consumeErr)

	code, err := fixture.Client.Get(context.Background(), "dev:bind_phone_otp:13800138000").Result()
	require.NoError(t, err)
	require.Equal(t, "123456", code)
}

type stubDevBindPhoneOTPGenerator struct {
	consumeErr    error
	consumedPhone string
	consumedCode  string
}

func (s *stubDevBindPhoneOTPGenerator) IssueCode(context.Context, string, user.SMSSender) error {
	return nil
}

func (s *stubDevBindPhoneOTPGenerator) CooldownSeconds() int {
	return 60
}

func (s *stubDevBindPhoneOTPGenerator) Check(context.Context, string, string) error {
	return nil
}

func (s *stubDevBindPhoneOTPGenerator) Consume(_ context.Context, phone, code string) error {
	s.consumedPhone = phone
	s.consumedCode = code
	return s.consumeErr
}
