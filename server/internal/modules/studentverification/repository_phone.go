package studentverification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type phoneOutboxJob struct {
	ID             int64
	OperationID    string
	ActionRevision int64
	AttemptCount   int
}

func (r *Repository) FindBUAAPhoneRosterEvidence(
	ctx context.Context,
	userID int64,
	schoolCode string,
	studentIDHash *string,
	nameHash *string,
	phoneHash string,
	now time.Time,
) (*PhoneRosterEvidence, error) {
	ctx = withTable(ctx, "student_roster_records")
	var evidence PhoneRosterEvidence
	var hardExpirySeconds int
	err := r.db.QueryRow(ctx, `
		SELECT subject.school_id, subject.id, active.snapshot_id,
		       active.activation_revision, snapshot.source_cutoff_at,
		       profile.snapshot_hard_expiry_seconds
		FROM student_enrollment_subjects subject
		JOIN schools school ON school.id = subject.school_id
		JOIN school_verification_profiles profile
		  ON profile.school_id = subject.school_id
		JOIN academic.student_roster_active active
		  ON active.school_id = subject.school_id
		JOIN academic.student_roster_snapshots snapshot
		  ON snapshot.id = active.snapshot_id
		 AND snapshot.school_id = active.school_id
		JOIN academic.student_roster_records record
		  ON record.school_id = active.school_id
		 AND record.snapshot_id = active.snapshot_id
		 AND record.student_id_hash = subject.student_id_hash
		WHERE subject.user_id = $1
		  AND school.code = $2
		  AND subject.binding_status = 'active'
		  AND profile.enabled
		  AND profile.validation_status = 'valid'
		  AND snapshot.status = 'active'
		  AND snapshot.source_cutoff_at <= $6::timestamptz + INTERVAL '5 minutes'
		  AND snapshot.source_cutoff_at
		      + make_interval(secs => profile.snapshot_hard_expiry_seconds) >= $6::timestamptz
		  AND record.eligibility_status = 'eligible'
		  AND record.hmac_key_version = $7
		  AND record.phone_hash = $5
		  AND ($3::text IS NULL OR record.student_id_hash = $3)
		  AND ($4::text IS NULL OR record.name_hash = $4)
		  AND EXISTS (
		      SELECT 1
		      FROM user_verification_credentials credential
		      WHERE credential.user_id = subject.user_id
		        AND credential.school_id = subject.school_id
		        AND credential.enrollment_subject_id = subject.id
		        AND credential.status = 'active'
		        AND credential.revoked_at IS NULL
		        AND (credential.expires_at IS NULL OR credential.expires_at > $6::timestamptz)
		  )
		  AND NOT EXISTS (
		      SELECT 1
		      FROM student_subject_conflicts conflict
		      WHERE conflict.school_id = subject.school_id
		        AND conflict.subject_hash = subject.subject_hash
		        AND conflict.status IN ('open', 'under_review')
		  )
		ORDER BY subject.activated_at DESC, subject.id DESC
		LIMIT 1
	`, userID, schoolCode, studentIDHash, nameHash, phoneHash, now, RosterHMACKeyVersion).Scan(
		&evidence.SchoolID,
		&evidence.EnrollmentSubjectID,
		&evidence.SnapshotID,
		&evidence.SnapshotRevision,
		&evidence.SourceCutoffAt,
		&hardExpirySeconds,
	)
	if err != nil {
		return nil, err
	}
	evidence.HardExpiry = time.Duration(hardExpirySeconds) * time.Second
	return &evidence, nil
}

