package studentverification

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetManualReviewCaseForUser(
	ctx context.Context,
	applicationID string,
	userID int64,
) (*ManualReviewCase, error) {
	ctx = withTable(ctx, "student_manual_review_cases")
	reviewCase, err := scanManualReviewCase(r.db.QueryRow(ctx, manualReviewCaseSelectSQL()+`
		WHERE review_case.application_id = $1 AND review_case.user_id = $2
	`, applicationID, userID))
	if err != nil {
		return nil, err
	}
	if err := r.hydrateManualReviewMaterials(ctx, []*ManualReviewCase{reviewCase}); err != nil {
		return nil, err
	}
	return reviewCase, nil
}

func (r *Repository) GetManualReviewCaseByID(
	ctx context.Context,
	caseID string,
) (*ManualReviewCase, error) {
	ctx = withTable(ctx, "student_manual_review_cases")
	reviewCase, err := scanManualReviewCase(r.db.QueryRow(ctx, manualReviewCaseSelectSQL()+`
		WHERE review_case.id = $1
	`, caseID))
	if err != nil {
		return nil, err
	}
	if err := r.hydrateManualReviewMaterials(ctx, []*ManualReviewCase{reviewCase}); err != nil {
		return nil, err
	}
	return reviewCase, nil
}

func (r *Repository) GetManualReviewCaseSchoolCode(
	ctx context.Context,
	caseID string,
) (string, error) {
	ctx = withTable(ctx, "student_manual_review_cases")
	var schoolCode string
	err := r.db.QueryRow(ctx, `
		SELECT school.code
		FROM student_manual_review_cases review_case
		JOIN schools school ON school.id = review_case.school_id
		WHERE review_case.id = $1
	`, caseID).Scan(&schoolCode)
	return schoolCode, err
}

func (r *Repository) GetManualReviewCaseForUpdateTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
) (*ManualReviewCase, error) {
	return scanManualReviewCase(tx.QueryRow(ctx, manualReviewCaseSelectSQL()+`
		WHERE review_case.id = $1
		FOR UPDATE OF review_case
	`, caseID))
}

func (r *Repository) GetManualReviewCaseForApplicationUpdateTx(
	ctx context.Context,
	tx pgx.Tx,
	applicationID string,
	userID int64,
) (*ManualReviewCase, error) {
	return scanManualReviewCase(tx.QueryRow(ctx, manualReviewCaseSelectSQL()+`
		WHERE review_case.application_id = $1 AND review_case.user_id = $2
		FOR UPDATE OF review_case
	`, applicationID, userID))
}

func manualReviewCaseSelectSQL() string {
	return `
		SELECT review_case.id, review_case.application_id, review_case.user_id,
		       review_case.school_id, school.code, school.name, review_case.status,
		       review_case.material_type, review_case.form_data_enc,
		       review_case.form_digest, review_case.encryption_key_version,
		       review_case.student_id_hash, review_case.student_id_display,
		       review_case.applicant_name_masked, review_case.email_hash,
		       review_case.email_display, review_case.email_verified_at,
		       review_case.privacy_notice_version, review_case.consented_at,
		       review_case.submitted_at, review_case.reviewed_by_user_id,
		       review_case.reviewed_at, review_case.user_visible_reason,
		       review_case.internal_risk_note_enc, review_case.credential_class,
		       review_case.credential_expires_at, review_case.credential_id,
		       review_case.revision, review_case.created_at, review_case.updated_at
		FROM student_manual_review_cases review_case
		JOIN schools school ON school.id = review_case.school_id
	`
}

