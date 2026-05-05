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
	_ *AdmissionPolicy,
	now time.Time,
) (int, error) {
	var count int
	err := tx.QueryRow(ctx, `
		INSERT INTO group_admission_failures (
			platform, guild_id, qq_id, failure_count, last_failure_at
		)
		VALUES ($1, $2, $3, 1, $4::timestamptz)
		ON CONFLICT (platform, guild_id, qq_id) DO UPDATE
		SET failure_count = group_admission_failures.failure_count + 1,
		    last_failure_at = $4::timestamptz,
		    updated_at = NOW()
		RETURNING failure_count
	`, session.Platform, session.GuildID, session.QQID, now).Scan(&count)
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

func (r *Repository) ResetAdmissionFailureCountTx(
	ctx context.Context,
	tx pgx.Tx,
	platform string,
	guildID string,
	qqID string,
	now time.Time,
) error {
	_, err := tx.Exec(ctx, `
		UPDATE group_admission_failures
		SET failure_count = 0, updated_at = $4
		WHERE platform = $1 AND guild_id = $2 AND qq_id = $3
	`, platform, guildID, qqID, now)
	if err != nil {
		return fmt.Errorf("ResetAdmissionFailureCountTx: %w", err)
	}
	return nil
}
