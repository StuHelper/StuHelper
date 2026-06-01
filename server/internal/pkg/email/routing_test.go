package email

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubOTPSender struct {
	err   error
	calls int
}

func (s *stubOTPSender) SendOTP(context.Context, string, string, string, string, string, int) error {
	s.calls++
	return s.err
}

func TestFailoverOTPSenderUsesPriorityFallback(t *testing.T) {
	tencent := &stubOTPSender{err: errors.New("tencent down")}
	resend := &stubOTPSender{}
	sender, err := NewFailoverOTPSender(
		[]OTPProvider{
			{Name: ProviderTencentSES, Sender: tencent},
			{Name: ProviderResend, Sender: resend},
		},
		DeliveryPolicy{
			Mode:        "priority",
			MaxAttempts: 2,
			Providers: []DeliveryPolicyEntry{
				{Name: ProviderTencentSES, Enabled: true, Priority: 10, Weight: 100},
				{Name: ProviderResend, Enabled: true, Priority: 20, Weight: 100},
			},
		},
		nil,
	)
	require.NoError(t, err)

	err = sender.SendOTP(context.Background(), "student@buaa.edu.cn", "学生认证验证码", "123456", "", "", 0)

	require.NoError(t, err)
	assert.Equal(t, 1, tencent.calls)
	assert.Equal(t, 1, resend.calls)
}

func TestResolveProviderOrderSupportsWeightedSamePriority(t *testing.T) {
	available := map[string]OTPSender{
		ProviderTencentSES: &stubOTPSender{},
		ProviderResend:     &stubOTPSender{},
	}
	policy := DeliveryPolicy{
		Mode:        "weighted",
		MaxAttempts: 2,
		Providers: []DeliveryPolicyEntry{
			{Name: ProviderTencentSES, Enabled: true, Priority: 10, Weight: 1},
			{Name: ProviderResend, Enabled: true, Priority: 10, Weight: 2},
		},
	}

	first := ResolveProviderOrder(policy, available, func(int, int) int { return 0 })
	second := ResolveProviderOrder(policy, available, func(int, int) int { return 1 })

	assert.Equal(t, []string{ProviderResend, ProviderTencentSES}, first)
	assert.Equal(t, []string{ProviderResend, ProviderTencentSES}, second)
}

func TestParseDeliveryPolicyRejectsInvalidJSON(t *testing.T) {
	_, err := ParseDeliveryPolicy("{")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse email delivery policy")
}
