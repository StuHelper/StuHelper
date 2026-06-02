package admission

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/schoolauth"
)

func (r *Repository) GetLinkedSessionByUserID(
	ctx context.Context,
	userID int64,
	now time.Time,
	admissionSessionID string,
) (*AdmissionSession, error) {
	ctx = withDBTable(ctx, "group_admission_sessions")
	admissionSessionID = strings.TrimSpace(admissionSessionID)
	if admissionSessionID != "" {
		session, err := scanAdmissionSession(r.db.QueryRow(ctx, `
			SELECT `+admissionSessionColumns+`
			FROM group_admission_sessions
			WHERE user_id = $1 AND id = $2 AND status = $3
			  AND cancelled_at IS NULL
			LIMIT 1
		`, userID, admissionSessionID, StatusLinked))
		if errors.Is(err, ErrAdmissionTokenNotFound) {
			return nil, nil
		}
		return session, err
	}
	query := "SELECT " + admissionSessionColumns + `
		FROM group_admission_sessions
		WHERE user_id = $1 AND status = $2
		  AND cancelled_at IS NULL
		ORDER BY
		  CASE WHEN submission_wait_deadline_at >= $3 THEN 0 ELSE 1 END,
		  updated_at DESC
		LIMIT 1`
	session, err := scanAdmissionSession(r.db.QueryRow(ctx, query, userID, StatusLinked, now))
	if errors.Is(err, ErrAdmissionTokenNotFound) {
		return nil, nil
	}
	return session, err
}

func (r *Repository) GetLinkedSessionByUserIDTx(
	ctx context.Context,
	tx pgx.Tx,
	userID int64,
	now time.Time,
	admissionSessionID string,
) (*AdmissionSession, error) {
	admissionSessionID = strings.TrimSpace(admissionSessionID)
	if admissionSessionID != "" {
		session, err := scanAdmissionSession(tx.QueryRow(ctx, `
			SELECT `+admissionSessionColumns+`
			FROM group_admission_sessions
			WHERE user_id = $1 AND id = $2 AND status = $3
			  AND cancelled_at IS NULL
			LIMIT 1
			FOR UPDATE
		`, userID, admissionSessionID, StatusLinked))
		if errors.Is(err, ErrAdmissionTokenNotFound) {
			return nil, nil
		}
		return session, err
	}
	query := "SELECT " + admissionSessionColumns + `
		FROM group_admission_sessions
		WHERE user_id = $1 AND status = $2
		  AND cancelled_at IS NULL
		ORDER BY
		  CASE WHEN submission_wait_deadline_at >= $3 THEN 0 ELSE 1 END,
		  updated_at DESC
		LIMIT 1
		FOR UPDATE`
	session, err := scanAdmissionSession(tx.QueryRow(ctx, query, userID, StatusLinked, now))
	if errors.Is(err, ErrAdmissionTokenNotFound) {
		return nil, nil
	}
	return session, err
}

func (r *Repository) CreateFreshmanApplication(ctx context.Context, app *FreshmanApplication) error {
	ctx = withDBTable(ctx, "freshman_verification_applications")
	_, err := r.db.Exec(ctx, `
		INSERT INTO freshman_verification_applications (
			id, user_id, school_id, admission_session_id, applicant_name, applicant_name_masked,
			department_or_major, material_type, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, app.ID, app.UserID, app.SchoolID, app.AdmissionSessionID, app.ApplicantName,
		app.ApplicantNameMasked, app.DepartmentOrMajor, app.MaterialType, app.Status)
	if err != nil {
		return fmt.Errorf("CreateFreshmanApplication: %w", err)
	}
	return nil
}

func (r *Repository) HasPendingFreshmanApplication(ctx context.Context, userID, schoolID int64) (bool, error) {
	ctx = withDBTable(ctx, "freshman_verification_applications")
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM freshman_verification_applications
			WHERE user_id = $1 AND school_id = $2 AND status = $3
		)
	`, userID, schoolID, FreshmanApplicationPending).Scan(&exists)
	return exists, err
}

