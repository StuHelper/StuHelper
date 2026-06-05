package serviceaccount

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

const bootstrapCredentialSQL = `
	WITH existing AS (
		SELECT token_hash
		FROM bot_service_credentials
		WHERE name = $1
	), upserted AS (
		INSERT INTO bot_service_credentials (
			name, token_hash, audience, scopes, expires_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, NOW(), NOW()
		)
		ON CONFLICT (name) DO UPDATE SET
			token_hash = EXCLUDED.token_hash,
			audience = EXCLUDED.audience,
			scopes = EXCLUDED.scopes,
			expires_at = EXCLUDED.expires_at,
			revoked_at = CASE
				WHEN bot_service_credentials.token_hash <> EXCLUDED.token_hash THEN NULL
				ELSE bot_service_credentials.revoked_at
			END,
			rotated_at = CASE
				WHEN bot_service_credentials.token_hash <> EXCLUDED.token_hash THEN NOW()
				ELSE bot_service_credentials.rotated_at
			END,
			updated_at = NOW()
		RETURNING id, name, token_hash
	)
	SELECT upserted.id, upserted.name,
		CASE
			WHEN existing.token_hash IS NULL THEN 'created'
			WHEN existing.token_hash <> upserted.token_hash THEN 'rotated'
			ELSE 'unchanged'
		END AS status
	FROM upserted
	LEFT JOIN existing ON TRUE
`

var errCredentialRecordNotFound = errors.New("service account credential record not found")

type Repository struct {
	db *db.DB
}

type credentialRecord struct {
	ID        int64
	Name      string
	Audiences []string
	Scopes    []string
	ExpiresAt sql.NullTime
	RevokedAt sql.NullTime
}

func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

func (r *Repository) EnsureBootstrapCredential(
	ctx context.Context,
	credential BootstrapCredential,
	tokenHash string,
) (BootstrapResult, error) {
	row := r.db.QueryRow(ctx, bootstrapCredentialSQL,
		credential.Name,
		tokenHash,
		credential.Audience,
		credential.Scopes,
		credential.ExpiresAt,
	)
	var result BootstrapResult
	var status string
	if err := row.Scan(&result.ID, &result.Name, &status); err != nil {
		return BootstrapResult{}, fmt.Errorf("bootstrap service account credential: %w", err)
	}
	result.Status = BootstrapStatus(status)
	return result, nil
}

func (r *Repository) RevokeCredential(ctx context.Context, name string) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		UPDATE bot_service_credentials
		SET revoked_at = NOW(), updated_at = NOW()
		WHERE name = $1 AND revoked_at IS NULL
		RETURNING id
	`, name).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, errCredentialRecordNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("revoke service account credential: %w", err)
	}
	return id, nil
}

func (r *Repository) LoadCredentialByTokenHash(ctx context.Context, tokenHash string) (*credentialRecord, error) {
	var record credentialRecord
	err := r.db.QueryRow(ctx, `
		SELECT id, name, audience, scopes, expires_at, revoked_at
		FROM bot_service_credentials
		WHERE token_hash = $1
	`, tokenHash).Scan(
		&record.ID,
		&record.Name,
		&record.Audiences,
		&record.Scopes,
		&record.ExpiresAt,
		&record.RevokedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errCredentialRecordNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load service account credential: %w", err)
	}
	return &record, nil
}

func (r *Repository) TouchLastUsed(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE bot_service_credentials
		SET last_used_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("touch service account credential usage: %w", err)
	}
	return nil
}
