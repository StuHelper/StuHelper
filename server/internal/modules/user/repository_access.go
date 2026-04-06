package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ReviewAccessSubject 是评课访问控制所需的最小用户事实集合。
type ReviewAccessSubject struct {
	InternalUserID   int64
	SchoolID         *string
	StudentVerified  bool
	IdentityVerified bool
}

// GetReviewAccessSubjectByExternalID 一次查询获取评课访问控制所需的用户事实。
func (r *Repository) GetReviewAccessSubjectByExternalID(ctx context.Context, externalID string) (*ReviewAccessSubject, error) {
	var subject ReviewAccessSubject
	err := r.db.QueryRow(ctx, `
		SELECT u.id,
		       up.school_id,
		       COALESCE(up.verification_status = 'verified', false) AS student_verified,
		       COALESCE(ui.verified, false) AS identity_verified
		FROM users u
		LEFT JOIN user_profiles up ON up.user_id = u.id
		LEFT JOIN user_identities ui ON ui.user_id = u.id
		WHERE u.external_id = $1
	`, externalID).Scan(
		&subject.InternalUserID,
		&subject.SchoolID,
		&subject.StudentVerified,
		&subject.IdentityVerified,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetReviewAccessSubjectByExternalID: %w", err)
	}
	return &subject, nil
}
