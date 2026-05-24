package admission

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

func (r *Repository) GetLinkedSessionByUserID(ctx context.Context, userID int64) (*AdmissionSession, error) {
	ctx = withDBTable(ctx, "group_admission_sessions")
	query := "SELECT " + admissionSessionColumns + `
		FROM group_admission_sessions
		WHERE user_id = $1 AND status = $2
		ORDER BY updated_at DESC
		LIMIT 1`
	session, err := scanAdmissionSession(r.db.QueryRow(ctx, query, userID, StatusLinked))
	if errors.Is(err, ErrAdmissionTokenNotFound) {
		return nil, nil
	}
	return session, err
}

func (r *Repository) GetLinkedSessionByUserIDTx(ctx context.Context, tx pgx.Tx, userID int64) (*AdmissionSession, error) {
	query := "SELECT " + admissionSessionColumns + `
		FROM group_admission_sessions
		WHERE user_id = $1 AND status = $2
		ORDER BY updated_at DESC
		LIMIT 1
		FOR UPDATE`
	session, err := scanAdmissionSession(tx.QueryRow(ctx, query, userID, StatusLinked))
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

func (r *Repository) MarkUserLinkedSessionsVerifiedTx(ctx context.Context, tx pgx.Tx, userID int64, now time.Time) error {
	_, err := tx.Exec(ctx, `
		UPDATE group_admission_sessions
		SET status = $2, verified_at = $3, updated_at = NOW()
		WHERE user_id = $1 AND status IN ($4, $5)
	`, userID, StatusVerified, now, StatusLinked, StatusMaterialSubmitted)
	return err
}

func (r *Repository) GetAdmissionSchoolConfig(ctx context.Context, schoolID int64) (*AdmissionSchoolConfig, error) {
	ctx = withDBTable(ctx, "school_configs")
	var enabled bool
	var raw sql.NullString
	err := r.db.QueryRow(ctx, `
		SELECT enabled, COALESCE(manual_form_fields::text, '')
		FROM school_configs
		WHERE school_id = $1
	`, schoolID).Scan(&enabled, &raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetAdmissionSchoolConfig: %w", err)
	}
	config := parseAdmissionSchoolConfig(schoolID, enabled, raw.String)
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

func parseAdmissionSchoolConfig(schoolID int64, enabled bool, raw string) AdmissionSchoolConfig {
	config := AdmissionSchoolConfig{SchoolID: schoolID, Enabled: enabled, FreshmanEnabled: true}
	var envelope struct {
		Admission struct {
			EmailDomains []string `json:"emailDomains"`
			SSOLoginURL  string   `json:"ssoLoginURL"`
		} `json:"admission"`
	}
	if strings.TrimSpace(raw) == "" || json.Unmarshal([]byte(raw), &envelope) != nil {
		return config
	}
	config.EmailDomains = normalizeEmailDomains(envelope.Admission.EmailDomains)
	config.SSOLoginURL = strings.TrimSpace(envelope.Admission.SSOLoginURL)
	return config
}

func normalizeEmailDomains(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		domain := strings.ToLower(strings.TrimSpace(value))
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		result = append(result, domain)
	}
	return result
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
