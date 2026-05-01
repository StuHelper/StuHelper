package user

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) ReplaceMFARecoveryCodesTx(
	ctx context.Context,
	tx pgx.Tx,
	params MFARecoveryCodeReplace,
) error {
	if err := validateMFARecoveryHashes(params.UserID, params.CodeHashes); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_mfa_recovery_codes WHERE user_id = $1`, params.UserID); err != nil {
		return fmt.Errorf("delete mfa recovery codes: %w", err)
	}
	for _, hash := range params.CodeHashes {
		if err := insertMFARecoveryCode(ctx, tx, mfaRecoveryCodeRecord{UserID: params.UserID, CodeHash: hash}); err != nil {
			return err
		}
	}
	if err := recordMFARecoveryIssuedAt(ctx, tx, params); err != nil {
		return err
	}
	return nil
}

func (r *Repository) ConsumeMFARecoveryCodeTx(
	ctx context.Context,
	tx pgx.Tx,
	params MFARecoveryCodeConsume,
) (bool, error) {
	if params.UserID <= 0 || strings.TrimSpace(params.CodeHash) == "" {
		return false, ErrMFARecoveryCodeInvalid
	}
	var id int64
	err := tx.QueryRow(ctx, `
		UPDATE user_mfa_recovery_codes
		SET used_at = $3, updated_at = NOW()
		WHERE user_id = $1 AND code_hash = $2 AND used_at IS NULL
		RETURNING id
	`, params.UserID, strings.TrimSpace(params.CodeHash), params.UsedAt).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("consume mfa recovery code: %w", err)
	}
	return true, nil
}

func validateMFARecoveryHashes(userID int64, codeHashes []string) error {
	if userID <= 0 || len(codeHashes) != mfaRecoveryCodeCount {
		return ErrMFARecoveryCodeInvalid
	}
	seen := make([]string, 0, len(codeHashes))
	for _, hash := range codeHashes {
		trimmed := strings.TrimSpace(hash)
		if trimmed == "" || slices.Contains(seen, trimmed) {
			return ErrMFARecoveryCodeInvalid
		}
		seen = append(seen, trimmed)
	}
	return nil
}

type mfaRecoveryCodeRecord struct {
	UserID   int64
	CodeHash string
}

func insertMFARecoveryCode(ctx context.Context, tx pgx.Tx, params mfaRecoveryCodeRecord) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO user_mfa_recovery_codes (user_id, code_hash, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`, params.UserID, strings.TrimSpace(params.CodeHash))
	if err != nil {
		return fmt.Errorf("insert mfa recovery code: %w", err)
	}
	return nil
}

func recordMFARecoveryIssuedAt(ctx context.Context, tx pgx.Tx, params MFARecoveryCodeReplace) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO user_mfa_enrollment (
			user_id, active, methods, recovery_codes_issued_at, reset_required, updated_at
		) VALUES ($1, false, '{}'::TEXT[], $2, false, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			recovery_codes_issued_at = EXCLUDED.recovery_codes_issued_at,
			updated_at = NOW()
	`, params.UserID, params.IssuedAt)
	if err != nil {
		return fmt.Errorf("record mfa recovery issue time: %w", err)
	}
	return nil
}