func scanManualReviewCase(row rowScanner) (*ManualReviewCase, error) {
	var reviewCase ManualReviewCase
	err := row.Scan(
		&reviewCase.ID, &reviewCase.ApplicationID, &reviewCase.UserID,
		&reviewCase.SchoolID, &reviewCase.SchoolCode, &reviewCase.SchoolName,
		&reviewCase.Status, &reviewCase.MaterialType, &reviewCase.FormDataEnc,
		&reviewCase.FormDigest, &reviewCase.EncryptionKeyVersion,
		&reviewCase.StudentIDHash, &reviewCase.StudentIDMasked,
		&reviewCase.ApplicantNameMasked, &reviewCase.EmailHash,
		&reviewCase.EmailMasked, &reviewCase.EmailVerifiedAt,
		&reviewCase.PrivacyNoticeVersion, &reviewCase.ConsentedAt,
		&reviewCase.SubmittedAt, &reviewCase.ReviewedByUserID,
		&reviewCase.ReviewedAt, &reviewCase.UserVisibleReason,
		&reviewCase.InternalRiskNoteEnc, &reviewCase.CredentialClass,
		&reviewCase.CredentialExpiresAt, &reviewCase.CredentialID,
		&reviewCase.Revision, &reviewCase.CreatedAt, &reviewCase.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	reviewCase.School = SchoolReference{Code: reviewCase.SchoolCode, Name: reviewCase.SchoolName}
	reviewCase.Materials = []ManualReviewMaterial{}
	reviewCase.NextActions = []string{}
	return &reviewCase, nil
}

func (r *Repository) hydrateManualReviewMaterials(
	ctx context.Context,
	cases []*ManualReviewCase,
) error {
	if len(cases) == 0 {
		return nil
	}
	caseIDs := make([]string, 0, len(cases))
	byID := make(map[string]*ManualReviewCase, len(cases))
	for _, reviewCase := range cases {
		if reviewCase == nil {
			continue
		}
		caseIDs = append(caseIDs, reviewCase.ID)
		byID[reviewCase.ID] = reviewCase
	}
	if len(caseIDs) == 0 {
		return nil
	}
	ctx = withTable(ctx, "student_manual_review_materials")
	rows, err := r.db.Query(ctx, manualReviewMaterialSelectSQL()+`
		WHERE material.case_id = ANY($1::varchar[]) AND material.status = 'active'
		ORDER BY material.case_id, material.created_at, material.id
	`, caseIDs)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		caseID, material, scanErr := scanManualReviewMaterial(rows)
		if scanErr != nil {
			return scanErr
		}
		if reviewCase := byID[caseID]; reviewCase != nil {
			reviewCase.Materials = append(reviewCase.Materials, material)
		}
	}
	return rows.Err()
}

func manualReviewMaterialSelectSQL() string {
	return `
		SELECT material.case_id, material.id, material.object_key,
		       material.content_type, material.size_bytes, material.sha256,
		       material.width, material.height, material.capture_source,
		       material.requested_facing_mode, material.retention_until,
		       material.created_at
		FROM student_manual_review_materials material
	`
}

func scanManualReviewMaterial(row rowScanner) (string, ManualReviewMaterial, error) {
	var caseID string
	var material ManualReviewMaterial
	err := row.Scan(
		&caseID, &material.ID, &material.ObjectKey, &material.ContentType,
		&material.SizeBytes, &material.SHA256, &material.Width, &material.Height,
		&material.CaptureSource, &material.FacingMode, &material.RetentionAt,
		&material.CapturedAt,
	)
	return caseID, material, err
}

