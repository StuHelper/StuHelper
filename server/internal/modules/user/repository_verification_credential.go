package user

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

func (r *Repository) EnsureVerificationCredentialTx(
	ctx context.Context,
	tx pgx.Tx,
	credential VerificationCredentialProjection,
) error {
	credentialID, err := id.New()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO user_verification_credentials (
			id, user_id, school_id, kind, subject_hash, subject_display, verified_at
		)
		SELECT
			$1::varchar(36),
			$2::bigint,
			$3::bigint,
			$4::text,
			$5::varchar(128),
			$6::text,
			$7::timestamptz
		WHERE NOT EXISTS (
			SELECT 1
			FROM user_verification_credentials
			WHERE user_id = $2::bigint
			  AND school_id = $3::bigint
			  AND kind = $4::text
			  AND subject_hash = $5::varchar(128)
			  AND revoked_at IS NULL
			  AND (expires_at IS NULL OR expires_at > $7::timestamptz)
		)
	`, credentialID, credential.UserID, credential.SchoolID, credential.Kind, credential.SubjectHash,
		credential.SubjectDisplay, credential.VerifiedAt)
	if err != nil {
		return fmt.Errorf("EnsureVerificationCredentialTx: %w", err)
	}
	return nil
}
