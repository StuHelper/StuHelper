package openplatform

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) HasActiveConsents(ctx context.Context, appID, userID int64, scopes []string) (bool, error) {
	if len(scopes) == 0 {
		return false, ErrInvalidScope
	}
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT scope)
		FROM open_platform_user_consents
		WHERE app_id = $1
		  AND user_id = $2
		  AND scope = ANY($3)
		  AND revoked_at IS NULL
	`, appID, userID, scopes).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("HasActiveConsents: %w", err)
	}
	return count == len(scopes), nil
}

func (r *Repository) GrantConsents(ctx context.Context, consent Consent, scopes []string) error {
	if len(scopes) == 0 {
		return ErrInvalidScope
	}
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		for _, scope := range scopes {
			if _, err := tx.Exec(ctx, `
				INSERT INTO open_platform_user_consents (
					app_id, user_id, scope, granted_at, revoked_at, grant_source,
					request_id
				) VALUES ($1, $2, $3, NOW(), NULL, $4, $5)
				ON CONFLICT (app_id, user_id, scope) DO UPDATE SET
					granted_at = NOW(),
					revoked_at = NULL,
					grant_source = EXCLUDED.grant_source,
					request_id = EXCLUDED.request_id
			`, consent.AppID, consent.UserID, scope,
				nonBlankOrDefault(consent.GrantSource, "web"), consent.RequestID,
			); err != nil {
				return fmt.Errorf("GrantConsents scope %s: %w", scope, err)
			}
		}
		return nil
	})
}

func (r *Repository) RevokeAppConsents(ctx context.Context, appID, userID int64, scopes []string) error {
	if len(scopes) == 0 {
		return r.revokeAllAppConsents(ctx, appID, userID)
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE open_platform_user_consents
		SET revoked_at = NOW()
		WHERE app_id = $1
		  AND user_id = $2
		  AND scope = ANY($3)
		  AND revoked_at IS NULL
	`, appID, userID, scopes)
	if err != nil {
		return fmt.Errorf("RevokeAppConsents: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConsentRequired
	}
	return nil
}

func (r *Repository) revokeAllAppConsents(ctx context.Context, appID, userID int64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE open_platform_user_consents
		SET revoked_at = NOW()
		WHERE app_id = $1
		  AND user_id = $2
		  AND revoked_at IS NULL
	`, appID, userID)
	if err != nil {
		return fmt.Errorf("RevokeAllAppConsents: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConsentRequired
	}
	return nil
}

func nonBlankOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
