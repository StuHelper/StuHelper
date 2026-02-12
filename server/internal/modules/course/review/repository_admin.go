package review

import (
	"context"

	"github.com/jackc/pgx/v5"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

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
	newID, err := id.New()
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO notifications (id, user_hash, type, title, content, related_type, related_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, newID, p.UserHash, p.Type, p.Title, p.Content, p.RelatedType, p.RelatedID)
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
func (r *Repository) MarkNotificationRead(ctx context.Context, id string, userHash string) error {
	result, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = true WHERE id = $1 AND user_hash = $2
	`, id, userHash)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

// MarkAllNotificationsRead 标记所有通知已读
func (r *Repository) MarkAllNotificationsRead(ctx context.Context, userHash string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = true WHERE user_hash = $1 AND is_read = false
	`, userHash)
	return err
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
	newID, err := id.New()
	if err != nil {
		return nil, err
	}
	err = r.db.QueryRow(ctx, `
		INSERT INTO review_drafts (id, user_hash, course_id, teacher_id, term_id, title, content, grade, ratings, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (user_hash, course_id) DO UPDATE SET
			teacher_id = EXCLUDED.teacher_id,
			term_id = EXCLUDED.term_id,
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			grade = EXCLUDED.grade,
			ratings = EXCLUDED.ratings,
			updated_at = NOW()
		RETURNING id, course_id, teacher_id, term_id, title, content, grade, ratings, updated_at
	`, newID, p.UserHash, p.CourseID, p.TeacherID, p.TermID, p.Title, p.Content, p.Grade, p.Ratings).Scan(
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

// AdjustCourseCountsForBatchHide 批量隐藏时减少相关课程的评论计数
// 仅对当前状态为 published 的评论所属课程进行计数调整
func (r *Repository) AdjustCourseCountsForBatchHide(ctx context.Context, tx pgx.Tx, ids []string) error {
	_, err := tx.Exec(ctx, `
		UPDATE courses SET review_count = GREATEST(review_count - sub.cnt, 0)
		FROM (
			SELECT course_id, COUNT(*) AS cnt
			FROM reviews
			WHERE id = ANY($1) AND status = 'published'
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
			WHERE id = ANY($1) AND status != 'published'
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
	newID, err := id.New()
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO admin_operation_logs
		(id, admin_user_id, admin_username, action, resource_type, resource_id, old_value, new_value, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, newID, p.AdminUserID, p.AdminUsername, p.Action, p.ResourceType, p.ResourceID, p.OldValue, p.NewValue, p.IPAddress, p.UserAgent)
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
