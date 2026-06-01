package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

const SystemConfigDeliveryPolicyKey = "email.delivery_policy"

type SystemConfigPolicyProvider struct {
	db            *db.DB
	defaultPolicy DeliveryPolicy
	available     map[string]OTPSender
}

func NewSystemConfigPolicyProvider(database *db.DB, defaultPolicy DeliveryPolicy, available map[string]OTPSender) *SystemConfigPolicyProvider {
	if database == nil {
		return nil
	}
	return &SystemConfigPolicyProvider{
		db:            database,
		defaultPolicy: defaultPolicy,
		available:     available,
	}
}

func (p *SystemConfigPolicyProvider) GetEmailDeliveryPolicy(ctx context.Context) (DeliveryPolicy, error) {
	if p == nil || p.db == nil {
		return p.defaultPolicy, nil
	}
	ctx = db.WithTableHint(ctx, "system_configs")
	var raw string
	err := p.db.QueryRow(ctx, `
		SELECT value
		FROM system_configs
		WHERE key = $1
	`, SystemConfigDeliveryPolicyKey).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return p.defaultPolicy, nil
		}
		return p.defaultPolicy, fmt.Errorf("load email delivery policy: %w", err)
	}
	policy, err := ParseDeliveryPolicy(raw)
	if err != nil {
		return p.defaultPolicy, err
	}
	return NormalizeDeliveryPolicy(policy, p.available), nil
}
