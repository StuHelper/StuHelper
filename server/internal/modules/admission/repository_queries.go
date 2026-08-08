package admission

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
	"github.com/StuHelper/StuHelper/server/internal/pkg/id"
)

func (r *Repository) ListPolicies(ctx context.Context) ([]AdmissionPolicy, error) {
	ctx = withDBTable(ctx, "group_admission_policies")
	rows, err := r.db.Query(ctx, `
		SELECT id, platform, guild_id, guard_enabled, school_id, auto_approve_join,
		       join_handling_strategy, auto_approve_verified_join, auto_approve_unverified_join,
		       unverified_join_reject_reason, initial_mute_duration_seconds,
		       link_wait_seconds, submission_wait_seconds, manual_review_timeout_seconds,
		       reminder_interval_seconds, failed_join_limit, blacklist_duration_seconds,
		       freshman_channel_enabled, freshman_channel_closes_at, freshman_default_expires_at,
		       forward_raw_material_to_qq, management_guild_ids, max_material_bytes, max_extension_days,
		       allow_temporary_freshman
		FROM group_admission_policies
		ORDER BY platform ASC, guild_id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ListPolicies: %w", err)
	}
	defer rows.Close()
	return scanAdmissionPolicies(rows)
}

func (r *Repository) ListPolicyTargets(ctx context.Context) ([]AdmissionPolicyTarget, error) {
	ctx = withDBTable(ctx, "group_admission_policies")
	rows, err := r.db.Query(ctx, `
		SELECT id, platform, guild_id, guard_enabled, join_handling_strategy, link_wait_seconds,
		       management_guild_ids
		FROM group_admission_policies
		ORDER BY platform ASC, guild_id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ListPolicyTargets: %w", err)
	}
	defer rows.Close()

	items := make([]AdmissionPolicyTarget, 0)
	for rows.Next() {
		var item AdmissionPolicyTarget
		if err := rows.Scan(
			&item.PolicyID,
			&item.Platform,
			&item.GuildID,
			&item.GuardEnabled,
			&item.JoinHandlingStrategy,
			&item.LinkWaitSeconds,
			&item.ManagementGuildIDs,
		); err != nil {
			return nil, fmt.Errorf("scan admission policy target: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admission policy targets: %w", err)
	}
	return items, nil
}

func (r *Repository) CreatePolicyFromSource(ctx context.Context, input AdmissionPolicyCreateRequest) (*AdmissionPolicy, error) {
	ctx = withDBTable(ctx, "group_admission_policies")
	policyID, err := id.New()
	if err != nil {
		return nil, fmt.Errorf("CreatePolicyFromSource id: %w", err)
	}

	created, err := scanAdmissionPolicy(r.db.QueryRow(ctx, `
		INSERT INTO group_admission_policies (
			id, platform, guild_id, guard_enabled, school_id, auto_approve_join,
			join_handling_strategy, auto_approve_verified_join, auto_approve_unverified_join,
			unverified_join_reject_reason, initial_mute_duration_seconds,
			link_wait_seconds, submission_wait_seconds, manual_review_timeout_seconds,
			reminder_interval_seconds, failed_join_limit, blacklist_duration_seconds,
			freshman_channel_enabled, freshman_channel_closes_at, freshman_default_expires_at,
			forward_raw_material_to_qq, management_guild_ids, max_material_bytes, max_extension_days,
			allow_temporary_freshman
		)
		SELECT $1, $2, $3, TRUE, school_id, auto_approve_join,
		       join_handling_strategy, auto_approve_verified_join, auto_approve_unverified_join,
		       unverified_join_reject_reason, initial_mute_duration_seconds,
		       link_wait_seconds, submission_wait_seconds, manual_review_timeout_seconds,
		       reminder_interval_seconds, failed_join_limit, blacklist_duration_seconds,
		       freshman_channel_enabled, freshman_channel_closes_at, freshman_default_expires_at,
		       forward_raw_material_to_qq, '{}'::text[], max_material_bytes, max_extension_days,
		       allow_temporary_freshman
		FROM group_admission_policies
		WHERE id = $4
		ON CONFLICT (platform, guild_id) DO NOTHING
		RETURNING id, platform, guild_id, guard_enabled, school_id, auto_approve_join,
		          join_handling_strategy, auto_approve_verified_join, auto_approve_unverified_join,
		          unverified_join_reject_reason,
		          initial_mute_duration_seconds, link_wait_seconds, submission_wait_seconds,
		          manual_review_timeout_seconds, reminder_interval_seconds, failed_join_limit,
		          blacklist_duration_seconds, freshman_channel_enabled, freshman_channel_closes_at,
		          freshman_default_expires_at, forward_raw_material_to_qq, management_guild_ids,
		          max_material_bytes, max_extension_days, allow_temporary_freshman
	`, policyID, input.Platform, input.GuildID, input.SourcePolicyID))
	if err != nil {
		return nil, fmt.Errorf("CreatePolicyFromSource: %w", err)
	}
	if created != nil {
		return created, nil
	}

	sourceExists, err := r.policyIDExists(ctx, input.SourcePolicyID)
	if err != nil {
		return nil, err
	}
	if !sourceExists {
		return nil, ErrAdmissionPolicyNotFound
	}
	targetExists, err := r.policyTargetExists(ctx, input.Platform, input.GuildID)
	if err != nil {
		return nil, err
	}
	if targetExists {
		return nil, ErrAdmissionPolicyAlreadyExists
	}
	return nil, ErrAdmissionPolicyNotFound
}

func (r *Repository) UpdatePolicy(ctx context.Context, policy AdmissionPolicy) (*AdmissionPolicy, error) {
	ctx = withDBTable(ctx, "group_admission_policies")
	updated, err := scanAdmissionPolicy(r.db.QueryRow(ctx, updateAdmissionPolicySQL(), updateAdmissionPolicyArgs(policy)...))
	if err != nil {
		return nil, fmt.Errorf("UpdatePolicy: %w", err)
	}
	if updated == nil {
		return nil, ErrAdmissionPolicyNotFound
	}
	return updated, nil
}

func (r *Repository) policyIDExists(ctx context.Context, policyID string) (bool, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM group_admission_policies WHERE id = $1
		)
	`, policyID).Scan(&exists); err != nil {
		return false, fmt.Errorf("policyIDExists: %w", err)
	}
	return exists, nil
}

func (r *Repository) policyTargetExists(ctx context.Context, platform, guildID string) (bool, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM group_admission_policies WHERE platform = $1 AND guild_id = $2
		)
	`, platform, guildID).Scan(&exists); err != nil {
		return false, fmt.Errorf("policyTargetExists: %w", err)
	}
	return exists, nil
}

