package review

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

// maxExportLimit 导出查询的最大行数上限
// M-50: 硬编码上限防止单次导出过大数据集导致 OOM，超出此限制的数据将被截断。
// 如需调整，应同步评估数据库查询性能和内存占用。
const maxExportLimit = 10000

// allowedStatuses 允许的 status 参数白名单
var allowedStatuses = map[string]bool{
	"all":       true,
	"published": true,
	"hidden":    true,
	"deleted":   true,
	"pending":   true,
	"resolved":  true,
	"rejected":  true,
}

// validateStatus 校验 status 参数是否在白名单内，不合法时回退为 fallback
func validateStatus(status, fallback string) string {
	if status == "" || !allowedStatuses[status] {
		return fallback
	}
	return status
}

// ListAllReviews 获取所有评论（管理员，含总数）
// M-86: 使用 strings.Builder 重构 SQL 构建逻辑，参数绑定更清晰
func (r *Repository) ListAllReviews(ctx context.Context, status string, limit, offset int) ([]Review, int, error) {
	status = validateStatus(status, "all")

	var qb strings.Builder
	qb.WriteString(`
		SELECT r.id, r.course_id, c.name, r.teacher_id, t.name, r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count,
		       r.reply_count,
		       r.status, r.created_at,
		       COUNT(*) OVER() AS total
		FROM reviews r
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
	`)

	var args []interface{}
	needFilter := status != "" && status != "all"

	if needFilter {
		qb.WriteString(` WHERE r.status = $1`)
		qb.WriteString(` ORDER BY r.created_at DESC LIMIT $2 OFFSET $3`)
		args = []interface{}{status, limit, offset}
	} else {
		qb.WriteString(` ORDER BY r.created_at DESC LIMIT $1 OFFSET $2`)
		args = []interface{}{limit, offset}
	}

	rows, err := r.db.Query(ctx, qb.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListAllReviews: %w", err)
	}
	defer rows.Close()
	return scanReviewsWithTotal(rows)
}

// GetAdminStats 获取管理统计（条件聚合，reviews 单次扫描 + reports 单次扫描）
// M-52: 性能优化依赖以下索引（应在 init.sql 中创建）：
//   - reviews: idx_reviews_status (status) — 加速 FILTER 条件聚合
//   - reviews: idx_reviews_created_at (created_at) — 加速 today/week 过滤
//   - review_reports: idx_review_reports_status (status) — 加速 pending 计数
func (r *Repository) GetAdminStats(ctx context.Context) (*AdminStats, error) {
	var stats AdminStats
	// reviews 表：6 个计数合并为单次全表扫描 + FILTER 条件聚合
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total_reviews,
			COUNT(*) FILTER (WHERE status = 'published') AS published_reviews,
			COUNT(*) FILTER (WHERE status = 'hidden') AS hidden_reviews,
			COUNT(*) FILTER (WHERE status = 'deleted') AS deleted_reviews,
			COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE) AS today_reviews,
			COUNT(*) FILTER (WHERE created_at >= date_trunc('week', CURRENT_DATE)) AS week_reviews
		FROM reviews
	`).Scan(
		&stats.TotalReviews,
		&stats.PublishedReviews,
		&stats.HiddenReviews,
		&stats.DeletedReviews,
		&stats.TodayReviews,
		&stats.WeekReviews,
	)
	if err != nil {
		return nil, fmt.Errorf("GetAdminStats reviews: %w", err)
	}
	// reports 表：独立扫描（与 reviews 无关联，无法合并）
	err = r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total_reports,
			COUNT(*) FILTER (WHERE status = 'pending') AS pending_reports
		FROM review_reports
	`).Scan(
		&stats.TotalReports,
		&stats.PendingReports,
	)
	if err != nil {
		return nil, fmt.Errorf("GetAdminStats reports: %w", err)
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
		return fmt.Errorf("CreateNotification generate id: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO notifications (id, user_hash, type, title, content, related_type, related_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, newID, p.UserHash, p.Type, p.Title, p.Content, p.RelatedType, p.RelatedID)
	if err != nil {
		return fmt.Errorf("CreateNotification: %w", err)
	}
	return nil
}

// ListNotificationsResult 通知列表查询结果（含总数和未读数）
type ListNotificationsResult struct {
	List   []Notification
	Total  int
	Unread int
}

// ListNotifications 获取通知列表（含总数和未读数，单次查询避免数据不一致）
func (r *Repository) ListNotifications(ctx context.Context, userHash string, limit, offset int) (*ListNotificationsResult, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, type, title, content, related_type, related_id, is_read, created_at,
		       COUNT(*) OVER() AS total,
		       SUM(CASE WHEN is_read = false THEN 1 ELSE 0 END) OVER() AS unread
		FROM notifications
		WHERE user_hash = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userHash, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("ListNotifications: %w", err)
	}
	defer rows.Close()

	result := &ListNotificationsResult{List: make([]Notification, 0, limit)}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(
			&n.ID, &n.Type, &n.Title, &n.Content,
			&n.RelatedType, &n.RelatedID, &n.IsRead, &n.CreatedAt,
			&result.Total, &result.Unread,
		); err != nil {
			return nil, fmt.Errorf("ListNotifications scan: %w", err)
		}
		result.List = append(result.List, n)
	}
	return result, rows.Err()
}

