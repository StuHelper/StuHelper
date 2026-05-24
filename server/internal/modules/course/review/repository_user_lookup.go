package review

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GetUserIDByUserHash 根据 user_hash 解析内部用户 ID。
// 未命中时返回 0，且不视为错误。
func (r *Repository) GetUserIDByUserHash(ctx context.Context, userHash string) (int64, error) {
	ctx = withDBTable(ctx, "users")
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
