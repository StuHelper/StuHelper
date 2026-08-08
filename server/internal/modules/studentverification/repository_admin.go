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

func (r *Repository) ListAdminVerificationSchools(ctx context.Context) ([]AdminVerificationSchoolConfig, error) {
	ctx = withTable(ctx, "school_verification_profiles")
	rows, err := r.db.Query(ctx, adminVerificationSchoolSelectSQL()+`
		ORDER BY school.name, school.code
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]AdminVerificationSchoolConfig, 0)
	for rows.Next() {
		profile, scanErr := scanAdminVerificationSchool(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, *profile)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for index := range result {
		methods, methodErr := r.listAdminVerificationMethods(ctx, result[index].SchoolID)
		if methodErr != nil {
			return nil, methodErr
		}
		result[index].Methods = methods
	}
	return result, nil
}

func (r *Repository) GetAdminVerificationSchool(ctx context.Context, schoolCode string) (*AdminVerificationSchoolConfig, error) {
	ctx = withTable(ctx, "school_verification_profiles")
	profile, err := scanAdminVerificationSchool(r.db.QueryRow(ctx, adminVerificationSchoolSelectSQL()+`
		WHERE school.code = $1
	`, schoolCode))
	if err != nil {
		return nil, err
	}
	profile.Methods, err = r.listAdminVerificationMethods(ctx, profile.SchoolID)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func adminVerificationSchoolSelectSQL() string {
	return `
		SELECT profile.school_id, school.code, school.name, school.location,
		       profile.adapter_id, profile.adapter_version, profile.email_domains,
		       profile.student_id_policy, profile.name_match_policy,
		       profile.enrollment_policy, profile.manual_form_schema,
		       profile.snapshot_sync_interval_seconds,
		       profile.snapshot_warning_after_seconds,
		       profile.snapshot_hard_expiry_seconds,
		       profile.snapshot_grace_seconds,
		       profile.snapshot_auto_activate,
		       profile.enabled, profile.validation_status, profile.validation_code,
		       profile.config_revision, profile.updated_at
		FROM school_verification_profiles profile
		JOIN schools school ON school.id = profile.school_id
	`
}

func scanAdminVerificationSchool(row rowScanner) (*AdminVerificationSchoolConfig, error) {
	var profile AdminVerificationSchoolConfig
	var studentIDPolicy, nameMatchPolicy, enrollmentPolicy, manualFormSchema []byte
	if err := row.Scan(
		&profile.SchoolID, &profile.SchoolCode, &profile.SchoolName, &profile.Location,
		&profile.AdapterID, &profile.AdapterVersion, &profile.EmailDomains,
		&studentIDPolicy, &nameMatchPolicy, &enrollmentPolicy, &manualFormSchema,
		&profile.SnapshotSyncIntervalSeconds, &profile.SnapshotWarningAfterSeconds,
		&profile.SnapshotHardExpirySeconds, &profile.SnapshotGraceSeconds,
		&profile.SnapshotAutoActivate,
		&profile.Enabled, &profile.ValidationStatus, &profile.ValidationCode,
		&profile.ConfigRevision, &profile.UpdatedAt,
	); err != nil {
		return nil, err
	}
	decoded, err := decodeAdminJSONObject(studentIDPolicy)
	if err != nil {
		return nil, err
	}
	profile.StudentIDPolicy = decoded
	decoded, err = decodeAdminJSONObject(nameMatchPolicy)
	if err != nil {
		return nil, err
	}
	profile.NameMatchPolicy = decoded
	decoded, err = decodeAdminJSONObject(enrollmentPolicy)
	if err != nil {
		return nil, err
	}
	profile.EnrollmentPolicy = decoded
	decoded, err = decodeAdminJSONObject(manualFormSchema)
	if err != nil {
		return nil, err
	}
	profile.ManualFormSchema = decoded
	profile.studentIDPolicyRaw = append(json.RawMessage(nil), studentIDPolicy...)
	profile.nameMatchPolicyRaw = append(json.RawMessage(nil), nameMatchPolicy...)
	profile.enrollmentPolicyRaw = append(json.RawMessage(nil), enrollmentPolicy...)
	profile.manualFormSchemaRaw = append(json.RawMessage(nil), manualFormSchema...)
	profile.Methods = []AdminVerificationMethodConfig{}
	return &profile, nil
}

func (r *Repository) listAdminVerificationMethods(ctx context.Context, schoolID int64) ([]AdminVerificationMethodConfig, error) {
	rows, err := r.db.Query(ctx, adminVerificationMethodSelectSQL()+`
		WHERE method.school_id = $1
		ORDER BY method.method
	`, schoolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]AdminVerificationMethodConfig, 0, 5)
	for rows.Next() {
		method, scanErr := scanAdminVerificationMethod(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, *method)
	}
	return result, rows.Err()
}

func (r *Repository) GetAdminVerificationMethod(ctx context.Context, schoolCode string, method Method) (*AdminVerificationMethodConfig, error) {
	return scanAdminVerificationMethod(r.db.QueryRow(withTable(ctx, "school_verification_methods"), adminVerificationMethodSelectSQL()+`
		WHERE school.code = $1 AND method.method = $2
	`, schoolCode, method))
}

func adminVerificationMethodSelectSQL() string {
	return `
		SELECT method.school_id, school.code, method.method, method.display_name,
		       method.description, method.adapter_id, method.adapter_version,
		       method.roster_dependency, method.conditional_policy,
		       method.public_form_schema, method.risk_policy,
		       method.credential_ttl_seconds, method.connector_operation_key,
		       (method.secret_reference IS NOT NULL), method.privacy_notice_version,
		       method.privacy_notice, method.enabled, method.validation_status,
		       method.validation_code, method.health_status, method.health_code,
		       method.health_checked_at, method.config_revision, method.updated_at
		FROM school_verification_methods method
		JOIN schools school ON school.id = method.school_id
	`
}

func scanAdminVerificationMethod(row rowScanner) (*AdminVerificationMethodConfig, error) {
	var method AdminVerificationMethodConfig
	var methodName string
	var conditionalPolicy, publicFormSchema, riskPolicy, privacyNotice []byte
	if err := row.Scan(
		&method.SchoolID, &method.SchoolCode, &methodName, &method.DisplayName,
		&method.Description, &method.AdapterID, &method.AdapterVersion,
		&method.RosterDependency, &conditionalPolicy, &publicFormSchema,
		&riskPolicy, &method.CredentialTTLSeconds, &method.ConnectorOperationKey,
		&method.SecretConfigured, &method.PrivacyNoticeVersion, &privacyNotice,
		&method.Enabled, &method.ValidationStatus, &method.ValidationCode,
		&method.HealthStatus, &method.HealthCode, &method.HealthCheckedAt,
		&method.ConfigRevision, &method.UpdatedAt,
	); err != nil {
		return nil, err
	}
	method.Method = Method(methodName)
	decoded, err := decodeAdminJSONObject(conditionalPolicy)
	if err != nil {
		return nil, err
	}
	method.ConditionalPolicy = decoded
	decoded, err = decodeAdminJSONObject(publicFormSchema)
	if err != nil {
		return nil, err
	}
	method.PublicFormSchema = decoded
	decoded, err = decodeAdminJSONObject(riskPolicy)
	if err != nil {
		return nil, err
	}
	method.RiskPolicy = decoded
	decoded, err = decodeAdminJSONObject(privacyNotice)
	if err != nil {
		return nil, err
	}
	method.PrivacyNotice = decoded
	method.conditionalPolicyRaw = append(json.RawMessage(nil), conditionalPolicy...)
	method.publicFormSchemaRaw = append(json.RawMessage(nil), publicFormSchema...)
	method.riskPolicyRaw = append(json.RawMessage(nil), riskPolicy...)
	method.privacyNoticeRaw = append(json.RawMessage(nil), privacyNotice...)
	return &method, nil
}

func decodeAdminJSONObject(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var target map[string]any
	if err := json.Unmarshal(raw, &target); err != nil || target == nil {
		return nil, fmt.Errorf("decode verification administration JSON")
	}
	return target, nil
}

func (r *Repository) CreateAdminVerificationSchool(
	ctx context.Context,
	input UpdateAdminVerificationSchoolConfigInput,
) error {
	studentIDPolicy, err := json.Marshal(input.StudentIDPolicy)
	if err != nil {
		return err
	}
	nameMatchPolicy, err := json.Marshal(input.NameMatchPolicy)
	if err != nil {
		return err
	}
	enrollmentPolicy, err := json.Marshal(input.EnrollmentPolicy)
	if err != nil {
		return err
	}
	manualFormSchema, err := json.Marshal(input.ManualFormSchema)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var schoolID int64
		if err := tx.QueryRow(ctx, `
			SELECT id FROM schools WHERE code = $1
		`, input.SchoolCode).Scan(&schoolID); err == pgx.ErrNoRows {
			return ErrSchoolNotFound
		} else if err != nil {
			return err
		}
		var revision int64
		err := tx.QueryRow(ctx, `
			INSERT INTO school_verification_profiles (
			    school_id, adapter_id, adapter_version, email_domains,
			    student_id_policy, name_match_policy, enrollment_policy,
			    manual_form_schema, snapshot_sync_interval_seconds,
			    snapshot_warning_after_seconds, snapshot_hard_expiry_seconds,
			    snapshot_grace_seconds, snapshot_auto_activate, enabled, validation_status,
			    config_revision, created_at, updated_at
			)
			VALUES (
			    $1, $2, $3, $4, $5::jsonb, $6::jsonb, $7::jsonb, $8::jsonb,
			    $9, $10, $11, $12, $13, false, 'pending', 1, $14, $14
			)
			ON CONFLICT (school_id) DO NOTHING
			RETURNING config_revision
		`, schoolID, input.AdapterID, input.AdapterVersion, input.EmailDomains,
			studentIDPolicy, nameMatchPolicy, enrollmentPolicy, manualFormSchema,
			input.SnapshotSyncIntervalSeconds, input.SnapshotWarningAfterSeconds,
			input.SnapshotHardExpirySeconds, input.SnapshotGraceSeconds,
			input.SnapshotAutoActivate, now,
		).Scan(&revision)
		if err == pgx.ErrNoRows {
			return ErrAdminConfigRevision
		}
		if err != nil {
			return err
		}
		return insertStudentVerificationAdminAuditTx(ctx, tx, adminAuditInput{
			ActorUserID: input.ActorUserID, EventType: "student_verification.school_config.created",
			Action: "create", ResourceType: "school_verification_profile",
			ResourceID: input.SchoolCode, SchoolID: schoolID, Reason: input.Reason,
			Before: map[string]any{},
			After:  map[string]any{"revision": revision, "enabled": false, "validationStatus": "pending"},
			Now:    now,
		})
	})
}

func (r *Repository) UpdateAdminVerificationSchool(
	ctx context.Context,
	input UpdateAdminVerificationSchoolConfigInput,
) error {
	studentIDPolicy, err := json.Marshal(input.StudentIDPolicy)
	if err != nil {
		return err
	}
	nameMatchPolicy, err := json.Marshal(input.NameMatchPolicy)
	if err != nil {
		return err
	}
	enrollmentPolicy, err := json.Marshal(input.EnrollmentPolicy)
	if err != nil {
		return err
	}
	manualFormSchema, err := json.Marshal(input.ManualFormSchema)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var schoolID int64
		var revision int64
		err := tx.QueryRow(ctx, `
			UPDATE school_verification_profiles profile
			SET adapter_id = $2, adapter_version = $3, email_domains = $4,
			    student_id_policy = $5::jsonb, name_match_policy = $6::jsonb,
			    enrollment_policy = $7::jsonb, manual_form_schema = $8::jsonb,
			    snapshot_sync_interval_seconds = $9,
			    snapshot_warning_after_seconds = $10,
			    snapshot_hard_expiry_seconds = $11,
			    snapshot_grace_seconds = $12,
			    snapshot_auto_activate = $13,
			    enabled = false, validation_status = 'pending', validation_code = NULL,
			    validated_at = NULL, validated_by_user_id = NULL,
			    config_revision = config_revision + 1, updated_at = $14
			FROM schools school
			WHERE profile.school_id = school.id AND school.code = $1
			  AND profile.config_revision = $15
			RETURNING profile.school_id, profile.config_revision
		`, input.SchoolCode, input.AdapterID, input.AdapterVersion, input.EmailDomains,
			studentIDPolicy, nameMatchPolicy, enrollmentPolicy, manualFormSchema,
			input.SnapshotSyncIntervalSeconds, input.SnapshotWarningAfterSeconds,
			input.SnapshotHardExpirySeconds, input.SnapshotGraceSeconds,
			input.SnapshotAutoActivate, now,
			input.ExpectedRevision).Scan(&schoolID, &revision)
		if err == pgx.ErrNoRows {
			return ErrAdminConfigRevision
		}
		if err != nil {
			return err
		}
		return insertStudentVerificationAdminAuditTx(ctx, tx, adminAuditInput{
			ActorUserID: input.ActorUserID, EventType: "student_verification.school_config.updated",
			Action: "update", ResourceType: "school_verification_profile",
			ResourceID: input.SchoolCode, SchoolID: schoolID, Reason: input.Reason,
			Before: map[string]any{"revision": input.ExpectedRevision},
			After:  map[string]any{"revision": revision, "enabled": false, "validationStatus": "pending"},
			Now:    now,
		})
	})
}

func (r *Repository) UpdateAdminVerificationMethod(
	ctx context.Context,
	input UpdateAdminVerificationMethodConfigInput,
) error {
	conditionalPolicy, err := json.Marshal(input.ConditionalPolicy)
	if err != nil {
		return err
	}
	publicFormSchema, err := json.Marshal(input.PublicFormSchema)
	if err != nil {
		return err
	}
	riskPolicy, err := json.Marshal(input.RiskPolicy)
	if err != nil {
		return err
	}
	privacyNotice, err := json.Marshal(input.PrivacyNotice)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var schoolID int64
		var revision int64
		if input.ExpectedRevision == 0 {
			if err := tx.QueryRow(ctx, `
				SELECT profile.school_id
				FROM school_verification_profiles profile
				JOIN schools school ON school.id = profile.school_id
				WHERE school.code = $1
			`, input.SchoolCode).Scan(&schoolID); err == pgx.ErrNoRows {
				return ErrSchoolNotFound
			} else if err != nil {
				return err
			}
			err := tx.QueryRow(ctx, `
				INSERT INTO school_verification_methods (
				    school_id, method, display_name, description, adapter_id,
				    adapter_version, roster_dependency, conditional_policy,
				    public_form_schema, risk_policy, credential_ttl_seconds,
				    connector_operation_key, privacy_notice_version, privacy_notice,
				    enabled, validation_status, health_status, config_revision,
				    created_at, updated_at
				)
				VALUES (
				    $1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb,
				    $10::jsonb, $11, $12, $13, $14::jsonb,
				    false, 'pending', 'unknown', 1, $15, $15
				)
				ON CONFLICT (school_id, method) DO NOTHING
				RETURNING config_revision
			`, schoolID, input.Method, input.DisplayName, input.Description,
				input.AdapterID, input.AdapterVersion, input.RosterDependency,
				conditionalPolicy, publicFormSchema, riskPolicy,
				input.CredentialTTLSeconds, input.ConnectorOperationKey,
				input.PrivacyNoticeVersion, privacyNotice, now,
			).Scan(&revision)
			if err == pgx.ErrNoRows {
				return ErrAdminConfigRevision
			}
			if err != nil {
				return err
			}
		} else {
			err := tx.QueryRow(ctx, `
			UPDATE school_verification_methods method
			SET display_name = $3, description = $4, adapter_id = $5,
			    adapter_version = $6, roster_dependency = $7,
			    conditional_policy = $8::jsonb, public_form_schema = $9::jsonb,
			    risk_policy = $10::jsonb, credential_ttl_seconds = $11,
			    connector_operation_key = $12, privacy_notice_version = $13,
			    privacy_notice = $14::jsonb, enabled = false,
			    validation_status = 'pending', validation_code = NULL,
			    health_status = 'unknown', health_code = NULL, health_checked_at = NULL,
			    validated_at = NULL, validated_by_user_id = NULL,
			    config_revision = config_revision + 1, updated_at = $15
			FROM schools school
			WHERE method.school_id = school.id AND school.code = $1
			  AND method.method = $2 AND method.config_revision = $16
			RETURNING method.school_id, method.config_revision
			`, input.SchoolCode, input.Method, input.DisplayName, input.Description,
				input.AdapterID, input.AdapterVersion, input.RosterDependency,
				conditionalPolicy, publicFormSchema, riskPolicy, input.CredentialTTLSeconds,
				input.ConnectorOperationKey, input.PrivacyNoticeVersion, privacyNotice,
				now, input.ExpectedRevision).Scan(&schoolID, &revision)
			if err == pgx.ErrNoRows {
				return ErrAdminConfigRevision
			}
			if err != nil {
				return err
			}
		}
		return insertStudentVerificationAdminAuditTx(ctx, tx, adminAuditInput{
			ActorUserID: input.ActorUserID, EventType: "student_verification.method_config.updated",
			Action: "update", ResourceType: "school_verification_method",
			ResourceID: input.SchoolCode + ":" + string(input.Method), SchoolID: schoolID,
			Reason: input.Reason, Before: map[string]any{"revision": input.ExpectedRevision},
			After: map[string]any{"revision": revision, "enabled": false, "validationStatus": "pending"},
			Now:   now,
		})
	})
}

func (r *Repository) ValidateAdminVerificationSchool(
	ctx context.Context,
	input ValidateAdminVerificationConfigInput,
	valid bool,
	validationCode string,
) error {
	now := time.Now().UTC()
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		status := "invalid"
		if valid {
			status = "valid"
		}
		var schoolID, enabledMethodCount int64
		if input.Enable && valid {
			if err := tx.QueryRow(ctx, `
				SELECT COUNT(*)
				FROM school_verification_methods method
				JOIN schools school ON school.id = method.school_id
				WHERE school.code = $1 AND method.enabled
				  AND method.validation_status = 'valid'
				  AND method.health_status IN ('healthy', 'degraded')
			`, input.SchoolCode).Scan(&enabledMethodCount); err != nil {
				return err
			}
			if enabledMethodCount == 0 {
				return ErrAdminConfigDependency
			}
		}
		var revision int64
		err := tx.QueryRow(ctx, `
			UPDATE school_verification_profiles profile
			SET validation_status = $2, validation_code = NULLIF($3, ''),
			    enabled = $4 AND $2 = 'valid', validated_at = $5,
			    validated_by_user_id = $6, config_revision = config_revision + 1,
			    updated_at = $5
			FROM schools school
			WHERE profile.school_id = school.id AND school.code = $1
			  AND profile.config_revision = $7
			RETURNING profile.school_id, profile.config_revision
		`, input.SchoolCode, status, validationCode, input.Enable, now,
			input.ActorUserID, input.ExpectedRevision).Scan(&schoolID, &revision)
		if err == pgx.ErrNoRows {
			return ErrAdminConfigRevision
		}
		if err != nil {
			return err
		}
		return insertStudentVerificationAdminAuditTx(ctx, tx, adminAuditInput{
			ActorUserID: input.ActorUserID, EventType: "student_verification.school_config.validated",
			Action: "validate", ResourceType: "school_verification_profile",
			ResourceID: input.SchoolCode, SchoolID: schoolID, Reason: input.Reason,
			Before: map[string]any{"revision": input.ExpectedRevision},
			After:  map[string]any{"revision": revision, "validationStatus": status, "enabled": input.Enable && valid},
			Now:    now,
		})
	})
}

func (r *Repository) ValidateAdminVerificationMethod(
	ctx context.Context,
	input ValidateAdminVerificationConfigInput,
	valid bool,
	validationCode string,
	healthStatus string,
	healthCode string,
) error {
	if input.Method == nil {
		return ErrAdminConfigInvalid
	}
	now := time.Now().UTC()
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		status := "invalid"
		if valid {
			status = "valid"
		}
		if input.Enable && (!valid || (healthStatus != "healthy" && healthStatus != "degraded")) {
			return ErrAdminConfigDependency
		}
		var schoolID, revision int64
		err := tx.QueryRow(ctx, `
			UPDATE school_verification_methods method
			SET validation_status = $3, validation_code = NULLIF($4, ''),
			    health_status = $5, health_code = NULLIF($6, ''), health_checked_at = $7,
			    enabled = $8 AND $3 = 'valid' AND $5 IN ('healthy', 'degraded'),
			    validated_at = $7, validated_by_user_id = $9,
			    config_revision = config_revision + 1, updated_at = $7
			FROM schools school
			WHERE method.school_id = school.id AND school.code = $1
			  AND method.method = $2 AND method.config_revision = $10
			RETURNING method.school_id, method.config_revision
		`, input.SchoolCode, *input.Method, status, validationCode, healthStatus,
			healthCode, now, input.Enable, input.ActorUserID, input.ExpectedRevision).Scan(&schoolID, &revision)
		if err == pgx.ErrNoRows {
			return ErrAdminConfigRevision
		}
		if err != nil {
			return err
		}
		return insertStudentVerificationAdminAuditTx(ctx, tx, adminAuditInput{
			ActorUserID: input.ActorUserID, EventType: "student_verification.method_config.validated",
			Action: "validate", ResourceType: "school_verification_method",
			ResourceID: input.SchoolCode + ":" + string(*input.Method), SchoolID: schoolID,
			Reason: input.Reason, Before: map[string]any{"revision": input.ExpectedRevision},
			After: map[string]any{"revision": revision, "validationStatus": status, "healthStatus": healthStatus, "enabled": input.Enable && valid},
			Now:   now,
		})
	})
}

func (r *Repository) ConnectorOperationAvailable(
	ctx context.Context,
	schoolID int64,
	operationKey string,
	adapterID string,
	adapterVersion string,
) bool {
	var available bool
	err := r.db.QueryRow(withTable(ctx, "campus_connector_school_operations"), `
		SELECT EXISTS (
			SELECT 1
			FROM campus_connector_school_operations operation
			JOIN campus_connector_nodes node ON node.id = operation.node_id
			WHERE operation.school_id = $1 AND operation.operation_key = $2
			  AND operation.adapter_id = $3 AND operation.adapter_version = $4
			  AND operation.enabled AND operation.validation_status = 'valid'
			  AND operation.health_status IN ('healthy', 'degraded')
			  AND node.status IN ('active', 'degraded')
			  AND node.revoked_at IS NULL AND node.certificate_not_after > NOW()
		)
	`, schoolID, operationKey, adapterID, adapterVersion).Scan(&available)
	return err == nil && available
}

func (r *Repository) ListAdminStudentCredentials(
	ctx context.Context,
	schoolCode string,
	status CredentialStatus,
	limit int,
	offset int,
) ([]AdminStudentCredential, error) {
	ctx = withTable(ctx, "user_verification_credentials")
	rows, err := r.db.Query(ctx, adminStudentCredentialSelectSQL()+`
		WHERE credential.verification_application_id IS NOT NULL
		  AND school.code = $1
		  AND ($2 = '' OR credential.status = $2)
		ORDER BY credential.verified_at DESC, credential.id DESC
		LIMIT $3 OFFSET $4
	`, schoolCode, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]AdminStudentCredential, 0)
	for rows.Next() {
		credential, scanErr := scanAdminStudentCredential(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, *credential)
	}
	return result, rows.Err()
}

func (r *Repository) GetAdminStudentCredential(ctx context.Context, credentialID string) (*AdminStudentCredential, error) {
	return scanAdminStudentCredential(r.db.QueryRow(withTable(ctx, "user_verification_credentials"), adminStudentCredentialSelectSQL()+`
		WHERE credential.verification_application_id IS NOT NULL
		  AND credential.id = $1
	`, credentialID))
}

func adminStudentCredentialSelectSQL() string {
	return `
		SELECT credential.id, credential.user_id, credential.school_id,
		       school.code, school.name, credential.kind, credential.status,
		       credential.credential_class, credential.subject_display,
		       credential.roster_dependency, credential.assurance,
		       credential.verified_at, credential.expires_at,
		       credential.review_required_at, credential.revoked_reason,
		       credential.revision
		FROM user_verification_credentials credential
		JOIN schools school ON school.id = credential.school_id
	`
}

func scanAdminStudentCredential(row rowScanner) (*AdminStudentCredential, error) {
	var credential AdminStudentCredential
	var method, status string
	if err := row.Scan(
		&credential.ID, &credential.UserID, &credential.SchoolID,
		&credential.SchoolCode, &credential.SchoolName, &method, &status,
		&credential.CredentialClass, &credential.SubjectDisplay,
		&credential.RosterDependency, &credential.Assurance,
		&credential.VerifiedAt, &credential.ExpiresAt,
		&credential.ReviewRequiredAt, &credential.RevokedReason,
		&credential.Revision,
	); err != nil {
		return nil, err
	}
	credential.Method = Method(method)
	credential.Status = CredentialStatus(status)
	return &credential, nil
}

func (r *Repository) RevokeAdminStudentCredential(
	ctx context.Context,
	input AdminCredentialRevokeInput,
) error {
	now := time.Now().UTC()
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var userID, schoolID int64
		var revision int64
		err := tx.QueryRow(ctx, `
			UPDATE user_verification_credentials
			SET status = 'revoked', revoked_at = $3, revoked_reason = $4,
			    review_required_at = NULL, review_required_reason = NULL,
			    revision = revision + 1, last_evaluated_at = $3, updated_at = $3
			WHERE id = $1 AND revision = $2
			  AND verification_application_id IS NOT NULL
			  AND status IN ('active', 'review_required')
			RETURNING user_id, school_id, revision
		`, input.CredentialID, input.ExpectedRevision, now, input.Reason).Scan(
			&userID, &schoolID, &revision,
		)
		if err == pgx.ErrNoRows {
			return ErrCredentialState
		}
		if err != nil {
			return err
		}
		if err := r.BumpEligibilityRevisionTx(ctx, tx, userID, schoolID, "credential_admin_revoked", now); err != nil {
			return err
		}
		return insertStudentVerificationAdminAuditTx(ctx, tx, adminAuditInput{
			ActorUserID: input.ActorUserID, EventType: "student_verification.credential.revoked",
			Action: "revoke", ResourceType: "student_verification_credential",
			ResourceID: input.CredentialID, SchoolID: schoolID, Reason: input.Reason,
			Before: map[string]any{"revision": input.ExpectedRevision, "status": "active_or_review_required"},
			After:  map[string]any{"revision": revision, "status": "revoked"}, Now: now,
		})
	})
}

func (r *Repository) ListAdminStudentSubjectConflicts(
	ctx context.Context,
	schoolCode string,
	status string,
	limit int,
	offset int,
) ([]AdminStudentSubjectConflict, error) {
	ctx = withTable(ctx, "student_subject_conflicts")
	rows, err := r.db.Query(ctx, adminSubjectConflictSelectSQL()+`
		WHERE school.code = $1 AND ($2 = '' OR conflict.status = $2)
		ORDER BY conflict.created_at DESC, conflict.id DESC
		LIMIT $3 OFFSET $4
	`, schoolCode, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]AdminStudentSubjectConflict, 0)
	for rows.Next() {
		conflict, scanErr := scanAdminSubjectConflict(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, *conflict)
	}
	return result, rows.Err()
}

func (r *Repository) GetAdminStudentSubjectConflict(ctx context.Context, conflictID string) (*AdminStudentSubjectConflict, error) {
	return scanAdminSubjectConflict(r.db.QueryRow(withTable(ctx, "student_subject_conflicts"), adminSubjectConflictSelectSQL()+`
		WHERE conflict.id = $1
	`, conflictID))
}

func adminSubjectConflictSelectSQL() string {
	return `
		SELECT conflict.id, conflict.school_id, school.code, school.name,
		       conflict.subject_hash, conflict.claimant_user_id,
		       conflict.incumbent_user_id, conflict.application_id,
		       conflict.status, conflict.reason_code, conflict.resolution_code,
		       conflict.resolved_at, conflict.created_at, conflict.updated_at
		FROM student_subject_conflicts conflict
		JOIN schools school ON school.id = conflict.school_id
	`
}

func scanAdminSubjectConflict(row rowScanner) (*AdminStudentSubjectConflict, error) {
	var conflict AdminStudentSubjectConflict
	if err := row.Scan(
		&conflict.ID, &conflict.SchoolID, &conflict.SchoolCode, &conflict.SchoolName,
		&conflict.subjectHash, &conflict.ClaimantUserID, &conflict.IncumbentUserID,
		&conflict.ApplicationID, &conflict.Status, &conflict.ReasonCode,
		&conflict.ResolutionCode, &conflict.ResolvedAt, &conflict.CreatedAt,
		&conflict.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &conflict, nil
}

func (r *Repository) DecideAdminStudentSubjectConflict(
	ctx context.Context,
	input AdminSubjectConflictDecisionInput,
) error {
	now := time.Now().UTC()
	return r.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var conflict AdminStudentSubjectConflict
		err := tx.QueryRow(ctx, `
			SELECT id, school_id, subject_hash, claimant_user_id,
			       incumbent_user_id, status
			FROM student_subject_conflicts
			WHERE id = $1
			FOR UPDATE
		`, input.ConflictID).Scan(
			&conflict.ID, &conflict.SchoolID, &conflict.subjectHash,
			&conflict.ClaimantUserID, &conflict.IncumbentUserID, &conflict.Status,
		)
		if err == pgx.ErrNoRows {
			return ErrAdminConflictNotFound
		}
		if err != nil {
			return err
		}
		if conflict.Status != "open" && conflict.Status != "under_review" {
			return ErrAdminConflictState
		}
		resolutionCode := ""
		targetStatus := "under_review"
		switch input.Action {
		case "start_review":
			if conflict.Status != "open" {
				return ErrAdminConflictState
			}
		case "dismiss_claim":
			targetStatus = "dismissed"
			resolutionCode = "claim_dismissed"
		case "release_subject_for_reverification":
			targetStatus = "resolved"
			resolutionCode = "released_for_reverification"
			if _, err := tx.Exec(ctx, `
				UPDATE student_enrollment_subjects
				SET binding_status = 'revoked', revoked_at = $3,
				    review_required_at = NULL, review_required_reason = NULL,
				    revision = revision + 1, updated_at = $3
				WHERE school_id = $1 AND subject_hash = $2
				  AND binding_status IN ('active', 'review_required')
			`, conflict.SchoolID, conflict.subjectHash, now); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				UPDATE user_verification_credentials
				SET status = 'revoked', revoked_at = $3,
				    revoked_reason = 'subject_conflict_reverification',
				    review_required_at = NULL, review_required_reason = NULL,
				    revision = revision + 1, last_evaluated_at = $3, updated_at = $3
				WHERE school_id = $1 AND subject_hash = $2
				  AND verification_application_id IS NOT NULL
				  AND status IN ('active', 'review_required')
			`, conflict.SchoolID, conflict.subjectHash, now); err != nil {
				return err
			}
			users := []int64{conflict.ClaimantUserID}
			if conflict.IncumbentUserID != nil {
				users = append(users, *conflict.IncumbentUserID)
			}
			for _, userID := range users {
				if err := r.BumpEligibilityRevisionTx(ctx, tx, userID, conflict.SchoolID, "subject_released_for_reverification", now); err != nil {
					return err
				}
			}
		default:
			return ErrAdminConflictState
		}
		_, err = tx.Exec(ctx, `
			UPDATE student_subject_conflicts
			SET status = $2, resolution_code = NULLIF($3, ''),
			    resolved_by_user_id = CASE WHEN $2 IN ('resolved', 'dismissed') THEN $4 ELSE NULL END,
			    resolved_at = CASE WHEN $2 IN ('resolved', 'dismissed') THEN $5 ELSE NULL END,
			    updated_at = $5
			WHERE id = $1
		`, input.ConflictID, targetStatus, resolutionCode, input.ActorUserID, now)
		if err != nil {
			return err
		}
		return insertStudentVerificationAdminAuditTx(ctx, tx, adminAuditInput{
			ActorUserID: input.ActorUserID, EventType: "student_verification.subject_conflict.decided",
			Action: input.Action, ResourceType: "student_subject_conflict",
			ResourceID: input.ConflictID, SchoolID: conflict.SchoolID, Reason: input.Reason,
			Before: map[string]any{"status": conflict.Status},
			After:  map[string]any{"status": targetStatus, "resolutionCode": resolutionCode}, Now: now,
		})
	})
}