func (r *Repository) CreatePhoneOperation(
	ctx context.Context,
	operation PhoneBindingOperation,
	evidence *PhoneRosterEvidence,
	now time.Time,
) error {
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var projectionState string
		var currentHash *string
		var hasCredential bool
		if err := tx.QueryRow(ctx, `
			SELECT u.phone_projection_state, u.phone_hash,
			       EXISTS (
			           SELECT 1 FROM phone_verification_credentials credential
			           WHERE credential.user_id = u.id
			             AND credential.status = 'active'
			             AND credential.revoked_at IS NULL
			             AND (credential.expires_at IS NULL OR credential.expires_at > $2)
			       )
			FROM users u
			WHERE u.id = $1
			FOR UPDATE
		`, operation.UserID, now).Scan(&projectionState, &currentHash, &hasCredential); err != nil {
			return err
		}

		switch operation.OperationKind {
		case PhoneOperationBind:
			if projectionState != "absent" || hasCredential {
				return ErrPhoneAlreadyBound
			}
		case PhoneOperationChange:
			if projectionState != "synced" || !hasCredential {
				return ErrPhoneNotBound
			}
			if currentHash != nil && operation.TargetPhoneHash != nil && *currentHash == *operation.TargetPhoneHash {
				return ErrPhoneAlreadyBound
			}
		default:
			return ErrPhoneOperationConflict
		}

		status := PhoneOperationPendingVerification
		var method *PhoneVerificationMethod
		var verifiedAt *time.Time
		if evidence != nil {
			status = PhoneOperationCasdoorUpdatePending
			value := PhoneMethodRosterMatch
			method = &value
			verifiedAt = &now
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO phone_binding_operations (
			    id, user_id, operation_kind, status, verification_method,
			    target_phone_enc, target_phone_hash, target_phone_masked,
			    encryption_key_version, hmac_key_version, revision, expires_at,
			    verified_at, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 1, $11, $12, $13, $13)
		`, operation.ID, operation.UserID, operation.OperationKind, status, method,
			operation.TargetPhoneEnc, operation.TargetPhoneHash, operation.TargetPhoneMasked,
			operation.EncryptionKeyVersion, operation.HMACKeyVersion, operation.ExpiresAt,
			verifiedAt, now)
		if err != nil {
			if isUniqueViolation(err, "phone_binding_operations_active_user_uidx") {
				return ErrPhoneOperationConflict
			}
			return fmt.Errorf("create phone operation: %w", err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO phone_number_claims (
			    phone_hash, user_id, claim_status, operation_id, revision,
			    expires_at, created_at, updated_at
			)
			VALUES ($1, $2, 'pending', $3, 1, $4, $5, $5)
		`, operation.TargetPhoneHash, operation.UserID, operation.ID, operation.ExpiresAt, now)
		if err != nil {
			if isUniqueViolation(err, "phone_number_claims_pkey") ||
				isUniqueViolation(err, "phone_number_claims_pending_user_uidx") {
				return ErrPhoneOwnershipConflict
			}
			return fmt.Errorf("reserve phone claim: %w", err)
		}
		if evidence == nil {
			return nil
		}
		attemptID, err := newID()
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO phone_verification_attempts (
			    id, operation_id, attempt_number, method, status, result_code,
			    school_id, enrollment_subject_id, roster_snapshot_id,
			    roster_snapshot_revision, started_at, completed_at, created_at, updated_at
			)
			VALUES (
			    $1, $2, 1, 'school_roster_phone_match', 'succeeded', 'matched',
			    $3, $4, $5, $6, $7, $7, $7, $7
			)
		`, attemptID, operation.ID, evidence.SchoolID, evidence.EnrollmentSubjectID,
			evidence.SnapshotID, evidence.SnapshotRevision, now)
		if err != nil {
			return fmt.Errorf("record roster phone attempt: %w", err)
		}
		return r.enqueuePhoneOperationTx(ctx, tx, operation.ID, 1, now)
	})
}

func (r *Repository) CreatePhoneUnbindOperation(
	ctx context.Context,
	operation PhoneBindingOperation,
	now time.Time,
) error {
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var projectionState string
		var hasCredential bool
		if err := tx.QueryRow(ctx, `
			SELECT u.phone_projection_state,
			       EXISTS (
			           SELECT 1 FROM phone_verification_credentials credential
			           WHERE credential.user_id = u.id
			             AND credential.status = 'active'
			             AND credential.revoked_at IS NULL
			       )
			FROM users u WHERE u.id = $1 FOR UPDATE
		`, operation.UserID).Scan(&projectionState, &hasCredential); err != nil {
			return err
		}
		if projectionState != "synced" || !hasCredential {
			return ErrPhoneNotBound
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO phone_binding_operations (
			    id, user_id, operation_kind, status, verification_method,
			    revision, expires_at, verified_at, created_at, updated_at
			)
			VALUES ($1, $2, 'unbind', 'casdoor_update_pending', 'step_up_mfa',
			        1, $3, $4, $4, $4)
		`, operation.ID, operation.UserID, operation.ExpiresAt, now)
		if err != nil {
			if isUniqueViolation(err, "phone_binding_operations_active_user_uidx") {
				return ErrPhoneOperationConflict
			}
			return fmt.Errorf("create phone unbind operation: %w", err)
		}
		attemptID, err := newID()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO phone_verification_attempts (
			    id, operation_id, attempt_number, method, status, result_code,
			    started_at, completed_at, created_at, updated_at
			)
			VALUES ($1, $2, 1, 'step_up_mfa', 'succeeded', 'step_up_confirmed', $3, $3, $3, $3)
		`, attemptID, operation.ID, now); err != nil {
			return fmt.Errorf("record phone unbind step-up: %w", err)
		}
		return r.enqueuePhoneOperationTx(ctx, tx, operation.ID, 1, now)
	})
}

func (r *Repository) GetPhoneOperation(ctx context.Context, operationID string, userID int64) (*PhoneBindingOperation, error) {
	ctx = withTable(ctx, "phone_binding_operations")
	return scanPhoneOperation(r.db.QueryRow(ctx, phoneOperationSelectSQL()+`
		WHERE operation.id = $1 AND operation.user_id = $2
	`, operationID, userID))
}

func (r *Repository) getPhoneOperationForUpdateTx(
	ctx context.Context,
	tx pgx.Tx,
	operationID string,
	userID int64,
) (*PhoneBindingOperation, error) {
	return scanPhoneOperation(tx.QueryRow(ctx, phoneOperationSelectSQL()+`
		WHERE operation.id = $1 AND operation.user_id = $2
		FOR UPDATE OF operation
	`, operationID, userID))
}

func phoneOperationSelectSQL() string {
	return `
		SELECT operation.id, operation.user_id, operation.operation_kind,
		       operation.status, operation.verification_method,
		       operation.target_phone_enc, operation.target_phone_hash,
		       operation.target_phone_masked, operation.encryption_key_version,
		       operation.hmac_key_version, operation.failure_code,
		       operation.attempt_count, operation.sms_resend_available_at,
		       operation.revision, operation.expires_at, operation.verified_at,
		       operation.casdoor_updated_at, operation.projection_synced_at,
		       operation.completed_at, operation.created_at, operation.updated_at
		FROM phone_binding_operations operation
	`
}

func scanPhoneOperation(row rowScanner) (*PhoneBindingOperation, error) {
	var operation PhoneBindingOperation
	var kind, status string
	var method *string
	err := row.Scan(
		&operation.ID, &operation.UserID, &kind, &status, &method,
		&operation.TargetPhoneEnc, &operation.TargetPhoneHash,
		&operation.TargetPhoneMasked, &operation.EncryptionKeyVersion,
		&operation.HMACKeyVersion, &operation.FailureCode,
		&operation.AttemptCount, &operation.SMSResendAvailableAt,
		&operation.Revision, &operation.ExpiresAt, &operation.VerifiedAt,
		&operation.CasdoorUpdatedAt, &operation.ProjectionSyncedAt,
		&operation.CompletedAt, &operation.CreatedAt, &operation.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	operation.OperationKind = PhoneOperationKind(kind)
	operation.Status = PhoneOperationStatus(status)
	if method != nil && (*method == string(PhoneMethodRosterMatch) ||
		*method == string(PhoneMethodSMS)) {
		value := PhoneVerificationMethod(*method)
		operation.VerificationMethod = &value
	}
	return &operation, nil
}

func (r *Repository) RecordPhoneSMSIssued(
	ctx context.Context,
	operationID string,
	userID int64,
	resendAt time.Time,
	now time.Time,
) error {
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		operation, err := r.getPhoneOperationForUpdateTx(ctx, tx, operationID, userID)
		if err != nil {
			return err
		}
		if operation.Status != PhoneOperationPendingVerification || !operation.ExpiresAt.After(now) {
			return ErrPhoneOperationConflict
		}
		if operation.VerificationMethod != nil && *operation.VerificationMethod != PhoneMethodSMS {
			return ErrPhoneOperationConflict
		}
		attemptNumber := operation.AttemptCount + 1
		attemptID, err := newID()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO phone_verification_attempts (
			    id, operation_id, attempt_number, method, status,
			    started_at, created_at, updated_at
			)
			VALUES ($1, $2, $3, 'sms_possession', 'pending', $4, $4, $4)
		`, attemptID, operation.ID, attemptNumber, now); err != nil {
			return fmt.Errorf("record phone SMS attempt: %w", err)
		}
		_, err = tx.Exec(ctx, `
			UPDATE phone_binding_operations
			SET verification_method = 'sms_possession', attempt_count = $2,
			    sms_resend_available_at = $3, revision = revision + 1,
			    updated_at = $4
			WHERE id = $1
		`, operation.ID, attemptNumber, resendAt, now)
		return err
	})
}

