package review

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// GetReviewContentFlagStateTx 在事务内读取评论审核状态和内容标记。
func (r *Repository) GetReviewContentFlagStateTx(ctx context.Context, tx pgx.Tx, reviewID string) (string, *string, int64, *int64, error) {
	var (
		status      string
		contentFlag *string
		courseID    int64
		teacherID   *int64
	)

	err := tx.QueryRow(ctx, `
		SELECT status, content_flag, course_id, teacher_id
		FROM reviews
		WHERE id = $1
		FOR UPDATE
	`, reviewID).Scan(&status, &contentFlag, &courseID, &teacherID)
	if err != nil {
		return "", nil, 0, nil, fmt.Errorf("GetReviewContentFlagStateTx: %w", err)
	}
	return status, contentFlag, courseID, teacherID, nil
}

// ClearContentFlagTx 管理员复核通过，清除 warn/review 内容标记。
func (r *Repository) ClearContentFlagTx(ctx context.Context, tx pgx.Tx, reviewID, adminUserID string) error {
	tag, err := tx.Exec(ctx, `
		UPDATE reviews
		SET content_flag = 'cleared',
		    content_flag_cleared_at = $2,
		    content_flag_cleared_by = $3,
		    updated_at = NOW()
		WHERE id = $1 AND content_flag IN ('warn', 'review')
	`, reviewID, time.Now().UTC(), adminUserID)
	if err != nil {
		return fmt.Errorf("ClearContentFlagTx: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrReviewNotFound
	}
	return nil
}

func (r *Repository) MarkContentFlagClearedTx(ctx context.Context, tx pgx.Tx, reviewID, adminUserID string) error {
	tag, err := tx.Exec(ctx, `
		UPDATE reviews
		SET content_flag = 'cleared',
		    content_flag_cleared_at = $2,
		    content_flag_cleared_by = $3,
		    updated_at = NOW()
		WHERE id = $1
	`, reviewID, time.Now().UTC(), adminUserID)
	if err != nil {
		return fmt.Errorf("MarkContentFlagClearedTx: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrReviewNotFound
	}
	return nil
}

// ListFlaggedReviews 获取待复核评课列表（content_flag in warn/review）。
func (r *Repository) ListFlaggedReviews(ctx context.Context, limit, offset int, schoolIDs []int64) ([]Review, int, error) {
	ctx = withDBTable(ctx, "reviews")
	var qb strings.Builder
	qb.WriteString(`
		SELECT r.id, r.course_id, r.title, r.content, r.status, r.content_flag,
		       r.user_hash, r.created_at, r.updated_at,
		       COUNT(*) OVER() AS total
	FROM reviews r
	WHERE r.content_flag IN ('warn', 'review')
	`)
	args := make([]interface{}, 0, 3)
	if len(schoolIDs) > 0 {
		qb.WriteString(` AND r.school_id = ANY($1)`)
		args = append(args, schoolIDs)
	}
	qb.WriteString(` ORDER BY CASE WHEN r.content_flag = 'review' THEN 0 ELSE 1 END, r.created_at DESC`)
	qb.WriteString(` LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2))
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, qb.String(), args...)
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
