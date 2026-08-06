package studentverification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/StuHelper/StuHelper/server/internal/pkg/db"
)

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository {
	if database == nil {
		panic("studentverification.NewRepository: database must not be nil")
	}
	return &Repository{db: database}
}

func (r *Repository) WithTx(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
	return r.db.WithTx(ctx, fn)
}

func withTable(ctx context.Context, table string) context.Context {
	return db.WithTableHint(ctx, table)
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && (constraint == "" || pgErr.ConstraintName == constraint)
}

func (r *Repository) ListAvailableSchools(ctx context.Context) ([]VerificationSchool, error) {
	ctx = withTable(ctx, "school_verification_methods")
	rows, err := r.db.Query(ctx, `
		SELECT s.id, s.code, s.name, COALESCE(s.location, ''),
		       m.method, m.display_name, m.description, m.adapter_id, m.public_form_schema,
		       m.privacy_notice_version, m.privacy_notice, m.health_status
		FROM school_verification_profiles p
		JOIN schools s ON s.id = p.school_id
		JOIN school_verification_methods m ON m.school_id = p.school_id
		WHERE p.enabled
		  AND p.validation_status = 'valid'
		  AND m.enabled
		  AND m.validation_status = 'valid'
		  AND EXISTS (
		      SELECT 1
		      FROM school_verification_methods healthy
		      WHERE healthy.school_id = p.school_id
		        AND healthy.enabled
		        AND healthy.validation_status = 'valid'
		        AND healthy.health_status IN ('healthy', 'degraded')
		  )
		ORDER BY s.name, m.method
	`)
	if err != nil {
		return nil, fmt.Errorf("list verification schools: %w", err)
	}
	defer rows.Close()

	result := make([]VerificationSchool, 0)
	bySchool := make(map[int64]int)
	for rows.Next() {
		var (
			schoolID                     int64
			schoolCode, schoolName       string
			location                     string
			method, display, description string
			adapterID                    string
			formRaw, noticeRaw           []byte
			noticeVersion                *string
			healthStatus                 string
		)
		if err := rows.Scan(
			&schoolID, &schoolCode, &schoolName, &location,
			&method, &display, &description, &adapterID, &formRaw,
			&noticeVersion, &noticeRaw, &healthStatus,
		); err != nil {
			return nil, fmt.Errorf("scan verification school: %w", err)
		}

		capability, err := decodeMethodCapability(
			Method(method), display, description, adapterID, formRaw, noticeVersion, noticeRaw, healthStatus,
		)
		if err != nil {
			return nil, fmt.Errorf("decode verification method %d/%s: %w", schoolID, method, err)
		}
		index, ok := bySchool[schoolID]
		if !ok {
			index = len(result)
			bySchool[schoolID] = index
			result = append(result, VerificationSchool{
				ID: schoolID, Code: schoolCode, Name: schoolName, Location: location,
				Methods: make([]MethodCapability, 0, 5),
			})
		}
		result[index].Methods = append(result[index].Methods, capability)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate verification schools: %w", err)
	}
	return result, nil
}

func decodeMethodCapability(
	method Method,
	display string,
	description string,
	adapterID string,
	formRaw []byte,
	noticeVersion *string,
	noticeRaw []byte,
	healthStatus string,
) (MethodCapability, error) {
	var form struct {
		Fields []VerificationField `json:"fields"`
	}
	if len(formRaw) > 0 {
		if err := json.Unmarshal(formRaw, &form); err != nil {
			return MethodCapability{}, err
		}
	}
	if form.Fields == nil {
		form.Fields = []VerificationField{}
	}

	availability := "temporarily_unavailable"
	unavailableCode := "method_temporarily_unavailable"
	if healthStatus == "healthy" || healthStatus == "degraded" {
		availability = "available"
		unavailableCode = ""
	}
	var notice *PrivacyNotice
	if noticeVersion != nil && len(noticeRaw) > 0 && string(noticeRaw) != "{}" {
		var parsed PrivacyNotice
		if err := json.Unmarshal(noticeRaw, &parsed); err != nil {
			return MethodCapability{}, err
		}
		parsed.Version = *noticeVersion
		if parsed.DataCategories == nil {
			parsed.DataCategories = []string{}
		}
		notice = &parsed
	}
	return MethodCapability{
		Method: method, DisplayName: display, Description: description,
		Availability: availability, UnavailableCode: unavailableCode,
		FormFields: form.Fields, PrivacyNotice: notice, adapterID: adapterID,
	}, nil
}