func (r *Repository) MarkPhoneSMSVerified(
	ctx context.Context,
	operationID string,
	userID int64,
	now time.Time,
) error {
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		operation, err := r.getPhoneOperationForUpdateTx(ctx, tx, operationID, userID)
		if err != nil {
			return err
		}
		if operation.Status == PhoneOperationCasdoorUpdatePending &&
			operation.VerificationMethod != nil && *operation.VerificationMethod == PhoneMethodSMS {
			return nil
		}
		if operation.Status != PhoneOperationPendingVerification ||
			operation.VerificationMethod == nil || *operation.VerificationMethod != PhoneMethodSMS ||
			!operation.ExpiresAt.After(now) {
			return ErrPhoneOperationConflict
		}
		result, err := tx.Exec(ctx, `
			UPDATE phone_verification_attempts
			SET status = 'succeeded', result_code = 'verified', completed_at = $3,
			    updated_at = $3
			WHERE operation_id = $1 AND attempt_number = $2 AND status = 'pending'
		`, operation.ID, operation.AttemptCount, now)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ErrPhoneOperationConflict
		}
		_, err = tx.Exec(ctx, `
			UPDATE phone_binding_operations
			SET status = 'casdoor_update_pending', verified_at = $2,
			    revision = revision + 1, updated_at = $2
			WHERE id = $1
		`, operation.ID, now)
		if err != nil {
			return err
		}
		return r.enqueuePhoneOperationTx(ctx, tx, operation.ID, operation.Revision+1, now)
	})
}

