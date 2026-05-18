package openplatform

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetInternalUserID(ctx context.Context, casdoorSubject string) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		SELECT id
		FROM users
		WHERE casdoor_subject = $1
	`, casdoorSubject).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrDisclosureUnavailable
	}
	if err != nil {
		return 0, fmt.Errorf("GetInternalUserID: %w", err)
	}
	return id, nil
}

func (r *Repository) GetUserProjection(ctx context.Context, userID int64) (*UserProjection, error) {
	var item UserProjection
	err := r.db.QueryRow(ctx, `
		SELECT u.casdoor_subject,
		       u.username,
		       COALESCE(u.email, ''),
		       u.avatar_url,
		       u.phone_enc,
		       u.phone_enc IS NOT NULL,
		       COALESCE(i.verified, false),
		       p.verification_status,
		       p.school_id,
		       s.name
		FROM users u
		LEFT JOIN user_identities i ON i.user_id = u.id
		LEFT JOIN user_profiles p ON p.user_id = u.id
		LEFT JOIN schools s ON s.id = p.school_id
		WHERE u.id = $1
	`, userID).Scan(&item.CasdoorSubject, &item.Username, &item.Email,
		&item.AvatarURL, &item.PhoneEnc, &item.PhoneVerified, &item.IdentityVerified,
		&item.ProfileStatus, &item.SchoolID, &item.SchoolName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDisclosureUnavailable
	}
	if err != nil {
		return nil, fmt.Errorf("GetUserProjection: %w", err)
	}
	return &item, nil
}
