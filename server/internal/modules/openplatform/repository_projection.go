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
		SELECT u.username,
		       COALESCE(u.email, ''),
		       u.avatar_url,
		       u.phone_enc,
		       u.phone_enc IS NOT NULL,
		       COALESCE(i.verified, false),
		       p.verification_status,
		       p.school_id,
		       s.name,
		       q.qq_id
		FROM users u
		LEFT JOIN user_identities i ON i.user_id = u.id
		LEFT JOIN user_profiles p ON p.user_id = u.id
		LEFT JOIN schools s ON s.id = p.school_id
		LEFT JOIN user_qq_bindings q ON q.user_id = u.id
		WHERE u.id = $1
	`, userID).Scan(&item.Username, &item.Email,
		&item.AvatarURL, &item.PhoneEnc, &item.PhoneVerified, &item.IdentityVerified,
		&item.ProfileStatus, &item.SchoolID, &item.SchoolName, &item.QQID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDisclosureUnavailable
	}
	if err != nil {
		return nil, fmt.Errorf("GetUserProjection: %w", err)
	}
	return &item, nil
}