func (r *Repository) MarkPhoneCasdoorUpdated(
	ctx context.Context,
	operationID string,
	userID int64,
	now time.Time,
) error {
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		operation, err := r.getPhoneOperationForUpdateTx(ctx, tx, operationID, userID)
		if err != nil {
			return err
		}
		if operation.Status == PhoneOperationProjectionPending || operation.Status == PhoneOperationCompleted {
			return nil
		}
		if operation.Status != PhoneOperationCasdoorUpdatePending {
			return ErrPhoneOperationConflict
		}
		if _, err := tx.Exec(ctx, `
			UPDATE phone_binding_operations
			SET status = 'projection_sync_pending', casdoor_updated_at = $2,
			    revision = revision + 1, updated_at = $2
			WHERE id = $1
		`, operation.ID, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE users
			SET phone_projection_state = 'syncing',
			    phone_projection_revision = phone_projection_revision + 1,
			    updated_at = $2
			WHERE id = $1
		`, userID, now); err != nil {
			return err
		}
		return r.bumpPhoneEligibilityTx(ctx, tx, userID, "casdoor_phone_changed", now)
	})
}

func (r *Repository) FinalizePhoneProjection(
	ctx context.Context,
	operationID string,
	userID int64,
	readbackEnc []byte,
	readbackHash *string,
	readbackMasked *string,
	encryptionKeyVersion *int,
	hmacKeyVersion *int,
	now time.Time,
) error {
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		operation, err := r.getPhoneOperationForUpdateTx(ctx, tx, operationID, userID)
		if err != nil {
			return err
		}
		if operation.Status == PhoneOperationCompleted {
			return nil
		}
		if operation.Status != PhoneOperationProjectionPending {
			return ErrPhoneOperationConflict
		}

		var credentialID *string
		if _, err := tx.Exec(ctx, `
			UPDATE phone_verification_credentials
			SET status = 'revoked', revoked_at = $2,
			    revoked_reason = $3, revision = revision + 1, updated_at = $2
			WHERE user_id = $1 AND status IN ('active', 'review_required')
			  AND revoked_at IS NULL
		`, userID, now, string(operation.OperationKind)); err != nil {
			return err
		}
		if operation.OperationKind == PhoneOperationUnbind {
			if _, err := tx.Exec(ctx, `
				UPDATE users
				SET phone_enc = NULL, phone_hash = NULL, phone_masked = NULL,
				    phone_projection_state = 'absent', phone_projection_synced_at = NULL,
				    phone_encryption_key_version = NULL, phone_hmac_key_version = NULL,
				    phone_projection_revision = phone_projection_revision + 1,
				    updated_at = $2
				WHERE id = $1
			`, userID, now); err != nil {
				return err
			}
		} else {
			if readbackHash == nil || readbackMasked == nil || encryptionKeyVersion == nil || hmacKeyVersion == nil || len(readbackEnc) == 0 {
				return ErrPhoneOperationConflict
			}
			if _, err := tx.Exec(ctx, `
				UPDATE users
				SET phone_enc = $2, phone_hash = $3, phone_masked = $4,
				    phone_projection_state = 'synced', phone_projection_synced_at = $5,
				    phone_encryption_key_version = $6, phone_hmac_key_version = $7,
				    phone_projection_revision = phone_projection_revision + 1,
				    updated_at = $5
				WHERE id = $1
			`, userID, readbackEnc, readbackHash, readbackMasked, now,
				encryptionKeyVersion, hmacKeyVersion); err != nil {
				if isUniqueViolation(err, "idx_users_phone_hash") {
					return ErrPhoneOwnershipConflict
				}
				return err
			}
			id, err := newID()
			if err != nil {
				return err
			}
			credentialID = &id
			evidence, err := r.phoneCredentialEvidenceTx(ctx, tx, operation)
			if err != nil {
				return err
			}
			method := string(PhoneMethodSMS)
			assurance := "current_possession"
			if operation.VerificationMethod != nil && *operation.VerificationMethod == PhoneMethodRosterMatch {
				method = string(PhoneMethodRosterMatch)
				assurance = "school_data_match"
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO phone_verification_credentials (
				    id, user_id, phone_hash, phone_display, method, assurance,
				    status, operation_id, school_id, enrollment_subject_id,
				    roster_snapshot_id, roster_snapshot_revision, verified_at,
				    last_confirmed_at, revision, created_at, updated_at
				)
				VALUES ($1, $2, $3, $4, $5, $6, 'active', $7, $8, $9, $10, $11,
				        $12, $12, 1, $12, $12)
			`, id, userID, readbackHash, readbackMasked, method, assurance,
				operation.ID, evidence.SchoolID, evidence.EnrollmentSubjectID,
				evidence.SnapshotID, evidence.SnapshotRevision, now); err != nil {
				return fmt.Errorf("activate phone credential: %w", err)
			}
		}

		if _, err := tx.Exec(ctx, `
			DELETE FROM phone_number_claims
			WHERE user_id = $1 AND claim_status IN ('active', 'releasing')
			  AND ($2::text IS NULL OR phone_hash <> $2)
		`, userID, readbackHash); err != nil {
			return err
		}
		if operation.OperationKind != PhoneOperationUnbind {
			result, err := tx.Exec(ctx, `
				UPDATE phone_number_claims
				SET claim_status = 'active', credential_id = $3, operation_id = NULL,
				    expires_at = NULL, revision = revision + 1, updated_at = $4
				WHERE phone_hash = $1 AND user_id = $2 AND claim_status = 'pending'
			`, readbackHash, userID, credentialID, now)
			if err != nil {
				return err
			}
			if result.RowsAffected() != 1 {
				return ErrPhoneOwnershipConflict
			}
		} else {
			if _, err := tx.Exec(ctx, `DELETE FROM phone_number_claims WHERE user_id = $1`, userID); err != nil {
				return err
			}
		}

		if _, err := tx.Exec(ctx, `
			UPDATE phone_binding_operations
			SET status = 'completed', target_phone_enc = NULL,
			    encryption_key_version = NULL, projection_synced_at = $2,
			    completed_at = $2, revision = revision + 1, updated_at = $2
			WHERE id = $1
		`, operation.ID, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE phone_binding_outbox
			SET status = 'completed', lease_owner = NULL, lease_expires_at = NULL,
			    completed_at = $2, updated_at = $2
			WHERE operation_id = $1 AND status IN ('pending', 'processing')
		`, operation.ID, now); err != nil {
			return err
		}
		return r.bumpPhoneEligibilityTx(ctx, tx, userID, "phone_operation_completed", now)
	})
}

func (r *Repository) phoneCredentialEvidenceTx(
	ctx context.Context,
	tx pgx.Tx,
	operation *PhoneBindingOperation,
) (phoneCredentialEvidence, error) {
	if operation.VerificationMethod == nil || *operation.VerificationMethod != PhoneMethodRosterMatch {
		return phoneCredentialEvidence{}, nil
	}
	var evidence phoneCredentialEvidence
	err := tx.QueryRow(ctx, `
		SELECT school_id, enrollment_subject_id, roster_snapshot_id, roster_snapshot_revision
		FROM phone_verification_attempts
		WHERE operation_id = $1 AND method = 'school_roster_phone_match' AND status = 'succeeded'
		ORDER BY attempt_number DESC
		LIMIT 1
	`, operation.ID).Scan(
		&evidence.SchoolID,
		&evidence.EnrollmentSubjectID,
		&evidence.SnapshotID,
		&evidence.SnapshotRevision,
	)
	return evidence, err
}

func (r *Repository) GetPhoneStatus(ctx context.Context, userID int64, now time.Time) (*PhoneStatus, error) {
	ctx = withTable(ctx, "phone_verification_credentials")
	var (
		projectionState string
		masked          *string
		method          *string
		verifiedAt      *time.Time
		expiresAt       *time.Time
		revision        int64
	)
	err := r.db.QueryRow(ctx, `
		SELECT u.phone_projection_state, u.phone_masked,
		       credential.method, credential.verified_at, credential.expires_at,
		       GREATEST(u.phone_projection_revision, COALESCE(fence.revision, 1))
		FROM users u
		LEFT JOIN phone_eligibility_revisions fence ON fence.user_id = u.id
		LEFT JOIN LATERAL (
		    SELECT c.method, c.verified_at, c.expires_at
		    FROM phone_verification_credentials c
		    WHERE c.user_id = u.id
		      AND c.status = 'active'
		      AND c.revoked_at IS NULL
		      AND (c.expires_at IS NULL OR c.expires_at > $2)
		      AND c.phone_hash = u.phone_hash
		    ORDER BY c.verified_at DESC, c.id DESC
		    LIMIT 1
		) credential ON true
		WHERE u.id = $1
	`, userID, now).Scan(&projectionState, &masked, &method, &verifiedAt, &expiresAt, &revision)
	if err != nil {
		return nil, err
	}
	result := &PhoneStatus{
		State: "unbound", MaskedPhone: masked, VerifiedAt: verifiedAt,
		ExpiresAt: expiresAt, Revision: revision,
	}
	switch projectionState {
	case "syncing":
		result.State = "syncing"
	case "synced":
		if method == nil {
			result.State = "review_required"
			break
		}
		result.State = "verified"
		value := PhoneVerificationMethod(*method)
		result.Method = &value
		result.PublishingRequirementSatisfied = true
	case "error", "legacy_unreconciled":
		result.State = "review_required"
	}
	return result, nil
}

func (r *Repository) bumpPhoneEligibilityTx(
	ctx context.Context,
	tx pgx.Tx,
	userID int64,
	reason string,
	now time.Time,
) error {
	var revision int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO phone_eligibility_revisions (user_id, revision, reason_code, updated_at)
		VALUES ($1, 1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE
		SET revision = phone_eligibility_revisions.revision + 1,
		    reason_code = EXCLUDED.reason_code,
		    updated_at = EXCLUDED.updated_at
		RETURNING revision
	`, userID, reason, now).Scan(&revision); err != nil {
		return err
	}
	eventID, err := newID()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO phone_eligibility_event_outbox (
		    event_id, user_id, revision, event_type, status, available_at,
		    created_at, updated_at
		)
		VALUES ($1, $2, $3, 'phone_eligibility.changed', 'pending', $4, $4, $4)
	`, eventID, userID, revision, now)
	return err
}

func (r *Repository) enqueuePhoneOperationTx(
	ctx context.Context,
	tx pgx.Tx,
	operationID string,
	revision int64,
	now time.Time,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO phone_binding_outbox (
		    operation_id, action_kind, action_revision, status,
		    available_at, created_at, updated_at
		)
		VALUES ($1, 'apply_casdoor', $2, 'pending', $3, $3, $3)
		ON CONFLICT (operation_id, action_kind, action_revision) DO NOTHING
	`, operationID, revision, now)
	return err
}

