package admission

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	admissionBotActionOutboxTable = "admission_bot_action_outbox"
	admissionBotActionMaxAttempts = 5
	botActionDispatchRetryAfter   = 30 * time.Second
)

func (r *Repository) QueueBotActionTx(
	ctx context.Context,
	tx pgx.Tx,
	input AdmissionBotActionQueueInput,
) error {
	if input.Session == nil {
		return ErrAdmissionSessionNotFound
	}
	if input.Action == "" || input.ScheduledAt.IsZero() {
		return ErrAdmissionInvalidInput
	}
	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}
	key := admissionBotActionKey(input.Session.ID, input.Action, input.ScheduledAt)
	_, err := tx.Exec(ctx, `
		INSERT INTO admission_bot_action_outbox (
			action_key, session_id, action, platform, bot_self_id, guild_id, channel_id, qq_id,
			scheduled_at, status, attempt_count, next_attempt_at, last_error, message_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, 'pending', 0, $10, NULL, NULL,
			$10, $10
		)
		ON CONFLICT (action_key)
		DO UPDATE SET
			action = EXCLUDED.action,
			platform = EXCLUDED.platform,
			bot_self_id = EXCLUDED.bot_self_id,
			guild_id = EXCLUDED.guild_id,
			channel_id = EXCLUDED.channel_id,
			qq_id = EXCLUDED.qq_id,
			scheduled_at = EXCLUDED.scheduled_at,
			status = CASE
				WHEN admission_bot_action_outbox.status IN ('dispatched', 'succeeded', 'dead_letter') THEN admission_bot_action_outbox.status
				ELSE 'pending'
			END,
			next_attempt_at = CASE
				WHEN admission_bot_action_outbox.status IN ('dispatched', 'succeeded', 'dead_letter') THEN admission_bot_action_outbox.next_attempt_at
				ELSE EXCLUDED.next_attempt_at
			END,
			last_error = CASE
				WHEN admission_bot_action_outbox.status IN ('dispatched', 'succeeded', 'dead_letter') THEN admission_bot_action_outbox.last_error
				ELSE NULL
			END,
			updated_at = EXCLUDED.updated_at
	`, key, input.Session.ID, string(input.Action), input.Session.Platform, input.Session.BotSelfID,
		input.Session.GuildID, input.Session.ChannelID, input.Session.QQID, input.ScheduledAt, now)
	if err != nil {
		return fmt.Errorf("QueueBotActionTx: %w", err)
	}
	return nil
}

