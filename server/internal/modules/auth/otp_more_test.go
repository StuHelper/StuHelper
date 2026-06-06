package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

type stubPhoneSMSSender func(ctx context.Context, phone, content string) error

func (s stubPhoneSMSSender) Send(ctx context.Context, phone, content string) error {
	return s(ctx, phone, content)
}

func newOTPServiceForTest(t *testing.T) (*OTPService, *redisfixture.Fixture) {
	t.Helper()
	require.NoError(t, crypto.InitHMACKey("test-auth-otp-secret-32-chars-long!", false))
	fixture := redisfixture.Start(t)
	return NewOTPService(fixture.Client), fixture
}

func TestOTPService_RateLimitAndCooldownHelpers(t *testing.T) {
	svc, _ := newOTPServiceForTest(t)
	ctx := context.Background()
	phone := "13800138000"

	assert.Equal(t, int(otpCooldown.Seconds()), OTPCooldownSeconds())

	for i := 0; i < otpPhoneLimit; i++ {
		require.NoError(t, svc.CheckPhoneRateLimit(ctx, phone))
	}
	err := svc.CheckPhoneRateLimit(ctx, phone)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOTPPhoneRateLimited)

	code, err := svc.Generate(ctx, phone)
	require.NoError(t, err)
	require.Len(t, code, otpLength)

	_, err = svc.Generate(ctx, phone)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOTPCooldown)
}

func TestOTPService_CleanupCodeOnlyAndVerifySuccess(t *testing.T) {
	svc, fixture := newOTPServiceForTest(t)
	ctx := context.Background()
	phone := "13800138000"

	_, err := svc.Generate(ctx, phone)
	require.NoError(t, err)
	phoneKey, err := phoneutil.HashLookup(phone)
	require.NoError(t, err)

	codeKey := otpCodePrefix + phoneKey
	cooldownKey := otpCooldownPrefix + phoneKey
	attemptsKey := otpAttemptsPrefix + phoneKey
	assert.True(t, fixture.Server.Exists(codeKey))
	assert.True(t, fixture.Server.Exists(cooldownKey))

	require.NoError(t, svc.CleanupCodeOnly(ctx, phone))
	assert.False(t, fixture.Server.Exists(codeKey))
	assert.True(t, fixture.Server.Exists(cooldownKey))

	fixture.Server.FastForward(otpCooldown)
	code, err := svc.Generate(ctx, phone)
	require.NoError(t, err)
	require.NoError(t, svc.Verify(ctx, phone, code))
	assert.False(t, fixture.Server.Exists(codeKey))
	assert.False(t, fixture.Server.Exists(attemptsKey))
}

func TestOTPService_CheckDoesNotConsumeCode(t *testing.T) {
	svc, fixture := newOTPServiceForTest(t)
	ctx := context.Background()
	phone := "13800138001"

	code, err := svc.Generate(ctx, phone)
	require.NoError(t, err)
	phoneKey, err := phoneutil.HashLookup(phone)
	require.NoError(t, err)
	codeKey := otpCodePrefix + phoneKey
	attemptsKey := otpAttemptsPrefix + phoneKey

	require.NoError(t, svc.Check(ctx, phone, code))
	assert.True(t, fixture.Server.Exists(codeKey))

	require.NoError(t, svc.Consume(ctx, phone, code))
	assert.False(t, fixture.Server.Exists(codeKey))
	assert.False(t, fixture.Server.Exists(attemptsKey))
}

