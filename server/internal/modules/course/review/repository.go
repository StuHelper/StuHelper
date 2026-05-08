package review

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
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

// CourseExistsTx 在事务内检查课程是否存在
func (r *Repository) CourseExistsTx(ctx context.Context, tx pgx.Tx, courseID int64) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM courses WHERE id = $1)`, courseID).Scan(&exists)
	return exists, err
}

// TeacherBelongsToCourseSchoolTx 在事务内检查教师是否与课程归属同一学校。
func (r *Repository) TeacherBelongsToCourseSchoolTx(ctx context.Context, tx pgx.Tx, teacherID int64, courseID int64) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM teachers t
			JOIN courses c ON c.school_id = t.school_id
			WHERE t.id = $1 AND c.id = $2
		)
	`, teacherID, courseID).Scan(&exists)
	return exists, err
}

// UserHasReviewedCourseTx 在事务内检查用户是否已对该课程发布评论
func (r *Repository) UserHasReviewedCourseTx(ctx context.Context, tx pgx.Tx, userHash string, courseID int64) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM reviews WHERE user_hash = $1 AND course_id = $2 AND status != 'deleted')
	`, userHash, courseID).Scan(&exists)
	return exists, err
}

// ReviewExistsTx 在事务内检查评论是否存在（已发布状态）
// 使用 FOR SHARE 防止并发事务在检查与写入之间删除评论（TOCTOU）
func (r *Repository) ReviewExistsTx(ctx context.Context, tx pgx.Tx, reviewID string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM reviews WHERE id = $1 AND status = 'published' FOR SHARE)`, reviewID).Scan(&exists)
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

// GetPortalStats 获取门户统计数据（课程数、评论数、院系数、评课用户数）
func (r *Repository) GetPortalStats(ctx context.Context) (courseCount, reviewCount, departmentCount, userCount int, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM courses),
			(SELECT COUNT(*) FROM reviews WHERE status = 'published'),
			(SELECT COUNT(*) FROM departments),
			(SELECT COUNT(DISTINCT user_hash) FROM reviews WHERE status = 'published')
	`).Scan(&courseCount, &reviewCount, &departmentCount, &userCount)
	return
}

// ListLatest 获取最新评论列表（含总数）
func (r *Repository) ListLatest(ctx context.Context, limit, offset int, sort string) ([]Review, int, error) {
	// SQL 注入安全保证：orderClause 的值 **仅** 来自 allowedSortOrders 硬编码 map，
	// 不包含任何用户输入。即使 sort 参数被篡改，最坏情况也只会命中默认排序。
	orderClause, ok := allowedSortOrders[sort]
	if !ok {
		orderClause = allowedSortOrders[SortTime]
	}

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM reviews WHERE status = 'published'
	`).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []Review{}, 0, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT r.id, r.course_id, c.name, r.teacher_id, t.name, r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count,
		       r.reply_count,
		       r.status, r.moderation_reason, r.created_at, r.updated_at
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
	list, err := scanReviews(rows)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// CreateParams 创建评论参数
type CreateParams struct {
	ID          string
	CourseID    int64
	TeacherID   *int64
	TermID      string
	Title       string
	Content     string
	Grade       string
	Ratings     []byte
	UserHash    string
	Status      string
	ContentFlag *string
}

