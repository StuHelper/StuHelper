package studentverification

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
)

type currentRosterAggregate struct {
	SnapshotID     string
	SourceCutoffAt time.Time
	RowCount       int64
}

func (r *Repository) GetRosterImportConfig(ctx context.Context, schoolCode string) (*rosterImportConfig, error) {
	ctx = withTable(ctx, "school_verification_profiles")
	var config rosterImportConfig
	var policy []byte
	err := r.db.QueryRow(ctx, `
		SELECT s.id, s.code, s.name, p.adapter_id, p.adapter_version,
		       p.enrollment_policy
		FROM school_verification_profiles p
		JOIN schools s ON s.id = p.school_id
		WHERE s.code = $1
	`, schoolCode).Scan(
		&config.SchoolID, &config.SchoolCode, &config.SchoolName,
		&config.AdapterID, &config.AdapterVersion, &policy,
	)
	if err != nil {
		return nil, err
	}
	config.EnrollmentPolicy = append(json.RawMessage(nil), policy...)
	return &config, nil
}

func (r *Repository) CreateRosterSnapshot(
	ctx context.Context,
	snapshot RosterSnapshot,
	sourceStartedAt *time.Time,
	connectorNodeID *string,
	encryptionKeyVersion int,
	hmacKeyVersion int,
	signatureAlgorithm string,
	signatureKeyID string,
	snapshotSignature []byte,
) error {
	var signatureAlgorithmValue, signatureKeyIDValue *string
	var snapshotSignatureValue []byte
	if signatureAlgorithm != "" {
		signatureAlgorithmValue = &signatureAlgorithm
		signatureKeyIDValue = &signatureKeyID
		snapshotSignatureValue = snapshotSignature
	}
	ctx = withTable(ctx, "student_roster_snapshots")
	_, err := r.db.Exec(ctx, `
		INSERT INTO academic.student_roster_snapshots (
		    id, school_id, source_kind, source_version, import_mode,
		    schema_version, mapping_version, status, source_started_at,
		    source_cutoff_at, import_started_at, checksum,
		    encryption_key_version, hmac_key_version, connector_node_id,
		    signature_algorithm, signature_key_id, snapshot_signature,
		    created_at, updated_at
		)
		VALUES (
		    $1, $2, $3, $4, $5,
		    $6, $7, 'staging', $8,
		    $9, $10, $11,
		    $12, $13, $14,
		    $15, $16, $17,
		    $10, $10
		)
	`,
		snapshot.ID, snapshot.SchoolID, snapshot.SourceKind, snapshot.SourceVersion,
		snapshot.ImportMode, snapshot.SchemaVersion, snapshot.MappingVersion,
		sourceStartedAt, snapshot.SourceCutoffAt, snapshot.ImportStartedAt,
		snapshot.Checksum, encryptionKeyVersion, hmacKeyVersion, connectorNodeID,
		signatureAlgorithmValue, signatureKeyIDValue, snapshotSignatureValue,
	)
	if err != nil {
		return fmt.Errorf("create roster snapshot: %w", err)
	}
	return nil
}

func (r *Repository) GetRosterSnapshotBySource(
	ctx context.Context,
	schoolID int64,
	sourceKind string,
	sourceVersion string,
) (*RosterSnapshot, error) {
	return r.getRosterSnapshot(ctx, `
		WHERE snapshot.school_id = $1
		  AND snapshot.source_kind = $2
		  AND snapshot.source_version = $3
	`, schoolID, sourceKind, sourceVersion)
}

func (r *Repository) GetRosterSnapshot(ctx context.Context, snapshotID string) (*RosterSnapshot, error) {
	return r.getRosterSnapshot(ctx, `WHERE snapshot.id = $1`, snapshotID)
}

