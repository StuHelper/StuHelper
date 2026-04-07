package review

import (
	"context"
	"fmt"
	"time"
)

// ClearContentFlag 管理员复核通过，清除 content_flag
func (r *Repository) ClearContentFlag(ctx context.Context, reviewID, adminUserID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE reviews
		SET content_flag = 'cleared',
		    content_flag_cleared_at = $2,
		    content_flag_cleared_by = $3,
		    updated_at = NOW()
		WHERE id = $1 AND content_flag = 'warn'
	`, reviewID, time.Now().UTC(), adminUserID)
	if err != nil {
		return fmt.Errorf("ClearContentFlag: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrReviewNotFound
	}
	return nil
}

// ListFlaggedReviews 获取待复核评课列表（content_flag = 'warn'）
func (r *Repository) ListFlaggedReviews(ctx context.Context, limit, offset int) ([]Review, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.id, r.course_id, r.title, r.content, r.status, r.content_flag,
		       r.user_hash, r.created_at, r.updated_at,
		       COUNT(*) OVER() AS total
		FROM reviews r
		WHERE r.content_flag = 'warn'
		ORDER BY r.created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("ListFlaggedReviews: %w", err)
	}
	defer rows.Close()

	var total int
	list := make([]Review, 0, limit)
	for rows.Next() {
		var review Review
		if err := rows.Scan(
			&review.ID, &review.CourseID, &review.Title, &review.Content,
			&review.Status, &review.ContentFlag,
			&review.UserHash, &review.CreatedAt, &review.UpdatedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("ListFlaggedReviews scan: %w", err)
		}
		review.UserHash = "" // 防御性清除，避免 user_hash 泄露到响应
		list = append(list, review)
	}
	return list, total, rows.Err()
}
