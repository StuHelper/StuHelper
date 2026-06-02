package admission

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

func (r *Repository) ListPolicies(ctx context.Context) ([]AdmissionPolicy, error) {
	ctx = withDBTable(ctx, "group_admission_policies")
	rows, err := r.db.Query(ctx, `
		SELECT id, platform, guild_id, school_id, auto_approve_join, initial_mute_duration_seconds,
		       link_wait_seconds, submission_wait_seconds, manual_review_timeout_seconds,
		       reminder_interval_seconds, failed_join_limit, blacklist_duration_seconds,
		       freshman_channel_enabled, freshman_channel_closes_at, freshman_default_expires_at,
		       forward_raw_material_to_qq, management_guild_ids, max_material_bytes, max_extension_days
		FROM group_admission_policies
		ORDER BY platform ASC, guild_id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ListPolicies: %w", err)
	}
	defer rows.Close()
	return scanAdmissionPolicies(rows)
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
		SET auto_approve_join = $2, initial_mute_duration_seconds = $3,
		    link_wait_seconds = $4, submission_wait_seconds = $5,
		    manual_review_timeout_seconds = $6, reminder_interval_seconds = $7,
		    failed_join_limit = $8, blacklist_duration_seconds = $9,
		    freshman_channel_enabled = $10, freshman_channel_closes_at = $11,
		    freshman_default_expires_at = $12, forward_raw_material_to_qq = $13,
		    management_guild_ids = $14, max_material_bytes = $15,
		    max_extension_days = $16, updated_at = NOW()
		WHERE id = $1
		RETURNING id, platform, guild_id, school_id, auto_approve_join,
		          initial_mute_duration_seconds, link_wait_seconds, submission_wait_seconds,
		          manual_review_timeout_seconds, reminder_interval_seconds, failed_join_limit,
		          blacklist_duration_seconds, freshman_channel_enabled, freshman_channel_closes_at,
		          freshman_default_expires_at, forward_raw_material_to_qq, management_guild_ids,
		          max_material_bytes, max_extension_days`
}

func updateAdmissionPolicyArgs(policy AdmissionPolicy) []any {
	return []any{
		policy.ID, policy.AutoApproveJoin, policy.InitialMuteDurationSeconds,
		policy.LinkWaitSeconds, policy.SubmissionWaitSeconds,
		policy.ManualReviewTimeoutSeconds, policy.ReminderIntervalSeconds,
		policy.FailedJoinLimit, policy.BlacklistDurationSeconds,
		policy.FreshmanChannelEnabled, policy.FreshmanChannelClosesAt,
		policy.FreshmanDefaultExpiresAt, policy.ForwardRawMaterialToQQ,
		policy.ManagementGuildIDs, policy.MaxMaterialBytes, policy.MaxExtensionDays,
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
