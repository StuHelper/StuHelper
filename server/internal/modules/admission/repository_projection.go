package admission

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/outbox"
)

const freshmanProjectionDedupePrefix = "freshman-provisional-role:"

func (r *Repository) GetLatestSessionByUserID(ctx context.Context, userID int64) (*AdmissionSession, error) {
	session, err := scanAdmissionSession(r.db.QueryRow(ctx, `
		SELECT `+admissionSessionColumns+`
		FROM group_admission_sessions
		WHERE user_id = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, userID))
	if errors.Is(err, ErrAdmissionTokenNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetLatestSessionByUserID: %w", err)
	}
	return session, nil
}

func (r *Repository) GetLatestCredentialForUser(ctx context.Context, userID int64) (*VerificationCredential, error) {
	credential, err := scanVerificationCredential(r.db.QueryRow(ctx, `
		SELECT user_id, school_id, kind, subject_hash, subject_display,
		       source_application_id, expires_at, verified_at
		FROM user_verification_credentials
		WHERE user_id = $1 AND revoked_at IS NULL
		ORDER BY verified_at DESC
		LIMIT 1
	`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetLatestCredentialForUser: %w", err)
	}
	return credential, nil
}

func (r *Repository) HasPendingFreshmanProjection(ctx context.Context, userID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM domain_event_outbox
			WHERE stream = $1
			  AND dedupe_key = $2
			  AND status IN ('pending', 'processing')
		)
	`, outbox.StreamIAMCasdoorRoleSync, freshmanProjectionDedupeKey(userID)).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("HasPendingFreshmanProjection: %w", err)
	}
	return exists, nil
}

func scanVerificationCredential(row pgx.Row) (*VerificationCredential, error) {
	var credential VerificationCredential
	err := row.Scan(
		&credential.UserID,
		&credential.SchoolID,
		&credential.Kind,
		&credential.SubjectHash,
		&credential.SubjectDisplay,
		&credential.SourceApplicationID,
		&credential.ExpiresAt,
		&credential.VerifiedAt,
	)
	if err != nil {
		return nil, err
	}
	return &credential, nil
}

func freshmanProjectionDedupeKey(userID int64) string {
	return fmt.Sprintf("%s%d", freshmanProjectionDedupePrefix, userID)
}
