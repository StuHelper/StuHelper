package review

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// GetReviewByID 根据ID获取已发布的评论（仅返回 published 状态）
// M-55: 包装 pgx.ErrNoRows 为业务层 sentinel error
func (r *Repository) GetReviewByID(ctx context.Context, reviewID string) (*Review, error) {
	var item Review
	err := r.db.QueryRow(ctx, `
		SELECT r.id, r.course_id, COALESCE(c.name, ''), r.teacher_id, COALESCE(t.name, ''), r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count,
		       r.reply_count,
		       r.status, r.moderation_reason, r.created_at, r.updated_at
		FROM reviews r
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
		WHERE r.id = $1 AND r.status IN ('published', 'hidden')
	`, reviewID).Scan(
		&item.ID, &item.CourseID, &item.CourseName, &item.TeacherID, &item.TeacherName,
		&item.TermID, &item.Title, &item.Content, &item.Grade, &item.Ratings,
		&item.LikeCount, &item.DislikeCount, &item.ReplyCount,
		&item.Status, &item.ModerationReason, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReviewNotFound
		}
		return nil, fmt.Errorf("GetReviewByID: %w", err)
	}
	return &item, nil
}

// GetReviewOwner 获取评论所有者哈希
func (r *Repository) GetReviewOwner(ctx context.Context, reviewID string) (string, error) {
	var userHash string
	err := r.db.QueryRow(ctx, `SELECT user_hash FROM reviews WHERE id = $1`, reviewID).Scan(&userHash)
	return userHash, err
}

// GetReviewOwnerAndStatus 获取评论所有者哈希和状态（单次查询）
func (r *Repository) GetReviewOwnerAndStatus(ctx context.Context, reviewID string) (string, string, error) {
	var userHash, status string
	err := r.db.QueryRow(ctx, `SELECT user_hash, status FROM reviews WHERE id = $1`, reviewID).Scan(&userHash, &status)
	return userHash, status, err
}

// GetReviewOwnerAndStatusTx 在事务内获取评论所有者和状态（带行锁）
func (r *Repository) GetReviewOwnerAndStatusTx(ctx context.Context, tx pgx.Tx, reviewID string) (string, string, error) {
	var userHash, status string
	err := tx.QueryRow(ctx, `SELECT user_hash, status FROM reviews WHERE id = $1 FOR UPDATE`, reviewID).Scan(&userHash, &status)
	return userHash, status, err
}

// GetReviewOwnerCourseIDAndStatusTx 在事务内获取评论所有者、课程ID和状态（带行锁）
func (r *Repository) GetReviewOwnerCourseIDAndStatusTx(ctx context.Context, tx pgx.Tx, reviewID string) (string, int64, string, error) {
	var userHash, status string
	var courseID int64
	err := tx.QueryRow(ctx, `SELECT user_hash, course_id, status FROM reviews WHERE id = $1 FOR UPDATE`, reviewID).Scan(&userHash, &courseID, &status)
	return userHash, courseID, status, err
}

// GetReviewOwnerAndCourseID 获取评论所有者哈希和课程ID（单次查询）
func (r *Repository) GetReviewOwnerAndCourseID(ctx context.Context, reviewID string) (string, int64, error) {
	var userHash string
	var courseID int64
	err := r.db.QueryRow(ctx, `SELECT user_hash, course_id FROM reviews WHERE id = $1`, reviewID).Scan(&userHash, &courseID)
	return userHash, courseID, err
}

// UpdateParams 更新评论参数
type UpdateParams struct {
	ID      string
	Title   string
	Content string
	Grade   string
	Ratings []byte
}

// Update 更新评论
func (r *Repository) Update(ctx context.Context, tx pgx.Tx, p UpdateParams) error {
	_, err := tx.Exec(ctx, `
		UPDATE reviews SET title = $2, content = $3, grade = $4, ratings = $5,
			avg_rating = COALESCE((SELECT AVG(value::numeric) FROM jsonb_each_text($5) WHERE value ~ '^\d+(\.\d+)?$'), 0),
			updated_at = NOW()
		WHERE id = $1
	`, p.ID, p.Title, p.Content, p.Grade, p.Ratings)
	return err
}

// SoftDeleteReview 软删除评论
func (r *Repository) SoftDeleteReview(ctx context.Context, tx pgx.Tx, reviewID string) error {
	_, err := tx.Exec(ctx, `UPDATE reviews SET status = 'deleted', updated_at = NOW() WHERE id = $1`, reviewID)
	return err
}

