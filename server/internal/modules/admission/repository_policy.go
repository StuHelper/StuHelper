package admission

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetPolicy(ctx context.Context, platform, guildID string) (*AdmissionPolicy, error) {
	ctx = withDBTable(ctx, "group_admission_policies")
	policy, err := scanAdmissionPolicy(r.db.QueryRow(ctx, `
		SELECT id, platform, guild_id, school_id, auto_approve_join, initial_mute_duration_seconds,
		       link_wait_seconds, submission_wait_seconds, manual_review_timeout_seconds,
		       reminder_interval_seconds, failed_join_limit, blacklist_duration_seconds,
		       freshman_channel_enabled, freshman_channel_closes_at, freshman_default_expires_at,
		       forward_raw_material_to_qq, management_guild_ids, max_material_bytes, max_extension_days
		FROM group_admission_policies
		WHERE platform = $1 AND guild_id = $2
	`, platform, guildID))
	if err != nil {
		return nil, fmt.Errorf("GetPolicy: %w", err)
	}
	return policy, nil
}

func scanAdmissionPolicy(row pgx.Row) (*AdmissionPolicy, error) {
	var policy AdmissionPolicy
	err := row.Scan(
		&policy.ID, &policy.Platform, &policy.GuildID, &policy.SchoolID, &policy.AutoApproveJoin,
		&policy.InitialMuteDurationSeconds, &policy.LinkWaitSeconds, &policy.SubmissionWaitSeconds,
		&policy.ManualReviewTimeoutSeconds, &policy.ReminderIntervalSeconds, &policy.FailedJoinLimit,
		&policy.BlacklistDurationSeconds, &policy.FreshmanChannelEnabled, &policy.FreshmanChannelClosesAt,
		&policy.FreshmanDefaultExpiresAt, &policy.ForwardRawMaterialToQQ, &policy.ManagementGuildIDs,
		&policy.MaxMaterialBytes, &policy.MaxExtensionDays,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &policy, nil
}