func (r *Repository) GetPendingFreshmanApplication(ctx context.Context, userID, schoolID int64) (*FreshmanApplication, error) {
	ctx = withDBTable(ctx, "freshman_verification_applications")
	app, err := scanFreshmanApplication(r.db.QueryRow(ctx, `
		SELECT id, user_id, school_id, admission_session_id, status, applicant_name, applicant_name_masked,
		       department_or_major, material_type, provisional_expires_at, reviewed_at, created_at
		FROM freshman_verification_applications
		WHERE user_id = $1 AND school_id = $2 AND status = $3
		ORDER BY created_at DESC
		LIMIT 1
	`, userID, schoolID, FreshmanApplicationPending))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetPendingFreshmanApplication: %w", err)
	}
	return app, nil
}

func (r *Repository) ReassignPendingFreshmanApplicationSession(
	ctx context.Context,
	applicationID string,
	sessionID string,
) (*FreshmanApplication, error) {
	ctx = withDBTable(ctx, "freshman_verification_applications")
	app, err := scanFreshmanApplication(r.db.QueryRow(ctx, `
		UPDATE freshman_verification_applications
		SET admission_session_id = $2
		WHERE id = $1 AND status = $3
		RETURNING id, user_id, school_id, admission_session_id, status, applicant_name, applicant_name_masked,
		          department_or_major, material_type, provisional_expires_at, reviewed_at, created_at
	`, applicationID, sessionID, FreshmanApplicationPending))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAdmissionSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("ReassignPendingFreshmanApplicationSession: %w", err)
	}
	return app, nil
}

func (r *Repository) GetFreshmanApplicationForUser(
	ctx context.Context,
	applicationID string,
	userID int64,
) (*FreshmanApplication, error) {
	ctx = withDBTable(ctx, "freshman_verification_applications")
	return scanFreshmanApplication(r.db.QueryRow(ctx, `
		SELECT id, user_id, school_id, admission_session_id, status, applicant_name, applicant_name_masked,
		       department_or_major, material_type, provisional_expires_at, reviewed_at, created_at
		FROM freshman_verification_applications
		WHERE id = $1 AND user_id = $2
	`, applicationID, userID))
}

func (r *Repository) CreateFreshmanMaterial(ctx context.Context, material FreshmanMaterialRecord) error {
	ctx = withDBTable(ctx, "freshman_verification_materials")
	_, err := r.db.Exec(ctx, `
		INSERT INTO freshman_verification_materials (
			id, application_id, object_key, content_type, size_bytes, sha256
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, material.ID, material.ApplicationID, material.ObjectKey, material.ContentType, material.SizeBytes, material.SHA256)
	if err != nil {
		return fmt.Errorf("CreateFreshmanMaterial: %w", err)
	}
	return nil
}

func (r *Repository) FreshmanApplicationHasMaterial(ctx context.Context, applicationID string) (bool, error) {
	ctx = withDBTable(ctx, "freshman_verification_materials")
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM freshman_verification_materials
			WHERE application_id = $1
		)
	`, applicationID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("FreshmanApplicationHasMaterial: %w", err)
	}
	return exists, nil
}