func (r *Repository) SaveManualReviewCaseTx(
	ctx context.Context,
	tx pgx.Tx,
	reviewCase ManualReviewCase,
	now time.Time,
) (*ManualReviewCase, bool, error) {
	existing, err := r.GetManualReviewCaseForApplicationUpdateTx(
		ctx, tx, reviewCase.ApplicationID, reviewCase.UserID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = tx.Exec(ctx, `
			INSERT INTO student_manual_review_cases (
			    id, application_id, user_id, school_id, status, material_type,
			    form_data_enc, form_digest, encryption_key_version,
			    student_id_hash, student_id_display, applicant_name_masked,
			    email_hash, email_display, privacy_notice_version, consented_at,
			    revision, created_at, updated_at
			)
			VALUES (
			    $1, $2, $3, $4, 'draft', $5,
			    $6, $7, $8, $9, $10, $11,
			    $12, $13, $14, $15, 1, $15, $15
			)
		`,
			reviewCase.ID, reviewCase.ApplicationID, reviewCase.UserID,
			reviewCase.SchoolID, reviewCase.MaterialType, reviewCase.FormDataEnc,
			reviewCase.FormDigest, reviewCase.EncryptionKeyVersion,
			reviewCase.StudentIDHash, reviewCase.StudentIDMasked,
			reviewCase.ApplicantNameMasked, reviewCase.EmailHash,
			reviewCase.EmailMasked, reviewCase.PrivacyNoticeVersion, now,
		)
		if err != nil {
			return nil, false, fmt.Errorf("insert manual review case: %w", err)
		}
		created, err := r.GetManualReviewCaseForApplicationUpdateTx(
			ctx, tx, reviewCase.ApplicationID, reviewCase.UserID,
		)
		return created, true, err
	}
	if err != nil {
		return nil, false, err
	}
	if existing.Status != ManualReviewDraft && existing.Status != ManualReviewSupplementRequired {
		return nil, false, ErrManualReviewState
	}
	result, err := tx.Exec(ctx, `
		UPDATE student_manual_review_cases
		SET material_type = $2, form_data_enc = $3, form_digest = $4,
		    encryption_key_version = $5, student_id_hash = $6,
		    student_id_display = $7, applicant_name_masked = $8,
		    email_hash = $9, email_display = $10,
		    email_verified_at = CASE WHEN email_hash IS NOT DISTINCT FROM $9 THEN email_verified_at ELSE NULL END,
		    email_verification_source = CASE WHEN email_hash IS NOT DISTINCT FROM $9 THEN email_verification_source ELSE NULL END,
		    privacy_notice_version = $11, consented_at = $12,
		    revision = revision + 1, updated_at = $12
		WHERE id = $1 AND status IN ('draft', 'supplement_required')
	`,
		existing.ID, reviewCase.MaterialType, reviewCase.FormDataEnc,
		reviewCase.FormDigest, reviewCase.EncryptionKeyVersion,
		reviewCase.StudentIDHash, reviewCase.StudentIDMasked,
		reviewCase.ApplicantNameMasked, reviewCase.EmailHash,
		reviewCase.EmailMasked, reviewCase.PrivacyNoticeVersion, now,
	)
	if err != nil {
		return nil, false, fmt.Errorf("update manual review case: %w", err)
	}
	if result.RowsAffected() != 1 {
		return nil, false, ErrManualReviewState
	}
	updated, err := r.GetManualReviewCaseForUpdateTx(ctx, tx, existing.ID)
	return updated, false, err
}

func (r *Repository) ProgressApplicationManualDraftTx(
	ctx context.Context,
	tx pgx.Tx,
	applicationID string,
	privacyNoticeVersion string,
	now time.Time,
) error {
	result, err := tx.Exec(ctx, `
		UPDATE student_verification_applications
		SET status = 'in_progress', current_method = 'manual_material_review',
		    privacy_notice_version = $2, consented_at = $3,
		    revision = revision + 1, updated_at = $3
		WHERE id = $1
		  AND status IN ('created', 'in_progress', 'pending_manual_review')
	`, applicationID, privacyNoticeVersion, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrApplicationState
	}
	return nil
}

func (r *Repository) InsertManualReviewEventTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	revision int64,
	actorType string,
	actorUserID *int64,
	action string,
	reasonCode *string,
	now time.Time,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO student_manual_review_events (
		    case_id, case_revision, actor_type, actor_user_id,
		    action, reason_code, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, caseID, revision, actorType, actorUserID, action, reasonCode, now)
	return err
}

func (r *Repository) AddManualReviewMaterialTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	userID int64,
	material ManualReviewMaterial,
	maxMaterials int,
	now time.Time,
) (*ManualReviewCase, error) {
	reviewCase, err := r.GetManualReviewCaseForUpdateTx(ctx, tx, caseID)
	if err != nil {
		return nil, err
	}
	if reviewCase.UserID != userID ||
		(reviewCase.Status != ManualReviewDraft && reviewCase.Status != ManualReviewSupplementRequired) {
		return nil, ErrManualReviewState
	}
	var materialCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM student_manual_review_materials
		WHERE case_id = $1 AND status = 'active'
	`, caseID).Scan(&materialCount); err != nil {
		return nil, err
	}
	if materialCount >= maxMaterials {
		return nil, ErrManualMaterialLimit
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO student_manual_review_materials (
		    id, case_id, object_key, content_type, size_bytes, sha256,
		    width, height, capture_source, requested_facing_mode,
		    retention_until, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, material.ID, caseID, material.ObjectKey, material.ContentType,
		material.SizeBytes, material.SHA256, material.Width, material.Height,
		material.CaptureSource, material.FacingMode, material.RetentionAt, now)
	if err != nil {
		return nil, fmt.Errorf("insert manual review material: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE student_manual_review_cases
		SET revision = revision + 1, updated_at = $2
		WHERE id = $1
	`, caseID, now); err != nil {
		return nil, err
	}
	updated, err := r.GetManualReviewCaseForUpdateTx(ctx, tx, caseID)
	if err != nil {
		return nil, err
	}
	applicant := userID
	if err := r.InsertManualReviewEventTx(
		ctx, tx, caseID, updated.Revision, "applicant", &applicant,
		"material_added", nil, now,
	); err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *Repository) SetManualReviewEmailVerifiedTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	userID int64,
	emailHash string,
	now time.Time,
) error {
	result, err := tx.Exec(ctx, `
		UPDATE student_manual_review_cases
		SET email_verified_at = $4, email_verification_source = 'outbound_otp',
		    revision = revision + 1, updated_at = $4
		WHERE id = $1 AND user_id = $2 AND email_hash = $3
		  AND status IN ('draft', 'supplement_required')
	`, caseID, userID, emailHash, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrManualReviewState
	}
	return nil
}

