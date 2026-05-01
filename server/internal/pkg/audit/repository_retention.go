package audit

import (
	"context"
	"fmt"
)

const (
	defaultLoginSuccessRetentionDays = 90
	defaultIAMSecurityRetentionDays  = 365
	defaultIAMHighValueRetentionDays = 1095
)

const (
	loginSuccessIAMCondition  = "event_type = 'iam.auth.login'"
	securityAuditIAMCondition = `(
		(event_type LIKE 'iam.auth.%' AND event_type <> 'iam.auth.login')
		OR event_type LIKE 'iam.mfa.%'
		OR event_type LIKE 'iam.token.%'
	)`
	casdoorAdminAPIIAMCondition = `(
		event_type = 'iam.casdoor.admin_api'
		OR event_type LIKE 'iam.casdoor_admin_api.%'
	)`
	highValueAdminIAMCondition = `(
		event_type LIKE 'iam.role.%'
		OR event_type LIKE 'iam.service_account.%'
		OR event_type LIKE 'iam.casdoor_app.%'
	)`
)

type IAMRetentionPolicy struct {
	LoginSuccessDays int
	SecurityDays     int
	HighValueDays    int
}

type iamCleanupSpec struct {
	category  string
	condition string
	days      int
}

func DefaultIAMRetentionPolicy() IAMRetentionPolicy {
	return IAMRetentionPolicy{
		LoginSuccessDays: defaultLoginSuccessRetentionDays,
		SecurityDays:     defaultIAMSecurityRetentionDays,
		HighValueDays:    defaultIAMHighValueRetentionDays,
	}
}

func (r *Repository) CleanupIAMEvents(ctx context.Context, policy IAMRetentionPolicy) (int64, error) {
	normalized, err := normalizeIAMRetentionPolicy(policy)
	if err != nil {
		return 0, err
	}

	var total int64
	for _, spec := range iamCleanupSpecs(normalized) {
		deleted, cleanupErr := r.cleanupIAMEventGroup(ctx, spec)
		if cleanupErr != nil {
			return 0, cleanupErr
		}
		total += deleted
	}
	return total, nil
}

func iamCleanupSpecs(policy IAMRetentionPolicy) []iamCleanupSpec {
	return []iamCleanupSpec{
		{category: defaultAuditCategory, condition: loginSuccessIAMCondition, days: policy.LoginSuccessDays},
		{category: defaultAuditCategory, condition: securityAuditIAMCondition, days: policy.SecurityDays},
		{category: adminOperationCategory, condition: casdoorAdminAPIIAMCondition, days: policy.SecurityDays},
		{category: adminOperationCategory, condition: highValueAdminIAMCondition, days: policy.HighValueDays},
	}
}

func normalizeIAMRetentionPolicy(policy IAMRetentionPolicy) (IAMRetentionPolicy, error) {
	defaults := DefaultIAMRetentionPolicy()
	if policy.LoginSuccessDays == 0 {
		policy.LoginSuccessDays = defaults.LoginSuccessDays
	}
	if policy.SecurityDays == 0 {
		policy.SecurityDays = defaults.SecurityDays
	}
	if policy.HighValueDays == 0 {
		policy.HighValueDays = defaults.HighValueDays
	}
	if policy.LoginSuccessDays < defaults.LoginSuccessDays ||
		policy.SecurityDays < defaults.SecurityDays ||
		policy.HighValueDays < defaults.HighValueDays {
		return IAMRetentionPolicy{}, fmt.Errorf("iam audit retention cannot be lower than baseline")
	}
	return policy, nil
}

func (r *Repository) cleanupIAMEventGroup(ctx context.Context, spec iamCleanupSpec) (int64, error) {
	query := fmt.Sprintf(`
		DELETE FROM audit_events
		WHERE category = $1
		  AND %s
		  AND created_at < NOW() - make_interval(days => $2)
	`, spec.condition)
	result, err := r.db.Exec(ctx, query, spec.category, spec.days)
	if err != nil {
		return 0, fmt.Errorf("cleanup iam audit events: %w", err)
	}
	return result.RowsAffected(), nil
}