func (r *Repository) FreshmanApplicationHasMaterialTx(ctx context.Context, tx pgx.Tx, applicationID string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM freshman_verification_materials
			WHERE application_id = $1
		)
	`, applicationID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("FreshmanApplicationHasMaterialTx: %w", err)
	}
	return exists, nil
}

func (r *Repository) GetActiveFreshmanCameraHandoff(ctx context.Context, applicationID string, now time.Time) (*FreshmanCameraHandoff, error) {
	ctx = withDBTable(ctx, "freshman_camera_handoffs")
	handoff, err := scanFreshmanCameraHandoff(r.db.QueryRow(ctx, freshmanCameraHandoffSelectSQL()+`
		WHERE application_id = $1
		  AND expires_at > $2
		  AND status IN ($3, $4)
		ORDER BY created_at DESC
		LIMIT 1
	`, applicationID, now, FreshmanCameraHandoffPending, FreshmanCameraHandoffUploaded))
	if errors.Is(err, ErrAdmissionCameraHandoffNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetActiveFreshmanCameraHandoff: %w", err)
	}
	return handoff, nil
}

func (r *Repository) ExpirePendingFreshmanCameraHandoffs(ctx context.Context, applicationID string, now time.Time) error {
	ctx = withDBTable(ctx, "freshman_camera_handoffs")
	_, err := r.db.Exec(ctx, `
		UPDATE freshman_camera_handoffs
		SET status = $2, updated_at = NOW()
		WHERE application_id = $1
		  AND status = $3
		  AND expires_at > $4
	`, applicationID, FreshmanCameraHandoffExpired, FreshmanCameraHandoffPending, now)
	if err != nil {
		return fmt.Errorf("ExpirePendingFreshmanCameraHandoffs: %w", err)
	}
	return nil
}

func (r *Repository) CreateFreshmanCameraHandoff(
	ctx context.Context,
	handoff FreshmanCameraHandoff,
	tokenHash string,
) error {
	ctx = withDBTable(ctx, "freshman_camera_handoffs")
	_, err := r.db.Exec(ctx, `
		INSERT INTO freshman_camera_handoffs (
			id, application_id, user_id, token_hash, status, continue_on, expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, handoff.ID, handoff.ApplicationID, handoff.UserID, tokenHash, handoff.Status,
		handoff.ContinueOn, handoff.ExpiresAt)
	if err != nil {
		return fmt.Errorf("CreateFreshmanCameraHandoff: %w", err)
	}
	return nil
}

