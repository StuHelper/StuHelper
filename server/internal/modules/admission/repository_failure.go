package admission

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) IncrementFailureFromKickEventTx(
	ctx context.Context,
	tx pgx.Tx,
	session *AdmissionSession,
	policy *AdmissionPolicy,
	now time.Time,
) (int, error) {
	var blacklistExpiresAt *time.Time
	if policy.BlacklistDurationSeconds != nil {
		expires := now.Add(time.Duration(*policy.BlacklistDurationSeconds) * time.Second)
		blacklistExpiresAt = &expires
	}

	var count int
	err := tx.QueryRow(ctx, `
		INSERT INTO group_admission_failures (
			platform, guild_id, qq_id, failure_count, blacklisted_at, blacklist_expires_at, last_failure_at
		)
		VALUES ($1, $2, $3, 1, CASE WHEN 1 >= $4 THEN $5::timestamptz ELSE NULL END,
		        CASE WHEN 1 >= $4 THEN $6::timestamptz ELSE NULL END, $5::timestamptz)
		ON CONFLICT (platform, guild_id, qq_id) DO UPDATE
		SET failure_count = group_admission_failures.failure_count + 1,
		    blacklisted_at = CASE
		        WHEN group_admission_failures.failure_count + 1 >= $4
		        THEN COALESCE(group_admission_failures.blacklisted_at, $5::timestamptz)
		        ELSE group_admission_failures.blacklisted_at
		    END,
		    blacklist_expires_at = CASE
		        WHEN group_admission_failures.failure_count + 1 >= $4 THEN $6::timestamptz
		        ELSE group_admission_failures.blacklist_expires_at
		    END,
		    last_failure_at = $5::timestamptz,
		    updated_at = NOW()
		RETURNING failure_count
	`, session.Platform, session.GuildID, session.QQID, policy.FailedJoinLimit, now, blacklistExpiresAt).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("IncrementFailureFromKickEventTx: %w", err)
	}
	return count, nil
}

func (r *Repository) UpdateLastBotErrorTx(ctx context.Context, tx pgx.Tx, sessionID string, botError string) error {
	_, err := tx.Exec(ctx, `
		UPDATE group_admission_sessions
		SET last_bot_error = $2, updated_at = NOW()
		WHERE id = $1
	`, sessionID, botError)
	if err != nil {
		return fmt.Errorf("UpdateLastBotErrorTx: %w", err)
	}
	return nil
}
