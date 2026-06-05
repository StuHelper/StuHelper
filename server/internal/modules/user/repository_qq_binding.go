package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const selectQQBindingByUserIDSQL = `
		SELECT user_id, qq_id, bound_at, created_at, updated_at
		FROM user_qq_bindings
		WHERE user_id = $1
	`

const selectQQBindingByQQIDSQL = `
		SELECT user_id, qq_id, bound_at, created_at, updated_at
		FROM user_qq_bindings
		WHERE qq_id = $1
	`

const selectQQBindingCodeByHashSQL = `
		SELECT user_id, code_hash, expires_at, consumed_at, created_at, updated_at
		FROM user_qq_binding_codes
		WHERE code_hash = $1
	`

func scanQQBindingRow(row pgx.Row) (*QQBinding, error) {
	var item QQBinding
	if err := row.Scan(
		&item.UserID,
		&item.QQID,
		&item.BoundAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func scanQQBindingCodeRow(row pgx.Row) (*QQBindingCode, error) {
	var item QQBindingCode
	if err := row.Scan(
		&item.UserID,
		&item.CodeHash,
		&item.ExpiresAt,
		&item.ConsumedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *Repository) GetQQBindingByUserID(ctx context.Context, userID int64) (*QQBinding, error) {
	ctx = withDBTable(ctx, "user_qq_bindings")
	item, err := scanQQBindingRow(r.db.QueryRow(ctx, selectQQBindingByUserIDSQL, userID))
	if err != nil {
		return nil, fmt.Errorf("GetQQBindingByUserID: %w", err)
	}
	return item, nil
}

func (r *Repository) GetQQBindingByQQID(ctx context.Context, qqID string) (*QQBinding, error) {
	ctx = withDBTable(ctx, "user_qq_bindings")
	item, err := scanQQBindingRow(r.db.QueryRow(ctx, selectQQBindingByQQIDSQL, qqID))
	if err != nil {
		return nil, fmt.Errorf("GetQQBindingByQQID: %w", err)
	}
	return item, nil
}

func (r *Repository) UpsertQQBindingCode(ctx context.Context, code *QQBindingCode) error {
	ctx = withDBTable(ctx, "user_qq_binding_codes")
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_qq_binding_codes (
			user_id, code_hash, expires_at, consumed_at, created_at, updated_at
		) VALUES ($1, $2, $3, NULL, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET code_hash = EXCLUDED.code_hash,
		    expires_at = EXCLUDED.expires_at,
		    consumed_at = NULL,
		    updated_at = NOW()
	`, code.UserID, code.CodeHash, code.ExpiresAt)
	if err != nil {
		return fmt.Errorf("UpsertQQBindingCode: %w", err)
	}
	return nil
}

func (r *Repository) GetQQBindingCodeByHashTx(ctx context.Context, tx pgx.Tx, codeHash string) (*QQBindingCode, error) {
	item, err := scanQQBindingCodeRow(tx.QueryRow(ctx, selectQQBindingCodeByHashSQL+" FOR UPDATE", codeHash))
	if err != nil {
		return nil, fmt.Errorf("GetQQBindingCodeByHashTx: %w", err)
	}
	return item, nil
}

func (r *Repository) GetQQBindingByUserIDTx(ctx context.Context, tx pgx.Tx, userID int64) (*QQBinding, error) {
	item, err := scanQQBindingRow(tx.QueryRow(ctx, selectQQBindingByUserIDSQL, userID))
	if err != nil {
		return nil, fmt.Errorf("GetQQBindingByUserIDTx: %w", err)
	}
	return item, nil
}

func (r *Repository) GetQQBindingByQQIDTx(ctx context.Context, tx pgx.Tx, qqID string) (*QQBinding, error) {
	item, err := scanQQBindingRow(tx.QueryRow(ctx, selectQQBindingByQQIDSQL, qqID))
	if err != nil {
		return nil, fmt.Errorf("GetQQBindingByQQIDTx: %w", err)
	}
	return item, nil
}

func (r *Repository) CreateQQBindingTx(ctx context.Context, tx pgx.Tx, binding *QQBinding) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO user_qq_bindings (
			user_id, qq_id, bound_at, created_at, updated_at
		) VALUES ($1, $2, $3, NOW(), NOW())
	`, binding.UserID, binding.QQID, binding.BoundAt)
	if err != nil {
		return fmt.Errorf("CreateQQBindingTx: %w", err)
	}
	return nil
}

func (r *Repository) MarkQQBindingCodeConsumedTx(ctx context.Context, tx pgx.Tx, userID int64, consumedAt time.Time) error {
	_, err := tx.Exec(ctx, `
		UPDATE user_qq_binding_codes
		SET consumed_at = $2,
		    updated_at = NOW()
		WHERE user_id = $1
	`, userID, consumedAt)
	if err != nil {
		return fmt.Errorf("MarkQQBindingCodeConsumedTx: %w", err)
	}
	return nil
}