// UpdateReviewStatus 更新评论状态
func (r *Repository) UpdateReviewStatus(ctx context.Context, tx pgx.Tx, reviewID, status string) error {
	_, err := tx.Exec(ctx, `UPDATE reviews SET status = $2, updated_at = NOW() WHERE id = $1`, reviewID, status)
	return err
}

// DecrementCourseReviewCount 减少课程评论计数
func (r *Repository) DecrementCourseReviewCount(ctx context.Context, tx pgx.Tx, courseID int64) error {
	_, err := tx.Exec(ctx, `UPDATE courses SET review_count = GREATEST(review_count - 1, 0) WHERE id = $1`, courseID)
	return err
}

// GetReviewCourseID 获取评论的课程ID
func (r *Repository) GetReviewCourseID(ctx context.Context, reviewID string) (int64, error) {
	var courseID int64
	err := r.db.QueryRow(ctx, `SELECT course_id FROM reviews WHERE id = $1`, reviewID).Scan(&courseID)
	return courseID, err
}

// GetReviewCourseIDTx 在事务内获取评论的课程ID
func (r *Repository) GetReviewCourseIDTx(ctx context.Context, tx pgx.Tx, reviewID string) (int64, error) {
	var courseID int64
	err := tx.QueryRow(ctx, `SELECT course_id FROM reviews WHERE id = $1`, reviewID).Scan(&courseID)
	return courseID, err
}

// GetReviewStatusAndCourseIDTx 在事务内获取评论状态和课程ID
func (r *Repository) GetReviewStatusAndCourseIDTx(ctx context.Context, tx pgx.Tx, reviewID string) (string, int64, error) {
	var status string
	var courseID int64
	err := tx.QueryRow(ctx, `SELECT status, course_id FROM reviews WHERE id = $1 FOR UPDATE`, reviewID).Scan(&status, &courseID)
	return status, courseID, err
}

// maxBatchCourseIDs limits the number of course IDs in a batch query
const maxBatchCourseIDs = 20

// ListByMultipleCourses 批量获取多个课程的测评列表（消除 N+1 查询）
// 返回按 courseID 分组的结果 map
func (r *Repository) ListByMultipleCourses(ctx context.Context, courseIDs []int64, sort string, limit int) (map[int64][]Review, map[int64]int, error) {
	if len(courseIDs) == 0 {
		return nil, nil, nil
	}
	if len(courseIDs) > maxBatchCourseIDs {
		courseIDs = courseIDs[:maxBatchCourseIDs]
	}

	orderBy, ok := allowedSortOrders[sort]
	if !ok {
		orderBy = allowedSortOrders[SortTime]
	}

	// 使用 LATERAL join 高效获取每个课程的 top-N 测评
	rows, err := r.db.Query(ctx, `
		SELECT sub.course_id, sub.id, c.name, sub.teacher_id, t.name, sub.term_id,
		       sub.title, sub.content, sub.grade, sub.ratings,
		       sub.like_count, sub.dislike_count,
		       sub.reply_count,
		       sub.status, sub.moderation_reason, sub.created_at, sub.updated_at,
		       sub.total
		FROM unnest($1::bigint[]) AS cid(id)
		CROSS JOIN LATERAL (
			SELECT r.*, COUNT(*) OVER() AS total
			FROM reviews r
			WHERE r.course_id = cid.id AND r.status IN ('published', 'hidden')
			ORDER BY `+orderBy+`
			LIMIT $2
		) sub
		LEFT JOIN courses c ON c.id = sub.course_id
		LEFT JOIN teachers t ON t.id = sub.teacher_id
	`, courseIDs, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("ListByMultipleCourses: %w", err)
	}
	defer rows.Close()

	reviewsMap := make(map[int64][]Review)
	totalsMap := make(map[int64]int)
	for rows.Next() {
		var courseID int64
		var item Review
		var total int
		if err := rows.Scan(
			&courseID,
			&item.ID, &item.CourseName, &item.TeacherID, &item.TeacherName,
			&item.TermID, &item.Title, &item.Content, &item.Grade, &item.Ratings,
			&item.LikeCount, &item.DislikeCount, &item.ReplyCount,
			&item.Status, &item.ModerationReason, &item.CreatedAt, &item.UpdatedAt,
			&total,
		); err != nil {
			return nil, nil, fmt.Errorf("ListByMultipleCourses scan: %w", err)
		}
		item.CourseID = courseID
		reviewsMap[courseID] = append(reviewsMap[courseID], item)
		totalsMap[courseID] = total
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("ListByMultipleCourses rows: %w", err)
	}

	return reviewsMap, totalsMap, nil
}