// CountUnreadNotifications 统计未读通知数量
func (r *Repository) CountUnreadNotifications(ctx context.Context, userHash string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications WHERE user_hash = $1 AND is_read = false
	`, userHash).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("CountUnreadNotifications: %w", err)
	}
	return count, nil
}

// MarkNotificationRead 标记通知已读
func (r *Repository) MarkNotificationRead(ctx context.Context, id string, userHash string) error {
	result, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = true WHERE id = $1 AND user_hash = $2
	`, id, userHash)
	if err != nil {
		return fmt.Errorf("MarkNotificationRead: %w", err)
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
	if err != nil {
		return fmt.Errorf("MarkAllNotificationsRead: %w", err)
	}
	return nil
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
// H-48: 通过 RETURNING xmax 判断是新建还是更新，保证幂等性
// xmax = 0 表示 INSERT（新建），xmax > 0 表示 UPDATE（更新已有记录）
func (r *Repository) UpsertDraft(ctx context.Context, p UpsertDraftParams) (*ReviewDraft, error) {
	var d ReviewDraft
	var xmax int64
	newID, err := id.New()
	if err != nil {
		return nil, fmt.Errorf("UpsertDraft generate id: %w", err)
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
		RETURNING id, course_id, teacher_id, term_id, title, content, grade, ratings, updated_at, xmax
	`, newID, p.UserHash, p.CourseID, p.TeacherID, p.TermID, p.Title, p.Content, p.Grade, p.Ratings).Scan(
		&d.ID, &d.CourseID, &d.TeacherID, &d.TermID, &d.Title, &d.Content, &d.Grade, &d.Ratings, &d.UpdatedAt, &xmax,
	)
	if err != nil {
		return nil, fmt.Errorf("UpsertDraft: %w", err)
	}
	// xmax == 0 表示新建，xmax > 0 表示更新（可用于调用方日志记录）
	_ = xmax
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDraftNotFound
		}
		return nil, fmt.Errorf("GetDraft: %w", err)
	}
	return &d, nil
}

// DeleteDraft 删除草稿
func (r *Repository) DeleteDraft(ctx context.Context, userHash string, courseID int64) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM review_drafts WHERE user_hash = $1 AND course_id = $2
	`, userHash, courseID)
	if err != nil {
		return fmt.Errorf("DeleteDraft: %w", err)
	}
	return nil
}

// maxBatchDBSize 数据库层批量操作的最大数量上限（L-36: 纵深防御，防止绕过 handler 直接调用）
const maxBatchDBSize = 1000

// BatchUpdateReviewStatusTx 批量更新评论状态（事务内执行）
// L-29: 使用 FOR UPDATE 行锁防止并发修改
// L-36: 校验批量上限
func (r *Repository) BatchUpdateReviewStatusTx(ctx context.Context, tx pgx.Tx, ids []string, status string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	if len(ids) > maxBatchDBSize {
		return 0, fmt.Errorf("batch size %d exceeds db limit of %d", len(ids), maxBatchDBSize)
	}

	// L-29: 先 SELECT FOR UPDATE 锁定行，再执行 UPDATE
	if _, err := tx.Exec(ctx, `SELECT id FROM reviews WHERE id = ANY($1) FOR UPDATE`, ids); err != nil {
		return 0, fmt.Errorf("BatchUpdateReviewStatusTx lock: %w", err)
	}

	result, err := tx.Exec(ctx, `
		UPDATE reviews SET status = $1, updated_at = NOW()
		WHERE id = ANY($2)
	`, status, ids)
	if err != nil {
		return 0, fmt.Errorf("BatchUpdateReviewStatusTx: %w", err)
	}
	return result.RowsAffected(), nil
}

// LockReviewsTx 在事务内对指定评论行加排他锁，防止并发操作导致计数不一致
func (r *Repository) LockReviewsTx(ctx context.Context, tx pgx.Tx, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `SELECT id FROM reviews WHERE id = ANY($1) FOR UPDATE`, ids)
	if err != nil {
		return fmt.Errorf("LockReviewsTx: %w", err)
	}
	return nil
}