func (r *Repository) ListRosterSnapshots(
	ctx context.Context,
	schoolCode string,
	limit int,
	offset int,
) ([]RosterSnapshot, error) {
	ctx = withTable(ctx, "student_roster_snapshots")
	rows, err := r.db.Query(ctx, rosterSnapshotSelectSQL()+`
		WHERE school.code = $1
		ORDER BY snapshot.source_cutoff_at DESC, snapshot.created_at DESC, snapshot.id DESC
		LIMIT $2 OFFSET $3
	`, schoolCode, limit, offset)
	if err != nil {
		return nil, err
	}
	snapshots := make([]RosterSnapshot, 0)
	for rows.Next() {
		snapshot, scanErr := scanRosterSnapshot(rows)
		if scanErr != nil {
			rows.Close()
			return nil, scanErr
		}
		snapshots = append(snapshots, *snapshot)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for index := range snapshots {
		checks, checkErr := r.listRosterQualityChecks(ctx, snapshots[index].ID)
		if checkErr != nil {
			return nil, checkErr
		}
		snapshots[index].QualityChecks = checks
	}
	return snapshots, nil
}

func (r *Repository) getRosterSnapshot(ctx context.Context, clause string, args ...any) (*RosterSnapshot, error) {
	ctx = withTable(ctx, "student_roster_snapshots")
	snapshot, err := scanRosterSnapshot(r.db.QueryRow(ctx, rosterSnapshotSelectSQL()+clause, args...))
	if err != nil {
		return nil, err
	}
	checks, err := r.listRosterQualityChecks(ctx, snapshot.ID)
	if err != nil {
		return nil, err
	}
	snapshot.QualityChecks = checks
	return snapshot, nil
}

func rosterSnapshotSelectSQL() string {
	return `
		SELECT snapshot.id, snapshot.school_id, school.code, school.name,
		       snapshot.source_kind, snapshot.source_version,
		       snapshot.import_mode, snapshot.schema_version,
		       snapshot.mapping_version, snapshot.status,
		       snapshot.source_cutoff_at, snapshot.import_started_at,
		       snapshot.import_completed_at, snapshot.activated_at,
		       snapshot.row_count, snapshot.eligible_row_count,
		       snapshot.deleted_row_count, snapshot.checksum,
		       snapshot.failure_code, active.activation_revision,
		       COALESCE(active.snapshot_id = snapshot.id, false),
		       snapshot.created_at, snapshot.updated_at
		FROM academic.student_roster_snapshots snapshot
		JOIN schools school ON school.id = snapshot.school_id
		LEFT JOIN academic.student_roster_active active
		  ON active.school_id = snapshot.school_id
		 AND active.snapshot_id = snapshot.id
	`
}

func scanRosterSnapshot(row rowScanner) (*RosterSnapshot, error) {
	var snapshot RosterSnapshot
	err := row.Scan(
		&snapshot.ID, &snapshot.SchoolID, &snapshot.SchoolCode, &snapshot.SchoolName,
		&snapshot.SourceKind, &snapshot.SourceVersion, &snapshot.ImportMode,
		&snapshot.SchemaVersion, &snapshot.MappingVersion, &snapshot.Status,
		&snapshot.SourceCutoffAt, &snapshot.ImportStartedAt, &snapshot.ImportCompletedAt,
		&snapshot.ActivatedAt, &snapshot.RowCount, &snapshot.EligibleRowCount,
		&snapshot.DeletedRowCount, &snapshot.Checksum, &snapshot.FailureCode,
		&snapshot.ActivationRevision, &snapshot.IsCurrent, &snapshot.CreatedAt,
		&snapshot.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	snapshot.QualityChecks = []RosterQualityCheck{}
	return &snapshot, nil
}

func (r *Repository) listRosterQualityChecks(ctx context.Context, snapshotID string) ([]RosterQualityCheck, error) {
	rows, err := r.db.Query(ctx, `
		SELECT check_key, status, measured, threshold,
		       COALESCE(detail_code, ''), checked_at
		FROM academic.student_roster_quality_checks
		WHERE snapshot_id = $1
		ORDER BY check_key
	`, snapshotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	checks := make([]RosterQualityCheck, 0)
	for rows.Next() {
		var check RosterQualityCheck
		var measuredRaw, thresholdRaw []byte
		if err := rows.Scan(
			&check.CheckKey, &check.Status, &measuredRaw, &thresholdRaw,
			&check.DetailCode, &check.CheckedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(measuredRaw, &check.Measured); err != nil {
			return nil, fmt.Errorf("decode roster quality measurement: %w", err)
		}
		if err := json.Unmarshal(thresholdRaw, &check.Threshold); err != nil {
			return nil, fmt.Errorf("decode roster quality threshold: %w", err)
		}
		checks = append(checks, check)
	}
	return checks, rows.Err()
}

func (r *Repository) GetCurrentRosterAggregate(ctx context.Context, schoolID int64) (*currentRosterAggregate, error) {
	ctx = withTable(ctx, "student_roster_active")
	var current currentRosterAggregate
	err := r.db.QueryRow(ctx, `
		SELECT snapshot.id, snapshot.source_cutoff_at, snapshot.row_count
		FROM academic.student_roster_active active
		JOIN academic.student_roster_snapshots snapshot
		  ON snapshot.id = active.snapshot_id
		 AND snapshot.school_id = active.school_id
		WHERE active.school_id = $1
		  AND snapshot.status = 'active'
	`, schoolID).Scan(&current.SnapshotID, &current.SourceCutoffAt, &current.RowCount)
	if err != nil {
		return nil, err
	}
	return &current, nil
}

func (r *Repository) FailRosterSnapshot(
	ctx context.Context,
	snapshotID string,
	checkKey string,
	detailCode string,
	now time.Time,
) error {
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		result, err := tx.Exec(ctx, `
			UPDATE academic.student_roster_snapshots
			SET status = 'failed', failure_code = $2,
			    import_completed_at = $3, updated_at = $3
			WHERE id = $1 AND status IN ('staging', 'validating')
		`, snapshotID, detailCode, now)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return ErrRosterSnapshotState
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO academic.student_roster_quality_checks (
			    snapshot_id, check_key, status, measured, threshold,
			    detail_code, checked_at, created_at
			)
			VALUES ($1, $2, 'failed', '{}'::jsonb, '{}'::jsonb, $3, $4, $4)
			ON CONFLICT (snapshot_id, check_key) DO UPDATE
			SET status = 'failed', detail_code = EXCLUDED.detail_code,
			    checked_at = EXCLUDED.checked_at
		`, snapshotID, checkKey, detailCode, now)
		return err
	})
}

func (r *Repository) StageRosterRecords(
	ctx context.Context,
	snapshot RosterSnapshot,
	prepared preparedRosterImport,
	encryptionKeyVersion int,
	hmacKeyVersion int,
	now time.Time,
) error {
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		result, err := tx.Exec(ctx, `
			UPDATE academic.student_roster_snapshots
			SET status = 'validating', updated_at = $2
			WHERE id = $1 AND school_id = $3 AND status = 'staging'
		`, snapshot.ID, now, snapshot.SchoolID)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ErrRosterSnapshotState
		}

		columns := []string{
			"snapshot_id", "school_id", "source_record_key_hash",
			"student_id_enc", "student_id_hash", "name_enc", "name_hash",
			"document_type", "document_number_enc", "document_number_hash",
			"phone_enc", "phone_hash", "encryption_key_version", "hmac_key_version",
			"student_status", "on_campus_status", "registration_status",
			"education_level", "student_category", "enrollment_year",
			"valid_from", "valid_until", "current_marker", "eligibility_status",
			"eligibility_code", "source_updated_at", "record_checksum",
			"created_at", "updated_at",
		}
		_, err = tx.CopyFrom(
			ctx,
			pgx.Identifier{"academic", "student_roster_records"},
			columns,
			pgx.CopyFromSlice(len(prepared.Rows), func(index int) ([]any, error) {
				row := prepared.Rows[index]
				return []any{
					snapshot.ID, snapshot.SchoolID, row.SourceRecordKeyHash,
					row.StudentIDEnc, row.StudentIDHash, row.NameEnc, row.NameHash,
					row.DocumentType, nullableBytes(row.DocumentNumberEnc), row.DocumentNumberHash,
					nullableBytes(row.PhoneEnc), row.PhoneHash,
					encryptionKeyVersion, hmacKeyVersion,
					row.StudentStatus, row.OnCampusStatus, row.RegistrationStatus,
					row.EducationLevel, row.StudentCategory, row.EnrollmentYear,
					row.ValidFrom, row.ValidUntil, row.CurrentMarker,
					row.EligibilityStatus, row.EligibilityCode, row.SourceUpdatedAt,
					row.RecordChecksum, now, now,
				}, nil
			}),
		)
		if err != nil {
			return fmt.Errorf("copy roster records: %w", err)
		}

		for _, check := range prepared.Checks {
			measured, marshalErr := json.Marshal(check.Measured)
			if marshalErr != nil {
				return marshalErr
			}
			threshold, marshalErr := json.Marshal(check.Threshold)
			if marshalErr != nil {
				return marshalErr
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO academic.student_roster_quality_checks (
				    snapshot_id, check_key, status, measured, threshold,
				    detail_code, checked_at, created_at
				)
				VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, NULLIF($6, ''), $7, $7)
			`, snapshot.ID, check.CheckKey, check.Status, measured, threshold,
				check.DetailCode, check.CheckedAt)
			if err != nil {
				return fmt.Errorf("insert roster quality check: %w", err)
			}
		}

		var previousCount int64
		err = tx.QueryRow(ctx, `
			SELECT COALESCE((
			    SELECT current.row_count
			    FROM academic.student_roster_active active
			    JOIN academic.student_roster_snapshots current
			      ON current.id = active.snapshot_id
			     AND current.school_id = active.school_id
			    WHERE active.school_id = $1
			), 0)
		`, snapshot.SchoolID).Scan(&previousCount)
		if err != nil {
			return err
		}
		deletedCount := previousCount - int64(len(prepared.Rows))
		if deletedCount < 0 {
			deletedCount = 0
		}
		status := "ready"
		var failureCode *string
		if hasFailedQualityCheck(prepared.Checks) {
			status = "failed"
			value := firstFailedRosterQualityCode(prepared.Checks)
			failureCode = &value
		}
		result, err = tx.Exec(ctx, `
			UPDATE academic.student_roster_snapshots
			SET status = $2, row_count = $3, eligible_row_count = $4,
			    deleted_row_count = $5, checksum = $6,
			    failure_code = $7, import_completed_at = $8, updated_at = $8
			WHERE id = $1 AND status = 'validating'
		`, snapshot.ID, status, int64(len(prepared.Rows)), prepared.EligibleCount,
			deletedCount, prepared.Checksum, failureCode, now)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ErrRosterSnapshotState
		}
		return nil
	})
}

