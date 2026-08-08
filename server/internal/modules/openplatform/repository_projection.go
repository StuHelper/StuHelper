package openplatform

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetUserProjection(ctx context.Context, userID int64) (*UserProjection, error) {
	ctx = withDBTable(ctx, "users")
	var item UserProjection
	err := r.db.QueryRow(ctx, `
		SELECT u.id,
		       u.username,
		       COALESCE(u.email, ''),
		       u.avatar_url,
		       phone_gate.user_id IS NOT NULL,
		       COALESCE(student.real_name_verified, false),
		       CASE WHEN student.school_id IS NOT NULL THEN 'verified'::text ELSE NULL END,
		       student.school_id,
		       student.school_name,
		       q.qq_id
		FROM users u
		LEFT JOIN LATERAL (
		    SELECT credential.user_id
		    FROM current_phone_gate_credentials credential
		    WHERE credential.user_id = u.id
		    ORDER BY credential.verified_at DESC, credential.id DESC
		    LIMIT 1
		) phone_gate ON true
		LEFT JOIN LATERAL (
		    SELECT credential.school_id,
		           school.name AS school_name,
		           bool_or(credential.kind = 'real_name_identity_check')
		               OVER (PARTITION BY credential.user_id) AS real_name_verified
		    FROM current_student_qualifying_credentials credential
		    JOIN schools school ON school.id = credential.school_id
		    WHERE credential.user_id = u.id
		    ORDER BY
		        CASE credential.credential_class WHEN 'formal_student' THEN 0 ELSE 1 END,
		        credential.verified_at DESC,
		        credential.id DESC
		    LIMIT 1
		) student ON true
		LEFT JOIN user_qq_bindings q ON q.user_id = u.id
		WHERE u.id = $1
	`, userID).Scan(&item.UserID, &item.Username, &item.Email,
		&item.AvatarURL, &item.PhoneVerified, &item.IdentityVerified,
		&item.ProfileStatus, &item.SchoolID, &item.SchoolName, &item.QQID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDisclosureUnavailable
	}
	if err != nil {
		return nil, fmt.Errorf("GetUserProjection: %w", err)
	}
	return &item, nil
}