func (r *Repository) GetMethodConfig(ctx context.Context, schoolCode string, method Method) (*MethodConfig, error) {
	ctx = withTable(ctx, "school_verification_methods")
	return scanMethodConfig(r.db.QueryRow(ctx, methodConfigSQL(false), schoolCode, string(method)))
}

func (r *Repository) GetMethodConfigTx(ctx context.Context, tx pgx.Tx, schoolCode string, method Method) (*MethodConfig, error) {
	return scanMethodConfig(tx.QueryRow(ctx, methodConfigSQL(true), schoolCode, string(method)))
}

func methodConfigSQL(forShare bool) string {
	query := `
		SELECT s.id, s.code, s.name, p.adapter_id, m.method, m.adapter_id, m.adapter_version,
		       m.roster_dependency, COALESCE(m.privacy_notice_version, ''),
		       m.public_form_schema, p.student_id_policy, p.enrollment_policy,
		       m.conditional_policy, m.risk_policy,
		       p.email_domains, m.connector_operation_key,
		       p.snapshot_hard_expiry_seconds, m.credential_ttl_seconds,
		       m.health_status
		FROM school_verification_profiles p
		JOIN schools s ON s.id = p.school_id
		JOIN school_verification_methods m ON m.school_id = p.school_id
		WHERE s.code = $1
		  AND m.method = $2
		  AND p.enabled
		  AND p.validation_status = 'valid'
		  AND m.enabled
		  AND m.validation_status = 'valid'
	`
	if forShare {
		query += " FOR SHARE OF p, m"
	}
	return query
}

func scanMethodConfig(row rowScanner) (*MethodConfig, error) {
	var (
		config               MethodConfig
		publicFormSchema     []byte
		studentIDPolicy      []byte
		enrollmentPolicy     []byte
		conditionalPolicy    []byte
		riskPolicy           []byte
		hardExpirySeconds    int
		credentialTTLSeconds *int
	)
	err := row.Scan(
		&config.SchoolID, &config.SchoolCode, &config.SchoolName, &config.SchoolAdapterID,
		&config.Method,
		&config.AdapterID, &config.AdapterVersion, &config.RosterDependency,
		&config.PrivacyNoticeVersion, &publicFormSchema, &studentIDPolicy,
		&enrollmentPolicy, &conditionalPolicy,
		&riskPolicy, &config.EmailDomains, &config.ConnectorOperation,
		&hardExpirySeconds,
		&credentialTTLSeconds, &config.HealthStatus,
	)
	if err != nil {
		return nil, err
	}
	config.PublicFormSchema = append(json.RawMessage(nil), publicFormSchema...)
	config.StudentIDPolicy = append(json.RawMessage(nil), studentIDPolicy...)
	config.EnrollmentPolicy = append(json.RawMessage(nil), enrollmentPolicy...)
	config.ConditionalPolicy = append(json.RawMessage(nil), conditionalPolicy...)
	config.RiskPolicy = append(json.RawMessage(nil), riskPolicy...)
	config.SnapshotHardExpiry = time.Duration(hardExpirySeconds) * time.Second
	if credentialTTLSeconds != nil {
		ttl := time.Duration(*credentialTTLSeconds) * time.Second
		config.CredentialTTL = &ttl
	}
	return &config, nil
}

