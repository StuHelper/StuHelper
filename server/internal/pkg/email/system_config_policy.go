package email

import (
	"context"
	"fmt"
)

type DeliveryPolicyValueStore interface {
	GetEmailDeliveryPolicyValue(ctx context.Context) (value string, found bool, err error)
}

type SystemConfigPolicyProvider struct {
	store         DeliveryPolicyValueStore
	defaultPolicy DeliveryPolicy
	available     map[string]OTPSender
}

func NewSystemConfigPolicyProvider(store DeliveryPolicyValueStore, defaultPolicy DeliveryPolicy, available map[string]OTPSender) *SystemConfigPolicyProvider {
	if store == nil {
		return nil
	}
	return &SystemConfigPolicyProvider{
		store:         store,
		defaultPolicy: defaultPolicy,
		available:     available,
	}
}

func (p *SystemConfigPolicyProvider) GetEmailDeliveryPolicy(ctx context.Context) (DeliveryPolicy, error) {
	if p == nil || p.store == nil {
		return p.defaultPolicy, nil
	}
	raw, found, err := p.store.GetEmailDeliveryPolicyValue(ctx)
	if err != nil {
		return p.defaultPolicy, fmt.Errorf("load email delivery policy: %w", err)
	}
	if !found {
		return p.defaultPolicy, nil
	}
	policy, err := ParseDeliveryPolicy(raw)
	if err != nil {
		return p.defaultPolicy, err
	}
	return NormalizeDeliveryPolicy(policy, p.available), nil
}
