package admission

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetVerifiedAdmissionUserByQQ(
	ctx context.Context,
	qqID string,
	schoolID int64,
	now time.Time,
) (*int64, error) {
	ctx = withDBTable(ctx, "user_verification_credentials")
	var userID int64
	err := r.db.QueryRow(ctx, `
		WITH bound AS (
			SELECT user_id
			FROM user_qq_bindings
			WHERE qq_id = $1
			LIMIT 1
		),
		verified_candidates AS (
			SELECT b.user_id, c.verified_at
			FROM bound b
			JOIN user_verification_credentials c ON c.user_id = b.user_id
			WHERE c.school_id = $2
			  AND c.status = 'active'
			  AND c.activated_at IS NOT NULL
			  AND c.revoked_at IS NULL
			  AND (c.expires_at IS NULL OR c.expires_at > $3)
			  AND NOT EXISTS (
			    SELECT 1
			    FROM student_subject_conflicts conflict
			    WHERE conflict.school_id = c.school_id
			      AND conflict.claimant_user_id = b.user_id
			      AND conflict.status IN ('open', 'under_review')
			  )
		)
		SELECT user_id
		FROM verified_candidates
		ORDER BY verified_at DESC NULLS LAST
		LIMIT 1
	`, qqID, schoolID, now).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetVerifiedAdmissionUserByQQ: %w", err)
	}
	return &userID, nil
}

func (r *Repository) GetBoundAdmissionUserByQQ(
	ctx context.Context,
	qqID string,
) (*int64, error) {
	ctx = withDBTable(ctx, "user_qq_bindings")
	var userID int64
	err := r.db.QueryRow(ctx, `
		SELECT user_id
		FROM user_qq_bindings
		WHERE qq_id = $1
		LIMIT 1
	`, qqID).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetBoundAdmissionUserByQQ: %w", err)
	}
	return &userID, nil
}