func (r *Repository) SchoolHasAvailableMethod(ctx context.Context, schoolCode string) (bool, error) {
	ctx = withTable(ctx, "school_verification_methods")
	var available bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
		    SELECT 1
		    FROM school_verification_profiles p
		    JOIN schools s ON s.id = p.school_id
		    JOIN school_verification_methods m ON m.school_id = p.school_id
		    WHERE s.code = $1
		      AND p.enabled
		      AND p.validation_status = 'valid'
		      AND m.enabled
		      AND m.validation_status = 'valid'
		      AND m.health_status IN ('healthy', 'degraded')
		)
	`, schoolCode).Scan(&available)
	return available, err
}

func (r *Repository) CreateApplication(ctx context.Context, app Application, continuationHash *string, continuationExpiry *time.Time) error {
	ctx = withTable(ctx, "student_verification_applications")
	_, err := r.db.Exec(ctx, `
		INSERT INTO student_verification_applications (
		    id, user_id, school_id, status, continuation_hash,
		    continuation_expires_at, expires_at, revision, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 1, $8, $8)
	`, app.ID, app.UserID, app.SchoolID, app.Status, continuationHash, continuationExpiry, app.ExpiresAt, app.CreatedAt)
	if err != nil {
		return fmt.Errorf("create verification application: %w", err)
	}
	return nil
}

func (r *Repository) GetActiveApplication(ctx context.Context, userID, schoolID int64) (*Application, error) {
	return r.getApplication(ctx, `
		WHERE a.user_id = $1 AND a.school_id = $2
		  AND a.status IN ('created', 'in_progress', 'pending_manual_review')
		ORDER BY a.created_at DESC
		LIMIT 1
	`, userID, schoolID)
}

func (r *Repository) GetApplication(ctx context.Context, id string, userID int64) (*Application, error) {
	return r.getApplication(ctx, `WHERE a.id = $1 AND a.user_id = $2`, id, userID)
}

func (r *Repository) getApplication(ctx context.Context, clause string, args ...any) (*Application, error) {
	ctx = withTable(ctx, "student_verification_applications")
	return scanApplication(r.db.QueryRow(ctx, applicationSelectSQL()+clause, args...))
}

func applicationSelectSQL() string {
	return `
		SELECT a.id, a.user_id, a.school_id, s.code, s.name, a.status,
		       a.current_method, a.privacy_notice_version, a.consented_at,
		       a.terminal_code, a.revision, a.created_at, a.updated_at,
		       a.expires_at, a.completed_at
		FROM student_verification_applications a
		JOIN schools s ON s.id = a.school_id
	`
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanApplication(row rowScanner) (*Application, error) {
	var (
		app           Application
		status        string
		currentMethod *string
	)
	err := row.Scan(
		&app.ID, &app.UserID, &app.SchoolID, &app.SchoolCode, &app.SchoolName,
		&status, &currentMethod, &app.PrivacyNoticeVersion, &app.ConsentedAt,
		&app.TerminalCode, &app.Revision, &app.CreatedAt, &app.UpdatedAt,
		&app.ExpiresAt, &app.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	app.Status = ApplicationStatus(status)
	if currentMethod != nil {
		method := Method(*currentMethod)
		app.CurrentMethod = &method
	}
	return &app, nil
}

func (r *Repository) GetApplicationForUpdateTx(ctx context.Context, tx pgx.Tx, id string, userID int64) (*Application, error) {
	return scanApplication(tx.QueryRow(ctx, applicationSelectSQL()+`
		WHERE a.id = $1 AND a.user_id = $2
		FOR UPDATE OF a
	`, id, userID))
}

func (r *Repository) GetActiveRosterRecord(ctx context.Context, schoolID int64, studentIDHash string) (*RosterRecord, error) {
	ctx = withTable(ctx, "student_roster_records")
	return scanRosterRecord(r.db.QueryRow(ctx, activeRosterRecordSQL(false), schoolID, studentIDHash))
}

func (r *Repository) GetActiveRosterState(ctx context.Context, schoolID int64) (*RosterSnapshotState, error) {
	ctx = withTable(ctx, "student_roster_active")
	var state RosterSnapshotState
	err := r.db.QueryRow(ctx, `
		SELECT active.snapshot_id, active.activation_revision, snapshot.source_cutoff_at
		FROM academic.student_roster_active active
		JOIN academic.student_roster_snapshots snapshot
		  ON snapshot.id = active.snapshot_id AND snapshot.school_id = active.school_id
		WHERE active.school_id = $1 AND snapshot.status = 'active'
	`, schoolID).Scan(&state.SnapshotID, &state.SnapshotRevision, &state.SourceCutoffAt)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *Repository) GetActiveRosterRecordTx(ctx context.Context, tx pgx.Tx, schoolID int64, studentIDHash string) (*RosterRecord, error) {
	return scanRosterRecord(tx.QueryRow(ctx, activeRosterRecordSQL(true), schoolID, studentIDHash))
}

func activeRosterRecordSQL(forShare bool) string {
	query := `
		SELECT active.snapshot_id, active.activation_revision, snapshot.source_cutoff_at,
		       record.student_id_hash, record.name_hash, record.document_type,
		       record.document_number_hash, record.eligibility_status,
		       record.eligibility_code, record.encryption_key_version,
		       record.hmac_key_version
		FROM academic.student_roster_active active
		JOIN academic.student_roster_snapshots snapshot
		  ON snapshot.id = active.snapshot_id AND snapshot.school_id = active.school_id
		JOIN academic.student_roster_records record
		  ON record.snapshot_id = active.snapshot_id AND record.school_id = active.school_id
		WHERE active.school_id = $1
		  AND snapshot.status = 'active'
		  AND record.student_id_hash = $2
	`
	if forShare {
		query += " FOR SHARE OF active, snapshot, record"
	}
	return query
}

func scanRosterRecord(row rowScanner) (*RosterRecord, error) {
	var record RosterRecord
	err := row.Scan(
		&record.SnapshotID, &record.SnapshotRevision, &record.SourceCutoffAt,
		&record.StudentIDHash, &record.NameHash, &record.DocumentType,
		&record.DocumentNumberHash, &record.EligibilityStatus,
		&record.EligibilityCode, &record.EncryptionKeyVersion,
		&record.HMACKeyVersion,
	)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *Repository) RecordAttempt(ctx context.Context, userID int64, applicationID string, result attemptResult, now time.Time) error {
	expired := false
	err := r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		app, err := r.GetApplicationForUpdateTx(ctx, tx, applicationID, userID)
		if err != nil {
			return err
		}
		if app.Status == ApplicationApproved || app.Status == ApplicationRejected || app.Status == ApplicationCancelled {
			return ErrApplicationState
		}
		if !app.ExpiresAt.After(now) {
			expired = true
			return r.expireApplicationTx(ctx, tx, app, now)
		}
		return r.insertAttemptAndProgressTx(ctx, tx, app, result, now)
	})
	if err != nil {
		return err
	}
	if expired {
		return ErrApplicationExpired
	}
	return nil
}

func (r *Repository) ExpireApplication(ctx context.Context, applicationID string, userID int64, now time.Time) (*Application, error) {
	err := r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		app, err := r.GetApplicationForUpdateTx(ctx, tx, applicationID, userID)
		if err != nil {
			return err
		}
		if app.Status == ApplicationApproved || app.Status == ApplicationRejected ||
			app.Status == ApplicationCancelled || app.Status == ApplicationExpired {
			return nil
		}
		if app.ExpiresAt.After(now) {
			return nil
		}
		return r.expireApplicationTx(ctx, tx, app, now)
	})
	if err != nil {
		return nil, err
	}
	return r.GetApplication(ctx, applicationID, userID)
}

// CancelApplication atomically closes a non-terminal verification application
// and every short-lived continuation that belongs to it. Repeating the request
// for an already-cancelled application is intentionally idempotent; other
// terminal states remain conflicts so a client cannot rewrite history.
func (r *Repository) CancelApplication(ctx context.Context, applicationID string, userID int64, now time.Time) (*Application, error) {
	err := r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		app, err := r.GetApplicationForUpdateTx(ctx, tx, applicationID, userID)
		if err != nil {
			return err
		}
		if app.Status == ApplicationCancelled {
			return nil
		}
		if !applicationIsNonTerminal(app) {
			return ErrApplicationState
		}

		if _, err := tx.Exec(ctx, `
			UPDATE student_email_inbound_challenges
			SET status = 'cancelled', updated_at = $2
			WHERE application_id = $1 AND status = 'waiting'
		`, app.ID, now); err != nil {
			return fmt.Errorf("cancel inbound email challenge: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE student_manual_camera_handoffs handoff
			SET status = CASE WHEN handoff.status = 'uploaded' THEN 'locked' ELSE 'expired' END,
			    token_enc = NULL, encryption_key_version = NULL, updated_at = $2
			FROM student_manual_review_cases review_case
			WHERE review_case.application_id = $1
			  AND handoff.case_id = review_case.id
			  AND handoff.status IN ('pending', 'uploaded')
		`, app.ID, now); err != nil {
			return fmt.Errorf("cancel manual camera handoff: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE student_manual_review_cases
			SET status = 'cancelled', submitted_at = COALESCE(submitted_at, $2),
			    revision = revision + 1, updated_at = $2
			WHERE application_id = $1
			  AND status IN ('draft', 'pending', 'supplement_required')
		`, app.ID, now); err != nil {
			return fmt.Errorf("cancel manual review case: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE student_verification_applications
			SET status = 'cancelled', terminal_code = 'user_cancelled',
			    completed_at = $2, revision = revision + 1, updated_at = $2
			WHERE id = $1
		`, app.ID, now); err != nil {
			return fmt.Errorf("cancel verification application: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetApplication(ctx, applicationID, userID)
}

func (r *Repository) expireApplicationTx(ctx context.Context, tx pgx.Tx, app *Application, now time.Time) error {
	_, err := tx.Exec(ctx, `
		UPDATE student_verification_applications
		SET status = 'expired', terminal_code = 'application_expired',
		    completed_at = $2, revision = revision + 1, updated_at = $2
		WHERE id = $1
	`, app.ID, now)
	if err != nil {
		return fmt.Errorf("expire verification application: %w", err)
	}
	return nil
}

func (r *Repository) insertAttemptAndProgressTx(
	ctx context.Context,
	tx pgx.Tx,
	app *Application,
	result attemptResult,
	now time.Time,
) error {
	var attemptNumber int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(attempt_number), 0) + 1
		FROM student_verification_attempts
		WHERE application_id = $1
	`, app.ID).Scan(&attemptNumber); err != nil {
		return fmt.Errorf("allocate verification attempt: %w", err)
	}
	attemptID, err := newID()
	if err != nil {
		return err
	}
	completedAt := now
	_, err = tx.Exec(ctx, `
		INSERT INTO student_verification_attempts (
		    id, application_id, attempt_number, method, status, result_code,
		    adapter_id, adapter_version, evidence_metadata,
		    started_roster_snapshot_id, started_roster_revision,
		    started_at, completed_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, '{}'::jsonb, $9, $10, $11, $12, $11, $12)
	`, attemptID, app.ID, attemptNumber, result.Method, result.Status, result.ResultCode,
		result.AdapterID, result.AdapterVersion, result.SnapshotID, result.SnapshotRevision,
		now, completedAt)
	if err != nil {
		return fmt.Errorf("insert verification attempt: %w", err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE student_verification_applications
		SET status = 'in_progress', current_method = $2,
		    privacy_notice_version = COALESCE($3, privacy_notice_version),
		    consented_at = COALESCE($4, consented_at),
		    revision = revision + 1, updated_at = $5
		WHERE id = $1
	`, app.ID, result.Method, result.PrivacyNotice, result.ConsentedAt, now)
	if err != nil {
		return fmt.Errorf("progress verification application: %w", err)
	}
	return nil
}