func (r *Repository) SubmitManualReviewTx(
	ctx context.Context,
	tx pgx.Tx,
	reviewCase *ManualReviewCase,
	requireVerifiedEmail bool,
	applicationExpiresAt time.Time,
	now time.Time,
) error {
	if reviewCase == nil ||
		(reviewCase.Status != ManualReviewDraft && reviewCase.Status != ManualReviewSupplementRequired) {
		return ErrManualReviewState
	}
	var materialCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM student_manual_review_materials
		WHERE case_id = $1 AND status = 'active'
	`, reviewCase.ID).Scan(&materialCount); err != nil {
		return err
	}
	if materialCount == 0 {
		return ErrManualMaterialRequired
	}
	if requireVerifiedEmail && reviewCase.EmailVerifiedAt == nil {
		return ErrManualEmailVerificationRequired
	}
	result, err := tx.Exec(ctx, `
		UPDATE student_manual_review_cases
		SET status = 'pending', submitted_at = COALESCE(submitted_at, $2),
		    reviewed_by_user_id = NULL, reviewed_at = NULL,
		    user_visible_reason = NULL, internal_risk_note_enc = NULL,
		    revision = revision + 1, updated_at = $2
		WHERE id = $1 AND status IN ('draft', 'supplement_required')
	`, reviewCase.ID, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrManualReviewState
	}
	result, err = tx.Exec(ctx, `
		UPDATE student_verification_applications
		SET status = 'pending_manual_review', current_method = 'manual_material_review',
		    expires_at = GREATEST(expires_at, $2), revision = revision + 1,
		    updated_at = $3
		WHERE id = $1 AND user_id = $4
		  AND status IN ('created', 'in_progress', 'pending_manual_review')
	`, reviewCase.ApplicationID, applicationExpiresAt, now, reviewCase.UserID)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrApplicationState
	}
	updated, err := r.GetManualReviewCaseForUpdateTx(ctx, tx, reviewCase.ID)
	if err != nil {
		return err
	}
	applicant := reviewCase.UserID
	return r.InsertManualReviewEventTx(
		ctx, tx, reviewCase.ID, updated.Revision, "applicant", &applicant,
		"submitted", nil, now,
	)
}

func (r *Repository) CreateManualCameraHandoffTx(
	ctx context.Context,
	tx pgx.Tx,
	handoff ManualCameraHandoff,
	tokenHash string,
	tokenEnc []byte,
	encryptionKeyVersion int,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO student_manual_camera_handoffs (
		    id, case_id, user_id, token_hash, token_enc,
		    encryption_key_version, status, expires_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7, $8, $8)
	`, handoff.ID, handoff.CaseID, handoff.UserID, tokenHash, tokenEnc,
		encryptionKeyVersion, handoff.ExpiresAt, handoff.CreatedAt)
	return err
}

func (r *Repository) ExpireManualCameraHandoffsTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	now time.Time,
) error {
	if _, err := tx.Exec(ctx, `
		UPDATE student_manual_camera_handoffs
		SET status = 'expired', token_enc = NULL, encryption_key_version = NULL,
		    updated_at = $2
		WHERE case_id = $1 AND status = 'pending'
		  AND expires_at <= $2
	`, caseID, now); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		UPDATE student_manual_camera_handoffs
		SET status = 'locked', token_enc = NULL, encryption_key_version = NULL,
		    updated_at = $2
		WHERE case_id = $1 AND status = 'uploaded'
		  AND expires_at <= $2
	`, caseID, now)
	return err
}

func (r *Repository) CloseExpiredManualCameraHandoffTx(
	ctx context.Context,
	tx pgx.Tx,
	handoffID string,
	now time.Time,
) error {
	if _, err := tx.Exec(ctx, `
		UPDATE student_manual_camera_handoffs
		SET status = 'expired', token_enc = NULL, encryption_key_version = NULL,
		    updated_at = $2
		WHERE id = $1 AND status = 'pending' AND expires_at <= $2
	`, handoffID, now); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		UPDATE student_manual_camera_handoffs
		SET status = 'locked', token_enc = NULL, encryption_key_version = NULL,
		    updated_at = $2
		WHERE id = $1 AND status = 'uploaded' AND expires_at <= $2
	`, handoffID, now)
	return err
}

func (r *Repository) ExpirePendingManualCameraHandoffTx(
	ctx context.Context,
	tx pgx.Tx,
	handoffID string,
	now time.Time,
) error {
	result, err := tx.Exec(ctx, `
		UPDATE student_manual_camera_handoffs
		SET status = 'expired', token_enc = NULL, encryption_key_version = NULL,
		    updated_at = $2
		WHERE id = $1 AND status = 'pending'
	`, handoffID, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrManualHandoffState
	}
	return nil
}

func (r *Repository) GetActiveManualCameraHandoffTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	now time.Time,
) (*storedManualCameraHandoff, error) {
	return scanStoredManualCameraHandoff(tx.QueryRow(ctx, manualCameraHandoffSelectSQL()+`
		WHERE handoff.case_id = $1 AND handoff.status IN ('pending', 'uploaded')
		  AND handoff.expires_at > $2
		ORDER BY handoff.created_at DESC
		LIMIT 1
		FOR UPDATE OF handoff
	`, caseID, now))
}

func (r *Repository) GetManualCameraHandoffForUser(
	ctx context.Context,
	applicationID string,
	handoffID string,
	userID int64,
) (*storedManualCameraHandoff, error) {
	ctx = withTable(ctx, "student_manual_camera_handoffs")
	return scanStoredManualCameraHandoff(r.db.QueryRow(ctx, manualCameraHandoffSelectSQL()+`
		WHERE handoff.id = $1 AND handoff.user_id = $2
		  AND review_case.application_id = $3
	`, handoffID, userID, applicationID))
}

func (r *Repository) GetManualCameraHandoffByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*storedManualCameraHandoff, error) {
	ctx = withTable(ctx, "student_manual_camera_handoffs")
	return scanStoredManualCameraHandoff(r.db.QueryRow(ctx, manualCameraHandoffSelectSQL()+`
		WHERE handoff.token_hash = $1
	`, tokenHash))
}

func (r *Repository) GetManualCameraHandoffByTokenHashForUpdateTx(
	ctx context.Context,
	tx pgx.Tx,
	tokenHash string,
) (*storedManualCameraHandoff, error) {
	return scanStoredManualCameraHandoff(tx.QueryRow(ctx, manualCameraHandoffSelectSQL()+`
		WHERE handoff.token_hash = $1
		FOR UPDATE OF handoff
	`, tokenHash))
}

type storedManualCameraHandoff struct {
	ManualCameraHandoff
	TokenEnc             []byte
	EncryptionKeyVersion *int
}

type expiredManualReviewMaterial struct {
	ID        string
	CaseID    string
	ObjectKey string
}

func (r *Repository) ListExpiredManualReviewMaterials(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]expiredManualReviewMaterial, error) {
	ctx = withTable(ctx, "student_manual_review_materials")
	rows, err := r.db.Query(ctx, `
		SELECT id, case_id, object_key
		FROM student_manual_review_materials
		WHERE status = 'active' AND retention_until <= $1
		ORDER BY retention_until, id
		LIMIT $2
	`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]expiredManualReviewMaterial, 0)
	for rows.Next() {
		var material expiredManualReviewMaterial
		if err := rows.Scan(&material.ID, &material.CaseID, &material.ObjectKey); err != nil {
			return nil, err
		}
		result = append(result, material)
	}
	return result, rows.Err()
}

