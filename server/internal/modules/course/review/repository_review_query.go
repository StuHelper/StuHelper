package review

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// ListRatingDimensions 获取评分维度列表
func (r *Repository) ListRatingDimensions(ctx context.Context) ([]RatingDimension, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, school_id, key, name, description, sort_order, is_active, created_at, updated_at
		FROM rating_dimensions
		WHERE is_active = true
		ORDER BY sort_order ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dimensions []RatingDimension
	for rows.Next() {
		var d RatingDimension
		if err := rows.Scan(
			&d.ID, &d.SchoolID, &d.Key, &d.Name, &d.Description,
			&d.SortOrder, &d.IsActive, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		dimensions = append(dimensions, d)
	}
	return dimensions, rows.Err()
}

// GetDimensionNames 获取维度名称映射
func (r *Repository) GetDimensionNames(ctx context.Context) (map[string]string, error) {
	rows, err := r.db.Query(ctx, `SELECT key, name FROM rating_dimensions WHERE is_active = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var key, name string
		if err := rows.Scan(&key, &name); err != nil {
			return nil, err
		}
		result[key] = name
	}
	return result, rows.Err()
}

// RatingStatRow 评分统计行
type RatingStatRow struct {
	TermID       *string
	DimensionKey string
	AvgRating    float64
	RatingCount  int
	RatingDist   []byte
}

// ListCourseRatingStats 获取课程评分统计
func (r *Repository) ListCourseRatingStats(ctx context.Context, courseID int64) ([]RatingStatRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT term_id, dimension_key, avg_rating, rating_count, rating_dist
		FROM course_rating_stats
		WHERE course_id = $1
		ORDER BY term_id NULLS FIRST
	`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []RatingStatRow
	for rows.Next() {
		var s RatingStatRow
		if err := rows.Scan(&s.TermID, &s.DimensionKey, &s.AvgRating, &s.RatingCount, &s.RatingDist); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetReviewByID 根据ID获取评论
func (r *Repository) GetReviewByID(ctx context.Context, reviewID string) (*Review, error) {
	var item Review
	err := r.db.QueryRow(ctx, `
		SELECT r.id, r.course_id, c.name, r.teacher_id, t.name, r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count,
		       (SELECT COUNT(*) FROM review_replies rr WHERE rr.review_id = r.id AND rr.status = 'published') AS reply_count,
		       r.status, r.created_at
		FROM reviews r
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
		WHERE r.id = $1
	`, reviewID).Scan(
		&item.ID, &item.CourseID, &item.CourseName, &item.TeacherID, &item.TeacherName,
		&item.TermID, &item.Title, &item.Content, &item.Grade, &item.Ratings,
		&item.LikeCount, &item.DislikeCount, &item.ReplyCount, &item.Status, &item.CreatedAt,
	)
	if err != nil {
		return nil, err
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

// allowedSortOrders 排序参数白名单（纵深防御：即使 handler 层已校验，Repository 层也独立校验）
var allowedSortOrders = map[string]string{
	"time":   "r.created_at DESC",
	"likes":  "r.like_count DESC, r.created_at DESC",
	"rating": "r.avg_rating DESC, r.created_at DESC",
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
func (r *Repository) ListByCourseWithSort(ctx context.Context, p ListByCourseWithSortParams) ([]Review, int, error) {
	query := `
		SELECT r.id, r.course_id, c.name, r.teacher_id, t.name, r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count,
		       (SELECT COUNT(*) FROM review_replies rr WHERE rr.review_id = r.id AND rr.status = 'published') AS reply_count,
		       r.status, r.created_at,
		       COUNT(*) OVER() AS total
		FROM reviews r
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
		WHERE r.course_id = $1 AND r.status = 'published'
	`
	args := []interface{}{p.CourseID}
	argIdx := 2

	if p.TermID != "" {
		query += ` AND r.term_id = $` + strconv.Itoa(argIdx)
		args = append(args, p.TermID)
		argIdx++
	}
	if p.TeacherID != nil {
		query += ` AND r.teacher_id = $` + strconv.Itoa(argIdx)
		args = append(args, *p.TeacherID)
		argIdx++
	}

	orderBy, ok := allowedSortOrders[p.Sort]
	if !ok {
		orderBy = allowedSortOrders["time"]
	}
	query += ` ORDER BY ` + orderBy

	query += ` LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
	args = append(args, p.Limit, p.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanReviewsWithTotal(rows)
}

// ListByUserHash 获取用户的评论列表（含总数）
func (r *Repository) ListByUserHash(ctx context.Context, userHash string, limit, offset int) ([]Review, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.id, r.course_id, c.name, r.teacher_id, t.name, r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count,
		       (SELECT COUNT(*) FROM review_replies rr WHERE rr.review_id = r.id AND rr.status = 'published') AS reply_count,
		       r.status, r.created_at,
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
		SELECT r.id, r.course_id, c.name, r.teacher_id, t.name, r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count,
		       (SELECT COUNT(*) FROM review_replies rr WHERE rr.review_id = r.id AND rr.status = 'published') AS reply_count,
		       r.status, r.created_at,
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