func TestOTPService_CheckRejectsMalformedCodeWithoutConsumingAttempt(t *testing.T) {
	svc, fixture := newOTPServiceForTest(t)
	ctx := context.Background()
	phone := "13800138004"

	code, err := svc.Generate(ctx, phone)
	require.NoError(t, err)
	phoneKey, err := phoneutil.HashLookup(phone)
	require.NoError(t, err)
	codeKey := otpCodePrefix + phoneKey
	attemptsKey := otpAttemptsPrefix + phoneKey

	for _, malformed := range []string{"12345", "1234567"} {
		err = svc.Check(ctx, phone, malformed)

		require.ErrorIs(t, err, ErrOTPInvalidCode)
	}
	assert.True(t, fixture.Server.Exists(codeKey))
	assert.False(t, fixture.Server.Exists(attemptsKey))
	require.NoError(t, svc.Check(ctx, phone, " "+code+" "))
	require.NoError(t, svc.Consume(ctx, phone, " "+code+" "))
	assert.False(t, fixture.Server.Exists(codeKey))
	assert.False(t, fixture.Server.Exists(attemptsKey))
}

func TestOTPService_ConsumeKeepsNewerCode(t *testing.T) {
	svc, fixture := newOTPServiceForTest(t)
	ctx := context.Background()
	phone := "13800138002"

	code, err := svc.Generate(ctx, phone)
	require.NoError(t, err)
	require.NoError(t, svc.Check(ctx, phone, code))

	fixture.Server.FastForward(otpCooldown)
	newCode, err := svc.Generate(ctx, phone)
	require.NoError(t, err)
	if newCode == code {
		newCode = "999999"
		if newCode == code {
			newCode = "000000"
		}
		phoneKey, keyErr := phoneutil.HashLookup(phone)
		require.NoError(t, keyErr)
		require.NoError(t, fixture.Client.Set(ctx, otpCodePrefix+phoneKey, newCode, otpTTL).Err())
	}

	require.NoError(t, svc.Consume(ctx, phone, code))

	phoneKey, err := phoneutil.HashLookup(phone)
	require.NoError(t, err)
	stored, err := fixture.Client.Get(ctx, otpCodePrefix+phoneKey).Result()
	require.NoError(t, err)
	assert.Equal(t, newCode, stored)
}

func TestOTPService_VerifyMaxAttemptsAndGenerateNumericCode(t *testing.T) {
	svc, fixture := newOTPServiceForTest(t)
	ctx := context.Background()
	phone := "13800138000"
	code, err := svc.Generate(ctx, phone)
	require.NoError(t, err)
	wrong := "000000"
	if wrong == code {
		wrong = "999999"
	}
	for i := 0; i < otpMaxAttempts-1; i++ {
		err = svc.Verify(ctx, phone, wrong)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrOTPInvalidCode)
	}
	err = svc.Verify(ctx, phone, wrong)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOTPMaxAttempts)

	phoneKey, err := phoneutil.HashLookup(phone)
	require.NoError(t, err)
	assert.False(t, fixture.Server.Exists(otpCodePrefix+phoneKey))

	numeric, err := generateNumericCode(otpLength)
	require.NoError(t, err)
	require.Len(t, numeric, otpLength)
	for _, ch := range numeric {
		assert.True(t, ch >= '0' && ch <= '9')
	}
}

func TestOTPService_VerifySuccessAfterPriorFailures(t *testing.T) {
	svc, fixture := newOTPServiceForTest(t)
	ctx := context.Background()
	phone := "13800138003"
	code, err := svc.Generate(ctx, phone)
	require.NoError(t, err)
	wrong := "000000"
	if wrong == code {
		wrong = "999999"
	}

	for i := 0; i < otpMaxAttempts-1; i++ {
		err = svc.Verify(ctx, phone, wrong)
		require.ErrorIs(t, err, ErrOTPInvalidCode)
	}
	require.NoError(t, svc.Verify(ctx, phone, code))

	phoneKey, err := phoneutil.HashLookup(phone)
	require.NoError(t, err)
	assert.False(t, fixture.Server.Exists(otpCodePrefix+phoneKey))
	assert.False(t, fixture.Server.Exists(otpAttemptsPrefix+phoneKey))
}

