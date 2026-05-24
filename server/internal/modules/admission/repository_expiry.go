package admission

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) ListExpiredFreshmanCredentials(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]ExpiredFreshmanCredential, error) {
	ctx = withDBTable(ctx, "user_verification_credentials")
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, expires_at
		FROM user_verification_credentials
		WHERE kind = $1
		  AND expires_at <= $2
		  AND revoked_at IS NULL
		  AND expiry_processed_at IS NULL
		ORDER BY expires_at ASC, id ASC
		LIMIT $3
	`, CredentialFreshmanMaterialManual, now, limit)
	if err != nil {
		return nil, fmt.Errorf("ListExpiredFreshmanCredentials: %w", err)
	}
	defer rows.Close()
	return scanExpiredFreshmanCredentials(rows)
}

func (r *Repository) MarkFreshmanCredentialExpiryProcessedTx(
	ctx context.Context,
	tx pgx.Tx,
	credentialID string,
	now time.Time,
) error {
	_, err := tx.Exec(ctx, `
		UPDATE user_verification_credentials
		SET revoked_at = $2, expiry_processed_at = $2, updated_at = NOW()
		WHERE id = $1
	`, credentialID, now)
	if err != nil {
		return fmt.Errorf("MarkFreshmanCredentialExpiryProcessedTx: %w", err)
	}
	return nil
}

func scanExpiredFreshmanCredentials(rows pgx.Rows) ([]ExpiredFreshmanCredential, error) {
	items := make([]ExpiredFreshmanCredential, 0)
	for rows.Next() {
		var item ExpiredFreshmanCredential
		if err := rows.Scan(&item.ID, &item.UserID, &item.ExpiresAt); err != nil {
			return nil, fmt.Errorf("scan expired freshman credential: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("expired freshman credential rows: %w", err)
	}
	return items, nil
}
