package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/reviewaccess"
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
	ctx = withDBTable(ctx, "users")
	var subject ReviewAccessSubject
	err := r.db.QueryRow(ctx, `
		SELECT u.id,
		       student.school_id,
		       student.school_id IS NOT NULL AS student_verified,
		       false AS identity_verified
		FROM users u
		LEFT JOIN LATERAL (
		    SELECT credential.school_id
		    FROM current_student_qualifying_credentials credential
		    WHERE credential.user_id = u.id
		    ORDER BY
		        CASE credential.credential_class WHEN 'formal_student' THEN 0 ELSE 1 END,
		        credential.verified_at DESC,
		        credential.id DESC
		    LIMIT 1
		) student ON true
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