func (r *Repository) LockEnrollmentSubjectTx(ctx context.Context, tx pgx.Tx, schoolID int64, subjectHash string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, $2))`, subjectHash, schoolID)
	return err
}

func (r *Repository) GetActiveEnrollmentSubjectTx(ctx context.Context, tx pgx.Tx, schoolID int64, subjectHash string) (*EnrollmentSubject, error) {
	var subject EnrollmentSubject
	err := tx.QueryRow(ctx, `
		SELECT id, user_id, school_id, subject_hash, student_id_hash,
		       student_id_display, binding_status
		FROM student_enrollment_subjects
		WHERE school_id = $1 AND subject_hash = $2
		  AND binding_status IN ('active', 'review_required')
		FOR UPDATE
	`, schoolID, subjectHash).Scan(
		&subject.ID, &subject.UserID, &subject.SchoolID, &subject.SubjectHash,
		&subject.StudentIDHash, &subject.StudentDisplay, &subject.BindingStatus,
	)
	if err != nil {
		return nil, err
	}
	return &subject, nil
}

func (r *Repository) CreateEnrollmentSubjectTx(
	ctx context.Context,
	tx pgx.Tx,
	subject EnrollmentSubject,
	personLinkHash *string,
	method Method,
	snapshotID *string,
	snapshotRevision *int64,
	now time.Time,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO student_enrollment_subjects (
		    id, user_id, school_id, subject_hash, person_link_hash,
		    student_id_hash, student_id_display, source_method,
		    binding_status, roster_snapshot_id, roster_snapshot_revision,
		    activated_at, revision, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'active', $9, $10, $11, 1, $11, $11)
	`, subject.ID, subject.UserID, subject.SchoolID, subject.SubjectHash, personLinkHash,
		subject.StudentIDHash, subject.StudentDisplay, method, snapshotID, snapshotRevision, now)
	if err != nil {
		return fmt.Errorf("create enrollment subject: %w", err)
	}
	return nil
}