// allowedSortOrders 排序参数白名单（纵深防御：即使 handler 层已校验，Repository 层也独立校验）
// L-56: 列别名约定 — 所有别名均使用 "r." 前缀引用 reviews 表列，
// 与 ListByCourseWithSort / ListLatest 的 SELECT 别名一致。
// 修改此 map 时需同步检查对应查询的 SELECT 列别名。
var allowedSortOrders = map[string]string{
	SortTime:   "r.created_at DESC",
	SortLikes:  "r.like_count DESC, r.created_at DESC",
	SortRating: "r.avg_rating DESC, r.created_at DESC",
}

// ListByCourseWithSortParams 带排序筛选的评论列表参数
type ListByCourseWithSortParams struct {
	CourseID  int64
	Sort      string // time, likes, rating
	TermID    string
	TeacherID *int64
	Limit     int
	Offset    int
}

// ListByCourseWithSort 获取课程评论列表（支持排序和筛选，含总数）
// L-33: 使用 strings.Builder 构建动态查询，提高可读性
func (r *Repository) ListByCourseWithSort(ctx context.Context, p ListByCourseWithSortParams) ([]Review, int, error) {
	var qb strings.Builder
	qb.WriteString(`
		SELECT r.id, r.course_id, COALESCE(c.name, ''), r.teacher_id, COALESCE(t.name, ''), r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count,
		       r.reply_count,
		       r.status, r.moderation_reason, r.created_at, r.updated_at,
		       COUNT(*) OVER() AS total
		FROM reviews r
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
		WHERE r.course_id = $1 AND r.status IN ('published', 'hidden')
	`)
	args := []interface{}{p.CourseID}
	argIdx := 2

	if p.TermID != "" {
		qb.WriteString(` AND r.term_id = $` + strconv.Itoa(argIdx))
		args = append(args, p.TermID)
		argIdx++
	}
	if p.TeacherID != nil {
		qb.WriteString(` AND r.teacher_id = $` + strconv.Itoa(argIdx))
		args = append(args, *p.TeacherID)
		argIdx++
	}

	// SQL 注入安全保证：orderBy 仅来自 allowedSortOrders 硬编码 map，不含用户输入
	orderBy, ok := allowedSortOrders[p.Sort]
	if !ok {
		orderBy = allowedSortOrders["time"]
	}
	qb.WriteString(` ORDER BY ` + orderBy)

	qb.WriteString(` LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1))
	args = append(args, p.Limit, p.Offset)

	rows, err := r.db.Query(ctx, qb.String(), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanReviewsWithTotal(rows)
}

// ListByUserHash 获取用户的评论列表（含总数）
func (r *Repository) ListByUserHash(ctx context.Context, userHash string, limit, offset int) ([]Review, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.id, r.course_id, COALESCE(c.name, ''), r.teacher_id, COALESCE(t.name, ''), r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count,
		       r.reply_count,
		       r.status, r.moderation_reason, r.created_at, r.updated_at,
		       COUNT(*) OVER() AS total
		FROM reviews r
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
		WHERE r.user_hash = $1 AND r.status != 'deleted'
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3
	`, userHash, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanReviewsWithTotal(rows)
}

// ListVotedReviews 获取用户点赞/踩的评论列表（含总数）
func (r *Repository) ListVotedReviews(ctx context.Context, userHash, voteType string, limit, offset int) ([]Review, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.id, r.course_id, COALESCE(c.name, ''), r.teacher_id, COALESCE(t.name, ''), r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count,
		       r.reply_count,
		       r.status, r.moderation_reason, r.created_at, r.updated_at,
		       COUNT(*) OVER() AS total
		FROM reviews r
		JOIN review_votes rv ON rv.review_id = r.id
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
		WHERE rv.user_hash = $1 AND rv.vote_type = $2 AND r.status = 'published'
		ORDER BY rv.created_at DESC
		LIMIT $3 OFFSET $4
	`, userHash, voteType, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanReviewsWithTotal(rows)
}
