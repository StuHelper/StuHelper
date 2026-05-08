package admission

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) ListPendingActionSessions(
	ctx context.Context,
	filter AdmissionPendingActionFilter,
	now time.Time,
) ([]AdmissionSession, error) {
	if strings.TrimSpace(filter.Platform) == "" || strings.TrimSpace(filter.BotSelfID) == "" {
		return nil, ErrAdmissionPendingActionFilterInvalid
	}
	query, args := pendingActionSessionsQuery(filter, now)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListPendingActionSessions: %w", err)
	}
	defer rows.Close()
	return scanAdmissionSessions(rows)
}

func pendingActionSessionsQuery(filter AdmissionPendingActionFilter, now time.Time) (string, []any) {
	clauses, args := pendingActionSessionFilterClauses(filter)
	joinedMutedParam := len(args) + 1
	linkedParam := joinedMutedParam + 1
	nowParam := linkedParam + 1
	materialSubmittedParam := nowParam + 1
	verifiedParam := materialSubmittedParam + 1
	limitParam := verifiedParam + 1
	args = append(args, StatusJoinedMuted, StatusLinked, now, StatusMaterialSubmitted, StatusVerified, filter.Limit)
	query := fmt.Sprintf(`
		SELECT `+admissionSessionColumns+`
		FROM group_admission_sessions
		WHERE %s
		  AND (
		    (status = $%d AND (next_reminder_at IS NULL OR next_reminder_at <= $%d))
		    OR (status = $%d AND link_wait_deadline_at <= $%d)
		    OR (status = $%d AND submission_wait_deadline_at <= $%d)
		    OR (status = $%d AND manual_review_deadline_at <= $%d)
		    OR (status = $%d AND cancelled_at IS NULL)
		  )
		ORDER BY updated_at ASC, id ASC
		LIMIT $%d
		`, strings.Join(clauses, "\n		  AND "), joinedMutedParam, nowParam, joinedMutedParam, nowParam,
		linkedParam, nowParam, materialSubmittedParam, nowParam, verifiedParam, limitParam)
	return query, args
}

func pendingActionSessionFilterClauses(filter AdmissionPendingActionFilter) ([]string, []any) {
	args := []any{filter.Platform, filter.BotSelfID}
	clauses := []string{"platform = $1", "bot_self_id = $2"}
	return clauses, args
}

func (r *Repository) ListPendingFreshmanForwards(ctx context.Context) ([]freshmanForwardRecord, error) {
	rows, err := r.db.Query(ctx, pendingFreshmanForwardSQL())
	if err != nil {
		return nil, fmt.Errorf("ListPendingFreshmanForwards: %w", err)
	}
	defer rows.Close()
	return scanFreshmanForwardRecords(rows)
}

func (r *Repository) GetAdmissionFailure(
	ctx context.Context,
	platform string,
	guildID string,
	qqID string,
) (*AdmissionFailure, error) {
	return scanAdmissionFailure(r.db.QueryRow(ctx, `
		SELECT platform, guild_id, qq_id, failure_count
		FROM group_admission_failures
		WHERE platform = $1 AND guild_id = $2 AND qq_id = $3
	`, platform, guildID, qqID))
}

func (r *Repository) MarkReminderSentTx(
	ctx context.Context,
	input markReminderSentTxInput,
) error {
	_, err := input.Tx.Exec(ctx, `
			UPDATE group_admission_sessions
			SET last_reminded_at = $2, next_reminder_at = $3, updated_at = NOW()
			WHERE id = $1
		`, input.Session.ID, input.Now, input.Now.Add(time.Duration(input.Policy.ReminderIntervalSeconds)*time.Second))
	return err
}

func (r *Repository) MarkBotReleaseCompletedTx(ctx context.Context, input markBotSessionTxInput) error {
	_, err := input.Tx.Exec(ctx, `
			UPDATE group_admission_sessions
			SET cancelled_at = $2, updated_at = NOW()
			WHERE id = $1 AND status = $3
	`, input.SessionID, input.Now, StatusVerified)
	return err
}

func (r *Repository) MarkBotKickCompletedTx(ctx context.Context, input markBotSessionTxInput) (bool, error) {
	tag, err := input.Tx.Exec(ctx, `
			UPDATE group_admission_sessions
			SET status = $2, cancelled_at = $3, updated_at = NOW()
			WHERE id = $1 AND status IN ($4, $5, $6)
		`, input.SessionID, StatusExpiredKicked, input.Now, StatusJoinedMuted, StatusLinked, StatusMaterialSubmitted)
	return tag.RowsAffected() > 0, err
}

type markReminderSentTxInput struct {
	Tx      pgx.Tx
	Session *AdmissionSession
	Policy  *AdmissionPolicy
	Now     time.Time
}

type markBotSessionTxInput struct {
	Tx        pgx.Tx
	SessionID string
	Now       time.Time
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