func (r *Repository) ClaimPhoneOutboxJobs(
	ctx context.Context,
	owner string,
	limit int,
	lease time.Duration,
	now time.Time,
) ([]phoneOutboxJob, error) {
	ctx = withTable(ctx, "phone_binding_outbox")
	rows, err := r.db.Query(ctx, `
		WITH candidates AS (
		    SELECT id
		    FROM phone_binding_outbox
		    WHERE (
		        status = 'pending' AND available_at <= $1
		    ) OR (
		        status = 'processing' AND lease_expires_at <= $1
		    )
		    ORDER BY available_at, id
		    FOR UPDATE SKIP LOCKED
		    LIMIT $2
		)
		UPDATE phone_binding_outbox outbox
		SET status = 'processing', lease_owner = $3,
		    lease_expires_at = $1 + make_interval(secs => $4),
		    attempt_count = outbox.attempt_count + 1,
		    updated_at = $1
		FROM candidates
		WHERE outbox.id = candidates.id
		RETURNING outbox.id, outbox.operation_id, outbox.action_revision, outbox.attempt_count
	`, now, limit, owner, int(lease.Seconds()))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := make([]phoneOutboxJob, 0, limit)
	for rows.Next() {
		var job phoneOutboxJob
		if err := rows.Scan(&job.ID, &job.OperationID, &job.ActionRevision, &job.AttemptCount); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *Repository) ResolvePhoneOperationUser(ctx context.Context, operationID string) (int64, error) {
	ctx = withTable(ctx, "phone_binding_operations")
	var userID int64
	err := r.db.QueryRow(ctx, `SELECT user_id FROM phone_binding_operations WHERE id = $1`, operationID).Scan(&userID)
	return userID, err
}

func (r *Repository) RetryPhoneOutboxJob(
	ctx context.Context,
	jobID int64,
	owner string,
	availableAt time.Time,
	errorCode string,
	now time.Time,
) error {
	result, err := r.db.Exec(withTable(ctx, "phone_binding_outbox"), `
		UPDATE phone_binding_outbox
		SET status = 'pending', available_at = $3, lease_owner = NULL,
		    lease_expires_at = NULL, last_error_code = $4, updated_at = $5
		WHERE id = $1 AND status = 'processing' AND lease_owner = $2
	`, jobID, owner, availableAt, errorCode, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return errors.New("phone outbox lease lost")
	}
	return nil
}

func (r *Repository) CompletePhoneOutboxJob(ctx context.Context, jobID int64, owner string, now time.Time) error {
	result, err := r.db.Exec(withTable(ctx, "phone_binding_outbox"), `
		UPDATE phone_binding_outbox
		SET status = 'completed', lease_owner = NULL, lease_expires_at = NULL,
		    completed_at = $3, updated_at = $3
		WHERE id = $1 AND status = 'processing' AND lease_owner = $2
	`, jobID, owner, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return errors.New("phone outbox lease lost")
	}
	return nil
}