func (r *Repository) CreateSubjectConflictTx(
	ctx context.Context,
	tx pgx.Tx,
	app *Application,
	subjectHash string,
	incumbentUserID int64,
	now time.Time,
) error {
	conflictID, err := newID()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO student_subject_conflicts (
		    id, school_id, subject_hash, claimant_user_id, incumbent_user_id,
		    application_id, status, reason_code, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, 'open', 'active_subject_owned_by_other_user', $7, $7)
		ON CONFLICT (school_id, subject_hash, claimant_user_id)
		    WHERE status IN ('open', 'under_review')
		DO UPDATE SET updated_at = EXCLUDED.updated_at
	`, conflictID, app.SchoolID, subjectHash, app.UserID, incumbentUserID, app.ID, now)
	return err
}

func (r *Repository) CreateCredentialTx(
	ctx context.Context,
	tx pgx.Tx,
	credential Credential,
	applicationID string,
	adapterID string,
	adapterVersion string,
	snapshotID *string,
	snapshotRevision *int64,
	metadata json.RawMessage,
	now time.Time,
) error {
	assurance := credential.Assurance
	if assurance == "" {
		assurance = "standard"
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO user_verification_credentials (
		    id, user_id, school_id, kind, subject_hash, subject_display,
		    verification_application_id, enrollment_subject_id, status,
		    credential_class, roster_dependency, assurance,
		    adapter_id, adapter_version, roster_snapshot_id,
		    roster_snapshot_revision, metadata, verified_at, expires_at,
		    activated_at, revision, last_evaluated_at, created_at, updated_at
		)
		VALUES (
		    $1, $2, $3, $4, $5, $6, $7, $8, 'active', $9, $10, $11,
		    $12, $13, $14, $15, $16, $17, $18, $17, 1, $17, $17, $17
		)
	`, credential.ID, credential.UserID, credential.SchoolID, credential.Method,
		credential.SubjectHash, credential.SubjectDisplay, applicationID, credential.EnrollmentID,
		credential.CredentialClass, credential.RosterDependency, assurance,
		adapterID, adapterVersion, snapshotID, snapshotRevision, metadata, now,
		credential.ExpiresAt)
	if err != nil {
		return fmt.Errorf("create verification credential: %w", err)
	}
	return nil
}

