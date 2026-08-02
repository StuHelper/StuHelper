package review

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/id"
)

// CreateReportParams 创建举报参数
type CreateReportParams struct {
	ReviewID     string
	SchoolID     int64
	ReporterHash string
	Reason       string
	Description  string
}

// CreateReport 创建举报
func (r *Repository) CreateReport(ctx context.Context, p CreateReportParams) error {
	ctx = withDBTable(ctx, "review_reports")
	newID, err := id.New()
	if err != nil {
		return fmt.Errorf("CreateReport generate id: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO review_reports (id, review_id, school_id, reporter_hash, reason, description, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending', NOW())
	`, newID, p.ReviewID, p.SchoolID, p.ReporterHash, p.Reason, p.Description)
	if err != nil {
		return fmt.Errorf("CreateReport: %w", err)
	}
	return nil
}

// CreateReportTx 创建举报记录（事务内），返回生成的举报 ID
func (r *Repository) CreateReportTx(ctx context.Context, tx pgx.Tx, p CreateReportParams) (string, error) {
	newID, err := id.New()
	if err != nil {
		return "", fmt.Errorf("CreateReportTx generate id: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO review_reports (id, review_id, school_id, reporter_hash, reason, description, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending', NOW())
	`, newID, p.ReviewID, p.SchoolID, p.ReporterHash, p.Reason, p.Description)
	if err != nil {
		return "", fmt.Errorf("CreateReportTx: %w", err)
	}
	return newID, nil
}

// ReportExists 检查是否已举报
func (r *Repository) ReportExists(ctx context.Context, reviewID, userHash string) (bool, error) {
	ctx = withDBTable(ctx, "review_reports")
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM review_reports WHERE review_id = $1 AND reporter_hash = $2)
	`, reviewID, userHash).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("ReportExists: %w", err)
	}
	return exists, nil
}

// ReportExistsTx 检查是否已举报（事务内）
func (r *Repository) ReportExistsTx(ctx context.Context, tx pgx.Tx, reviewID, userHash string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM review_reports WHERE review_id = $1 AND reporter_hash = $2)
	`, reviewID, userHash).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("ReportExistsTx: %w", err)
	}
	return exists, nil
}

// ListReports 获取举报列表（包含评论信息，含总数）
// 当 reviewID 为 nil（评论已被物理删除）时，跳过 Review 对象赋值，标记为已删除
func (r *Repository) ListReports(ctx context.Context, status string, limit, offset int, schoolIDs []int64) ([]ReviewReport, int, error) {
	ctx = withDBTable(ctx, "review_reports")
	if schoolIDs != nil && len(schoolIDs) == 0 {
		return []ReviewReport{}, 0, nil
	}
	var qb strings.Builder
	qb.WriteString(`
		SELECT rr.id, rr.review_id, rr.reason, rr.description, rr.status,
		       rr.resolved_by, rr.resolved_at, rr.resolution_note, rr.created_at,
		       rv.id, rv.course_id, COALESCE(c.name, ''), rv.teacher_id, COALESCE(t.name, ''),
		       rv.term_id, rv.title, rv.content, rv.grade, rv.ratings,
		       rv.like_count, rv.dislike_count,
		       rv.reply_count,
		       rv.status, rv.created_at, rv.updated_at,
		       COUNT(*) OVER() AS total
		FROM review_reports rr
		LEFT JOIN reviews rv ON rv.id = rr.review_id
		LEFT JOIN courses c ON c.id = rv.course_id
		LEFT JOIN teachers t ON t.id = rv.teacher_id
	`)
	var args []interface{}

	hasWhere := false
	if len(schoolIDs) > 0 {
		qb.WriteString(` WHERE rr.school_id = ANY($1)`)
		args = append(args, schoolIDs)
		hasWhere = true
	}
	if status != "" && status != "all" {
		if hasWhere {
			qb.WriteString(` AND`)
		} else {
			qb.WriteString(` WHERE`)
		}
		qb.WriteString(` rr.status = $` + strconv.Itoa(len(args)+1))
		args = append(args, status)
	}
	qb.WriteString(` ORDER BY rr.created_at DESC LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2))
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, qb.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListReports: %w", err)
	}
	defer rows.Close()

	reports := make([]ReviewReport, 0, 20)
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
			&review.LikeCount, &review.DislikeCount, &review.ReplyCount, &review.Status, &review.CreatedAt, &review.UpdatedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("ListReports scan: %w", err)
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
func (r *Repository) GetReportByID(ctx context.Context, reportID string) (*ReviewReport, error) {
	ctx = withDBTable(ctx, "review_reports")
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
		return nil, fmt.Errorf("GetReportByID: %w", err)
	}
	return &rp, nil
}

// GetReportByIDForUpdate 在事务内获取举报并加行锁。
func (r *Repository) GetReportByIDForUpdate(ctx context.Context, tx pgx.Tx, reportID string) (*ReviewReport, error) {
	var rp ReviewReport
	err := tx.QueryRow(ctx, `
		SELECT id, review_id, reason, description, status,
		       resolved_by, resolved_at, resolution_note, created_at
		FROM review_reports WHERE id = $1
		FOR UPDATE
	`, reportID).Scan(
		&rp.ID, &rp.ReviewID, &rp.Reason, &rp.Description, &rp.Status,
		&rp.ResolvedBy, &rp.ResolvedAt, &rp.ResolutionNote, &rp.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("GetReportByIDForUpdate: %w", err)
	}
	return &rp, nil
}

func (r *Repository) GetReportSchoolID(ctx context.Context, reportID string) (int64, error) {
	ctx = withDBTable(ctx, "review_reports")
	var schoolID int64
	err := r.db.QueryRow(ctx, `
		SELECT school_id
		FROM review_reports
		WHERE id = $1
	`, reportID).Scan(&schoolID)
	return schoolID, err
}

// UpdateReportParams 更新举报参数
type UpdateReportParams struct {
	ID         string
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
	if err != nil {
		return fmt.Errorf("UpdateReport: %w", err)
	}
	return nil
}

// CreateFavorite 创建收藏
func (r *Repository) CreateFavorite(ctx context.Context, userHash string, courseID int64) error {
	ctx = withDBTable(ctx, "course_favorites")
	newID, err := id.New()
	if err != nil {
		return fmt.Errorf("CreateFavorite generate id: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO course_favorites (id, user_hash, course_id, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_hash, course_id) DO NOTHING
	`, newID, userHash, courseID)
	if err != nil {
		return fmt.Errorf("CreateFavorite: %w", err)
	}
	return nil
}