// Create 创建评论（在事务中执行）
func (r *Repository) Create(ctx context.Context, tx pgx.Tx, p CreateParams) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO reviews (
			id, course_id, teacher_id, term_id, title, content, grade,
			ratings, avg_rating, user_hash, status, content_flag, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,
			COALESCE((SELECT AVG(value::numeric) FROM jsonb_each_text($8) WHERE value ~ '^\d+(\.\d+)?$'), 0),
			$9,$10,$11,NOW())
	`, p.ID, p.CourseID, p.TeacherID, p.TermID, p.Title,
		p.Content, p.Grade, p.Ratings, p.UserHash, p.Status, p.ContentFlag)
	return err
}

// CreateReturning 创建评论并通过 RETURNING 返回完整记录。
func (r *Repository) CreateReturning(ctx context.Context, tx pgx.Tx, p CreateParams) (*Review, error) {
	var review Review
	err := tx.QueryRow(ctx, `
		INSERT INTO reviews (
			id, course_id, teacher_id, term_id, title, content, grade,
			ratings, avg_rating, user_hash, status, content_flag, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,
			COALESCE((SELECT AVG(value::numeric) FROM jsonb_each_text($8) WHERE value ~ '^\d+(\.\d+)?$'), 0),
			$9,$10,$11,NOW())
		RETURNING id, course_id, teacher_id, term_id, title, content, grade,
			ratings, like_count, dislike_count, reply_count, status, content_flag, created_at, updated_at
	`, p.ID, p.CourseID, p.TeacherID, p.TermID, p.Title,
		p.Content, p.Grade, p.Ratings, p.UserHash, p.Status, p.ContentFlag,
	).Scan(
		&review.ID, &review.CourseID, &review.TeacherID, &review.TermID,
		&review.Title, &review.Content, &review.Grade, &review.Ratings,
		&review.LikeCount, &review.DislikeCount, &review.ReplyCount,
		&review.Status, &review.ContentFlag, &review.CreatedAt, &review.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateReturning: %w", err)
	}
	return &review, nil
}

// IncrementCourseReviewCount 增加课程评论计数
func (r *Repository) IncrementCourseReviewCount(ctx context.Context, tx pgx.Tx, courseID int64) error {
	_, err := tx.Exec(ctx, `UPDATE courses SET review_count = review_count + 1 WHERE id = $1`, courseID)
	return err
}

// CreateVote 创建投票记录
func (r *Repository) CreateVote(ctx context.Context, tx pgx.Tx, reviewID, userHash, voteType string) (bool, error) {
	newID, err := id.New()
	if err != nil {
		return false, err
	}
	result, err := tx.Exec(ctx, `
		INSERT INTO review_votes (id, review_id, user_hash, vote_type, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (review_id, user_hash) DO NOTHING
	`, newID, reviewID, userHash, voteType)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

// GetVoteType 获取用户对评论的投票类型，不存在返回空字符串
func (r *Repository) GetVoteType(ctx context.Context, tx pgx.Tx, reviewID, userHash string) (string, error) {
	var voteType string
	err := tx.QueryRow(ctx, `
		SELECT vote_type
		FROM review_votes
		WHERE review_id = $1 AND user_hash = $2
		FOR UPDATE
	`,
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

// scanReviews 扫描评论行
func scanReviews(rows interface {
	Next() bool
	Scan(...interface{}) error
	Err() error
}) ([]Review, error) {
	list := make([]Review, 0, 20)
	for rows.Next() {
		var item Review
		if err := rows.Scan(
			&item.ID, &item.CourseID, &item.CourseName, &item.TeacherID, &item.TeacherName,
			&item.TermID, &item.Title, &item.Content, &item.Grade, &item.Ratings,
			&item.LikeCount, &item.DislikeCount, &item.ReplyCount,
			&item.Status, &item.ModerationReason, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// scanReviewsWithTotal 扫描评论行（含 COUNT(*) OVER() 窗口函数总数）
func scanReviewsWithTotal(rows interface {
	Next() bool
	Scan(...interface{}) error
	Err() error
}) ([]Review, int, error) {
	list := make([]Review, 0, 20)
	var total int
	for rows.Next() {
		var item Review
		if err := rows.Scan(
			&item.ID, &item.CourseID, &item.CourseName, &item.TeacherID, &item.TeacherName,
			&item.TermID, &item.Title, &item.Content, &item.Grade, &item.Ratings,
			&item.LikeCount, &item.DislikeCount, &item.ReplyCount,
			&item.Status, &item.ModerationReason, &item.CreatedAt, &item.UpdatedAt,
			&total,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
