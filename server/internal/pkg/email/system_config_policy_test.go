package email

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDeliveryPolicyValueStore struct {
	value string
	found bool
	err   error
}

func (s fakeDeliveryPolicyValueStore) GetEmailDeliveryPolicyValue(context.Context) (string, bool, error) {
	return s.value, s.found, s.err
}

func TestSystemConfigPolicyProviderReturnsDefaultWhenValueMissing(t *testing.T) {
	available := map[string]OTPSender{
		ProviderTencentSES: &stubOTPSender{},
	}
	defaultPolicy := DeliveryPolicy{
		Mode:        "priority",
		MaxAttempts: 1,
		Providers: []DeliveryPolicyEntry{
			{Name: ProviderTencentSES, Enabled: true, Priority: 10, Weight: 100},
		},
	}
	provider := NewSystemConfigPolicyProvider(fakeDeliveryPolicyValueStore{}, defaultPolicy, available)

	policy, err := provider.GetEmailDeliveryPolicy(context.Background())

	require.NoError(t, err)
	assert.Equal(t, defaultPolicy, policy)
}

func TestSystemConfigPolicyProviderUsesStoredPolicy(t *testing.T) {
	available := map[string]OTPSender{
		ProviderTencentSES: &stubOTPSender{},
		ProviderResend:     &stubOTPSender{},
	}
	defaultPolicy := DeliveryPolicy{
		Mode:        "priority",
		MaxAttempts: 1,
		Providers: []DeliveryPolicyEntry{
			{Name: ProviderTencentSES, Enabled: true, Priority: 10, Weight: 100},
		},
	}
	provider := NewSystemConfigPolicyProvider(fakeDeliveryPolicyValueStore{
		found: true,
		value: `{"mode":"priority","maxAttempts":1,"providers":[{"name":"resend","enabled":true,"priority":1,"weight":100}]}`,
	}, defaultPolicy, available)

	policy, err := provider.GetEmailDeliveryPolicy(context.Background())

	require.NoError(t, err)
	require.Len(t, policy.Providers, 2)
	assert.Equal(t, ProviderResend, policy.Providers[0].Name)
	assert.Equal(t, 1, policy.Providers[0].Priority)
}

func TestSystemConfigPolicyProviderWrapsStoreError(t *testing.T) {
	storeErr := errors.New("database unavailable")
	provider := NewSystemConfigPolicyProvider(fakeDeliveryPolicyValueStore{err: storeErr}, DeliveryPolicy{}, nil)

	policy, err := provider.GetEmailDeliveryPolicy(context.Background())

	require.Error(t, err)
	assert.ErrorIs(t, err, storeErr)
	assert.Contains(t, err.Error(), "load email delivery policy")
	assert.Empty(t, policy.Providers)
}
