package admission

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const admissionSessionColumns = `
	id, platform, bot_self_id, guild_id, channel_id, qq_id, user_id, token_hash, auth_url,
	token_expires_at, token_consumed_at, status, link_wait_deadline_at,
	submission_wait_deadline_at, manual_review_deadline_at, initial_mute_until,
	verified_at, cancelled_at, last_bot_error
`

func (r *Repository) CreateSession(ctx context.Context, session *AdmissionSession) error {
	ctx = withDBTable(ctx, "group_admission_sessions")
	_, err := r.db.Exec(ctx, `
		INSERT INTO group_admission_sessions (
			id, platform, bot_self_id, guild_id, channel_id, qq_id, user_id, token_hash, auth_url,
			token_expires_at, token_consumed_at, status, link_wait_deadline_at,
			submission_wait_deadline_at, manual_review_deadline_at, initial_mute_until,
			verified_at, cancelled_at, last_bot_error, next_reminder_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`, session.ID, session.Platform, session.BotSelfID, session.GuildID, session.ChannelID, session.QQID,
		session.UserID, session.TokenHash, session.AuthURL, session.TokenExpiresAt, session.TokenConsumedAt, session.Status,
		session.LinkWaitDeadlineAt, session.SubmissionWaitDeadlineAt, session.ManualReviewDeadlineAt,
		session.InitialMuteUntil, session.VerifiedAt, session.CancelledAt, session.LastBotError, session.nextReminderAt)
	if err != nil {
		return fmt.Errorf("CreateSession: %w", err)
	}
	return nil
}

