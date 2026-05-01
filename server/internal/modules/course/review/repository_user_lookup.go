package review

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GetInternalUserIDByCasdoorSubject 根据外部用户 ID 查询内部 user_id。
func (r *Repository) GetInternalUserIDByCasdoorSubject(ctx context.Context, casdoorSubject string) (int64, error) {
	var userID int64
	err := r.db.QueryRow(ctx, `SELECT id FROM users WHERE casdoor_subject = $1`, casdoorSubject).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("GetInternalUserIDByCasdoorSubject: %w", err)
	}
	return userID, nil
}

// GetInternalUserIDByCasdoorSubjectTx resolves the OIDC subject to users.id inside a business transaction.
func (r *Repository) GetInternalUserIDByCasdoorSubjectTx(ctx context.Context, tx pgx.Tx, casdoorSubject string) (int64, error) {
	var userID int64
	err := tx.QueryRow(ctx, `SELECT id FROM users WHERE casdoor_subject = $1`, casdoorSubject).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("GetInternalUserIDByCasdoorSubjectTx: %w", err)
	}
	return userID, nil
}

// GetUserIDByUserHash 根据 user_hash 解析内部用户 ID。
// 未命中时返回 0，且不视为错误。
func (r *Repository) GetUserIDByUserHash(ctx context.Context, userHash string) (int64, error) {
	var userID int64
	err := r.db.QueryRow(ctx, `SELECT id FROM users WHERE user_hash = $1`, userHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("GetUserIDByUserHash: %w", err)
	}
	return userID, nil
}