// DeleteFavorite 删除收藏
func (r *Repository) DeleteFavorite(ctx context.Context, userHash string, courseID int64) error {
	ctx = withDBTable(ctx, "course_favorites")
	_, err := r.db.Exec(ctx, `
		DELETE FROM course_favorites WHERE user_hash = $1 AND course_id = $2
	`, userHash, courseID)
	if err != nil {
		return fmt.Errorf("DeleteFavorite: %w", err)
	}
	return nil
}

// FavoriteExists 检查是否已收藏
func (r *Repository) FavoriteExists(ctx context.Context, userHash string, courseID int64) (bool, error) {
	ctx = withDBTable(ctx, "course_favorites")
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM course_favorites WHERE user_hash = $1 AND course_id = $2)
	`, userHash, courseID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("FavoriteExists: %w", err)
	}
	return exists, nil
}

// ListFavorites 获取用户收藏列表（含总数）
func (r *Repository) ListFavorites(ctx context.Context, userHash string, limit, offset int) ([]FavoriteCourse, int, error) {
	ctx = withDBTable(ctx, "course_favorites")
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
		return nil, 0, fmt.Errorf("ListFavorites: %w", err)
	}
	defer rows.Close()

	list := make([]FavoriteCourse, 0, limit)
	var total int
	for rows.Next() {
		var f FavoriteCourse
		if err := rows.Scan(&f.ID, &f.Name, &f.Code, &f.Credits, &f.DepartmentID, &f.DepartmentName, &f.ReviewCount, &f.FavoritedAt, &total); err != nil {
			return nil, 0, fmt.Errorf("ListFavorites scan: %w", err)
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
	Status   string
}

// ReplyTimestamps 保存数据库为新回复生成的时间戳。
type ReplyTimestamps struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateReply 创建回复
func (r *Repository) CreateReply(ctx context.Context, tx pgx.Tx, p CreateReplyParams) (string, *ReplyTimestamps, error) {
	replyID, err := id.New()
	if err != nil {
		return "", nil, fmt.Errorf("CreateReply generate id: %w", err)
	}
	var ts ReplyTimestamps
	err = tx.QueryRow(ctx, `
		INSERT INTO review_replies (id, review_id, parent_id, user_hash, content, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at
	`, replyID, p.ReviewID, p.ParentID, p.UserHash, p.Content, p.Status).Scan(&ts.CreatedAt, &ts.UpdatedAt)
	if err != nil {
		return "", nil, fmt.Errorf("CreateReply: %w", err)
	}
	return replyID, &ts, nil
}

func (r *Repository) ReplyBelongsToReviewTx(ctx context.Context, tx pgx.Tx, replyID string, reviewID string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM review_replies
			WHERE id = $1 AND review_id = $2 AND status = 'published'
			FOR SHARE
		)
	`, replyID, reviewID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("ReplyBelongsToReviewTx: %w", err)
	}
	return exists, nil
}

