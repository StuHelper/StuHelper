package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// SetUserPhone 将手机号写入 users 表的加密列和哈希列。
func (r *Repository) SetUserPhone(ctx context.Context, userID int64, phoneEnc []byte, phoneHash string) error {
	var conflictID int64
	err := r.db.QueryRow(ctx, `
		SELECT id
		FROM users
		WHERE id != $1
		  AND phone_hash = $2
		LIMIT 1
	`, userID, phoneHash).Scan(&conflictID)
	if err == nil {
		return ErrPhoneAlreadyBound
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("SetUserPhone check conflict: %w", err)
	}

	_, err = r.db.Exec(ctx, `
		UPDATE users
		SET phone_enc = $2,
		    phone_hash = $3,
		    updated_at = NOW()
		WHERE id = $1
	`, userID, phoneEnc, phoneHash)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrPhoneAlreadyBound
		}
		return fmt.Errorf("SetUserPhone: %w", err)
	}
	return nil
}

func (r *Repository) SetUserPhoneTx(ctx context.Context, tx pgx.Tx, userID int64, phoneEnc []byte, phoneHash string) error {
	var conflictID int64
	err := tx.QueryRow(ctx, `
		SELECT id
		FROM users
		WHERE id != $1
		  AND phone_hash = $2
		LIMIT 1
	`, userID, phoneHash).Scan(&conflictID)
	if err == nil {
		return ErrPhoneAlreadyBound
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("SetUserPhoneTx check conflict: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE users
		SET phone_enc = $2,
		    phone_hash = $3,
		    updated_at = NOW()
		WHERE id = $1
	`, userID, phoneEnc, phoneHash)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrPhoneAlreadyBound
		}
		return fmt.Errorf("SetUserPhoneTx: %w", err)
	}
	return nil
}