func nullableBytes(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func firstFailedRosterQualityCode(checks []RosterQualityCheck) string {
	for _, check := range checks {
		if check.Status == "failed" {
			if check.DetailCode != "" {
				return check.DetailCode
			}
			return "quality_gate_failed"
		}
	}
	return "quality_gate_failed"
}

func (r *Repository) SwitchRosterSnapshot(
	ctx context.Context,
	input RosterSnapshotSwitchInput,
	rollback bool,
	now time.Time,
) (*RosterSnapshot, error) {
	actorUserID := input.ActorUserID
	return r.switchRosterSnapshot(ctx, rosterSnapshotSwitchCommand{
		SchoolCode: input.SchoolCode, SnapshotID: input.SnapshotID,
		ActorUserID: &actorUserID, ActorType: "admin", Reason: input.Reason,
		AllowSourceRegression: input.AllowSourceRegression,
	}, rollback, now)
}

func (r *Repository) AutoActivateRosterSnapshot(
	ctx context.Context,
	schoolCode string,
	snapshotID string,
	now time.Time,
) (*RosterSnapshot, error) {
	return r.switchRosterSnapshot(ctx, rosterSnapshotSwitchCommand{
		SchoolCode: schoolCode, SnapshotID: snapshotID, ActorType: "system",
		Reason:                "automatic activation after all roster quality gates passed",
		RequireAutoActivation: true,
	}, false, now)
}

func (r *Repository) switchRosterSnapshot(
	ctx context.Context,
	input rosterSnapshotSwitchCommand,
	rollback bool,
	now time.Time,
) (*RosterSnapshot, error) {
	var activatedID string
	err := r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var (
			targetID          string
			schoolID          int64
			targetStatus      string
			targetCutoff      time.Time
			hardExpirySeconds int
			profileValidation string
			autoActivate      bool
		)
		err := tx.QueryRow(ctx, `
			SELECT snapshot.id, snapshot.school_id, snapshot.status,
			       snapshot.source_cutoff_at, profile.snapshot_hard_expiry_seconds,
			       profile.validation_status, profile.snapshot_auto_activate
			FROM academic.student_roster_snapshots snapshot
			JOIN schools school ON school.id = snapshot.school_id
			JOIN school_verification_profiles profile
			  ON profile.school_id = snapshot.school_id
			WHERE snapshot.id = $1 AND school.code = $2
			FOR UPDATE OF snapshot, profile
		`, input.SnapshotID, input.SchoolCode).Scan(
			&targetID, &schoolID, &targetStatus, &targetCutoff,
			&hardExpirySeconds, &profileValidation, &autoActivate,
		)
		if err != nil {
			if err == pgx.ErrNoRows {
				return ErrRosterSnapshotNotFound
			}
			return err
		}
		var (
			previousID       *string
			previousCutoff   *time.Time
			previousRevision int64
		)
		err = tx.QueryRow(ctx, `
			SELECT active.snapshot_id, snapshot.source_cutoff_at,
			       active.activation_revision
			FROM academic.student_roster_active active
			JOIN academic.student_roster_snapshots snapshot
			  ON snapshot.id = active.snapshot_id
			 AND snapshot.school_id = active.school_id
			WHERE active.school_id = $1
			FOR UPDATE OF active, snapshot
		`, schoolID).Scan(&previousID, &previousCutoff, &previousRevision)
		if err != nil && err != pgx.ErrNoRows {
			return err
		}
		if err == pgx.ErrNoRows {
			previousID = nil
			previousCutoff = nil
			previousRevision = 0
		}
		if previousID != nil && *previousID == targetID {
			activatedID = targetID
			return nil
		}
		if input.RequireAutoActivation && (!autoActivate || targetStatus != "ready") {
			activatedID = targetID
			return nil
		}
		if profileValidation != "valid" {
			return ErrRosterPolicyInvalid
		}

		if rollback {
			if previousID == nil || (targetStatus != "superseded" && targetStatus != "rolled_back") {
				return ErrRosterSnapshotState
			}
		} else {
			if targetStatus != "ready" {
				if targetStatus != "failed" || !input.AllowSourceRegression {
					return ErrRosterSnapshotState
				}
				var unauthorizedFailures int
				if err := tx.QueryRow(ctx, `
					SELECT COUNT(*)
					FROM academic.student_roster_quality_checks
					WHERE snapshot_id = $1
					  AND status = 'failed'
					  AND check_key <> 'source_time.monotonic'
				`, targetID).Scan(&unauthorizedFailures); err != nil {
					return err
				}
				if unauthorizedFailures != 0 {
					return ErrRosterQualityFailed
				}
				var sourceRegressionFailures int
				if err := tx.QueryRow(ctx, `
					SELECT COUNT(*)
					FROM academic.student_roster_quality_checks
					WHERE snapshot_id = $1
					  AND check_key = 'source_time.monotonic'
					  AND status = 'failed'
				`, targetID).Scan(&sourceRegressionFailures); err != nil {
					return err
				}
				if sourceRegressionFailures != 1 {
					return ErrRosterQualityFailed
				}
			}
			if targetCutoff.Add(time.Duration(hardExpirySeconds) * time.Second).Before(now) {
				return ErrRosterQualityFailed
			}
			if previousCutoff != nil && targetCutoff.Before(*previousCutoff) &&
				!input.AllowSourceRegression {
				return ErrRosterSourceRegression
			}
		}

		var nonPassingChecks int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM academic.student_roster_quality_checks
			WHERE snapshot_id = $1 AND status IN ('pending', 'failed')
		`, targetID).Scan(&nonPassingChecks); err != nil {
			return err
		}
		if nonPassingChecks != 0 && (!input.AllowSourceRegression || rollback || nonPassingChecks != 1) {
			return ErrRosterQualityFailed
		}

		if previousID != nil {
			previousState := "superseded"
			if rollback {
				previousState = "rolled_back"
			}
			if _, err := tx.Exec(ctx, `
				UPDATE academic.student_roster_snapshots
				SET status = $2, updated_at = $3
				WHERE id = $1 AND status = 'active'
			`, *previousID, previousState, now); err != nil {
				return err
			}
		}
		if input.AllowSourceRegression && !rollback && targetStatus == "failed" {
			if _, err := tx.Exec(ctx, `
				UPDATE academic.student_roster_quality_checks
				SET status = 'warning', detail_code = 'source_time_regression_authorized',
				    checked_at = $2
				WHERE snapshot_id = $1 AND check_key = 'source_time.monotonic'
			`, targetID, now); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `
			UPDATE academic.student_roster_snapshots
			SET status = 'active', activated_at = $2,
			    activation_authorized_by_user_id = $3,
			    failure_code = NULL, updated_at = $2
			WHERE id = $1
		`, targetID, now, input.ActorUserID); err != nil {
			return err
		}

		var revision int64
		err = tx.QueryRow(ctx, `
			INSERT INTO academic.student_roster_active (
			    school_id, snapshot_id, previous_snapshot_id,
			    activation_revision, activated_at, activated_by_user_id, updated_at
			)
			VALUES ($1, $2, $3, 1, $4, $5, $4)
			ON CONFLICT (school_id) DO UPDATE
			SET snapshot_id = EXCLUDED.snapshot_id,
			    previous_snapshot_id = academic.student_roster_active.snapshot_id,
			    activation_revision = academic.student_roster_active.activation_revision + 1,
			    activated_at = EXCLUDED.activated_at,
			    activated_by_user_id = EXCLUDED.activated_by_user_id,
			    updated_at = EXCLUDED.updated_at
			RETURNING activation_revision
		`, schoolID, targetID, previousID, now, input.ActorUserID).Scan(&revision)
		if err != nil {
			return err
		}
		action := "activate"
		if rollback {
			action = "rollback"
		}
		if err := insertRosterSwitchAuditTx(
			ctx, tx, action, input, schoolID, previousID, targetID,
			previousRevision, revision, now,
		); err != nil {
			return err
		}
		// The active pointer and every roster-dependent subject, credential,
		// revision and outbox event are one authority change. Keeping this in
		// the same transaction prevents a newly active snapshot from becoming
		// visible while consumers still observe eligibility derived from the
		// previous one.
		if err := r.reevaluateRosterDependentsTx(ctx, tx, schoolID, now); err != nil {
			return err
		}
		activatedID = targetID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetRosterSnapshot(ctx, activatedID)
}

func insertRosterSwitchAuditTx(
	ctx context.Context,
	tx pgx.Tx,
	action string,
	input rosterSnapshotSwitchCommand,
	schoolID int64,
	previousID *string,
	targetID string,
	previousRevision int64,
	revision int64,
	now time.Time,
) error {
	eventID, err := newID()
	if err != nil {
		return err
	}
	actorUserID := ""
	category := "domain_event"
	if input.ActorUserID != nil {
		actorUserID = strconv.FormatInt(*input.ActorUserID, 10)
		category = "admin_operation"
	}
	event := audit.EventFromContext(ctx, audit.Event{
		Type:          audit.EventType("student_roster.snapshot." + action),
		Category:      category,
		ActorType:     input.ActorType,
		UserID:        actorUserID,
		Action:        action,
		ResourceType:  "student_roster_snapshot",
		ResourceID:    targetID,
		ScopeSchoolID: strconv.FormatInt(schoolID, 10),
		Result:        "success",
		Reason:        input.Reason,
		Before: map[string]any{
			"snapshotID": previousID,
			"revision":   previousRevision,
		},
		After: map[string]any{
			"snapshotID": targetID,
			"revision":   revision,
		},
		Details: map[string]any{
			"sourceRegressionOverride": input.AllowSourceRegression,
			"automaticActivation":      input.RequireAutoActivation,
		},
		Timestamp: now,
	})
	beforeJSON, err := json.Marshal(event.Before)
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(event.After)
	if err != nil {
		return err
	}
	detailsJSON, err := json.Marshal(event.Details)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events (
		    id, category, event_type, actor_type, actor_user_id,
		    actor_username, action, resource_type, resource_id,
		    scope_school_id, before_data, after_data, result, reason,
		    trace_id, request_id, details, created_at
		)
		VALUES (
		    $1, $2, $3, $4, $5,
		    '', $6, $7, $8,
		    $9, $10::jsonb, $11::jsonb, $12, $13,
		    NULLIF($14, ''), NULLIF($15, ''), $16::jsonb, $17
		)
	`, eventID, event.Category, event.Type, event.ActorType, event.UserID,
		event.Action, event.ResourceType, event.ResourceID, event.ScopeSchoolID,
		beforeJSON, afterJSON, event.Result, event.Reason,
		event.TraceID, event.RequestID, detailsJSON, event.Timestamp)
	return err
}

