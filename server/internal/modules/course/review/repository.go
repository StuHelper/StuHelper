package review

import (
	"context"
	"errors"
	"strconv"

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

// ReviewExists 检查已发布的评论是否存在（用于用户侧操作：投票、举报、回复）
func (r *Repository) ReviewExists(ctx context.Context, reviewID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM reviews WHERE id = $1 AND status = 'published')`, reviewID).Scan(&exists)
	return exists, err
}

// ReviewExistsTx 在事务内检查评论是否存在（已发布状态）
func (r *Repository) ReviewExistsTx(ctx context.Context, tx pgx.Tx, reviewID string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM reviews WHERE id = $1 AND status = 'published')`, reviewID).Scan(&exists)
	return exists, err
}

// ReviewExistsAny 检查评论是否存在（不过滤状态，用于管理员操作）
func (r *Repository) ReviewExistsAny(ctx context.Context, reviewID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM reviews WHERE id = $1)`, reviewID).Scan(&exists)
	return exists, err
}

// UserHasReviewedCourse 检查用户是否已对该课程发布评论
func (r *Repository) UserHasReviewedCourse(ctx context.Context, userHash string, courseID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM reviews WHERE user_hash = $1 AND course_id = $2 AND status != 'deleted')
	`, userHash, courseID).Scan(&exists)
	return exists, err
}

// CountByCourse 统计课程评论数量
func (r *Repository) CountByCourse(ctx context.Context, courseID int64) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE course_id = $1 AND status = 'published'`, courseID).Scan(&count)
	return count, err
}

// CountAll 统计所有已发布评论数量
func (r *Repository) CountAll(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE status = 'published'`).Scan(&count)
	return count, err
}

// GetPortalStats 获取门户统计数据（课程数、评论数、院系数）
func (r *Repository) GetPortalStats(ctx context.Context) (courseCount, reviewCount, departmentCount int, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM courses),
			(SELECT COUNT(*) FROM reviews WHERE status = 'published'),
			(SELECT COUNT(*) FROM departments)
	`).Scan(&courseCount, &reviewCount, &departmentCount)
	return
}

// ListByCourse 获取课程评论列表
func (r *Repository) ListByCourse(ctx context.Context, courseID int64, limit, offset int) ([]Review, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.id, r.course_id, c.name, r.teacher_id, t.name, r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count,
		       (SELECT COUNT(*) FROM review_replies rr WHERE rr.review_id = r.id AND rr.status = 'published') AS reply_count,
		       r.status, r.created_at
		FROM reviews r
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
		WHERE r.course_id = $1 AND r.status = 'published'
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3
	`, courseID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReviews(rows)
}

