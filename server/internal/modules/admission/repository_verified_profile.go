package admission

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/outbox"
)

const (
	admissionProfileProjectionJobType   = "user_profile_projection"
	admissionVerifiedStudentRoleJobType = "verified_student_role"
	admissionVerifiedStudentRoleName    = "verified_student"
)

type verifiedRoleSyncPayload struct {
	UserID   int64  `json:"userID"`
	Role     string `json:"role"`
	Approved bool   `json:"approved"`
}

type profileProjectionPayload struct {
	UserID   int64 `json:"userID"`
	Approved bool  `json:"approved"`
}

func (r *Repository) ProjectVerifiedUserProfileTx(
	ctx context.Context,
	tx pgx.Tx,
	credential VerificationCredential,
) error {
	method := profileVerificationMethodForCredential(credential.Kind)
	profileFields, err := verifiedProfileProjectionFields(credential)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
			INSERT INTO user_profiles (
				user_id, school_id, student_ids, active_student_id, manual_form_data,
				verification_status, verification_method, rejection_reason, reviewed_at,
				consent_given_at, verified_at, created_at, updated_at
			)
			VALUES (
				$1, $2, $3::jsonb, $4, $5::jsonb,
				'verified', $6, NULL, $7, $7, $7, NOW(), NOW()
			)
			ON CONFLICT (user_id)
			DO UPDATE SET
				school_id = EXCLUDED.school_id,
				student_ids = CASE
					WHEN jsonb_array_length(EXCLUDED.student_ids) > 0 THEN EXCLUDED.student_ids
					ELSE COALESCE(user_profiles.student_ids, '[]'::jsonb)
				END,
				active_student_id = COALESCE(EXCLUDED.active_student_id, user_profiles.active_student_id),
				verification_status = 'verified',
				verification_method = EXCLUDED.verification_method,
				rejection_reason = NULL,
				reviewed_at = EXCLUDED.reviewed_at,
				consent_given_at = COALESCE(user_profiles.consent_given_at, EXCLUDED.consent_given_at),
				verified_at = EXCLUDED.verified_at,
				manual_form_data = COALESCE(user_profiles.manual_form_data, '{}'::jsonb) || EXCLUDED.manual_form_data,
				updated_at = NOW()
		`, credential.UserID, credential.SchoolID, profileFields.studentIDsJSON, profileFields.activeStudentID,
		profileFields.manualFormDataJSON, method, credential.VerifiedAt)
	if err != nil {
		return fmt.Errorf("ProjectVerifiedUserProfileTx: %w", err)
	}
	if err := r.upsertVerifiedUserProfileProjectionJobsTx(ctx, tx, credential.UserID); err != nil {
		return err
	}
	return nil
}

type verifiedProfileProjectionFieldsOutput struct {
	studentIDsJSON     json.RawMessage
	activeStudentID    *string
	manualFormDataJSON json.RawMessage
}

func verifiedProfileProjectionFields(credential VerificationCredential) (verifiedProfileProjectionFieldsOutput, error) {
	studentID := strings.TrimSpace(credential.StudentID)
	studentName := strings.TrimSpace(credential.StudentName)
	subject := strings.TrimSpace(credential.Subject)
	studentIDs := []string{}
	var activeStudentID *string
	if studentID != "" {
		studentIDs = []string{studentID}
		activeStudentID = &studentID
	}
	studentIDsJSON, err := json.Marshal(studentIDs)
	if err != nil {
		return verifiedProfileProjectionFieldsOutput{}, fmt.Errorf("marshal admission profile student ids: %w", err)
	}
	fields := map[string]string{
		"source":         "admission_credential",
		"credentialKind": string(credential.Kind),
		"subjectDisplay": credential.SubjectDisplay,
	}
	if credential.Kind == CredentialSchoolEmailOTP && subject != "" {
		fields["schoolEmail"] = subject
	}
	if studentID != "" {
		fields["studentID"] = studentID
	}
	if studentName != "" {
		fields["studentName"] = studentName
	}
	manualFormDataJSON, err := json.Marshal(fields)
	if err != nil {
		return verifiedProfileProjectionFieldsOutput{}, fmt.Errorf("marshal admission profile manual form data: %w", err)
	}
	return verifiedProfileProjectionFieldsOutput{
		studentIDsJSON:     studentIDsJSON,
		activeStudentID:    activeStudentID,
		manualFormDataJSON: manualFormDataJSON,
	}, nil
}

func (r *Repository) upsertVerifiedUserProfileProjectionJobsTx(ctx context.Context, tx pgx.Tx, userID int64) error {
	profilePayload, err := json.Marshal(profileProjectionPayload{UserID: userID, Approved: true})
	if err != nil {
		return fmt.Errorf("marshal profile projection payload: %w", err)
	}
	if err := outbox.UpsertJobTx(
		ctx,
		tx,
		outbox.StreamIAMOpenFGATupleSync,
		admissionProfileProjectionJobType,
		admissionProfileProjectionDedupeKey(userID),
		profilePayload,
	); err != nil {
		return fmt.Errorf("upsert profile projection job: %w", err)
	}

	rolePayload, err := json.Marshal(verifiedRoleSyncPayload{
		UserID:   userID,
		Role:     admissionVerifiedStudentRoleName,
		Approved: true,
	})
	if err != nil {
		return fmt.Errorf("marshal verified student role payload: %w", err)
	}
	if err := outbox.UpsertJobTx(
		ctx,
		tx,
		outbox.StreamIAMCasdoorRoleSync,
		admissionVerifiedStudentRoleJobType,
		admissionVerifiedStudentRoleDedupeKey(userID),
		rolePayload,
	); err != nil {
		return fmt.Errorf("upsert verified student role job: %w", err)
	}
	return nil
}

func profileVerificationMethodForCredential(kind VerificationCredentialKind) string {
	switch kind {
	case CredentialSchoolEmailOTP:
		return "school_email_otp"
	case CredentialSchoolSSO:
		return "school_sso"
	default:
		return "manual"
	}
}