func (r *Repository) MarkManualReviewMaterialDeleted(
	ctx context.Context,
	material expiredManualReviewMaterial,
	now time.Time,
) error {
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		reviewCase, err := r.GetManualReviewCaseForUpdateTx(ctx, tx, material.CaseID)
		if err != nil {
			return err
		}
		result, err := tx.Exec(ctx, `
			UPDATE student_manual_review_materials
			SET status = 'deleted', deleted_at = $3
			WHERE id = $1 AND case_id = $2 AND status = 'active'
		`, material.ID, material.CaseID, now)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return nil
		}
		if _, err := tx.Exec(ctx, `
			UPDATE student_manual_review_cases
			SET revision = revision + 1, updated_at = $2
			WHERE id = $1
		`, material.CaseID, now); err != nil {
			return err
		}
		reason := "retention_expired"
		return r.InsertManualReviewEventTx(
			ctx, tx, material.CaseID, reviewCase.Revision+1,
			"system", nil, "material_deleted", &reason, now,
		)
	})
}

func manualCameraHandoffSelectSQL() string {
	return `
		SELECT handoff.id, handoff.case_id, review_case.application_id,
		       handoff.user_id, handoff.status, handoff.material_id,
		       handoff.continue_on, handoff.token_enc,
		       handoff.encryption_key_version, handoff.expires_at,
		       handoff.uploaded_at, handoff.chosen_at, handoff.created_at,
		       material.id, material.object_key, material.content_type,
		       material.size_bytes, material.sha256, material.width,
		       material.height, material.capture_source,
		       material.requested_facing_mode, material.retention_until,
		       material.created_at
		FROM student_manual_camera_handoffs handoff
		JOIN student_manual_review_cases review_case ON review_case.id = handoff.case_id
		LEFT JOIN student_manual_review_materials material
		  ON material.id = handoff.material_id AND material.status = 'active'
	`
}

func scanStoredManualCameraHandoff(row rowScanner) (*storedManualCameraHandoff, error) {
	var handoff storedManualCameraHandoff
	var material ManualReviewMaterial
	var materialID, objectKey, contentType, sha256Value, captureSource, facingMode *string
	var sizeBytes *int64
	var width, height *int
	var retentionUntil, capturedAt *time.Time
	err := row.Scan(
		&handoff.ID, &handoff.CaseID, &handoff.ApplicationID,
		&handoff.UserID, &handoff.Status, &handoff.MaterialID,
		&handoff.ContinueOn, &handoff.TokenEnc, &handoff.EncryptionKeyVersion,
		&handoff.ExpiresAt, &handoff.UploadedAt, &handoff.ChosenAt,
		&handoff.CreatedAt, &materialID, &objectKey, &contentType,
		&sizeBytes, &sha256Value, &width, &height, &captureSource,
		&facingMode, &retentionUntil, &capturedAt,
	)
	if err != nil {
		return nil, err
	}
	if materialID != nil && objectKey != nil && contentType != nil && sizeBytes != nil &&
		sha256Value != nil && width != nil && height != nil && captureSource != nil &&
		facingMode != nil && retentionUntil != nil && capturedAt != nil {
		material = ManualReviewMaterial{
			ID: *materialID, ObjectKey: *objectKey, ContentType: *contentType,
			SizeBytes: *sizeBytes, SHA256: *sha256Value, Width: *width,
			Height: *height, CaptureSource: *captureSource,
			FacingMode: *facingMode, RetentionAt: *retentionUntil,
			CapturedAt: *capturedAt,
		}
		handoff.Material = &material
	}
	return &handoff, nil
}

func (r *Repository) MarkManualHandoffUploadedTx(
	ctx context.Context,
	tx pgx.Tx,
	handoffID string,
	materialID string,
	now time.Time,
) error {
	result, err := tx.Exec(ctx, `
		UPDATE student_manual_camera_handoffs
		SET status = 'uploaded', material_id = $2, uploaded_at = $3,
		    updated_at = $3
		WHERE id = $1 AND status = 'pending' AND expires_at > $3
	`, handoffID, materialID, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrManualHandoffState
	}
	return nil
}

func (r *Repository) ChooseManualHandoffContinuationTx(
	ctx context.Context,
	tx pgx.Tx,
	handoffID string,
	continueOn string,
	now time.Time,
) error {
	result, err := tx.Exec(ctx, `
		UPDATE student_manual_camera_handoffs
		SET status = 'locked', continue_on = $2, chosen_at = $3,
		    token_enc = NULL, encryption_key_version = NULL, updated_at = $3
		WHERE id = $1 AND status = 'uploaded' AND expires_at > $3
	`, handoffID, continueOn, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrManualHandoffState
	}
	return nil
}