// ListReplies 获取回复列表（含总数）
func (r *Repository) ListReplies(ctx context.Context, reviewID string, limit, offset int) ([]Reply, int, error) {
	ctx = withDBTable(ctx, "review_replies")
	rows, err := r.db.Query(ctx, `
		SELECT id, review_id, parent_id, user_hash, content, like_count, status, created_at, updated_at,
		       COUNT(*) OVER() AS total
		FROM review_replies
		WHERE review_id = $1 AND status = 'published'
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`, reviewID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("ListReplies: %w", err)
	}
	defer rows.Close()

	list := make([]Reply, 0, limit)
	var total int
	for rows.Next() {
		var reply Reply
		if err := rows.Scan(
			&reply.ID, &reply.ReviewID, &reply.ParentID, &reply.UserHash,
			&reply.Content, &reply.LikeCount, &reply.Status, &reply.CreatedAt, &reply.UpdatedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("ListReplies scan: %w", err)
		}
		list = append(list, reply)
	}
	return list, total, rows.Err()
}

// GetReplyOwner 获取回复所有者
func (r *Repository) GetReplyOwner(ctx context.Context, replyID string) (string, error) {
	ctx = withDBTable(ctx, "review_replies")
	var userHash string
	err := r.db.QueryRow(ctx, `
		SELECT user_hash FROM review_replies WHERE id = $1
	`, replyID).Scan(&userHash)
	if err != nil {
		return "", fmt.Errorf("GetReplyOwner: %w", err)
	}
	return userHash, nil
}

// GetReplyReviewID 获取回复所属的评论ID
func (r *Repository) GetReplyReviewID(ctx context.Context, replyID string) (string, error) {
	ctx = withDBTable(ctx, "review_replies")
	var reviewID string
	err := r.db.QueryRow(ctx, `
		SELECT review_id FROM review_replies WHERE id = $1
	`, replyID).Scan(&reviewID)
	if err != nil {
		return "", fmt.Errorf("GetReplyReviewID: %w", err)
	}
	return reviewID, nil
}

// GetReplyOwnerAndReviewID 获取回复所有者和所属评论ID（单次查询）
func (r *Repository) GetReplyOwnerAndReviewID(ctx context.Context, replyID string) (string, string, error) {
	ctx = withDBTable(ctx, "review_replies")
	var userHash, reviewID string
	err := r.db.QueryRow(ctx, `
		SELECT user_hash, review_id FROM review_replies WHERE id = $1
	`, replyID).Scan(&userHash, &reviewID)
	if err != nil {
		return "", "", fmt.Errorf("GetReplyOwnerAndReviewID: %w", err)
	}
	return userHash, reviewID, nil
}

// GetReplyOwnerAndReviewIDTx 在事务内获取回复所有者和评论ID
func (r *Repository) GetReplyOwnerAndReviewIDTx(ctx context.Context, tx pgx.Tx, replyID string) (string, string, string, error) {
	var userHash, reviewID, status string
	err := tx.QueryRow(ctx, `
		SELECT user_hash, review_id, status
		FROM review_replies
		WHERE id = $1
		FOR UPDATE
	`, replyID).Scan(&userHash, &reviewID, &status)
	if err != nil {
		return "", "", "", fmt.Errorf("GetReplyOwnerAndReviewIDTx: %w", err)
	}
	return userHash, reviewID, status, nil
}

// SoftDeleteReply 软删除回复
func (r *Repository) SoftDeleteReply(ctx context.Context, tx pgx.Tx, replyID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE review_replies SET status = 'deleted', updated_at = NOW() WHERE id = $1
	`, replyID)
	if err != nil {
		return fmt.Errorf("SoftDeleteReply: %w", err)
	}
	return nil
}

// UpdateReplyStatusTx 在事务内更新回复状态。
func (r *Repository) UpdateReplyStatusTx(ctx context.Context, tx pgx.Tx, replyID string, status string) error {
	_, err := tx.Exec(ctx, `
		UPDATE review_replies SET status = $2, updated_at = NOW() WHERE id = $1
	`, replyID, status)
	if err != nil {
		return fmt.Errorf("UpdateReplyStatusTx: %w", err)
	}
	return nil
}

// IncrementReplyCount 增加评论回复计数
func (r *Repository) IncrementReplyCount(ctx context.Context, tx pgx.Tx, reviewID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE reviews SET reply_count = reply_count + 1 WHERE id = $1
	`, reviewID)
	if err != nil {
		return fmt.Errorf("IncrementReplyCount: %w", err)
	}
	return nil
}

// DecrementReplyCount 减少评论回复计数
func (r *Repository) DecrementReplyCount(ctx context.Context, tx pgx.Tx, reviewID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE reviews SET reply_count = GREATEST(reply_count - 1, 0) WHERE id = $1
	`, reviewID)
	if err != nil {
		return fmt.Errorf("DecrementReplyCount: %w", err)
	}
	return nil
}
