package review

import (
	"context"
	"fmt"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

// maxExportLimit 导出查询的最大行数上限
const maxExportLimit = 10000

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

// defaultLogRetentionDays is the default retention period for admin operation logs
const defaultLogRetentionDays = 90

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

// ListOperationLogs 获取操作日志列表
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

// CleanupOldOperationLogs 清理历史操作日志
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

// buildExportQuery 构建导出查询
func buildExportQuery(status string) (string, []interface{}) {
	status = validateStatus(status, "all")
	baseQuery := `
		SELECT r.id, r.course_id, COALESCE(c.name, '') as course_name,
			r.teacher_id, COALESCE(t.name, '') as teacher_name,
			r.term_id, r.title, r.content, r.grade, r.ratings,
			r.like_count, r.dislike_count,
			r.reply_count,
			r.status, r.moderation_reason, r.created_at, r.updated_at
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

// ForEachReviewForExport 流式遍历导出评论
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
			&review.ReplyCount, &review.Status, &review.ModerationReason,
			&review.CreatedAt, &review.UpdatedAt,
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
