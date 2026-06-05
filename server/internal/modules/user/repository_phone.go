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

type queryRowFn func(ctx context.Context, sql string, args ...any) rowScanner

func ensureUserPhoneAvailable(ctx context.Context, queryRow queryRowFn, userID int64, phoneHash, op string) error {
	var conflictID int64
	err := queryRow(ctx, `
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
		return fmt.Errorf("%s check conflict: %w", op, err)
	}
	return nil
}

func setUserPhone(ctx context.Context, queryRow queryRowFn, exec execFn, userID int64, phoneEnc []byte, phoneHash, op string) error {
	if err := ensureUserPhoneAvailable(ctx, queryRow, userID, phoneHash, op); err != nil {
		return err
	}
	_, err := exec(ctx, `
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
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// EnsureUserPhoneAvailable 检查手机号投影是否已被其他用户绑定。
func (r *Repository) EnsureUserPhoneAvailable(ctx context.Context, userID int64, phoneHash string) error {
	ctx = withDBTable(ctx, "users")
	return ensureUserPhoneAvailable(ctx, func(ctx context.Context, sql string, args ...any) rowScanner { return r.db.QueryRow(ctx, sql, args...) }, userID, phoneHash, "EnsureUserPhoneAvailable")
}

// SetUserPhone 将手机号写入 users 表的加密列和哈希列。
func (r *Repository) SetUserPhone(ctx context.Context, userID int64, phoneEnc []byte, phoneHash string) error {
	ctx = withDBTable(ctx, "users")
	return setUserPhone(ctx, func(ctx context.Context, sql string, args ...any) rowScanner { return r.db.QueryRow(ctx, sql, args...) }, r.db.Exec, userID, phoneEnc, phoneHash, "SetUserPhone")
}

func (r *Repository) SetUserPhoneTx(ctx context.Context, tx pgx.Tx, userID int64, phoneEnc []byte, phoneHash string) error {
	return setUserPhone(ctx, func(ctx context.Context, sql string, args ...any) rowScanner { return tx.QueryRow(ctx, sql, args...) }, tx.Exec, userID, phoneEnc, phoneHash, "SetUserPhoneTx")
}
