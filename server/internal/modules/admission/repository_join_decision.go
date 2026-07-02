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
	// profile 回退分支覆盖未写入凭证表的历史验证来源；但若该用户在本校的新生临时凭证
	// 已过期或被撤销，profile 残留的 verified 状态不可作为放行依据（F007）——
	// 此时仅凭证分支（仍持有其他有效凭证）可判定为已验证。
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
			  AND NOT EXISTS (
				SELECT 1
				FROM user_verification_credentials fc
				WHERE fc.user_id = b.user_id
				  AND fc.school_id = $2
				  AND fc.kind = $4
				  AND (fc.revoked_at IS NOT NULL
				    OR (fc.expires_at IS NOT NULL AND fc.expires_at <= $3))
			  )
		)
		SELECT user_id
		FROM verified_candidates
		ORDER BY verified_at DESC NULLS LAST, source_priority ASC
		LIMIT 1
	`, qqID, schoolID, now, CredentialFreshmanMaterialManual).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetVerifiedAdmissionUserByQQ: %w", err)
	}
	return &userID, nil
}
