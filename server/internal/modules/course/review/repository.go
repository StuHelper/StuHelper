package review

import (
	"context"

	"github.com/jackc/pgx/v5"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

// Repository 评课数据访问层
type Repository struct {
	db *db.DB
}

// NewRepository 创建数据访问层
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

// CourseExists 检查课程是否存在
func (r *Repository) CourseExists(ctx context.Context, courseID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM courses WHERE id = $1)`, courseID).Scan(&exists)
	return exists, err
}

// ReviewExists 检查评论是否存在
func (r *Repository) ReviewExists(ctx context.Context, reviewID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM reviews WHERE id = $1)`, reviewID).Scan(&exists)
	return exists, err
}

// CountByCourse 统计课程评论数量
func (r *Repository) CountByCourse(ctx context.Context, courseID int64) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE course_id = $1`, courseID).Scan(&count)
	return count, err
}

// CountAll 统计所有评论数量
func (r *Repository) CountAll(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM reviews`).Scan(&count)
	return count, err
}

// ListByCourse 获取课程评论列表
func (r *Repository) ListByCourse(ctx context.Context, courseID int64, limit, offset int) ([]Review, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.id, r.course_id, c.name, r.teacher_id, t.name, r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count, r.status, r.created_at
		FROM reviews r
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
		WHERE r.course_id = $1
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3
	`, courseID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReviews(rows)
}

// ListLatest 获取最新评论列表
func (r *Repository) ListLatest(ctx context.Context, limit, offset int) ([]Review, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.id, r.course_id, c.name, r.teacher_id, t.name, r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count, r.status, r.created_at
		FROM reviews r
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
		ORDER BY r.created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReviews(rows)
}

// CreateParams 创建评论参数
type CreateParams struct {
	ID        string
	CourseID  int64
	TeacherID *int64
	TermID    string
	Title     string
	Content   string
	Grade     string
	Ratings   []byte
	UserHash  string
}

// Create 创建评论（在事务中执行）
func (r *Repository) Create(ctx context.Context, tx pgx.Tx, p CreateParams) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO reviews (
			id, course_id, teacher_id, term_id, title, content, grade,
			ratings, user_hash, status, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())
	`, p.ID, p.CourseID, p.TeacherID, p.TermID, p.Title,
		p.Content, p.Grade, p.Ratings, p.UserHash, "published")
	return err
}

// IncrementCourseReviewCount 增加课程评论计数
func (r *Repository) IncrementCourseReviewCount(ctx context.Context, tx pgx.Tx, courseID int64) error {
	_, err := tx.Exec(ctx, `UPDATE courses SET review_count = review_count + 1 WHERE id = $1`, courseID)
	return err
}

// CreateVote 创建投票记录
func (r *Repository) CreateVote(ctx context.Context, tx pgx.Tx, reviewID, userHash, voteType string) (bool, error) {
	result, err := tx.Exec(ctx, `
		INSERT INTO review_votes (review_id, user_hash, vote_type, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT DO NOTHING
	`, reviewID, userHash, voteType)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

// IncrementLikeCount 增加点赞数
func (r *Repository) IncrementLikeCount(ctx context.Context, tx pgx.Tx, reviewID string) error {
	_, err := tx.Exec(ctx, `UPDATE reviews SET like_count = like_count + 1 WHERE id = $1`, reviewID)
	return err
}

// IncrementDislikeCount 增加踩数
func (r *Repository) IncrementDislikeCount(ctx context.Context, tx pgx.Tx, reviewID string) error {
	_, err := tx.Exec(ctx, `UPDATE reviews SET dislike_count = dislike_count + 1 WHERE id = $1`, reviewID)
	return err
}

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