func (r *Repository) ListSessions(
	ctx context.Context,
	filter AdmissionSessionListFilter,
) ([]AdmissionSession, int, error) {
	ctx = withDBTable(ctx, "group_admission_sessions")
	rows, err := r.db.Query(ctx, `
		SELECT `+admissionSessionColumns+`, COUNT(*) OVER() AS total
		FROM group_admission_sessions
		WHERE ($1::text = '' OR status = $1)
		  AND ($2::text = '' OR platform = $2)
		  AND ($3::text = '' OR bot_self_id = $3)
		  AND ($4::text = '' OR guild_id = $4)
		  AND ($5::text = '' OR qq_id = $5)
		ORDER BY created_at DESC, id ASC
		LIMIT $6 OFFSET $7
	`, string(filter.Status), filter.Platform, filter.BotSelfID, filter.GuildID, filter.QQID, filter.PageSize, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("ListSessions: %w", err)
	}
	defer rows.Close()
	return scanAdmissionSessionList(rows)
}

func (r *Repository) GetFreshmanApplicationByID(ctx context.Context, applicationID string) (*FreshmanApplication, error) {
	ctx = withDBTable(ctx, "freshman_verification_applications")
	app, err := scanFreshmanApplication(r.db.QueryRow(ctx, freshmanApplicationSelectSQL()+" WHERE id = $1", applicationID))
	if err != nil {
		return nil, mapFreshmanApplicationScanError("GetFreshmanApplicationByID", err)
	}
	return app, nil
}

func (r *Repository) ListFreshmanApplications(
	ctx context.Context,
	filter FreshmanApplicationListFilter,
) ([]FreshmanApplication, int, error) {
	ctx = withDBTable(ctx, "freshman_verification_applications")
	rows, err := r.db.Query(ctx, freshmanApplicationListSQL(), string(filter.Status), filter.PageSize, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("ListFreshmanApplications: %w", err)
	}
	defer rows.Close()
	return scanFreshmanApplicationList(rows)
}