// AdjustCourseCountsForBatchDelete 批量删除时减少相关课程的评论计数
// 仅对当前状态非 deleted 的评论所属课程进行计数调整
// H-49: 此子查询依赖索引 idx_reviews_id_status (id, status) 或主键索引 + status 过滤，
// 确保 WHERE id = ANY($1) 走索引扫描而非全表扫描
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
	if err != nil {
		return fmt.Errorf("AdjustCourseCountsForBatchDelete: %w", err)
	}
	return nil
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
	if err != nil {
		return fmt.Errorf("AdjustCourseCountsForBatchHide: %w", err)
	}
	return nil
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
	if err != nil {
		return fmt.Errorf("AdjustCourseCountsForBatchRestore: %w", err)
	}
	return nil
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
		return fmt.Errorf("CreateOperationLog generate id: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO admin_operation_logs
		(id, admin_user_id, admin_username, action, resource_type, resource_id, old_value, new_value, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, newID, p.AdminUserID, p.AdminUsername, p.Action, p.ResourceType, p.ResourceID, p.OldValue, p.NewValue, p.IPAddress, p.UserAgent)
	if err != nil {
		return fmt.Errorf("CreateOperationLog: %w", err)
	}
	return nil
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
		return nil, 0, fmt.Errorf("ListOperationLogs: %w", err)
	}
	defer rows.Close()

	list := make([]AdminOperationLog, 0, limit)
	var total int
	for rows.Next() {
		var log AdminOperationLog
		if err := rows.Scan(
			&log.ID, &log.AdminUserID, &log.AdminUsername, &log.Action, &log.ResourceType, &log.ResourceID,
			&log.OldValue, &log.NewValue, &log.IPAddress, &log.UserAgent, &log.CreatedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("ListOperationLogs scan: %w", err)
		}
		list = append(list, log)
	}
	return list, total, rows.Err()
}

// defaultLogRetentionDays is the default retention period for admin operation logs
const defaultLogRetentionDays = 90

// CleanupOldOperationLogs deletes operation logs older than the specified retention period.
// Returns the number of deleted rows.
func (r *Repository) CleanupOldOperationLogs(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = defaultLogRetentionDays
	}
	result, err := r.db.Exec(ctx, `
		DELETE FROM admin_operation_logs
		WHERE created_at < NOW() - make_interval(days => $1)
	`, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("CleanupOldOperationLogs: %w", err)
	}
	return result.RowsAffected(), nil
}

// buildExportQuery 构建导出查询的 SQL 和参数
func buildExportQuery(status string) (string, []interface{}) {
	status = validateStatus(status, "all")
	baseQuery := `
		SELECT r.id, r.course_id, COALESCE(c.name, '') as course_name,
			r.teacher_id, COALESCE(t.name, '') as teacher_name,
			r.term_id, r.title, r.content, r.grade, r.ratings,
			r.like_count, r.dislike_count,
			r.reply_count,
			r.status, r.created_at, r.updated_at
		FROM reviews r
		LEFT JOIN courses c ON r.course_id = c.id
		LEFT JOIN teachers t ON r.teacher_id = t.id`

	if status != "" && status != "all" {
		return baseQuery + `
			WHERE r.status = $1
			ORDER BY r.created_at DESC
			LIMIT $2`, []interface{}{status, maxExportLimit}
	}
	return baseQuery + `
		ORDER BY r.created_at DESC
		LIMIT $1`, []interface{}{maxExportLimit}
}

// ForEachReviewForExport 流式遍历导出评论，逐行回调避免全量加载到内存
func (r *Repository) ForEachReviewForExport(ctx context.Context, status string, fn func(Review) error) error {
	query, args := buildExportQuery(status)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("ForEachReviewForExport: %w", err)
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
			&review.ReplyCount, &review.Status, &review.CreatedAt, &review.UpdatedAt,
		); err != nil {
			return fmt.Errorf("ForEachReviewForExport scan: %w", err)
		}
		if err := fn(review); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("ForEachReviewForExport rows iteration: %w", err)
	}
	return nil
}