func TestOTPService_IssueCode(t *testing.T) {
	svc, fixture := newOTPServiceForTest(t)
	ctx := context.Background()
	phone := "13800138000"
	var gotPhone string
	var gotCode string

	require.NoError(t, svc.IssueCode(ctx, phone, stubPhoneSMSSender(func(_ context.Context, smsPhone, content string) error {
		gotPhone = smsPhone
		gotCode = content
		return nil
	})))

	assert.Equal(t, "+86"+phone, gotPhone)
	require.Len(t, gotCode, otpLength)
	assert.Equal(t, OTPCooldownSeconds(), svc.CooldownSeconds())

	phoneKey, err := phoneutil.HashLookup(phone)
	require.NoError(t, err)
	assert.True(t, fixture.Server.Exists(otpCodePrefix+phoneKey))
	assert.True(t, fixture.Server.Exists(otpCooldownPrefix+phoneKey))
}

func TestOTPService_IssueCode_SendFailureCleansCodeOnly(t *testing.T) {
	svc, fixture := newOTPServiceForTest(t)
	ctx := context.Background()
	phone := "13800138000"
	sendErr := errors.New("sms down")

	err := svc.IssueCode(ctx, phone, stubPhoneSMSSender(func(_ context.Context, _, _ string) error {
		return sendErr
	}))
	require.Error(t, err)
	assert.ErrorIs(t, err, sendErr)

	phoneKey, hashErr := phoneutil.HashLookup(phone)
	require.NoError(t, hashErr)
	assert.False(t, fixture.Server.Exists(otpCodePrefix+phoneKey))
	assert.True(t, fixture.Server.Exists(otpCooldownPrefix+phoneKey))
}

func TestOTPService_IssueCode_SendFailureCleanupSurvivesRequestCancellation(t *testing.T) {
	svc, fixture := newOTPServiceForTest(t)
	phone := "13800138000"
	sendErr := errors.New("sms down")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := svc.IssueCode(ctx, phone, stubPhoneSMSSender(func(_ context.Context, _, _ string) error {
		cancel()
		return sendErr
	}))
	require.Error(t, err)
	assert.ErrorIs(t, err, sendErr)

	phoneKey, hashErr := phoneutil.HashLookup(phone)
	require.NoError(t, hashErr)
	assert.False(t, fixture.Server.Exists(otpCodePrefix+phoneKey))
	assert.True(t, fixture.Server.Exists(otpCooldownPrefix+phoneKey))

	for i := 0; i < otpPhoneLimit; i++ {
		require.NoError(t, svc.CheckPhoneRateLimit(context.Background(), phone))
	}
	err = svc.CheckPhoneRateLimit(context.Background(), phone)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOTPPhoneRateLimited)
}

func TestOTPService_IssueCode_CooldownDoesNotConsumeHourlyQuota(t *testing.T) {
	svc, _ := newOTPServiceForTest(t)
	ctx := context.Background()
	phone := "13800138001"

	require.NoError(t, svc.IssueCode(ctx, phone, stubPhoneSMSSender(func(_ context.Context, _, _ string) error {
		return nil
	})))

	err := svc.IssueCode(ctx, phone, stubPhoneSMSSender(func(_ context.Context, _, _ string) error {
		return nil
	}))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOTPCooldown)

	for i := 0; i < otpPhoneLimit-1; i++ {
		require.NoError(t, svc.CheckPhoneRateLimit(ctx, phone))
	}
	err = svc.CheckPhoneRateLimit(ctx, phone)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOTPPhoneRateLimited)
}

func TestOTPService_IssueCode_SendFailureDoesNotConsumeHourlyQuota(t *testing.T) {
	svc, _ := newOTPServiceForTest(t)
	ctx := context.Background()
	phone := "13800138002"
	sendErr := errors.New("sms down")

	err := svc.IssueCode(ctx, phone, stubPhoneSMSSender(func(_ context.Context, _, _ string) error {
		return sendErr
	}))
	require.Error(t, err)
	assert.ErrorIs(t, err, sendErr)

	for i := 0; i < otpPhoneLimit; i++ {
		require.NoError(t, svc.CheckPhoneRateLimit(ctx, phone))
	}
	err = svc.CheckPhoneRateLimit(ctx, phone)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOTPPhoneRateLimited)
}
