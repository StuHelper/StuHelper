package admission

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func scanAdmissionPolicies(rows pgx.Rows) ([]AdmissionPolicy, error) {
	items := make([]AdmissionPolicy, 0)
	for rows.Next() {
		policy, err := scanAdmissionPolicy(rows)
		if err != nil {
			return nil, fmt.Errorf("scan admission policy: %w", err)
		}
		items = append(items, *policy)
	}
	return items, rows.Err()
}

func scanAdmissionSessionList(rows pgx.Rows) ([]AdmissionSession, int, error) {
	items := make([]AdmissionSession, 0)
	total := 0
	for rows.Next() {
		session, rowTotal, err := scanAdmissionSessionWithTotal(rows)
		if err != nil {
			return nil, 0, err
		}
		total = rowTotal
		items = append(items, *session)
	}
	return items, total, rows.Err()
}

func scanAdmissionSessionWithTotal(row pgx.Row) (*AdmissionSession, int, error) {
	var session AdmissionSession
	var total int
	err := row.Scan(
		&session.ID, &session.Platform, &session.BotSelfID, &session.GuildID, &session.ChannelID, &session.QQID,
		&session.QQNickname, &session.UserID, &session.TokenHash, &session.AuthURL, &session.TokenExpiresAt,
		&session.TokenConsumedAt, &session.Status, &session.LinkWaitDeadlineAt,
		&session.SubmissionWaitDeadlineAt, &session.ManualReviewDeadlineAt, &session.InitialMuteUntil,
		&session.VerifiedAt, &session.CancelledAt, &session.LastBotError, &total,
	)
	if err != nil {
		return nil, 0, err
	}
	return &session, total, nil
}

func freshmanApplicationSelectSQL() string {
	return `
		SELECT id, user_id, school_id, admission_session_id, status, applicant_name,
		       applicant_name_masked, department_or_major, material_type,
		       provisional_expires_at, reviewed_at, created_at
		FROM freshman_verification_applications`
}

func freshmanApplicationListSQL() string {
	return `
		SELECT id, user_id, school_id, admission_session_id, status, applicant_name,
		       applicant_name_masked, department_or_major, material_type,
		       provisional_expires_at, reviewed_at, created_at, COUNT(*) OVER() AS total
		FROM freshman_verification_applications
		WHERE ($1::text = '' OR status = $1)
		ORDER BY created_at DESC, id ASC
		LIMIT $2 OFFSET $3`
}

func scanFreshmanApplicationList(rows pgx.Rows) ([]FreshmanApplication, int, error) {
	items := make([]FreshmanApplication, 0)
	total := 0
	for rows.Next() {
		app, rowTotal, err := scanFreshmanApplicationWithTotal(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan freshman application: %w", err)
		}
		total = rowTotal
		items = append(items, *app)
	}
	return items, total, rows.Err()
}

func scanFreshmanApplicationWithTotal(row pgx.Row) (*FreshmanApplication, int, error) {
	var app FreshmanApplication
	var total int
	err := row.Scan(
		&app.ID, &app.UserID, &app.SchoolID, &app.AdmissionSessionID, &app.Status, &app.ApplicantName,
		&app.ApplicantNameMasked, &app.DepartmentOrMajor, &app.MaterialType, &app.ProvisionalExpiresAt,
		&app.ReviewedAt, &app.CreatedAt, &total,
	)
	if err != nil {
		return nil, 0, err
	}
	return &app, total, nil
}

func mapFreshmanApplicationScanError(op string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAdmissionApplicationNotFound
	}
	return fmt.Errorf("%s: %w", op, err)
}