// ListLatest 获取最新评论列表（含总数）
func (r *Repository) ListLatest(ctx context.Context, limit, offset int, sort string) ([]Review, int, error) {
	orderClause := "r.created_at DESC"
	switch sort {
	case "likes":
		orderClause = "r.like_count DESC, r.created_at DESC"
	case "rating":
		orderClause = "r.avg_rating DESC, r.created_at DESC"
	}

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
		WHERE r.status = 'published'
		ORDER BY `+orderClause+`
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanReviewsWithTotal(rows)
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
			ratings, avg_rating, user_hash, status, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,
			COALESCE((SELECT AVG(value::numeric) FROM jsonb_each_text($8) WHERE value ~ '^\d+(\.\d+)?$'), 0),
			$9,$10,NOW())
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
		ON CONFLICT (review_id, user_hash) DO NOTHING
	`, reviewID, userHash, voteType)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

// GetVoteType 获取用户对评论的投票类型，不存在返回空字符串
func (r *Repository) GetVoteType(ctx context.Context, tx pgx.Tx, reviewID, userHash string) (string, error) {
	var voteType string
	err := tx.QueryRow(ctx, `SELECT vote_type FROM review_votes WHERE review_id = $1 AND user_hash = $2`,
		reviewID, userHash).Scan(&voteType)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return voteType, err
}

// DeleteVote 删除投票记录
func (r *Repository) DeleteVote(ctx context.Context, tx pgx.Tx, reviewID, userHash string) error {
	_, err := tx.Exec(ctx, `DELETE FROM review_votes WHERE review_id = $1 AND user_hash = $2`, reviewID, userHash)
	return err
}

// UpdateVoteType 更新投票类型
func (r *Repository) UpdateVoteType(ctx context.Context, tx pgx.Tx, reviewID, userHash, voteType string) error {
	_, err := tx.Exec(ctx, `UPDATE review_votes SET vote_type = $3 WHERE review_id = $1 AND user_hash = $2`,
		reviewID, userHash, voteType)
	return err
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

// DecrementLikeCount 减少点赞数
func (r *Repository) DecrementLikeCount(ctx context.Context, tx pgx.Tx, reviewID string) error {
	_, err := tx.Exec(ctx, `UPDATE reviews SET like_count = GREATEST(like_count - 1, 0) WHERE id = $1`, reviewID)
	return err
}

// DecrementDislikeCount 减少踩数
func (r *Repository) DecrementDislikeCount(ctx context.Context, tx pgx.Tx, reviewID string) error {
	_, err := tx.Exec(ctx, `UPDATE reviews SET dislike_count = GREATEST(dislike_count - 1, 0) WHERE id = $1`, reviewID)
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

// GetReviewOwnerAndCourseID 获取评论所有者哈希和课程ID（单次查询）
func (r *Repository) GetReviewOwnerAndCourseID(ctx context.Context, reviewID string) (string, int64, error) {
	var userHash string
	var courseID int64
	err := r.db.QueryRow(ctx, `SELECT user_hash, course_id FROM reviews WHERE id = $1`, reviewID).Scan(&userHash, &courseID)
	return userHash, courseID, err
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
	err := tx.QueryRow(ctx, `SELECT status, course_id FROM reviews WHERE id = $1`, reviewID).Scan(&status, &courseID)
	return status, courseID, err
}

// CreateReportParams 创建举报参数
type CreateReportParams struct {
	ReviewID     string
	ReporterHash string
	Reason       string
	Description  string
}

// CreateReport 创建举报
func (r *Repository) CreateReport(ctx context.Context, p CreateReportParams) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO review_reports (review_id, reporter_hash, reason, description, status, created_at)
		VALUES ($1, $2, $3, $4, 'pending', NOW())
	`, p.ReviewID, p.ReporterHash, p.Reason, p.Description)
	return err
}

