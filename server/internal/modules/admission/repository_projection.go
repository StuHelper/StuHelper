package admission

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/outbox"
)

const (
	admissionVerificationProjectionStream       = "admission_verification_projection"
	admissionVerificationProjectionDedupePrefix = "admission-verification-projection:"
	admissionProfileProjectionDedupePrefix      = "user-profile-projection:"
)

func (r *Repository) GetLatestSessionByUserID(
	ctx context.Context,
	userID int64,
	admissionSessionID string,
) (*AdmissionSession, error) {
	ctx = withDBTable(ctx, "group_admission_sessions")
	admissionSessionID = strings.TrimSpace(admissionSessionID)
	if admissionSessionID != "" {
		session, err := scanAdmissionSession(r.db.QueryRow(ctx, `
			SELECT `+admissionSessionColumns+`
			FROM group_admission_sessions
			WHERE user_id = $1 AND id = $2
			LIMIT 1
		`, userID, admissionSessionID))
		if errors.Is(err, ErrAdmissionTokenNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("GetLatestSessionByUserID: %w", err)
		}
		return session, nil
	}
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

func (r *Repository) GetLatestCredentialForUser(
	ctx context.Context,
	userID int64,
	now time.Time,
) (*VerificationCredential, error) {
	ctx = withDBTable(ctx, "user_verification_credentials")
	query, args := latestCredentialForUserQuery(userID, 0, now)
	credential, err := scanVerificationCredential(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetLatestCredentialForUser: %w", err)
	}
	return credential, nil
}

func (r *Repository) GetLatestCredentialForUserSchool(
	ctx context.Context,
	userID int64,
	schoolID int64,
	now time.Time,
) (*VerificationCredential, error) {
	ctx = withDBTable(ctx, "user_verification_credentials")
	query, args := latestCredentialForUserQuery(userID, schoolID, now)
	credential, err := scanVerificationCredential(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetLatestCredentialForUserSchool: %w", err)
	}
	return credential, nil
}

func (r *Repository) GetLatestCredentialForUserSchoolTx(
	ctx context.Context,
	tx pgx.Tx,
	userID int64,
	schoolID int64,
	now time.Time,
) (*VerificationCredential, error) {
	query, args := latestCredentialForUserQuery(userID, schoolID, now)
	credential, err := scanVerificationCredential(tx.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetLatestCredentialForUserSchoolTx: %w", err)
	}
	return credential, nil
}

func latestCredentialForUserQuery(
	userID int64,
	schoolID int64,
	now time.Time,
) (string, []any) {
	schoolClause := ""
	args := []any{userID, now}
	if schoolID > 0 {
		schoolClause = " AND credential.school_id = $3"
		args = append(args, schoolID)
	}
	return `
		SELECT user_id, school_id, kind, subject_hash, subject_display,
		       source_application_id, expires_at, verified_at
		FROM user_verification_credentials AS credential
		WHERE user_id = $1
		  AND credential.status = 'active'
		  AND credential.activated_at IS NOT NULL
		  AND credential.revoked_at IS NULL
		  AND (credential.expires_at IS NULL OR credential.expires_at > $2)
		  AND NOT EXISTS (
		      SELECT 1
		      FROM student_subject_conflicts AS conflict
		      WHERE conflict.school_id = credential.school_id
		        AND conflict.claimant_user_id = credential.user_id
		        AND conflict.status IN ('open', 'under_review')
		  )
		  ` + schoolClause + `
		ORDER BY verified_at DESC
		LIMIT 1
	`, args
}

func (r *Repository) HasPendingAdmissionProjection(ctx context.Context, userID int64) (bool, error) {
	ctx = withDBTable(ctx, "domain_event_outbox")
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM domain_event_outbox
			WHERE status IN ('pending', 'processing')
			  AND (
			    (stream = $1 AND dedupe_key = $2)
			    OR (stream = $3 AND dedupe_key = $4)
			  )
		)
	`,
		admissionVerificationProjectionStream,
		admissionVerificationProjectionDedupeKey(userID),
		outbox.StreamIAMOpenFGATupleSync,
		admissionProfileProjectionDedupeKey(userID),
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("HasPendingAdmissionProjection: %w", err)
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

func admissionVerificationProjectionDedupeKey(userID int64) string {
	return fmt.Sprintf("%s%d", admissionVerificationProjectionDedupePrefix, userID)
}

func admissionProfileProjectionDedupeKey(userID int64) string {
	return fmt.Sprintf("%s%d", admissionProfileProjectionDedupePrefix, userID)
}
