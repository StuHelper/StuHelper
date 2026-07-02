package admission

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const (
	defaultPendingActionLimit = 50
	maxPendingActionLimit     = 200
	pendingForwardLimit       = 100
)

type AdmissionFailure struct {
	Platform     string
	GuildID      string
	QQID         string
	FailureCount int
}

type freshmanForwardRecord struct {
	Application        FreshmanApplication
	ObjectKey          string
	ManagementGuildIDs []string
	Platform           string
	BotSelfID          string
	SchoolName         string
	QQID               string
}

func (r *Repository) scanAdmissionSessions(rows pgx.Rows) ([]AdmissionSession, error) {
	items := make([]AdmissionSession, 0)
	for rows.Next() {
		session, err := r.scanAdmissionSession(rows)
		if err != nil {
			return nil, fmt.Errorf("scan admission session: %w", err)
		}
		items = append(items, *session)
	}
	return items, rows.Err()
}

func scanFreshmanForwardRecords(rows pgx.Rows) ([]freshmanForwardRecord, error) {
	items := make([]freshmanForwardRecord, 0)
	for rows.Next() {
		record, err := scanFreshmanForwardRecord(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, record)
	}
	return items, rows.Err()
}

func scanFreshmanForwardRecord(row pgx.Row) (freshmanForwardRecord, error) {
	var record freshmanForwardRecord
	err := row.Scan(
		&record.Application.ID, &record.Application.UserID, &record.Application.SchoolID,
		&record.Application.AdmissionSessionID, &record.Application.Status, &record.Application.ApplicantName,
		&record.Application.ApplicantNameMasked, &record.Application.DepartmentOrMajor,
		&record.Application.MaterialType, &record.Application.ProvisionalExpiresAt,
		&record.Application.ReviewedAt, &record.Application.CreatedAt, &record.ObjectKey,
		&record.ManagementGuildIDs, &record.Platform, &record.BotSelfID, &record.SchoolName, &record.QQID,
	)
	return record, err
}

func scanAdmissionFailure(row pgx.Row) (*AdmissionFailure, error) {
	var failure AdmissionFailure
	err := row.Scan(
		&failure.Platform, &failure.GuildID, &failure.QQID, &failure.FailureCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &failure, nil
}

func pendingFreshmanForwardSQL() string {
	return fmt.Sprintf(`
		SELECT app.id, app.user_id, app.school_id, app.admission_session_id, app.status,
		       app.applicant_name, app.applicant_name_masked, app.department_or_major,
		       app.material_type, app.provisional_expires_at, app.reviewed_at, app.created_at,
		       material.object_key, policy.management_guild_ids, session.platform,
		       session.bot_self_id, schools.name, session.qq_id
		FROM freshman_verification_applications app
		JOIN freshman_verification_materials material ON material.application_id = app.id
		JOIN group_admission_sessions session ON session.id = app.admission_session_id
		JOIN group_admission_policies policy ON policy.platform = session.platform AND policy.guild_id = session.guild_id
		JOIN schools ON schools.id = app.school_id
		WHERE app.status = 'pending'
		  AND app.forwarded_at IS NULL
		  AND policy.forward_raw_material_to_qq = TRUE
		ORDER BY app.created_at ASC
		LIMIT %d`, pendingForwardLimit)
}