// ReportExists 检查是否已举报
func (r *Repository) ReportExists(ctx context.Context, reviewID, userHash string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM review_reports WHERE review_id = $1 AND reporter_hash = $2)
	`, reviewID, userHash).Scan(&exists)
	return exists, err
}

// ListReports 获取举报列表（包含评论信息，含总数）
func (r *Repository) ListReports(ctx context.Context, status string, limit, offset int) ([]ReviewReport, int, error) {
	baseQuery := `
		SELECT rr.id, rr.review_id, rr.reason, rr.description, rr.status,
		       rr.resolved_by, rr.resolved_at, rr.resolution_note, rr.created_at,
		       rv.id, rv.course_id, COALESCE(c.name, ''), rv.teacher_id, COALESCE(t.name, ''),
		       rv.term_id, rv.title, rv.content, rv.grade, rv.ratings,
		       rv.like_count, rv.dislike_count,
		       (SELECT COUNT(*) FROM review_replies rpl WHERE rpl.review_id = rv.id AND rpl.status = 'published'),
		       rv.status, rv.created_at,
		       COUNT(*) OVER() AS total
		FROM review_reports rr
		LEFT JOIN reviews rv ON rv.id = rr.review_id
		LEFT JOIN courses c ON c.id = rv.course_id
		LEFT JOIN teachers t ON t.id = rv.teacher_id
	`
	var args []interface{}

	if status != "" && status != "all" {
		baseQuery += ` WHERE rr.status = $1 ORDER BY rr.created_at DESC LIMIT $2 OFFSET $3`
		args = []interface{}{status, limit, offset}
	} else {
		baseQuery += ` ORDER BY rr.created_at DESC LIMIT $1 OFFSET $2`
		args = []interface{}{limit, offset}
	}

	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reports []ReviewReport
	var total int
	for rows.Next() {
		var rp ReviewReport
		var review Review
		var reviewID *string
		if err := rows.Scan(
			&rp.ID, &rp.ReviewID, &rp.Reason, &rp.Description, &rp.Status,
			&rp.ResolvedBy, &rp.ResolvedAt, &rp.ResolutionNote, &rp.CreatedAt,
			&reviewID, &review.CourseID, &review.CourseName, &review.TeacherID, &review.TeacherName,
			&review.TermID, &review.Title, &review.Content, &review.Grade, &review.Ratings,
			&review.LikeCount, &review.DislikeCount, &review.ReplyCount, &review.Status, &review.CreatedAt,
			&total,
		); err != nil {
			return nil, 0, err
		}
		if reviewID != nil {
			review.ID = *reviewID
			rp.Review = &review
		}
		reports = append(reports, rp)
	}
	return reports, total, rows.Err()
}

// GetReportByID 根据ID获取举报
func (r *Repository) GetReportByID(ctx context.Context, reportID int64) (*ReviewReport, error) {
	var rp ReviewReport
	err := r.db.QueryRow(ctx, `
		SELECT id, review_id, reason, description, status,
		       resolved_by, resolved_at, resolution_note, created_at
		FROM review_reports WHERE id = $1
	`, reportID).Scan(
		&rp.ID, &rp.ReviewID, &rp.Reason, &rp.Description, &rp.Status,
		&rp.ResolvedBy, &rp.ResolvedAt, &rp.ResolutionNote, &rp.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rp, nil
}

// GetReportByIDForUpdate 在事务内获取举报并加行锁，仅返回 pending 状态的举报
func (r *Repository) GetReportByIDForUpdate(ctx context.Context, tx pgx.Tx, reportID int64) (*ReviewReport, error) {
	var rp ReviewReport
	err := tx.QueryRow(ctx, `
		SELECT id, review_id, reason, description, status,
		       resolved_by, resolved_at, resolution_note, created_at
		FROM review_reports WHERE id = $1 AND status = 'pending'
		FOR UPDATE
	`, reportID).Scan(
		&rp.ID, &rp.ReviewID, &rp.Reason, &rp.Description, &rp.Status,
		&rp.ResolvedBy, &rp.ResolvedAt, &rp.ResolutionNote, &rp.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rp, nil
}

// UpdateReportParams 更新举报参数
type UpdateReportParams struct {
	ID         int64
	Status     string
	ResolvedBy string
	Note       string
}

// UpdateReport 更新举报状态
func (r *Repository) UpdateReport(ctx context.Context, tx pgx.Tx, p UpdateReportParams) error {
	_, err := tx.Exec(ctx, `
		UPDATE review_reports
		SET status = $2, resolved_by = $3, resolution_note = $4, resolved_at = NOW()
		WHERE id = $1
	`, p.ID, p.Status, p.ResolvedBy, p.Note)
	return err
}

// ListAllReviews 获取所有评论（管理员，含总数）
func (r *Repository) ListAllReviews(ctx context.Context, status string, limit, offset int) ([]Review, int, error) {
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
	`
	var args []interface{}

	if status != "" && status != "all" {
		query += ` WHERE r.status = $1 ORDER BY r.created_at DESC LIMIT $2 OFFSET $3`
		args = []interface{}{status, limit, offset}
	} else {
		query += ` ORDER BY r.created_at DESC LIMIT $1 OFFSET $2`
		args = []interface{}{limit, offset}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanReviewsWithTotal(rows)
}

// GetAdminStats 获取管理统计（单次查询，减少数据库往返）
func (r *Repository) GetAdminStats(ctx context.Context) (*AdminStats, error) {
	var stats AdminStats
	err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM reviews) AS total_reviews,
			(SELECT COUNT(*) FROM reviews WHERE status = 'published') AS published_reviews,
			(SELECT COUNT(*) FROM reviews WHERE status = 'hidden') AS hidden_reviews,
			(SELECT COUNT(*) FROM reviews WHERE status = 'deleted') AS deleted_reviews,
			(SELECT COUNT(*) FROM review_reports WHERE status = 'pending') AS pending_reports,
			(SELECT COUNT(*) FROM review_reports) AS total_reports,
			(SELECT COUNT(*) FROM reviews WHERE created_at >= CURRENT_DATE) AS today_reviews,
			(SELECT COUNT(*) FROM reviews WHERE created_at >= date_trunc('week', CURRENT_DATE)) AS week_reviews
	`).Scan(
		&stats.TotalReviews,
		&stats.PublishedReviews,
		&stats.HiddenReviews,
		&stats.DeletedReviews,
		&stats.PendingReports,
		&stats.TotalReports,
		&stats.TodayReviews,
		&stats.WeekReviews,
	)
	if err != nil {
		return nil, err
	}
	return &stats, nil
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

// CreateFavorite 创建收藏
func (r *Repository) CreateFavorite(ctx context.Context, userHash string, courseID int64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO course_favorites (user_hash, course_id, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_hash, course_id) DO NOTHING
	`, userHash, courseID)
	return err
}