func (r *Repository) CreateSessionTx(ctx context.Context, tx pgx.Tx, session *AdmissionSession) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO group_admission_sessions (
			id, platform, bot_self_id, guild_id, channel_id, qq_id, user_id, token_hash, auth_url,
			token_expires_at, token_consumed_at, status, link_wait_deadline_at,
			submission_wait_deadline_at, manual_review_deadline_at, initial_mute_until,
			verified_at, cancelled_at, last_bot_error, next_reminder_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`, session.ID, session.Platform, session.BotSelfID, session.GuildID, session.ChannelID, session.QQID,
		session.UserID, session.TokenHash, session.AuthURL, session.TokenExpiresAt, session.TokenConsumedAt, session.Status,
		session.LinkWaitDeadlineAt, session.SubmissionWaitDeadlineAt, session.ManualReviewDeadlineAt,
		session.InitialMuteUntil, session.VerifiedAt, session.CancelledAt, session.LastBotError, session.nextReminderAt)
	if err != nil {
		return fmt.Errorf("CreateSessionTx: %w", err)
	}
	return nil
}

func (r *Repository) GetSessionByID(ctx context.Context, id string) (*AdmissionSession, error) {
	ctx = withDBTable(ctx, "group_admission_sessions")
	query := "SELECT " + admissionSessionColumns + " FROM group_admission_sessions WHERE id = $1"
	session, err := scanAdmissionSession(r.db.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("GetSessionByID: %w", err)
	}
	return session, nil
}

func (r *Repository) GetSessionByIDForUpdate(ctx context.Context, tx pgx.Tx, id string) (*AdmissionSession, error) {
	query := "SELECT " + admissionSessionColumns + " FROM group_admission_sessions WHERE id = $1 FOR UPDATE"
	session, err := scanAdmissionSession(tx.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("GetSessionByIDForUpdate: %w", err)
	}
	return session, nil
}

func (r *Repository) GetLatestSessionBySubject(
	ctx context.Context,
	input BotSessionSubjectInput,
) (*AdmissionSession, error) {
	ctx = withDBTable(ctx, "group_admission_sessions")
	query := `
		SELECT ` + admissionSessionColumns + `
		FROM group_admission_sessions
		WHERE platform = $1 AND guild_id = $2 AND qq_id = $3
		ORDER BY created_at DESC, updated_at DESC, id DESC
		LIMIT 1
	`
	session, err := scanAdmissionSession(r.db.QueryRow(ctx, query, input.Platform, input.GuildID, input.QQID))
	if err != nil {
		if errors.Is(err, ErrAdmissionTokenNotFound) {
			return nil, ErrAdmissionSessionNotFound
		}
		return nil, fmt.Errorf("GetLatestSessionBySubject: %w", err)
	}
	return session, nil
}

func (r *Repository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*AdmissionSession, error) {
	ctx = withDBTable(ctx, "group_admission_sessions")
	query := "SELECT " + admissionSessionColumns + " FROM group_admission_sessions WHERE token_hash = $1"
	session, err := scanAdmissionSession(r.db.QueryRow(ctx, query, tokenHash))
	if err != nil {
		return nil, fmt.Errorf("GetSessionByTokenHash: %w", err)
	}
	return session, nil
}

func (r *Repository) GetSessionByTokenHashForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	tokenHash string,
) (*AdmissionSession, error) {
	query := "SELECT " + admissionSessionColumns + " FROM group_admission_sessions WHERE token_hash = $1 FOR UPDATE"
	session, err := scanAdmissionSession(tx.QueryRow(ctx, query, tokenHash))
	if err != nil {
		return nil, fmt.Errorf("GetSessionByTokenHashForUpdate: %w", err)
	}
	return session, nil
}

func (r *Repository) CancelInProgressSessionsBySubjectTx(
	ctx context.Context,
	tx pgx.Tx,
	input BotSessionSubjectInput,
	now time.Time,
) ([]string, error) {
	rows, err := tx.Query(ctx, `
		UPDATE group_admission_sessions
		SET status = $4, cancelled_at = $5, next_reminder_at = NULL, updated_at = NOW()
		WHERE platform = $1 AND guild_id = $2 AND qq_id = $3
		  AND status IN ($6, $7, $8)
		RETURNING id
	`, input.Platform, input.GuildID, input.QQID, StatusCancelled, now,
		StatusJoinedMuted, StatusLinked, StatusMaterialSubmitted)
	if err != nil {
		return nil, fmt.Errorf("CancelInProgressSessionsBySubjectTx: %w", err)
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("CancelInProgressSessionsBySubjectTx scan: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("CancelInProgressSessionsBySubjectTx rows: %w", err)
	}
	return ids, nil
}

func (r *Repository) CancelInProgressSessionByID(
	ctx context.Context,
	sessionID string,
	now time.Time,
) (*AdmissionSession, error) {
	ctx = withDBTable(ctx, "group_admission_sessions")
	query := `
		UPDATE group_admission_sessions
		SET status = $2, cancelled_at = $3, next_reminder_at = NULL, last_bot_error = NULL, updated_at = NOW()
		WHERE id = $1 AND status IN ($4, $5, $6)
		RETURNING ` + admissionSessionColumns
	session, err := scanAdmissionSession(r.db.QueryRow(
		ctx,
		query,
		sessionID,
		StatusCancelled,
		now,
		StatusJoinedMuted,
		StatusLinked,
		StatusMaterialSubmitted,
	))
	if err != nil {
		return nil, fmt.Errorf("CancelInProgressSessionByID: %w", err)
	}
	return session, nil
}

func (r *Repository) MarkTokenConsumedAndLinked(
	ctx context.Context,
	tx pgx.Tx,
	sessionID string,
	userID int64,
	now time.Time,
	submissionDeadline time.Time,
) (*AdmissionSession, error) {
	return r.updateSessionTx(ctx, tx, `
		UPDATE group_admission_sessions
		SET user_id = $2, token_consumed_at = $3, status = $4,
		    submission_wait_deadline_at = $5, next_reminder_at = NULL, updated_at = NOW()
		WHERE id = $1
		RETURNING `+admissionSessionColumns, sessionID, userID, now, StatusLinked, submissionDeadline)
}

func (r *Repository) MarkMaterialSubmitted(
	ctx context.Context,
	sessionID string,
	manualReviewDeadline time.Time,
) (*AdmissionSession, error) {
	ctx = withDBTable(ctx, "group_admission_sessions")
	query := `
		UPDATE group_admission_sessions
		SET status = $2, manual_review_deadline_at = $3, next_reminder_at = NULL, updated_at = NOW()
		WHERE id = $1
		RETURNING ` + admissionSessionColumns
	session, err := scanAdmissionSession(r.db.QueryRow(ctx, query, sessionID, StatusMaterialSubmitted, manualReviewDeadline))
	if err != nil {
		return nil, fmt.Errorf("MarkMaterialSubmitted: %w", err)
	}
	return session, nil
}

func (r *Repository) MarkVerified(ctx context.Context, sessionID string, now time.Time) (*AdmissionSession, error) {
	ctx = withDBTable(ctx, "group_admission_sessions")
	query := `
		UPDATE group_admission_sessions
		SET status = $2, verified_at = $3, next_reminder_at = NULL, updated_at = NOW()
		WHERE id = $1
		RETURNING ` + admissionSessionColumns
	session, err := scanAdmissionSession(r.db.QueryRow(ctx, query, sessionID, StatusVerified, now))
	if err != nil {
		return nil, fmt.Errorf("MarkVerified: %w", err)
	}
	return session, nil
}

func (r *Repository) updateSessionTx(ctx context.Context, tx pgx.Tx, query string, args ...any) (*AdmissionSession, error) {
	session, err := scanAdmissionSession(tx.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, fmt.Errorf("updateSessionTx: %w", err)
	}
	return session, nil
}

func scanAdmissionSession(row pgx.Row) (*AdmissionSession, error) {
	var session AdmissionSession
	err := row.Scan(
		&session.ID, &session.Platform, &session.BotSelfID, &session.GuildID, &session.ChannelID, &session.QQID,
		&session.UserID, &session.TokenHash, &session.AuthURL, &session.TokenExpiresAt,
		&session.TokenConsumedAt, &session.Status, &session.LinkWaitDeadlineAt,
		&session.SubmissionWaitDeadlineAt, &session.ManualReviewDeadlineAt, &session.InitialMuteUntil,
		&session.VerifiedAt, &session.CancelledAt, &session.LastBotError,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAdmissionTokenNotFound
		}
		return nil, err
	}
	return &session, nil
}