func (r *Repository) ListAdminCampusConnectorHealth(
	ctx context.Context,
	schoolCode string,
) ([]AdminCampusConnectorHealth, error) {
	ctx = withTable(ctx, "campus_connector_nodes")
	rows, err := r.db.Query(ctx, `
		SELECT node.id, node.display_name, node.status, node.protocol_version,
		       node.software_version, node.certificate_not_after,
		       node.max_concurrency, node.heartbeat_interval_seconds,
		       node.last_heartbeat_at, node.last_health_code, node.revision,
		       school.code, operation.operation_key, operation.operation_type,
		       operation.adapter_id, operation.adapter_version, operation.enabled,
		       operation.validation_status, operation.health_status,
		       operation.health_code, operation.health_checked_at,
		       operation.config_revision
		FROM campus_connector_nodes node
		LEFT JOIN campus_connector_school_operations operation ON operation.node_id = node.id
		LEFT JOIN schools school ON school.id = operation.school_id
		WHERE ($1 = '' OR school.code = $1)
		ORDER BY node.display_name, node.id, school.code, operation.operation_key
	`, schoolCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]AdminCampusConnectorHealth, 0)
	indexByID := map[string]int{}
	for rows.Next() {
		var node AdminCampusConnectorHealth
		var operation AdminCampusConnectorOperationHealth
		var school, operationKey, operationType, adapterID, adapterVersion *string
		var operationEnabled *bool
		var validationStatus, healthStatus *string
		var configRevision *int64
		if err := rows.Scan(
			&node.ID, &node.DisplayName, &node.Status, &node.ProtocolVersion,
			&node.SoftwareVersion, &node.CertificateNotAfter, &node.MaxConcurrency,
			&node.HeartbeatIntervalSeconds, &node.LastHeartbeatAt, &node.LastHealthCode,
			&node.Revision, &school, &operationKey, &operationType, &adapterID,
			&adapterVersion, &operationEnabled, &validationStatus, &healthStatus,
			&operation.HealthCode, &operation.HealthCheckedAt, &configRevision,
		); err != nil {
			return nil, err
		}
		index, exists := indexByID[node.ID]
		if !exists {
			index = len(result)
			indexByID[node.ID] = index
			node.Operations = []AdminCampusConnectorOperationHealth{}
			result = append(result, node)
		}
		if school != nil && operationKey != nil && operationType != nil && adapterID != nil &&
			adapterVersion != nil && operationEnabled != nil && validationStatus != nil &&
			healthStatus != nil && configRevision != nil {
			operation.SchoolCode = *school
			operation.OperationKey = *operationKey
			operation.OperationType = *operationType
			operation.AdapterID = *adapterID
			operation.AdapterVersion = *adapterVersion
			operation.Enabled = *operationEnabled
			operation.ValidationStatus = *validationStatus
			operation.HealthStatus = *healthStatus
			operation.ConfigRevision = *configRevision
			result[index].Operations = append(result[index].Operations, operation)
		}
	}
	return result, rows.Err()
}

