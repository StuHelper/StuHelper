package admission

import (
	"context"
	"encoding/json"
	"fmt"

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
	_, err := tx.Exec(ctx, `
		INSERT INTO user_profiles (
			user_id, school_id, student_ids, active_student_id, manual_form_data,
			verification_status, verification_method, rejection_reason, reviewed_at,
			consent_given_at, verified_at, created_at, updated_at
		)
		VALUES (
			$1, $2, '[]'::jsonb, NULL,
			jsonb_build_object(
				'source', 'admission_credential',
				'credentialKind', $3::text,
				'subjectDisplay', $4::text
			),
			'verified', $5, NULL, $6, $6, $6, NOW(), NOW()
		)
		ON CONFLICT (user_id)
		DO UPDATE SET
			school_id = EXCLUDED.school_id,
			verification_status = 'verified',
			verification_method = EXCLUDED.verification_method,
			rejection_reason = NULL,
			reviewed_at = EXCLUDED.reviewed_at,
			consent_given_at = COALESCE(user_profiles.consent_given_at, EXCLUDED.consent_given_at),
			verified_at = EXCLUDED.verified_at,
			manual_form_data = CASE
				WHEN user_profiles.manual_form_data IS NULL OR user_profiles.manual_form_data = '{}'::jsonb
					THEN EXCLUDED.manual_form_data
				ELSE user_profiles.manual_form_data
			END,
			updated_at = NOW()
	`, credential.UserID, credential.SchoolID, credential.Kind, credential.SubjectDisplay, method, credential.VerifiedAt)
	if err != nil {
		return fmt.Errorf("ProjectVerifiedUserProfileTx: %w", err)
	}
	if err := r.upsertVerifiedUserProfileProjectionJobsTx(ctx, tx, credential.UserID); err != nil {
		return err
	}
	return nil
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
		fmt.Sprintf("user-profile-projection:%d", userID),
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
		fmt.Sprintf("verified-student-role:%d", userID),
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
