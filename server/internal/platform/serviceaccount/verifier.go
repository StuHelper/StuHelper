package serviceaccount

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

const (
	wildcardSuffix         = "*"
	bootstrapCredentialSQL = `
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
)

type BootstrapStatus string

const (
	BootstrapUnchanged BootstrapStatus = "unchanged"
	BootstrapCreated   BootstrapStatus = "created"
	BootstrapRotated   BootstrapStatus = "rotated"
)

type BootstrapCredential struct {
	Name      string
	RawToken  string
	Audience  []string
	Scopes    []string
	ExpiresAt *time.Time
}

type BootstrapResult struct {
	ID     int64
	Name   string
	Status BootstrapStatus
}

type Verifier struct {
	db      *db.DB
	hmacKey []byte
}

func NewVerifier(database *db.DB, hmacKey []byte) (*Verifier, error) {
	if database == nil {
		return nil, fmt.Errorf("service account verifier: database is required")
	}
	if len(hmacKey) == 0 {
		return nil, fmt.Errorf("service account verifier: HMAC key is required")
	}
	keyCopy := append([]byte(nil), hmacKey...)
	return &Verifier{db: database, hmacKey: keyCopy}, nil
}

func (v *Verifier) EnsureBootstrapCredential(ctx context.Context, input BootstrapCredential) (BootstrapResult, error) {
	normalized, tokenHash, err := v.normalizeBootstrap(input)
	if err != nil {
		return BootstrapResult{}, err
	}

	row := v.db.QueryRow(ctx, bootstrapCredentialSQL,
		normalized.Name,
		tokenHash,
		normalized.Audience,
		normalized.Scopes,
		normalized.ExpiresAt,
	)
	var result BootstrapResult
	var status string
	if err := row.Scan(&result.ID, &result.Name, &status); err != nil {
		return BootstrapResult{}, fmt.Errorf("bootstrap service account credential: %w", err)
	}
	result.Status = BootstrapStatus(status)
	return result, nil
}

func (v *Verifier) Verify(ctx context.Context, rawToken, audience, scope string) error {
	tokenHash, err := v.hashToken(strings.TrimSpace(rawToken))
	if err != nil {
		return err
	}

	var id int64
	var audiences []string
	var scopes []string
	var expiresAt sql.NullTime
	var revokedAt sql.NullTime
	err = v.db.QueryRow(ctx, `
		SELECT id, audience, scopes, expires_at, revoked_at
		FROM bot_service_credentials
		WHERE token_hash = $1
	`, tokenHash).Scan(&id, &audiences, &scopes, &expiresAt, &revokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCredentialInvalid
	}
	if err != nil {
		return fmt.Errorf("verify service account credential: %w", err)
	}

	if revokedAt.Valid || isExpired(expiresAt) {
		return ErrCredentialInvalid
	}
	if !audienceAllowed(audiences, audience) || !slices.Contains(scopes, scope) {
		return ErrCredentialForbidden
	}
	if err := v.touchLastUsed(ctx, id); err != nil {
		return err
	}
	return nil
}

func (v *Verifier) Revoke(ctx context.Context, name string) error {
	normalizedName := strings.TrimSpace(name)
	if normalizedName == "" {
		return ErrCredentialNotConfigured
	}
	tag, err := v.db.Exec(ctx, `
		UPDATE bot_service_credentials
		SET revoked_at = NOW(), updated_at = NOW()
		WHERE name = $1 AND revoked_at IS NULL
	`, normalizedName)
	if err != nil {
		return fmt.Errorf("revoke service account credential: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCredentialInvalid
	}
	return nil
}

func (v *Verifier) normalizeBootstrap(input BootstrapCredential) (BootstrapCredential, string, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.RawToken = strings.TrimSpace(input.RawToken)
	input.Audience = compactNonEmpty(input.Audience)
	input.Scopes = compactNonEmpty(input.Scopes)
	if input.Name == "" || input.RawToken == "" || len(input.Audience) == 0 || len(input.Scopes) == 0 {
		return BootstrapCredential{}, "", ErrCredentialNotConfigured
	}
	tokenHash, err := v.hashToken(input.RawToken)
	if err != nil {
		return BootstrapCredential{}, "", err
	}
	return input, tokenHash, nil
}

func (v *Verifier) hashToken(rawToken string) (string, error) {
	if rawToken == "" {
		return "", ErrCredentialInvalid
	}
	tokenHash, err := crypto.HMACHashWithKey(rawToken, v.hmacKey)
	if err != nil {
		return "", fmt.Errorf("hash service account token: %w", err)
	}
	return tokenHash, nil
}

func (v *Verifier) touchLastUsed(ctx context.Context, id int64) error {
	_, err := v.db.Exec(ctx, `
		UPDATE bot_service_credentials
		SET last_used_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("touch service account credential usage: %w", err)
	}
	return nil
}

func compactNonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" && !slices.Contains(result, trimmed) {
			result = append(result, trimmed)
		}
	}
	return result
}

func audienceAllowed(audiences []string, required string) bool {
	required = strings.TrimSpace(required)
	for _, audience := range audiences {
		audience = strings.TrimSpace(audience)
		if audience == required {
			return true
		}
		if strings.HasSuffix(audience, wildcardSuffix) && strings.HasPrefix(required, strings.TrimSuffix(audience, wildcardSuffix)) {
			return true
		}
	}
	return false
}

func isExpired(expiresAt sql.NullTime) bool {
	return expiresAt.Valid && !expiresAt.Time.After(time.Now())
}
