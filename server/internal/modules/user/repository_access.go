package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/reviewaccess"
)

// ReviewAccessSubject 是评课访问控制所需的最小用户事实集合。
type ReviewAccessSubject struct {
	InternalUserID   int64
	SchoolID         *int64
	StudentVerified  bool
	IdentityVerified bool
}

// GetReviewAccessSubjectByCasdoorSubject 一次查询获取评课访问控制所需的用户事实。
func (r *Repository) GetReviewAccessSubjectByCasdoorSubject(ctx context.Context, casdoorSubject string) (*ReviewAccessSubject, error) {
	var subject ReviewAccessSubject
	err := r.db.QueryRow(ctx, `
		SELECT u.id,
		       up.school_id,
		       COALESCE(up.verification_status = 'verified', false) AS student_verified,
		       COALESCE(ui.verified, false) AS identity_verified
		FROM users u
		LEFT JOIN user_profiles up ON up.user_id = u.id
		LEFT JOIN user_identities ui ON ui.user_id = u.id
		WHERE u.casdoor_subject = $1
	`, casdoorSubject).Scan(
		&subject.InternalUserID,
		&subject.SchoolID,
		&subject.StudentVerified,
		&subject.IdentityVerified,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetReviewAccessSubjectByCasdoorSubject: %w", err)
	}
	return &subject, nil
}

func (r *Repository) GetReviewAccessSubject(ctx context.Context, casdoorSubject string) (*reviewaccess.Subject, error) {
	subject, err := r.GetReviewAccessSubjectByCasdoorSubject(ctx, casdoorSubject)
	if err != nil || subject == nil {
		return nil, err
	}
	return &reviewaccess.Subject{
		InternalUserID:   subject.InternalUserID,
		SchoolID:         subject.SchoolID,
		StudentVerified:  subject.StudentVerified,
		IdentityVerified: subject.IdentityVerified,
	}, nil
}