// DeleteFavorite 删除收藏
func (r *Repository) DeleteFavorite(ctx context.Context, userHash string, courseID int64) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM course_favorites WHERE user_hash = $1 AND course_id = $2
	`, userHash, courseID)
	return err
}

// FavoriteExists 检查是否已收藏
func (r *Repository) FavoriteExists(ctx context.Context, userHash string, courseID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM course_favorites WHERE user_hash = $1 AND course_id = $2)
	`, userHash, courseID).Scan(&exists)
	return exists, err
}

// ListFavorites 获取用户收藏列表（含总数）
func (r *Repository) ListFavorites(ctx context.Context, userHash string, limit, offset int) ([]FavoriteCourse, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.name, c.code, c.credits, c.department_id,
		       d.name, c.review_count, cf.created_at,
		       COUNT(*) OVER() AS total
		FROM course_favorites cf
		JOIN courses c ON c.id = cf.course_id
		LEFT JOIN departments d ON d.id = c.department_id
		WHERE cf.user_hash = $1
		ORDER BY cf.created_at DESC
		LIMIT $2 OFFSET $3
	`, userHash, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []FavoriteCourse
	var total int
	for rows.Next() {
		var f FavoriteCourse
		if err := rows.Scan(&f.ID, &f.Name, &f.Code, &f.Credits, &f.DepartmentID, &f.DepartmentName, &f.ReviewCount, &f.FavoritedAt, &total); err != nil {
			return nil, 0, err
		}
		list = append(list, f)
	}
	return list, total, rows.Err()
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

// UpsertDraftParams 保存草稿参数
type UpsertDraftParams struct {
	UserHash  string
	CourseID  int64
	TeacherID *int64
	TermID    string
	Title     string
	Content   string
	Grade     string
	Ratings   []byte
}

// UpsertDraft 保存或更新草稿
func (r *Repository) UpsertDraft(ctx context.Context, p UpsertDraftParams) (*ReviewDraft, error) {
	var d ReviewDraft
	err := r.db.QueryRow(ctx, `
		INSERT INTO review_drafts (user_hash, course_id, teacher_id, term_id, title, content, grade, ratings, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (user_hash, course_id) DO UPDATE SET
			teacher_id = EXCLUDED.teacher_id,
			term_id = EXCLUDED.term_id,
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			grade = EXCLUDED.grade,
			ratings = EXCLUDED.ratings,
			updated_at = NOW()
		RETURNING id, course_id, teacher_id, term_id, title, content, grade, ratings, updated_at
	`, p.UserHash, p.CourseID, p.TeacherID, p.TermID, p.Title, p.Content, p.Grade, p.Ratings).Scan(
		&d.ID, &d.CourseID, &d.TeacherID, &d.TermID, &d.Title, &d.Content, &d.Grade, &d.Ratings, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// GetDraft 获取草稿
func (r *Repository) GetDraft(ctx context.Context, userHash string, courseID int64) (*ReviewDraft, error) {
	var d ReviewDraft
	err := r.db.QueryRow(ctx, `
		SELECT id, course_id, teacher_id, term_id, title, content, grade, ratings, updated_at
		FROM review_drafts
		WHERE user_hash = $1 AND course_id = $2
	`, userHash, courseID).Scan(
		&d.ID, &d.CourseID, &d.TeacherID, &d.TermID, &d.Title, &d.Content, &d.Grade, &d.Ratings, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// DeleteDraft 删除草稿
func (r *Repository) DeleteDraft(ctx context.Context, userHash string, courseID int64) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM review_drafts WHERE user_hash = $1 AND course_id = $2
	`, userHash, courseID)
	return err
}

// CreateReplyParams 创建回复参数
type CreateReplyParams struct {
	ReviewID string
	ParentID *int64
	UserHash string
	Content  string
}

// CreateReply 创建回复
func (r *Repository) CreateReply(ctx context.Context, tx pgx.Tx, p CreateReplyParams) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO review_replies (review_id, parent_id, user_hash, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, p.ReviewID, p.ParentID, p.UserHash, p.Content).Scan(&id)
	return id, err
}