func (r *Repository) CompleteApplicationTx(
	ctx context.Context,
	tx pgx.Tx,
	app *Application,
	result attemptResult,
	now time.Time,
) error {
	if err := r.insertAttemptAndProgressTx(ctx, tx, app, result, now); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		UPDATE student_verification_applications
		SET status = 'approved', current_method = $2,
		    terminal_code = NULL, completed_at = $3,
		    revision = revision + 1, updated_at = $3
		WHERE id = $1
	`, app.ID, result.Method, now)
	return err
}

func (r *Repository) BumpEligibilityRevisionTx(ctx context.Context, tx pgx.Tx, userID, schoolID int64, reason string, now time.Time) error {
	eventID, err := newID()
	if err != nil {
		return err
	}
	var revision int64
	err = tx.QueryRow(ctx, `
		INSERT INTO student_eligibility_revisions (user_id, school_id, revision, reason_code, updated_at)
		VALUES ($1, $2, 1, $3, $4)
		ON CONFLICT (user_id, school_id) DO UPDATE
		SET revision = student_eligibility_revisions.revision + 1,
		    reason_code = EXCLUDED.reason_code,
		    updated_at = EXCLUDED.updated_at
		RETURNING revision
	`, userID, schoolID, reason, now).Scan(&revision)
	if err != nil {
		return fmt.Errorf("bump student eligibility revision: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO student_verification_event_outbox (
		    event_id, aggregate_type, aggregate_id, user_id, school_id,
		    event_type, aggregate_revision, payload, available_at,
		    created_at, updated_at
		)
		VALUES (
		    $1, 'student_eligibility', format('%s:%s', $2::bigint, $3::bigint),
		    $2::bigint, $3::bigint, 'student_eligibility.changed', $4,
		    jsonb_build_object('reasonCode', $5::text), $6, $6, $6
		)
	`, eventID, userID, schoolID, revision, reason, now)
	if err != nil {
		return fmt.Errorf("enqueue student eligibility event: %w", err)
	}
	return nil
}