func (r *Repository) ListAdminFreshmanApplications(
	ctx context.Context,
	filter FreshmanApplicationListFilter,
) ([]adminFreshmanApplicationRow, int, error) {
	ctx = withDBTable(ctx, "freshman_verification_applications")
	rows, err := r.db.Query(ctx, adminFreshmanApplicationListSQL(), string(filter.Status), filter.PageSize, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("ListAdminFreshmanApplications: %w", err)
	}
	defer rows.Close()
	return scanAdminFreshmanApplicationList(rows)
}

func (r *Repository) MarkFreshmanApplicationForwarded(ctx context.Context, applicationID string, now time.Time) error {
	ctx = withDBTable(ctx, "freshman_verification_applications")
	tag, err := r.db.Exec(ctx, `
		UPDATE freshman_verification_applications
		SET forwarded_at = $2, updated_at = NOW()
		WHERE id = $1
	`, applicationID, now)
	if err != nil {
		return fmt.Errorf("MarkFreshmanApplicationForwarded: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAdmissionApplicationNotFound
	}
	return nil
}

func (r *Repository) InsertAuditEventTx(ctx context.Context, tx pgx.Tx, event audit.Event) error {
	eventID, err := id.New()
	if err != nil {
		return err
	}
	details, err := json.Marshal(event.Details)
	if err != nil {
		return fmt.Errorf("marshal admission audit details: %w", err)
	}
	return insertAuditEventTx(ctx, tx, eventID, event, details)
}

func updateAdmissionPolicySQL() string {
	return `
		UPDATE group_admission_policies
		SET guard_enabled = $2, join_handling_strategy = $3, auto_approve_join = $4,
		    auto_approve_verified_join = $5, auto_approve_unverified_join = $6,
		    unverified_join_reject_reason = $7, initial_mute_duration_seconds = $8,
		    link_wait_seconds = $9, submission_wait_seconds = $10,
		    manual_review_timeout_seconds = $11, reminder_interval_seconds = $12,
		    failed_join_limit = $13, blacklist_duration_seconds = $14,
		    freshman_channel_enabled = $15, freshman_channel_closes_at = $16,
		    freshman_default_expires_at = $17, forward_raw_material_to_qq = $18,
		    management_guild_ids = $19, max_material_bytes = $20,
		    max_extension_days = $21, allow_temporary_freshman = $22, updated_at = NOW()
		WHERE id = $1
		RETURNING id, platform, guild_id, guard_enabled, school_id, auto_approve_join,
		          join_handling_strategy, auto_approve_verified_join, auto_approve_unverified_join,
		          unverified_join_reject_reason,
		          initial_mute_duration_seconds, link_wait_seconds, submission_wait_seconds,
		          manual_review_timeout_seconds, reminder_interval_seconds, failed_join_limit,
		          blacklist_duration_seconds, freshman_channel_enabled, freshman_channel_closes_at,
		          freshman_default_expires_at, forward_raw_material_to_qq, management_guild_ids,
		          max_material_bytes, max_extension_days, allow_temporary_freshman`
}

func updateAdmissionPolicyArgs(policy AdmissionPolicy) []any {
	return []any{
		policy.ID, policy.GuardEnabled, policy.JoinHandlingStrategy, policy.AutoApproveJoin,
		policy.AutoApproveVerifiedJoin, policy.AutoApproveUnverifiedJoin,
		policy.UnverifiedJoinRejectReason, policy.InitialMuteDurationSeconds,
		policy.LinkWaitSeconds, policy.SubmissionWaitSeconds, policy.ManualReviewTimeoutSeconds,
		policy.ReminderIntervalSeconds, policy.FailedJoinLimit, policy.BlacklistDurationSeconds,
		policy.FreshmanChannelEnabled, policy.FreshmanChannelClosesAt, policy.FreshmanDefaultExpiresAt,
		policy.ForwardRawMaterialToQQ, policy.ManagementGuildIDs, policy.MaxMaterialBytes,
		policy.MaxExtensionDays, policy.AllowTemporaryFreshman,
	}
}

func insertAuditEventTx(ctx context.Context, tx pgx.Tx, eventID string, event audit.Event, details []byte) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			id, category, event_type, actor_type, actor_user_id, action,
			resource_type, resource_id, result, reason, request_id, trace_id, details
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, eventID, event.Category, event.Type, event.ActorType, event.UserID, event.Action,
		event.ResourceType, event.ResourceID, event.Result, event.Reason, event.RequestID, event.TraceID, details)
	if err != nil {
		return fmt.Errorf("InsertAuditEventTx: %w", err)
	}
	return nil
}