// ListReplies 获取回复列表（含总数）
func (r *Repository) ListReplies(ctx context.Context, reviewID string, limit, offset int) ([]Reply, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, review_id, parent_id, user_hash, content, like_count, status, created_at, updated_at,
		       COUNT(*) OVER() AS total
		FROM review_replies
		WHERE review_id = $1 AND status = 'published'
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`, reviewID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []Reply
	var total int
	for rows.Next() {
		var reply Reply
		if err := rows.Scan(
			&reply.ID, &reply.ReviewID, &reply.ParentID, &reply.UserHash,
			&reply.Content, &reply.LikeCount, &reply.Status, &reply.CreatedAt, &reply.UpdatedAt,
			&total,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, reply)
	}
	return list, total, rows.Err()
}

// GetReplyOwner 获取回复所有者
func (r *Repository) GetReplyOwner(ctx context.Context, replyID int64) (string, error) {
	var userHash string
	err := r.db.QueryRow(ctx, `
		SELECT user_hash FROM review_replies WHERE id = $1
	`, replyID).Scan(&userHash)
	return userHash, err
}

// GetReplyReviewID 获取回复所属的评论ID
func (r *Repository) GetReplyReviewID(ctx context.Context, replyID int64) (string, error) {
	var reviewID string
	err := r.db.QueryRow(ctx, `
		SELECT review_id FROM review_replies WHERE id = $1
	`, replyID).Scan(&reviewID)
	return reviewID, err
}

// GetReplyOwnerAndReviewID 获取回复所有者和所属评论ID（单次查询）
func (r *Repository) GetReplyOwnerAndReviewID(ctx context.Context, replyID int64) (string, string, error) {
	var userHash, reviewID string
	err := r.db.QueryRow(ctx, `
		SELECT user_hash, review_id FROM review_replies WHERE id = $1
	`, replyID).Scan(&userHash, &reviewID)
	return userHash, reviewID, err
}

// SoftDeleteReply 软删除回复
func (r *Repository) SoftDeleteReply(ctx context.Context, tx pgx.Tx, replyID int64) error {
	_, err := tx.Exec(ctx, `
		UPDATE review_replies SET status = 'deleted', updated_at = NOW() WHERE id = $1
	`, replyID)
	return err
}

// IncrementReplyCount 增加评论回复计数
func (r *Repository) IncrementReplyCount(ctx context.Context, tx pgx.Tx, reviewID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE reviews SET reply_count = reply_count + 1 WHERE id = $1
	`, reviewID)
	return err
}

// DecrementReplyCount 减少评论回复计数
func (r *Repository) DecrementReplyCount(ctx context.Context, tx pgx.Tx, reviewID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE reviews SET reply_count = GREATEST(reply_count - 1, 0) WHERE id = $1
	`, reviewID)
	return err
}

// CreateNotificationParams 创建通知参数
type CreateNotificationParams struct {
	UserHash    string
	Type        string
	Title       string
	Content     string
	RelatedType string
	RelatedID   string
}

// CreateNotification 创建通知
func (r *Repository) CreateNotification(ctx context.Context, p CreateNotificationParams) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO notifications (user_hash, type, title, content, related_type, related_id)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, p.UserHash, p.Type, p.Title, p.Content, p.RelatedType, p.RelatedID)
	return err
}

// ListNotifications 获取通知列表（含总数）
func (r *Repository) ListNotifications(ctx context.Context, userHash string, limit, offset int) ([]Notification, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, type, title, content, related_type, related_id, is_read, created_at,
		       COUNT(*) OVER() AS total
		FROM notifications
		WHERE user_hash = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userHash, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []Notification
	var total int
	for rows.Next() {
		var n Notification
		if err := rows.Scan(
			&n.ID, &n.Type, &n.Title, &n.Content,
			&n.RelatedType, &n.RelatedID, &n.IsRead, &n.CreatedAt,
			&total,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, n)
	}
	return list, total, rows.Err()
}

// CountUnreadNotifications 统计未读通知数量
func (r *Repository) CountUnreadNotifications(ctx context.Context, userHash string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications WHERE user_hash = $1 AND is_read = false
	`, userHash).Scan(&count)
	return count, err
}

// MarkNotificationRead 标记通知已读
func (r *Repository) MarkNotificationRead(ctx context.Context, id int64, userHash string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = true WHERE id = $1 AND user_hash = $2
	`, id, userHash)
	return err
}

// MarkAllNotificationsRead 标记所有通知已读
func (r *Repository) MarkAllNotificationsRead(ctx context.Context, userHash string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = true WHERE user_hash = $1 AND is_read = false
	`, userHash)
	return err
}

// RatingTrendItem 评分趋势项
type RatingTrendItem struct {
	TermID   string  `json:"termID"`
	TermName string  `json:"termName"`
	AvgRating float64 `json:"avgRating"`
	Count    int     `json:"count"`
}