func (r *Repository) ListManualReviewCases(
	ctx context.Context,
	schoolCode string,
	status ManualReviewStatus,
	limit int,
	offset int,
) ([]*ManualReviewCase, error) {
	ctx = withTable(ctx, "student_manual_review_cases")
	rows, err := r.db.Query(ctx, manualReviewCaseSelectSQL()+`
		WHERE school.code = $1
		  AND ($2 = '' OR review_case.status = $2)
		  AND review_case.status <> 'draft'
		ORDER BY COALESCE(review_case.submitted_at, review_case.created_at), review_case.id
		LIMIT $3 OFFSET $4
	`, schoolCode, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cases := make([]*ManualReviewCase, 0)
	for rows.Next() {
		reviewCase, scanErr := scanManualReviewCase(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		cases = append(cases, reviewCase)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.hydrateManualReviewMaterials(ctx, cases); err != nil {
		return nil, err
	}
	return cases, nil
}

func (r *Repository) GetManualMaterialForCase(
	ctx context.Context,
	caseID string,
	materialID string,
) (*ManualReviewMaterial, int64, string, error) {
	ctx = withTable(ctx, "student_manual_review_materials")
	var userID int64
	var schoolCode string
	caseIDResult, material, err := scanManualReviewMaterial(r.db.QueryRow(ctx, manualReviewMaterialSelectSQL()+`
		JOIN student_manual_review_cases review_case ON review_case.id = material.case_id
		JOIN schools school ON school.id = review_case.school_id
		WHERE material.case_id = $1 AND material.id = $2 AND material.status = 'active'
	`, caseID, materialID))
	if err != nil {
		return nil, 0, "", err
	}
	if caseIDResult != caseID {
		return nil, 0, "", pgx.ErrNoRows
	}
	if err := r.db.QueryRow(ctx, `
		SELECT review_case.user_id, school.code
		FROM student_manual_review_cases review_case
		JOIN schools school ON school.id = review_case.school_id
		WHERE review_case.id = $1
	`, caseID).Scan(&userID, &schoolCode); err != nil {
		return nil, 0, "", err
	}
	return &material, userID, schoolCode, nil
}

func (r *Repository) RecordManualMaterialAccessEvent(
	ctx context.Context,
	caseID string,
	actorUserID int64,
	action string,
	reasonCode *string,
	now time.Time,
) error {
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		reviewCase, err := r.GetManualReviewCaseForUpdateTx(ctx, tx, caseID)
		if err != nil {
			return err
		}
		return r.InsertManualReviewEventTx(
			ctx, tx, caseID, reviewCase.Revision, "reviewer", &actorUserID,
			action, reasonCode, now,
		)
	})
}

func (r *Repository) RequestManualReviewSupplementTx(
	ctx context.Context,
	tx pgx.Tx,
	reviewCase *ManualReviewCase,
	reviewerUserID int64,
	userVisibleReason string,
	internalRiskNoteEnc []byte,
	applicationExpiresAt time.Time,
	now time.Time,
) error {
	result, err := tx.Exec(ctx, `
		UPDATE student_manual_review_cases
		SET status = 'supplement_required', reviewed_by_user_id = $2,
		    user_visible_reason = $3, internal_risk_note_enc = $4,
		    revision = revision + 1, updated_at = $5
		WHERE id = $1 AND status = 'pending'
	`, reviewCase.ID, reviewerUserID, userVisibleReason, internalRiskNoteEnc, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrManualReviewState
	}
	if _, err := tx.Exec(ctx, `
		UPDATE student_verification_applications
		SET expires_at = GREATEST(expires_at, $2), revision = revision + 1,
		    updated_at = $3
		WHERE id = $1 AND status = 'pending_manual_review'
	`, reviewCase.ApplicationID, applicationExpiresAt, now); err != nil {
		return err
	}
	updated, err := r.GetManualReviewCaseForUpdateTx(ctx, tx, reviewCase.ID)
	if err != nil {
		return err
	}
	reasonCode := "additional_material_required"
	return r.InsertManualReviewEventTx(
		ctx, tx, reviewCase.ID, updated.Revision, "reviewer", &reviewerUserID,
		"supplement_requested", &reasonCode, now,
	)
}

func (r *Repository) RejectManualReviewTx(
	ctx context.Context,
	tx pgx.Tx,
	reviewCase *ManualReviewCase,
	reviewerUserID int64,
	userVisibleReason string,
	internalRiskNoteEnc []byte,
	config *MethodConfig,
	now time.Time,
) error {
	result, err := tx.Exec(ctx, `
		UPDATE student_manual_review_cases
		SET status = 'rejected', reviewed_by_user_id = $2, reviewed_at = $3,
		    user_visible_reason = $4, internal_risk_note_enc = $5,
		    revision = revision + 1, updated_at = $3
		WHERE id = $1 AND status = 'pending'
	`, reviewCase.ID, reviewerUserID, now, userVisibleReason, internalRiskNoteEnc)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrManualReviewState
	}
	application, err := r.GetApplicationForUpdateTx(ctx, tx, reviewCase.ApplicationID, reviewCase.UserID)
	if err != nil {
		return err
	}
	if err := r.insertAttemptAndProgressTx(ctx, tx, application, attemptResultFor(
		config, "failed", "manual_review_rejected", nil, nil,
		reviewCase.PrivacyNoticeVersion, now,
	), now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE student_verification_applications
		SET status = 'rejected', terminal_code = 'manual_review_not_approved',
		    completed_at = $2, revision = revision + 1, updated_at = $2
		WHERE id = $1
	`, reviewCase.ApplicationID, now); err != nil {
		return err
	}
	updated, err := r.GetManualReviewCaseForUpdateTx(ctx, tx, reviewCase.ID)
	if err != nil {
		return err
	}
	reasonCode := "not_approved"
	return r.InsertManualReviewEventTx(
		ctx, tx, reviewCase.ID, updated.Revision, "reviewer", &reviewerUserID,
		"rejected", &reasonCode, now,
	)
}

func (r *Repository) ApproveManualReviewCaseTx(
	ctx context.Context,
	tx pgx.Tx,
	reviewCase *ManualReviewCase,
	reviewerUserID int64,
	userVisibleReason string,
	internalRiskNoteEnc []byte,
	credential Credential,
	config *MethodConfig,
	now time.Time,
) error {
	result, err := tx.Exec(ctx, `
		UPDATE student_manual_review_cases
		SET status = 'approved', reviewed_by_user_id = $2, reviewed_at = $3,
		    user_visible_reason = $4, internal_risk_note_enc = $5,
		    credential_class = $6, credential_expires_at = $7,
		    credential_id = $8, revision = revision + 1, updated_at = $3
		WHERE id = $1 AND status = 'pending'
	`, reviewCase.ID, reviewerUserID, now, userVisibleReason, internalRiskNoteEnc,
		credential.CredentialClass, credential.ExpiresAt, credential.ID)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrManualReviewState
	}
	application, err := r.GetApplicationForUpdateTx(ctx, tx, reviewCase.ApplicationID, reviewCase.UserID)
	if err != nil {
		return err
	}
	if err := r.CompleteApplicationTx(ctx, tx, application, attemptResultFor(
		config, "succeeded", "manual_review_approved", nil, nil,
		reviewCase.PrivacyNoticeVersion, now,
	), now); err != nil {
		return err
	}
	updated, err := r.GetManualReviewCaseForUpdateTx(ctx, tx, reviewCase.ID)
	if err != nil {
		return err
	}
	reasonCode := "approved"
	return r.InsertManualReviewEventTx(
		ctx, tx, reviewCase.ID, updated.Revision, "reviewer", &reviewerUserID,
		"approved", &reasonCode, now,
	)
}

func (r *Repository) CreateSchoolVerificationSuggestion(
	ctx context.Context,
	userID int64,
	suggestionID string,
	schoolName string,
	schoolLocation *string,
	now time.Time,
) error {
	ctx = withTable(ctx, "school_verification_suggestions")
	_, err := r.db.Exec(ctx, `
		INSERT INTO school_verification_suggestions (
		    id, user_id, school_name, school_location, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, 'pending', $5, $5)
	`, suggestionID, userID, schoolName, schoolLocation, now)
	return err
}

func normalizeManualReviewStatus(value string) ManualReviewStatus {
	return ManualReviewStatus(strings.TrimSpace(value))
}