func (r *Repository) ReevaluateRosterDependents(ctx context.Context, schoolID int64, now time.Time) error {
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return r.reevaluateRosterDependentsTx(ctx, tx, schoolID, now)
	})
}

func (r *Repository) reevaluateRosterDependentsTx(
	ctx context.Context,
	tx pgx.Tx,
	schoolID int64,
	now time.Time,
) error {
	studentUsers := make(map[int64]struct{})
	phoneUsers := make(map[int64]struct{})

	rows, err := tx.Query(ctx, `
			WITH validity AS (
			    SELECT subject.id,
			           EXISTS (
			               SELECT 1
			               FROM academic.student_roster_active active
			               JOIN academic.student_roster_records record
			                 ON record.snapshot_id = active.snapshot_id
			                AND record.school_id = active.school_id
			               WHERE active.school_id = subject.school_id
			                 AND record.student_id_hash = subject.student_id_hash
			                 AND record.eligibility_status = 'eligible'
			           ) AS eligible
			    FROM student_enrollment_subjects subject
			    WHERE subject.school_id = $1
			      AND (
			          subject.binding_status = 'active'
			          OR (
			              subject.binding_status = 'review_required'
			              AND subject.review_required_reason = 'roster_ineligible'
			          )
			      )
			)
			UPDATE student_enrollment_subjects subject
			SET binding_status = CASE WHEN validity.eligible THEN 'active' ELSE 'review_required' END,
			    review_required_at = CASE WHEN validity.eligible THEN NULL::timestamptz ELSE $2::timestamptz END,
			    review_required_reason = CASE WHEN validity.eligible THEN NULL::text ELSE 'roster_ineligible'::text END,
			    revision = revision + 1, updated_at = $2
			FROM validity
			WHERE subject.id = validity.id
			  AND (
			      (validity.eligible AND subject.binding_status <> 'active')
			      OR (NOT validity.eligible AND subject.binding_status <> 'review_required')
			  )
			RETURNING subject.user_id
	`, schoolID, now)
	if err != nil {
		return err
	}
	if err := collectUserIDs(rows, studentUsers); err != nil {
		return err
	}

	rows, err = tx.Query(ctx, `
			WITH validity AS (
			    SELECT credential.id,
			           EXISTS (
			               SELECT 1
			               FROM student_enrollment_subjects subject
			               JOIN academic.student_roster_active active
			                 ON active.school_id = subject.school_id
			               JOIN academic.student_roster_records record
			                 ON record.snapshot_id = active.snapshot_id
			                AND record.school_id = active.school_id
			                AND record.student_id_hash = subject.student_id_hash
			               WHERE subject.id = credential.enrollment_subject_id
			                 AND subject.binding_status = 'active'
			                 AND record.eligibility_status = 'eligible'
			           ) AS eligible
			    FROM user_verification_credentials credential
			    WHERE credential.school_id = $1
			      AND credential.roster_dependency = 'required'
			      AND credential.revoked_at IS NULL
			      AND (
			          credential.status = 'active'
			          OR (
			              credential.status = 'review_required'
			              AND credential.review_required_reason = 'roster_ineligible'
			          )
			      )
			)
			UPDATE user_verification_credentials credential
			SET status = CASE WHEN validity.eligible THEN 'active' ELSE 'review_required' END,
			    review_required_at = CASE WHEN validity.eligible THEN NULL::timestamptz ELSE $2::timestamptz END,
			    review_required_reason = CASE WHEN validity.eligible THEN NULL::text ELSE 'roster_ineligible'::text END,
			    revision = revision + 1, last_evaluated_at = $2, updated_at = $2
			FROM validity
			WHERE credential.id = validity.id
			  AND (
			      (validity.eligible AND credential.status <> 'active')
			      OR (NOT validity.eligible AND credential.status <> 'review_required')
			  )
			RETURNING credential.user_id
	`, schoolID, now)
	if err != nil {
		return err
	}
	if err := collectUserIDs(rows, studentUsers); err != nil {
		return err
	}

	rows, err = tx.Query(ctx, `
			WITH validity AS (
			    SELECT credential.id,
			           EXISTS (
			               SELECT 1
			               FROM student_enrollment_subjects subject
			               JOIN academic.student_roster_active active
			                 ON active.school_id = subject.school_id
			               JOIN academic.student_roster_records record
			                 ON record.snapshot_id = active.snapshot_id
			                AND record.school_id = active.school_id
			                AND record.student_id_hash = subject.student_id_hash
			               WHERE subject.id = credential.enrollment_subject_id
			                 AND subject.binding_status = 'active'
			                 AND record.eligibility_status = 'eligible'
			                 AND record.phone_hash = credential.phone_hash
			           ) AS eligible
			    FROM phone_verification_credentials credential
			    WHERE credential.school_id = $1
			      AND credential.method = 'school_roster_phone_match'
			      AND credential.revoked_at IS NULL
			      AND (
			          credential.status = 'active'
			          OR (
			              credential.status = 'review_required'
			              AND credential.review_required_reason = 'roster_ineligible'
			          )
			      )
			)
			UPDATE phone_verification_credentials credential
			SET status = CASE WHEN validity.eligible THEN 'active' ELSE 'review_required' END,
			    review_required_at = CASE WHEN validity.eligible THEN NULL::timestamptz ELSE $2::timestamptz END,
			    review_required_reason = CASE WHEN validity.eligible THEN NULL::text ELSE 'roster_ineligible'::text END,
			    revision = revision + 1, last_confirmed_at = $2, updated_at = $2
			FROM validity
			WHERE credential.id = validity.id
			  AND (
			      (validity.eligible AND credential.status <> 'active')
			      OR (NOT validity.eligible AND credential.status <> 'review_required')
			  )
			RETURNING credential.user_id
	`, schoolID, now)
	if err != nil {
		return err
	}
	if err := collectUserIDs(rows, phoneUsers); err != nil {
		return err
	}

	for userID := range studentUsers {
		if err := r.BumpEligibilityRevisionTx(
			ctx, tx, userID, schoolID, "roster_snapshot_changed", now,
		); err != nil {
			return err
		}
	}
	for userID := range phoneUsers {
		if err := r.bumpPhoneEligibilityTx(ctx, tx, userID, "roster_snapshot_changed", now); err != nil {
			return err
		}
	}
	return nil
}

func collectUserIDs(rows pgx.Rows, target map[int64]struct{}) error {
	defer rows.Close()
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return err
		}
		target[userID] = struct{}{}
	}
	return rows.Err()
}