// GetRatingTrend 获取课程评分趋势
func (r *Repository) GetRatingTrend(ctx context.Context, courseID int64) ([]RatingTrendItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.term_id, COALESCE(t.name, r.term_id) as term_name,
			AVG(r.avg_rating) as avg_rating,
			COUNT(*) as count
		FROM reviews r
		LEFT JOIN terms t ON r.term_id = t.id
		WHERE r.course_id = $1 AND r.status = 'published' AND r.term_id IS NOT NULL
		GROUP BY r.term_id, t.name
		ORDER BY r.term_id ASC
	`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []RatingTrendItem
	for rows.Next() {
		var item RatingTrendItem
		if err := rows.Scan(&item.TermID, &item.TermName, &item.AvgRating, &item.Count); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// HotCourse 热门课程
type HotCourse struct {
	CourseID    int64   `json:"courseID"`
	CourseName  string  `json:"courseName"`
	ReviewCount int     `json:"reviewCount"`
	AvgRating   float64 `json:"avgRating"`
}

// ListHotCourses 获取热门课程排行
func (r *Repository) ListHotCourses(ctx context.Context, period string, limit int) ([]HotCourse, error) {
	var timeFilter string
	switch period {
	case "week":
		timeFilter = "AND r.created_at >= NOW() - INTERVAL '7 days'"
	case "month":
		timeFilter = "AND r.created_at >= NOW() - INTERVAL '30 days'"
	default:
		timeFilter = ""
	}

	query := `
		SELECT c.id, c.name, COUNT(r.id) as review_count,
			COALESCE(AVG(r.avg_rating), 0) as avg_rating
		FROM courses c
		LEFT JOIN reviews r ON c.id = r.course_id AND r.status = 'published' ` + timeFilter + `
		GROUP BY c.id, c.name
		HAVING COUNT(r.id) > 0
		ORDER BY review_count DESC, avg_rating DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []HotCourse
	for rows.Next() {
		var item HotCourse
		if err := rows.Scan(&item.CourseID, &item.CourseName, &item.ReviewCount, &item.AvgRating); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// GetTeacherName 获取教师名称
func (r *Repository) GetTeacherName(ctx context.Context, teacherID int64) (string, error) {
	var name string
	err := r.db.QueryRow(ctx, `
		SELECT name FROM teachers WHERE id = $1
	`, teacherID).Scan(&name)
	return name, err
}

// GetTeacherInfo 获取教师基本信息（名称 + 院系名称）
func (r *Repository) GetTeacherInfo(ctx context.Context, teacherID int64) (name, departmentName string, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT t.name, COALESCE(d.name, '')
		FROM teachers t
		LEFT JOIN departments d ON d.id = t.department_id
		WHERE t.id = $1
	`, teacherID).Scan(&name, &departmentName)
	return
}

// ListTeacherCourses 获取教师授课课程列表（含评分和评论数）
func (r *Repository) ListTeacherCourses(ctx context.Context, teacherID int64) ([]TeacherCourse, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.name,
			AVG(CASE WHEN r.avg_rating > 0 THEN r.avg_rating END) AS avg_rating,
			COUNT(r.id) AS review_count
		FROM reviews r
		JOIN courses c ON c.id = r.course_id
		WHERE r.teacher_id = $1 AND r.status = 'published'
		GROUP BY c.id, c.name
		ORDER BY review_count DESC
	`, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []TeacherCourse
	for rows.Next() {
		var tc TeacherCourse
		if err := rows.Scan(&tc.ID, &tc.Name, &tc.AvgRating, &tc.ReviewCount); err != nil {
			return nil, err
		}
		list = append(list, tc)
	}
	return list, rows.Err()
}

// GetTeacherReviewCount 获取教师的评论总数
func (r *Repository) GetTeacherReviewCount(ctx context.Context, teacherID int64) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM reviews
		WHERE teacher_id = $1 AND status = 'published'
	`, teacherID).Scan(&count)
	return count, err
}

// GetTeacherRatingStats 获取教师评分统计
func (r *Repository) GetTeacherRatingStats(ctx context.Context, teacherID int64) ([]TeacherRatingStats, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, teacher_id, term_id, dimension_key, avg_rating, rating_count, rating_dist, updated_at
		FROM teacher_rating_stats
		WHERE teacher_id = $1
		ORDER BY term_id NULLS FIRST, dimension_key
	`, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []TeacherRatingStats
	for rows.Next() {
		var s TeacherRatingStats
		if err := rows.Scan(
			&s.ID, &s.TeacherID, &s.TermID, &s.DimensionKey,
			&s.AvgRating, &s.RatingCount, &s.RatingDist, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// RefreshTeacherRatingStats 刷新教师评分统计（从 reviews 表聚合）
func (r *Repository) RefreshTeacherRatingStats(ctx context.Context, teacherID int64) error {
	_, err := r.db.Exec(ctx, `
		WITH base AS (
			SELECT r.term_id, d.key AS dimension_key,
				(r.ratings->>d.key)::numeric AS rating_num,
				(r.ratings->>d.key)::text AS rating_val
			FROM reviews r
			CROSS JOIN rating_dimensions d
			WHERE r.teacher_id = $1 AND r.status = 'published' AND d.is_active = true
				AND r.ratings ? d.key
		),
		stats AS (
			SELECT term_id, dimension_key,
				AVG(rating_num) AS avg_rating,
				COUNT(*) AS rating_count
			FROM base
			GROUP BY term_id, dimension_key
		),
		dist AS (
			SELECT term_id, dimension_key,
				jsonb_object_agg(rating_val, cnt) AS rating_dist
			FROM (
				SELECT term_id, dimension_key, rating_val, COUNT(*) AS cnt
				FROM base
				GROUP BY term_id, dimension_key, rating_val
			) sub
			GROUP BY term_id, dimension_key
		)
		INSERT INTO teacher_rating_stats (teacher_id, term_id, dimension_key, avg_rating, rating_count, rating_dist, updated_at)
		SELECT $1, s.term_id, s.dimension_key, s.avg_rating, s.rating_count,
			COALESCE(d.rating_dist, '{}'::jsonb), NOW()
		FROM stats s
		LEFT JOIN dist d ON s.term_id IS NOT DISTINCT FROM d.term_id AND s.dimension_key = d.dimension_key
		ON CONFLICT (teacher_id, term_id, dimension_key)
		DO UPDATE SET
			avg_rating = EXCLUDED.avg_rating,
			rating_count = EXCLUDED.rating_count,
			rating_dist = EXCLUDED.rating_dist,
			updated_at = NOW()
	`, teacherID)
	return err
}

// ListActiveSensitiveWords 获取所有启用的敏感词
func (r *Repository) ListActiveSensitiveWords(ctx context.Context) ([]SensitiveWord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, word, category, level, is_active, created_at
		FROM sensitive_words
		WHERE is_active = true
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []SensitiveWord
	for rows.Next() {
		var w SensitiveWord
		if err := rows.Scan(&w.ID, &w.Word, &w.Category, &w.Level, &w.IsActive, &w.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

// BatchUpdateReviewStatusTx 批量更新评论状态（事务内执行）
func (r *Repository) BatchUpdateReviewStatusTx(ctx context.Context, tx pgx.Tx, ids []string, status string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	result, err := tx.Exec(ctx, `
		UPDATE reviews SET status = $1, updated_at = NOW()
		WHERE id = ANY($2)
	`, status, ids)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// AdjustCourseCountsForBatchDelete 批量删除时减少相关课程的评论计数
// 仅对当前状态非 deleted 的评论所属课程进行计数调整
func (r *Repository) AdjustCourseCountsForBatchDelete(ctx context.Context, tx pgx.Tx, ids []string) error {
	_, err := tx.Exec(ctx, `
		UPDATE courses SET review_count = GREATEST(review_count - sub.cnt, 0)
		FROM (
			SELECT course_id, COUNT(*) AS cnt
			FROM reviews
			WHERE id = ANY($1) AND status != 'deleted'
			GROUP BY course_id
		) sub
		WHERE courses.id = sub.course_id
	`, ids)
	return err
}

// AdjustCourseCountsForBatchRestore 批量恢复时增加相关课程的评论计数
// 仅对当前状态为 deleted 的评论所属课程进行计数调整
func (r *Repository) AdjustCourseCountsForBatchRestore(ctx context.Context, tx pgx.Tx, ids []string) error {
	_, err := tx.Exec(ctx, `
		UPDATE courses SET review_count = review_count + sub.cnt
		FROM (
			SELECT course_id, COUNT(*) AS cnt
			FROM reviews
			WHERE id = ANY($1) AND status = 'deleted'
			GROUP BY course_id
		) sub
		WHERE courses.id = sub.course_id
	`, ids)
	return err
}

// CreateOperationLogParams 创建操作日志参数
type CreateOperationLogParams struct {
	AdminUserID   string
	AdminUsername string
	Action        string
	ResourceType  string
	ResourceID    string
	OldValue      []byte
	NewValue      []byte
	IPAddress     string
	UserAgent     string
}

// CreateOperationLog 创建操作日志
func (r *Repository) CreateOperationLog(ctx context.Context, p CreateOperationLogParams) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO admin_operation_logs
		(admin_user_id, admin_username, action, resource_type, resource_id, old_value, new_value, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, p.AdminUserID, p.AdminUsername, p.Action, p.ResourceType, p.ResourceID, p.OldValue, p.NewValue, p.IPAddress, p.UserAgent)
	return err
}

// ListOperationLogs 获取操作日志列表（含总数）
func (r *Repository) ListOperationLogs(ctx context.Context, limit, offset int) ([]AdminOperationLog, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, admin_user_id, admin_username, action, resource_type, resource_id,
			old_value, new_value, ip_address, user_agent, created_at,
			COUNT(*) OVER() AS total
		FROM admin_operation_logs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []AdminOperationLog
	var total int
	for rows.Next() {
		var log AdminOperationLog
		if err := rows.Scan(
			&log.ID, &log.AdminUserID, &log.AdminUsername, &log.Action, &log.ResourceType, &log.ResourceID,
			&log.OldValue, &log.NewValue, &log.IPAddress, &log.UserAgent, &log.CreatedAt,
			&total,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, log)
	}
	return list, total, rows.Err()
}

// ListAllReviewsForExport 获取所有评论用于导出
func (r *Repository) ListAllReviewsForExport(ctx context.Context, status string) ([]Review, error) {
	baseQuery := `
		SELECT r.id, r.course_id, COALESCE(c.name, '') as course_name,
			r.teacher_id, COALESCE(t.name, '') as teacher_name,
			r.term_id, r.title, r.content, r.grade, r.ratings,
			r.like_count, r.dislike_count,
			(SELECT COUNT(*) FROM review_replies rr WHERE rr.review_id = r.id AND rr.status = 'published') AS reply_count,
			r.status, r.created_at
		FROM reviews r
		LEFT JOIN courses c ON r.course_id = c.id
		LEFT JOIN teachers t ON r.teacher_id = t.id`

	var rows *db.RowsWithCancel
	var err error
	if status != "" && status != "all" {
		rows, err = r.db.Query(ctx, baseQuery+`
			WHERE r.status = $1
			ORDER BY r.created_at DESC
			LIMIT 10000`, status)
	} else {
		rows, err = r.db.Query(ctx, baseQuery+`
			ORDER BY r.created_at DESC
			LIMIT 10000`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Review
	for rows.Next() {
		var review Review
		if err := rows.Scan(
			&review.ID, &review.CourseID, &review.CourseName,
			&review.TeacherID, &review.TeacherName,
			&review.TermID, &review.Title, &review.Content, &review.Grade, &review.Ratings,
			&review.LikeCount, &review.DislikeCount, &review.ReplyCount, &review.Status, &review.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, review)
	}
	return list, rows.Err()
}

// ForEachReviewForExport 流式遍历导出评论，逐行回调避免全量加载到内存
func (r *Repository) ForEachReviewForExport(ctx context.Context, status string, fn func(Review) error) error {
	baseQuery := `
		SELECT r.id, r.course_id, COALESCE(c.name, '') as course_name,
			r.teacher_id, COALESCE(t.name, '') as teacher_name,
			r.term_id, r.title, r.content, r.grade, r.ratings,
			r.like_count, r.dislike_count,
			(SELECT COUNT(*) FROM review_replies rr WHERE rr.review_id = r.id AND rr.status = 'published') AS reply_count,
			r.status, r.created_at
		FROM reviews r
		LEFT JOIN courses c ON r.course_id = c.id
		LEFT JOIN teachers t ON r.teacher_id = t.id`

	var rows *db.RowsWithCancel
	var err error
	if status != "" && status != "all" {
		rows, err = r.db.Query(ctx, baseQuery+`
			WHERE r.status = $1
			ORDER BY r.created_at DESC
			LIMIT 10000`, status)
	} else {
		rows, err = r.db.Query(ctx, baseQuery+`
			ORDER BY r.created_at DESC
			LIMIT 10000`)
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var review Review
		if err := rows.Scan(
			&review.ID, &review.CourseID, &review.CourseName,
			&review.TeacherID, &review.TeacherName,
			&review.TermID, &review.Title, &review.Content,
			&review.Grade, &review.Ratings,
			&review.LikeCount, &review.DislikeCount,
			&review.ReplyCount, &review.Status, &review.CreatedAt,
		); err != nil {
			return err
		}
		if err := fn(review); err != nil {
			return err
		}
	}
	return rows.Err()
}

