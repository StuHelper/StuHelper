package review

import (
	"context"

	"github.com/jackc/pgx/v5"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

// CreateReportParams 创建举报参数
type CreateReportParams struct {
	ReviewID     string
	ReporterHash string
	Reason       string
	Description  string
}

// CreateReport 创建举报
func (r *Repository) CreateReport(ctx context.Context, p CreateReportParams) error {
	newID, err := id.New()
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO review_reports (id, review_id, reporter_hash, reason, description, status, created_at)
		VALUES ($1, $2, $3, $4, $5, 'pending', NOW())
	`, newID, p.ReviewID, p.ReporterHash, p.Reason, p.Description)
	return err
}

// CreateReportTx 创建举报记录（事务内）
func (r *Repository) CreateReportTx(ctx context.Context, tx pgx.Tx, p CreateReportParams) error {
	newID, err := id.New()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO review_reports (id, review_id, reporter_hash, reason, description, status, created_at)
		VALUES ($1, $2, $3, $4, $5, 'pending', NOW())
	`, newID, p.ReviewID, p.ReporterHash, p.Reason, p.Description)
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

// ReportExistsTx 检查是否已举报（事务内）
func (r *Repository) ReportExistsTx(ctx context.Context, tx pgx.Tx, reviewID, userHash string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
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

// CreateFavorite 创建收藏
func (r *Repository) CreateFavorite(ctx context.Context, userHash string, courseID int64) error {
	newID, err := id.New()
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO course_favorites (id, user_hash, course_id, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_hash, course_id) DO NOTHING
	`, newID, userHash, courseID)
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

// CreateReplyParams 创建回复参数
type CreateReplyParams struct {
	ReviewID string
	ParentID *string
	UserHash string
	Content  string
}

// CreateReply 创建回复
func (r *Repository) CreateReply(ctx context.Context, tx pgx.Tx, p CreateReplyParams) (string, error) {
	replyID, err := id.New()
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO review_replies (id, review_id, parent_id, user_hash, content)
		VALUES ($1, $2, $3, $4, $5)
	`, replyID, p.ReviewID, p.ParentID, p.UserHash, p.Content)
	return replyID, err
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
func (r *Repository) GetReplyOwner(ctx context.Context, replyID string) (string, error) {
	var userHash string
	err := r.db.QueryRow(ctx, `
		SELECT user_hash FROM review_replies WHERE id = $1
	`, replyID).Scan(&userHash)
	return userHash, err
}

// GetReplyReviewID 获取回复所属的评论ID
func (r *Repository) GetReplyReviewID(ctx context.Context, replyID string) (string, error) {
	var reviewID string
	err := r.db.QueryRow(ctx, `
		SELECT review_id FROM review_replies WHERE id = $1
	`, replyID).Scan(&reviewID)
	return reviewID, err
}

// GetReplyOwnerAndReviewID 获取回复所有者和所属评论ID（单次查询）
func (r *Repository) GetReplyOwnerAndReviewID(ctx context.Context, replyID string) (string, string, error) {
	var userHash, reviewID string
	err := r.db.QueryRow(ctx, `
		SELECT user_hash, review_id FROM review_replies WHERE id = $1
	`, replyID).Scan(&userHash, &reviewID)
	return userHash, reviewID, err
}

// GetReplyOwnerAndReviewIDTx 在事务内获取回复所有者和评论ID
func (r *Repository) GetReplyOwnerAndReviewIDTx(ctx context.Context, tx pgx.Tx, replyID string) (string, string, string, error) {
	var userHash, reviewID, status string
	err := tx.QueryRow(ctx, `
		SELECT user_hash, review_id, status FROM review_replies WHERE id = $1
	`, replyID).Scan(&userHash, &reviewID, &status)
	return userHash, reviewID, status, err
}

// SoftDeleteReply 软删除回复
func (r *Repository) SoftDeleteReply(ctx context.Context, tx pgx.Tx, replyID string) error {
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
