package openplatform

import (
	"context"
	"fmt"
)

const (
	defaultOpenPlatformDisclosureAuditRetentionDays  = 365
	defaultOpenPlatformOperationalAuditRetentionDays = 1095
	defaultOpenPlatformAuditCleanupChunkSize         = 5000
)

const (
	openPlatformDisclosureAuditCondition = `(
		event_type LIKE 'open_platform.disclosure.%'
		OR event_type = 'open_platform.resource_access.checked'
	)`
	openPlatformOperationalAuditCondition = `(
		event_type NOT LIKE 'open_platform.disclosure.%'
		AND event_type <> 'open_platform.resource_access.checked'
	)`
)

type AuditRetentionPolicy struct {
	DisclosureDays  int
	OperationalDays int
}

type auditCleanupSpec struct {
	condition string
	days      int
}

func DefaultAuditRetentionPolicy() AuditRetentionPolicy {
	return AuditRetentionPolicy{
		DisclosureDays:  defaultOpenPlatformDisclosureAuditRetentionDays,
		OperationalDays: defaultOpenPlatformOperationalAuditRetentionDays,
	}
}

func (r *Repository) CleanupAuditEvents(ctx context.Context, policy AuditRetentionPolicy) (int64, error) {
	normalized, err := normalizeAuditRetentionPolicy(policy)
	if err != nil {
		return 0, err
	}

	var total int64
	for _, spec := range auditCleanupSpecs(normalized) {
		deleted, cleanupErr := r.cleanupAuditEventGroup(ctx, spec)
		if cleanupErr != nil {
			return 0, cleanupErr
		}
		total += deleted
	}
	return total, nil
}

func auditCleanupSpecs(policy AuditRetentionPolicy) []auditCleanupSpec {
	return []auditCleanupSpec{
		{condition: openPlatformDisclosureAuditCondition, days: policy.DisclosureDays},
		{condition: openPlatformOperationalAuditCondition, days: policy.OperationalDays},
	}
}

func normalizeAuditRetentionPolicy(policy AuditRetentionPolicy) (AuditRetentionPolicy, error) {
	defaults := DefaultAuditRetentionPolicy()
	if policy.DisclosureDays == 0 {
		policy.DisclosureDays = defaults.DisclosureDays
	}
	if policy.OperationalDays == 0 {
		policy.OperationalDays = defaults.OperationalDays
	}
	if policy.DisclosureDays < defaults.DisclosureDays ||
		policy.OperationalDays < defaults.OperationalDays {
		return AuditRetentionPolicy{}, fmt.Errorf("open platform audit retention cannot be lower than baseline")
	}
	return policy, nil
}

func (r *Repository) cleanupAuditEventGroup(ctx context.Context, spec auditCleanupSpec) (int64, error) {
	query := fmt.Sprintf(`
		DELETE FROM open_platform_audit_events
		WHERE id IN (
			SELECT id
			FROM open_platform_audit_events
			WHERE %s
			  AND created_at < NOW() - make_interval(days => $1)
			ORDER BY created_at ASC, id ASC
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
	`, spec.condition)
	deleted, err := r.cleanupAuditEventsInChunks(ctx, query, spec.days)
	if err != nil {
		return 0, fmt.Errorf("cleanup open platform audit events: %w", err)
	}
	return deleted, nil
}

func (r *Repository) cleanupAuditEventsInChunks(ctx context.Context, query string, args ...any) (int64, error) {
	ctx = withDBTable(ctx, "open_platform_audit_events")
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}

		queryArgs := make([]any, 0, len(args)+1)
		queryArgs = append(queryArgs, args...)
		queryArgs = append(queryArgs, defaultOpenPlatformAuditCleanupChunkSize)

		result, err := r.db.Exec(ctx, query, queryArgs...)
		if err != nil {
			return total, fmt.Errorf("cleanup open platform audit events chunk: %w", err)
		}

		deleted := result.RowsAffected()
		total += deleted
		if deleted < defaultOpenPlatformAuditCleanupChunkSize {
			return total, nil
		}
	}
}
