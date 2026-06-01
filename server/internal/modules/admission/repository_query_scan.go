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
		&session.UserID, &session.TokenHash, &session.AuthURL, &session.TokenExpiresAt,
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

func adminFreshmanApplicationListSQL() string {
	return `
		SELECT app.id, app.user_id, app.school_id, app.admission_session_id, app.status,
		       app.applicant_name, app.applicant_name_masked, app.department_or_major,
		       app.material_type, app.provisional_expires_at, app.reviewed_at,
		       app.created_at, material.object_key, session.qq_id,
		       failure.failure_count, COUNT(*) OVER() AS total
		FROM freshman_verification_applications app
		LEFT JOIN freshman_verification_materials material ON material.application_id = app.id
		LEFT JOIN group_admission_sessions session ON session.id = app.admission_session_id
		LEFT JOIN group_admission_failures failure
		  ON failure.platform = session.platform
		 AND failure.guild_id = session.guild_id
		 AND failure.qq_id = session.qq_id
		WHERE ($1::text = '' OR app.status = $1)
		ORDER BY app.created_at DESC, app.id ASC
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

func scanAdminFreshmanApplicationList(rows pgx.Rows) ([]adminFreshmanApplicationRow, int, error) {
	items := make([]adminFreshmanApplicationRow, 0)
	total := 0
	for rows.Next() {
		item, rowTotal, err := scanAdminFreshmanApplicationRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan admin freshman application: %w", err)
		}
		total = rowTotal
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func scanAdminFreshmanApplicationRow(row pgx.Row) (*adminFreshmanApplicationRow, int, error) {
	var item adminFreshmanApplicationRow
	var total int
	err := row.Scan(
		&item.Application.ID, &item.Application.UserID, &item.Application.SchoolID,
		&item.Application.AdmissionSessionID, &item.Application.Status, &item.Application.ApplicantName,
		&item.Application.ApplicantNameMasked, &item.Application.DepartmentOrMajor,
		&item.Application.MaterialType, &item.Application.ProvisionalExpiresAt,
		&item.Application.ReviewedAt, &item.Application.CreatedAt, &item.ObjectKey,
		&item.QQID, &item.FailureCount, &total,
	)
	return &item, total, err
}

func mapFreshmanApplicationScanError(op string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAdmissionApplicationNotFound
	}
	return fmt.Errorf("%s: %w", op, err)
}
