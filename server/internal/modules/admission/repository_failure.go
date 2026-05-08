package admission

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) IncrementFailureFromKickEventTx(
	ctx context.Context,
	input admissionFailureIncrementTxInput,
) (int, error) {
	var count int
	err := input.Tx.QueryRow(ctx, `
			INSERT INTO group_admission_failures (
				platform, guild_id, qq_id, failure_count, last_failure_at
			)
			VALUES ($1, $2, $3, 1, $4::timestamptz)
			ON CONFLICT (platform, guild_id, qq_id) DO UPDATE
			SET failure_count = group_admission_failures.failure_count + 1,
			    last_failure_at = $4::timestamptz,
			    updated_at = NOW()
			RETURNING failure_count
			`,
		input.Session.Platform, input.Session.GuildID, input.Session.QQID, input.Now,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("IncrementFailureFromKickEventTx: %w", err)
	}
	return count, nil
}

func (r *Repository) UpdateLastBotErrorTx(ctx context.Context, input updateLastBotErrorTxInput) error {
	_, err := input.Tx.Exec(ctx, `
			UPDATE group_admission_sessions
			SET last_bot_error = $2, updated_at = NOW()
			WHERE id = $1
		`, input.SessionID, input.BotError)
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
) (int, error) {
	var previousCount int
	err := tx.QueryRow(ctx, `
		SELECT failure_count
		FROM group_admission_failures
		WHERE platform = $1 AND guild_id = $2 AND qq_id = $3
		FOR UPDATE
	`, platform, guildID, qqID).Scan(&previousCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("ResetAdmissionFailureCountTx: %w", err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE group_admission_failures
		SET failure_count = 0, updated_at = $4
		WHERE platform = $1 AND guild_id = $2 AND qq_id = $3
	`, platform, guildID, qqID, now)
	if err != nil {
		return 0, fmt.Errorf("ResetAdmissionFailureCountTx: %w", err)
	}
	return previousCount, nil
}

type admissionFailureIncrementTxInput struct {
	Tx      pgx.Tx
	Session *AdmissionSession
	Now     time.Time
}

type updateLastBotErrorTxInput struct {
	Tx        pgx.Tx
	SessionID string
	BotError  string
}