func (r *Repository) GetCredentialByApplication(ctx context.Context, applicationID string, userID int64) (*Credential, error) {
	ctx = withTable(ctx, "user_verification_credentials")
	return scanCredential(r.db.QueryRow(ctx, credentialSelectSQL()+`
		WHERE c.verification_application_id = $1 AND c.user_id = $2
		ORDER BY c.verified_at DESC LIMIT 1
	`, applicationID, userID))
}

func (r *Repository) ListCredentials(ctx context.Context, userID int64) ([]Credential, error) {
	ctx = withTable(ctx, "user_verification_credentials")
	rows, err := r.db.Query(ctx, credentialSelectSQL()+`
		WHERE c.user_id = $1
		ORDER BY c.verified_at DESC, c.id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	credentials := make([]Credential, 0)
	for rows.Next() {
		credential, err := scanCredential(rows)
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, *credential)
	}
	return credentials, rows.Err()
}

func credentialSelectSQL() string {
	return `
		SELECT c.id, c.user_id, c.school_id, s.code, s.name, c.kind,
		       c.status, c.credential_class, c.subject_hash, c.subject_display,
		       c.enrollment_subject_id, c.roster_dependency, c.verified_at,
		       c.expires_at, c.review_required_at, c.revision
		FROM user_verification_credentials c
		JOIN schools s ON s.id = c.school_id
	`
}

func scanCredential(row rowScanner) (*Credential, error) {
	var (
		credential     Credential
		method, status string
	)
	err := row.Scan(
		&credential.ID, &credential.UserID, &credential.SchoolID,
		&credential.SchoolCode, &credential.SchoolName, &method, &status,
		&credential.CredentialClass, &credential.SubjectHash, &credential.SubjectDisplay,
		&credential.EnrollmentID, &credential.RosterDependency, &credential.VerifiedAt,
		&credential.ExpiresAt, &credential.ReviewRequiredAt, &credential.Revision,
	)
	if err != nil {
		return nil, err
	}
	credential.Method = Method(method)
	credential.Status = CredentialStatus(status)
	return &credential, nil
}

func (r *Repository) RevokeCredential(ctx context.Context, credentialID string, userID int64, now time.Time) (*Credential, error) {
	var schoolID int64
	err := r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var currentStatus string
		if err := tx.QueryRow(ctx, `
			SELECT school_id, status
			FROM user_verification_credentials
			WHERE id = $1 AND user_id = $2
			FOR UPDATE
		`, credentialID, userID).Scan(&schoolID, &currentStatus); err != nil {
			return err
		}
		if currentStatus == string(CredentialRevoked) || currentStatus == string(CredentialExpired) {
			return ErrCredentialState
		}
		_, err := tx.Exec(ctx, `
			UPDATE user_verification_credentials
			SET status = 'revoked', revoked_at = $3,
			    revoked_reason = 'user_requested', revision = revision + 1,
			    last_evaluated_at = $3, updated_at = $3
			WHERE id = $1 AND user_id = $2
		`, credentialID, userID, now)
		if err != nil {
			return err
		}
		return r.BumpEligibilityRevisionTx(ctx, tx, userID, schoolID, "credential_revoked", now)
	})
	if err != nil {
		return nil, err
	}
	return scanCredential(r.db.QueryRow(withTable(ctx, "user_verification_credentials"), credentialSelectSQL()+`
		WHERE c.id = $1 AND c.user_id = $2
	`, credentialID, userID))
}

func (r *Repository) ListQualifyingCredentials(ctx context.Context, userID, schoolID int64, now time.Time) ([]Credential, error) {
	ctx = withTable(ctx, "user_verification_credentials")
	rows, err := r.db.Query(ctx, credentialSelectSQL()+`
		WHERE c.user_id = $1
		  AND c.school_id = $2
		  AND c.verification_application_id IS NOT NULL
		  AND c.status = 'active'
		  AND c.revoked_at IS NULL
		  AND (c.expires_at IS NULL OR c.expires_at > $3)
		  AND NOT EXISTS (
		      SELECT 1 FROM student_subject_conflicts conflict
		      WHERE conflict.school_id = c.school_id
		        AND conflict.subject_hash = c.subject_hash
		        AND conflict.status IN ('open', 'under_review')
		  )
		  AND (
		      c.roster_dependency = 'independent'
		      OR (
		          c.roster_dependency = 'required'
		          AND c.enrollment_subject_id IS NOT NULL
		          AND EXISTS (
		              SELECT 1
		              FROM student_enrollment_subjects subject
		              JOIN academic.student_roster_active active
		                ON active.school_id = subject.school_id
		              JOIN academic.student_roster_snapshots snapshot
		                ON snapshot.id = active.snapshot_id
		               AND snapshot.school_id = active.school_id
		              JOIN academic.student_roster_records record
		                ON record.school_id = active.school_id
		               AND record.snapshot_id = active.snapshot_id
		               AND record.student_id_hash = subject.student_id_hash
		              JOIN school_verification_profiles profile
		                ON profile.school_id = subject.school_id
		              WHERE subject.id = c.enrollment_subject_id
		                AND subject.binding_status = 'active'
		                AND record.eligibility_status = 'eligible'
		                AND snapshot.status = 'active'
		                AND snapshot.source_cutoff_at <= $3 + INTERVAL '5 minutes'
		                AND snapshot.source_cutoff_at +
		                    make_interval(secs => profile.snapshot_hard_expiry_seconds) >= $3
		          )
		      )
		      OR (
		          c.roster_dependency = 'conditional'
		          AND c.metadata @> '{"qualification_satisfied": true}'::jsonb
		      )
		  )
		ORDER BY c.verified_at DESC, c.id DESC
	`, userID, schoolID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	credentials := make([]Credential, 0)
	for rows.Next() {
		credential, err := scanCredential(rows)
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, *credential)
	}
	return credentials, rows.Err()
}

func (r *Repository) GetSchoolByCode(ctx context.Context, schoolCode string) (int64, string, error) {
	ctx = withTable(ctx, "schools")
	var id int64
	var name string
	err := r.db.QueryRow(ctx, `SELECT id, name FROM schools WHERE code = $1`, schoolCode).Scan(&id, &name)
	return id, name, err
}

func (r *Repository) GetSchoolCodeByID(ctx context.Context, schoolID int64) (string, error) {
	ctx = withTable(ctx, "schools")
	var schoolCode string
	err := r.db.QueryRow(ctx, `
		SELECT code
		FROM schools
		WHERE id = $1
	`, schoolID).Scan(&schoolCode)
	if err != nil {
		return "", fmt.Errorf("get verification school code: %w", err)
	}
	return schoolCode, nil
}

func (r *Repository) GetEligibilityRevision(ctx context.Context, userID, schoolID int64) (int64, error) {
	ctx = withTable(ctx, "student_eligibility_revisions")
	var revision int64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE((
		    SELECT revision FROM student_eligibility_revisions
		    WHERE user_id = $1 AND school_id = $2
		), 1)
	`, userID, schoolID).Scan(&revision)
	return revision, err
}

func (r *Repository) GetCurrentStudentStatus(ctx context.Context, userID int64) (*CurrentStudentStatus, error) {
	ctx = withTable(ctx, "current_student_qualifying_credentials")
	var result CurrentStudentStatus
	var schoolID int64
	var credentialClass string
	err := r.db.QueryRow(ctx, `
		SELECT school_id, credential_class
		FROM current_student_qualifying_credentials
		WHERE user_id = $1
		ORDER BY
		    CASE credential_class WHEN 'formal_student' THEN 0 ELSE 1 END,
		    verified_at DESC,
		    id DESC
		LIMIT 1
	`, userID).Scan(&schoolID, &credentialClass)
	if errors.Is(err, pgx.ErrNoRows) {
		return &result, nil
	}
	if err != nil {
		return nil, err
	}
	result.Eligible = true
	result.SchoolID = &schoolID
	result.CredentialClass = &credentialClass
	return &result, nil
}