func (r *Repository) ClaimDueBotActions(
	ctx context.Context,
	filter AdmissionPendingActionFilter,
	now time.Time,
) ([]AdmissionBotActionOutboxRow, error) {
	if filter.Limit <= 0 {
		return []AdmissionBotActionOutboxRow{}, nil
	}
	ctx = withDBTable(ctx, admissionBotActionOutboxTable)
	var rows []AdmissionBotActionOutboxRow
	err := r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		claimed, err := tx.Query(
			ctx,
			claimDueBotActionsSQL(),
			filter.Platform,
			filter.BotSelfID,
			now,
			filter.Limit,
			now.Add(botActionDispatchRetryAfter),
			admissionBotActionMaxAttempts,
		)
		if err != nil {
			return fmt.Errorf("ClaimDueBotActions query: %w", err)
		}
		defer claimed.Close()
		rows, err = scanBotActionOutboxRows(claimed)
		return err
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func claimDueBotActionsSQL() string {
	return `
		WITH terminal AS (
			UPDATE admission_bot_action_outbox
			SET status = 'dead_letter',
			    last_error = CASE
			      WHEN status = 'dispatched' THEN 'bot action dispatch timed out'
			      ELSE COALESCE(last_error, 'bot action exceeded max attempts')
			    END,
			    updated_at = $3
			WHERE platform = $1
			  AND bot_self_id = $2
			  AND scheduled_at <= $3
			  AND next_attempt_at <= $3
			  AND status IN ('pending', 'failed', 'dispatched')
			  AND attempt_count >= $6
			RETURNING id
		),
		candidates AS (
			SELECT id
			FROM admission_bot_action_outbox
			WHERE platform = $1
			  AND bot_self_id = $2
			  AND scheduled_at <= $3
			  AND next_attempt_at <= $3
			  AND status IN ('pending', 'failed', 'dispatched')
			  AND attempt_count < $6
			ORDER BY scheduled_at ASC, id ASC
			LIMIT $4
			FOR UPDATE SKIP LOCKED
		),
		claimed AS (
			UPDATE admission_bot_action_outbox AS o
			SET status = 'dispatched',
			    attempt_count = attempt_count + 1,
			    next_attempt_at = $5,
			    updated_at = $3
			FROM candidates
			WHERE o.id = candidates.id
			RETURNING ` + admissionBotActionColumns("o") + `
		)
		SELECT ` + admissionBotActionColumns("claimed") + `, ` + admissionSessionColumnsWithAlias("s") + `
		FROM claimed
		JOIN group_admission_sessions AS s ON s.id = claimed.session_id
		ORDER BY claimed.scheduled_at ASC, claimed.id ASC
	`
}

func (r *Repository) GetBotActionForUpdateTx(
	ctx context.Context,
	tx pgx.Tx,
	actionID int64,
) (*AdmissionBotActionOutboxRow, error) {
	row := tx.QueryRow(ctx, `
		SELECT `+admissionBotActionColumns("o")+`, `+admissionSessionColumnsWithAlias("s")+`
		FROM admission_bot_action_outbox AS o
		JOIN group_admission_sessions AS s ON s.id = o.session_id
		WHERE o.id = $1
		FOR UPDATE OF o, s
	`, actionID)
	action, err := scanBotActionOutboxRow(row)
	if err != nil {
		return nil, fmt.Errorf("GetBotActionForUpdateTx: %w", err)
	}
	return &action, nil
}

func (r *Repository) MarkBotActionSucceededTx(
	ctx context.Context,
	tx pgx.Tx,
	actionID int64,
	dispatchAttempt int,
	event BotEventInput,
	now time.Time,
) error {
	tag, err := tx.Exec(ctx, `
		UPDATE admission_bot_action_outbox
		SET status = 'succeeded',
		    last_error = NULL,
		    message_id = NULLIF($2, ''),
		    updated_at = $3
		WHERE id = $1
		  AND status = 'dispatched'
		  AND attempt_count = $4
	`, actionID, event.MessageID, now, dispatchAttempt)
	if err != nil {
		return fmt.Errorf("MarkBotActionSucceededTx: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrAdmissionBotActionLeaseLost
	}
	return nil
}

func (r *Repository) MarkBotActionFailedTx(
	ctx context.Context,
	tx pgx.Tx,
	actionID int64,
	dispatchAttempt int,
	event BotEventInput,
	now time.Time,
	attemptCount int,
) error {
	errMsg := normalizeBotEventError(event)
	tag, err := tx.Exec(ctx, `
		UPDATE admission_bot_action_outbox
		SET status = CASE WHEN attempt_count >= $5 THEN 'dead_letter' ELSE 'failed' END,
		    last_error = $2,
		    next_attempt_at = $3,
		    updated_at = $4
		WHERE id = $1
		  AND status = 'dispatched'
		  AND attempt_count = $6
	`, actionID, errMsg, now.Add(botActionRetryBackoff(attemptCount)), now, admissionBotActionMaxAttempts, dispatchAttempt)
	if err != nil {
		return fmt.Errorf("MarkBotActionFailedTx: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrAdmissionBotActionLeaseLost
	}
	return nil
}

func (r *Repository) MarkBotActionPreparationFailed(
	ctx context.Context,
	actionID int64,
	dispatchAttempt int,
	lastError string,
	now time.Time,
) error {
	ctx = withDBTable(ctx, admissionBotActionOutboxTable)
	tag, err := r.db.Exec(ctx, `
		UPDATE admission_bot_action_outbox
		SET status = CASE WHEN attempt_count >= $5 THEN 'dead_letter' ELSE 'failed' END,
		    last_error = $2,
		    next_attempt_at = $3,
		    updated_at = $4
		WHERE id = $1
		  AND status = 'dispatched'
		  AND attempt_count = $6
	`, actionID, lastError, now.Add(botActionRetryBackoff(dispatchAttempt)), now,
		admissionBotActionMaxAttempts, dispatchAttempt)
	if err != nil {
		return fmt.Errorf("MarkBotActionPreparationFailed: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrAdmissionBotActionLeaseLost
	}
	return nil
}

func (r *Repository) MarkBotActionStale(
	ctx context.Context,
	actionID int64,
	dispatchAttempt int,
	now time.Time,
) error {
	ctx = withDBTable(ctx, admissionBotActionOutboxTable)
	tag, err := r.db.Exec(ctx, `
		UPDATE admission_bot_action_outbox
		SET status = 'stale', updated_at = $2
		WHERE id = $1
		  AND status = 'dispatched'
		  AND attempt_count = $3
	`, actionID, now, dispatchAttempt)
	if err != nil {
		return fmt.Errorf("MarkBotActionStale: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrAdmissionBotActionLeaseLost
	}
	return nil
}

// abandonBotActionClaims is only valid for a synchronous, single-shot cleanup
// of claims that ClaimQueuedAdmissionActions has not exposed to a bot. Returning
// the retry budget reuses the numeric attempt on the next claim, so this helper
// must never be retried asynchronously or used after a response may be visible.
func (r *Repository) abandonBotActionClaims(
	ctx context.Context,
	rows []AdmissionBotActionOutboxRow,
	now time.Time,
) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	ids := make([]int64, len(rows))
	attempts := make([]int32, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
		attempts[i] = int32(rows[i].AttemptCount)
	}
	ctx = withDBTable(ctx, admissionBotActionOutboxTable)
	tag, err := r.db.Exec(ctx, `
		WITH claims(id, attempt_count) AS (
			SELECT *
			FROM unnest($1::bigint[], $2::integer[])
		)
		UPDATE admission_bot_action_outbox AS o
		SET status = CASE WHEN o.attempt_count <= 1 THEN 'pending' ELSE 'failed' END,
		    attempt_count = o.attempt_count - 1,
		    next_attempt_at = $4,
		    updated_at = $3
		FROM claims
		WHERE o.id = claims.id
		  AND o.status = 'dispatched'
		  AND o.attempt_count = claims.attempt_count
	`, ids, attempts, now, now.Add(botActionDispatchRetryAfter))
	if err != nil {
		return 0, fmt.Errorf("abandonBotActionClaims: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *Repository) MarkBotActionStaleTx(ctx context.Context, tx pgx.Tx, actionID int64, now time.Time) error {
	_, err := tx.Exec(ctx, `
		UPDATE admission_bot_action_outbox
		SET status = 'stale', updated_at = $2
		WHERE id = $1 AND status <> 'succeeded'
	`, actionID, now)
	if err != nil {
		return fmt.Errorf("MarkBotActionStaleTx: %w", err)
	}
	return nil
}

func (r *Repository) ListVerifiedUnreleasedSessionsByUserSchoolTx(
	ctx context.Context,
	tx pgx.Tx,
	userID int64,
	schoolID int64,
) ([]AdmissionSession, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+admissionSessionScanColumnsWithAlias("s")+`
		FROM group_admission_sessions AS s
		JOIN group_admission_policies AS p
		  ON p.platform = s.platform AND p.guild_id = s.guild_id
		WHERE s.user_id = $1
		  AND p.school_id = $2
		  AND s.status = $3
		  AND s.cancelled_at IS NULL
		ORDER BY s.updated_at ASC, s.id ASC
	`, userID, schoolID, StatusVerified)
	if err != nil {
		return nil, fmt.Errorf("ListVerifiedUnreleasedSessionsByUserSchoolTx: %w", err)
	}
	defer rows.Close()
	return scanAdmissionSessions(rows)
}

func admissionBotActionKey(sessionID string, action BotAction, scheduledAt time.Time) string {
	if action == BotActionRelease {
		return sessionID + ":release"
	}
	return sessionID + ":" + string(action) + ":" + strconv.FormatInt(scheduledAt.Unix(), 10)
}

func botActionRetryBackoff(attempts int) time.Duration {
	if attempts < 1 {
		return botActionDispatchRetryAfter
	}
	backoff := time.Duration(1<<min(attempts, 5)) * botActionDispatchRetryAfter
	if backoff > 10*time.Minute {
		return 10 * time.Minute
	}
	return backoff
}

func admissionBotActionColumns(alias string) string {
	return alias + `.id, ` + alias + `.action_key, ` + alias + `.session_id, ` + alias + `.action,
		` + alias + `.platform, ` + alias + `.bot_self_id, ` + alias + `.guild_id, ` + alias + `.channel_id, ` + alias + `.qq_id,
		` + alias + `.scheduled_at, ` + alias + `.status, ` + alias + `.attempt_count, ` + alias + `.next_attempt_at,
		` + alias + `.last_error, ` + alias + `.message_id, ` + alias + `.created_at, ` + alias + `.updated_at`
}

func admissionSessionColumnsWithAlias(alias string) string {
	return alias + `.id, ` + alias + `.platform, ` + alias + `.bot_self_id, ` + alias + `.guild_id, ` + alias + `.channel_id, ` + alias + `.qq_id,
		` + alias + `.user_id, ` + alias + `.token_hash, ` + alias + `.auth_url, ` + alias + `.token_expires_at,
		` + alias + `.token_consumed_at, ` + alias + `.status, ` + alias + `.link_wait_deadline_at,
		` + alias + `.submission_wait_deadline_at, ` + alias + `.manual_review_deadline_at, ` + alias + `.initial_mute_until,
		` + alias + `.verified_at, ` + alias + `.cancelled_at, ` + alias + `.last_bot_error, ` + alias + `.next_reminder_at`
}

func admissionSessionScanColumnsWithAlias(alias string) string {
	return alias + `.id, ` + alias + `.platform, ` + alias + `.bot_self_id, ` + alias + `.guild_id, ` + alias + `.channel_id, ` + alias + `.qq_id,
		` + alias + `.user_id, ` + alias + `.token_hash, ` + alias + `.auth_url,
		` + alias + `.token_expires_at, ` + alias + `.token_consumed_at, ` + alias + `.status, ` + alias + `.link_wait_deadline_at,
		` + alias + `.submission_wait_deadline_at, ` + alias + `.manual_review_deadline_at, ` + alias + `.initial_mute_until,
		` + alias + `.verified_at, ` + alias + `.cancelled_at, ` + alias + `.last_bot_error`
}

func scanBotActionOutboxRows(rows pgx.Rows) ([]AdmissionBotActionOutboxRow, error) {
	items := []AdmissionBotActionOutboxRow{}
	for rows.Next() {
		item, err := scanBotActionOutboxRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan bot action outbox row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan bot action outbox rows: %w", err)
	}
	return items, nil
}

func scanBotActionOutboxRow(row pgx.Row) (AdmissionBotActionOutboxRow, error) {
	var item AdmissionBotActionOutboxRow
	err := row.Scan(
		&item.ID, &item.ActionKey, &item.SessionID, &item.Action,
		&item.Platform, &item.BotSelfID, &item.GuildID, &item.ChannelID, &item.QQID,
		&item.ScheduledAt, &item.Status, &item.AttemptCount, &item.NextAttemptAt,
		&item.LastError, &item.MessageID, &item.CreatedAt, &item.UpdatedAt,
		&item.Session.ID, &item.Session.Platform, &item.Session.BotSelfID, &item.Session.GuildID,
		&item.Session.ChannelID, &item.Session.QQID, &item.Session.UserID, &item.Session.TokenHash,
		&item.Session.AuthURL, &item.Session.TokenExpiresAt, &item.Session.TokenConsumedAt,
		&item.Session.Status, &item.Session.LinkWaitDeadlineAt, &item.Session.SubmissionWaitDeadlineAt,
		&item.Session.ManualReviewDeadlineAt, &item.Session.InitialMuteUntil, &item.Session.VerifiedAt,
		&item.Session.CancelledAt, &item.Session.LastBotError, &item.Session.nextReminderAt,
	)
	return item, err
}
