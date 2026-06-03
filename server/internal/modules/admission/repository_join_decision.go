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
			SELECT b.user_id, c.verified_at, 1 AS source_priority
			FROM bound b
			JOIN user_verification_credentials c ON c.user_id = b.user_id
			WHERE c.school_id = $2
			  AND c.revoked_at IS NULL
			  AND (c.expires_at IS NULL OR c.expires_at > $3)

			UNION ALL

			SELECT b.user_id, p.verified_at, 2 AS source_priority
			FROM bound b
			JOIN user_profiles p ON p.user_id = b.user_id
			WHERE p.school_id = $2
			  AND p.verification_status = 'verified'
		)
		SELECT user_id
		FROM verified_candidates
		ORDER BY verified_at DESC NULLS LAST, source_priority ASC
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
