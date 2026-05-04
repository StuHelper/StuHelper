package admission

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) ListPendingActionSessions(
	ctx context.Context,
	filter AdmissionPendingActionFilter,
	now time.Time,
) ([]AdmissionSession, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+admissionSessionColumns+`
		FROM group_admission_sessions
		WHERE ($1::text = '' OR platform = $1)
		  AND ($2::text = '' OR bot_self_id = $2)
		  AND (
		    (status = $3 AND (next_reminder_at IS NULL OR next_reminder_at <= $5))
		    OR (status = $3 AND link_wait_deadline_at <= $5)
		    OR (status = $4 AND submission_wait_deadline_at <= $5)
		    OR (status = $6 AND manual_review_deadline_at <= $5)
		    OR (status = $7 AND cancelled_at IS NULL)
		  )
		ORDER BY updated_at ASC, id ASC
		LIMIT $8
		`, filter.Platform, filter.BotSelfID, StatusJoinedMuted, StatusLinked, now,
		StatusMaterialSubmitted, StatusVerified, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("ListPendingActionSessions: %w", err)
	}
	defer rows.Close()
	return scanAdmissionSessions(rows)
}

func (r *Repository) ListPendingFreshmanForwards(ctx context.Context) ([]freshmanForwardRecord, error) {
	rows, err := r.db.Query(ctx, pendingFreshmanForwardSQL())
	if err != nil {
		return nil, fmt.Errorf("ListPendingFreshmanForwards: %w", err)
	}
	defer rows.Close()
	return scanFreshmanForwardRecords(rows)
}

func (r *Repository) GetActiveAdmissionFailure(ctx context.Context, qqID string) (*AdmissionFailure, error) {
	failure, err := scanAdmissionFailure(r.db.QueryRow(ctx, `
		SELECT platform, guild_id, qq_id, failure_count, blacklisted_at, blacklist_expires_at, released_at
		FROM group_admission_failures
		WHERE qq_id = $1
		  AND blacklisted_at IS NOT NULL
		  AND released_at IS NULL
		  AND (blacklist_expires_at IS NULL OR blacklist_expires_at > NOW())
		ORDER BY blacklisted_at DESC
		LIMIT 1
	`, qqID))
	if err != nil {
		return nil, err
	}
	return failure, nil
}

func (r *Repository) GetAdmissionFailure(
	ctx context.Context,
	platform string,
	guildID string,
	qqID string,
) (*AdmissionFailure, error) {
	return scanAdmissionFailure(r.db.QueryRow(ctx, `
		SELECT platform, guild_id, qq_id, failure_count, blacklisted_at, blacklist_expires_at, released_at
		FROM group_admission_failures
		WHERE platform = $1 AND guild_id = $2 AND qq_id = $3 AND released_at IS NULL
	`, platform, guildID, qqID))
}

func (r *Repository) ReleaseAdmissionBlacklist(ctx context.Context, qqID string, now time.Time) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE group_admission_failures
		SET released_at = $2, updated_at = NOW()
		WHERE qq_id = $1
		  AND blacklisted_at IS NOT NULL
		  AND released_at IS NULL
		  AND (blacklist_expires_at IS NULL OR blacklist_expires_at > $2)
	`, qqID, now)
	if err != nil {
		return fmt.Errorf("ReleaseAdmissionBlacklist: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAdmissionBlacklistNotFound
	}
	return nil
}

func (r *Repository) MarkReminderSentTx(
	ctx context.Context,
	tx pgx.Tx,
	session *AdmissionSession,
	policy *AdmissionPolicy,
	now time.Time,
) error {
	_, err := tx.Exec(ctx, `
		UPDATE group_admission_sessions
		SET last_reminded_at = $2, next_reminder_at = $3, updated_at = NOW()
		WHERE id = $1
	`, session.ID, now, now.Add(time.Duration(policy.ReminderIntervalSeconds)*time.Second))
	return err
}

func (r *Repository) MarkBotReleaseCompletedTx(ctx context.Context, tx pgx.Tx, sessionID string, now time.Time) error {
	_, err := tx.Exec(ctx, `
		UPDATE group_admission_sessions
		SET cancelled_at = $2, updated_at = NOW()
		WHERE id = $1 AND status = $3
	`, sessionID, now, StatusVerified)
	return err
}

func (r *Repository) MarkBotKickCompletedTx(ctx context.Context, tx pgx.Tx, sessionID string, now time.Time) error {
	_, err := tx.Exec(ctx, `
		UPDATE group_admission_sessions
		SET status = $2, cancelled_at = $3, updated_at = NOW()
		WHERE id = $1 AND status IN ($4, $5, $6)
	`, sessionID, StatusExpiredKicked, now, StatusJoinedMuted, StatusLinked, StatusMaterialSubmitted)
	return err
}

func (r *Repository) ManagementGuildAllowed(ctx context.Context, guildID string) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM group_admission_policies WHERE $1 = ANY(management_guild_ids)
		)
	`, guildID).Scan(&allowed)
	return allowed, err
}