func (r *Repository) HasActiveFreshmanCameraHandoff(ctx context.Context, applicationID string, now time.Time) (bool, error) {
	ctx = withDBTable(ctx, "freshman_camera_handoffs")
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM freshman_camera_handoffs
			WHERE application_id = $1
			  AND expires_at > $2
			  AND status IN ($3, $4)
		)
	`, applicationID, now, FreshmanCameraHandoffPending, FreshmanCameraHandoffUploaded).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("HasActiveFreshmanCameraHandoff: %w", err)
	}
	return exists, nil
}

func (r *Repository) GetFreshmanCameraHandoffForUser(
	ctx context.Context,
	handoffID string,
	userID int64,
) (*FreshmanCameraHandoff, error) {
	ctx = withDBTable(ctx, "freshman_camera_handoffs")
	return scanFreshmanCameraHandoff(r.db.QueryRow(ctx, freshmanCameraHandoffSelectSQL()+`
		WHERE id = $1 AND user_id = $2
	`, handoffID, userID))
}

func (r *Repository) GetFreshmanCameraHandoffByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*FreshmanCameraHandoff, error) {
	ctx = withDBTable(ctx, "freshman_camera_handoffs")
	return scanFreshmanCameraHandoff(r.db.QueryRow(ctx, freshmanCameraHandoffSelectSQL()+`
		WHERE token_hash = $1
	`, tokenHash))
}

func (r *Repository) GetFreshmanCameraHandoffByID(
	ctx context.Context,
	handoffID string,
) (*FreshmanCameraHandoff, error) {
	ctx = withDBTable(ctx, "freshman_camera_handoffs")
	return scanFreshmanCameraHandoff(r.db.QueryRow(ctx, freshmanCameraHandoffSelectSQL()+`
		WHERE id = $1
	`, handoffID))
}

func (r *Repository) MarkFreshmanCameraHandoffUploaded(ctx context.Context, handoffID string, now time.Time) error {
	ctx = withDBTable(ctx, "freshman_camera_handoffs")
	tag, err := r.db.Exec(ctx, `
		UPDATE freshman_camera_handoffs
		SET status = $2, uploaded_at = COALESCE(uploaded_at, $3), updated_at = NOW()
		WHERE id = $1 AND status = $4
	`, handoffID, FreshmanCameraHandoffUploaded, now, FreshmanCameraHandoffPending)
	if err != nil {
		return fmt.Errorf("MarkFreshmanCameraHandoffUploaded: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAdmissionCameraHandoffLocked
	}
	return nil
}

func (r *Repository) ChooseFreshmanCameraHandoffContinuation(
	ctx context.Context,
	handoffID string,
	continuation FreshmanCameraContinuation,
	now time.Time,
) (*FreshmanCameraHandoff, error) {
	ctx = withDBTable(ctx, "freshman_camera_handoffs")
	return scanFreshmanCameraHandoff(r.db.QueryRow(ctx, `
		UPDATE freshman_camera_handoffs
		SET status = $2, continue_on = $3, chosen_at = $4, updated_at = NOW()
		WHERE id = $1 AND status = $5 AND continue_on IS NULL
		RETURNING id, application_id, user_id, status, continue_on, expires_at, uploaded_at, chosen_at, created_at
	`, handoffID, FreshmanCameraHandoffLocked, continuation, now, FreshmanCameraHandoffUploaded))
}

func (r *Repository) CreateVerificationCredentialTx(ctx context.Context, tx pgx.Tx, credential VerificationCredential) error {
	credentialID, err := id.New()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO user_verification_credentials (
			id, user_id, school_id, kind, subject_hash, subject_display,
			source_application_id, expires_at, verified_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, credentialID, credential.UserID, credential.SchoolID, credential.Kind, credential.SubjectHash,
		credential.SubjectDisplay, credential.SourceApplicationID, credential.ExpiresAt, credential.VerifiedAt)
	if err != nil {
		return fmt.Errorf("CreateVerificationCredentialTx: %w", err)
	}
	return nil
}

func freshmanCameraHandoffSelectSQL() string {
	return `
		SELECT id, application_id, user_id, status, continue_on, expires_at, uploaded_at, chosen_at, created_at
		FROM freshman_camera_handoffs
	`
}

func scanFreshmanCameraHandoff(row pgx.Row) (*FreshmanCameraHandoff, error) {
	var handoff FreshmanCameraHandoff
	var continueOn sql.NullString
	if err := row.Scan(
		&handoff.ID,
		&handoff.ApplicationID,
		&handoff.UserID,
		&handoff.Status,
		&continueOn,
		&handoff.ExpiresAt,
		&handoff.UploadedAt,
		&handoff.ChosenAt,
		&handoff.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAdmissionCameraHandoffNotFound
		}
		return nil, err
	}
	if continueOn.Valid {
		value := FreshmanCameraContinuation(continueOn.String)
		handoff.ContinueOn = &value
	}
	return &handoff, nil
}

func (r *Repository) MarkUserLinkedSessionsVerifiedTx(
	ctx context.Context,
	tx pgx.Tx,
	userID int64,
	schoolID int64,
	now time.Time,
) error {
	_, err := tx.Exec(ctx, `
		UPDATE group_admission_sessions AS gas
		SET status = $2, verified_at = $3, updated_at = NOW()
		FROM group_admission_policies AS gap
		WHERE gas.user_id = $1
		  AND gas.platform = gap.platform
		  AND gas.guild_id = gap.guild_id
		  AND gap.school_id = $4
		  AND gas.status IN ($5, $6)
		  AND gas.cancelled_at IS NULL
		  AND (
		    (gas.status = $5 AND gas.submission_wait_deadline_at >= $3)
		    OR (gas.status = $6 AND (gas.manual_review_deadline_at IS NULL OR gas.manual_review_deadline_at >= $3))
		  )
	`, userID, StatusVerified, now, schoolID, StatusLinked, StatusMaterialSubmitted)
	return err
}

func (r *Repository) GetAdmissionSchoolConfig(ctx context.Context, schoolID int64) (*AdmissionSchoolConfig, error) {
	ctx = withDBTable(ctx, "school_configs")
	var enabled bool
	var raw sql.NullString
	var academicDBTable sql.NullString
	var schoolCode string
	err := r.db.QueryRow(ctx, `
		SELECT sc.enabled, COALESCE(sc.manual_form_fields::text, ''), sc.academic_db_table,
			COALESCE(s.code, sc.school_id::text) AS school_code
		FROM school_configs sc
		LEFT JOIN schools s ON s.id = sc.school_id
		WHERE sc.school_id = $1
	`, schoolID).Scan(&enabled, &raw, &academicDBTable, &schoolCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetAdmissionSchoolConfig: %w", err)
	}
	config := parseAdmissionSchoolConfig(schoolID, schoolCode, enabled, raw.String)
	if academicDBTable.Valid {
		value := strings.TrimSpace(academicDBTable.String)
		if value != "" {
			config.AcademicDBTable = &value
		}
	}
	return &config, nil
}

func (r *Repository) GetAdmissionSchoolConfigByCode(ctx context.Context, schoolCode string) (*AdmissionSchoolConfig, error) {
	ctx = withDBTable(ctx, "school_configs")
	var schoolID int64
	var enabled bool
	var raw sql.NullString
	var academicDBTable sql.NullString
	var resolvedSchoolCode string
	err := r.db.QueryRow(ctx, `
		SELECT sc.school_id, sc.enabled, COALESCE(sc.manual_form_fields::text, ''), sc.academic_db_table,
			COALESCE(s.code, sc.school_id::text) AS school_code
		FROM school_configs sc
		JOIN schools s ON s.id = sc.school_id
		WHERE s.code = $1
	`, strings.TrimSpace(schoolCode)).Scan(&schoolID, &enabled, &raw, &academicDBTable, &resolvedSchoolCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetAdmissionSchoolConfigByCode: %w", err)
	}
	config := parseAdmissionSchoolConfig(schoolID, resolvedSchoolCode, enabled, raw.String)
	if academicDBTable.Valid {
		value := strings.TrimSpace(academicDBTable.String)
		if value != "" {
			config.AcademicDBTable = &value
		}
	}
	return &config, nil
}

func scanFreshmanApplication(row pgx.Row) (*FreshmanApplication, error) {
	var app FreshmanApplication
	if err := row.Scan(
		&app.ID, &app.UserID, &app.SchoolID, &app.AdmissionSessionID, &app.Status, &app.ApplicantName,
		&app.ApplicantNameMasked, &app.DepartmentOrMajor, &app.MaterialType, &app.ProvisionalExpiresAt,
		&app.ReviewedAt, &app.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &app, nil
}

func parseAdmissionSchoolConfig(schoolID int64, schoolCode string, enabled bool, raw string) AdmissionSchoolConfig {
	config := AdmissionSchoolConfig{
		SchoolID:        schoolID,
		SchoolCode:      strings.TrimSpace(schoolCode),
		Enabled:         enabled,
		FreshmanEnabled: true,
	}
	settings := schoolauth.ParseAdmissionSettings([]byte(raw))
	config.EmailDomains = settings.EmailDomains
	config.SSOLoginURL = settings.SSOLoginURL
	if settings.EmailIdentityPolicy != nil {
		config.EmailIdentityPolicy = &SchoolEmailIdentityPolicy{
			Type:                 settings.EmailIdentityPolicy.Type,
			StudentIDEmailDomain: settings.EmailIdentityPolicy.StudentIDEmailDomain,
			RequireStudentName:   settings.EmailIdentityPolicy.RequireStudentName,
		}
	}
	return config
}

type FreshmanMaterialRecord struct {
	ID            string
	ApplicationID string
	ObjectKey     string
	ContentType   string
	SizeBytes     int64
	SHA256        string
}

type VerificationCredential struct {
	UserID              int64
	SchoolID            int64
	Kind                VerificationCredentialKind
	SubjectHash         string
	SubjectDisplay      string
	SourceApplicationID *string
	ExpiresAt           *time.Time
	VerifiedAt          time.Time
}