type adminAuditInput struct {
	ActorUserID  int64
	EventType    string
	Action       string
	ResourceType string
	ResourceID   string
	SchoolID     int64
	Reason       string
	Before       map[string]any
	After        map[string]any
	Now          time.Time
}

func insertStudentVerificationAdminAuditTx(ctx context.Context, tx pgx.Tx, input adminAuditInput) error {
	eventID, err := newID()
	if err != nil {
		return err
	}
	event := audit.EventFromContext(ctx, audit.Event{
		Type: audit.EventType(input.EventType), Category: "admin_operation",
		ActorType: "admin", UserID: strconv.FormatInt(input.ActorUserID, 10),
		Action: input.Action, ResourceType: input.ResourceType,
		ResourceID: input.ResourceID, ScopeSchoolID: strconv.FormatInt(input.SchoolID, 10),
		Result: "success", Reason: input.Reason, Before: input.Before, After: input.After,
		Timestamp: input.Now,
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
		VALUES ($1, $2, $3, $4, $5, '', $6, $7, $8, $9,
		        $10::jsonb, $11::jsonb, $12, $13, NULLIF($14, ''),
		        NULLIF($15, ''), $16::jsonb, $17)
	`, eventID, event.Category, event.Type, event.ActorType, event.UserID,
		event.Action, event.ResourceType, event.ResourceID, event.ScopeSchoolID,
		beforeJSON, afterJSON, event.Result, event.Reason, event.TraceID,
		event.RequestID, detailsJSON, event.Timestamp)
	return err
}
