package review

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	auditpkg "git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
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
	RequestID     string
	TraceID       string
	IPAddress     string
	UserAgent     string
}

// CreateOperationLog 创建操作日志
func (r *Repository) CreateOperationLog(ctx context.Context, p CreateOperationLogParams) error {
	return auditpkg.NewRepository(r.db).WriteEvent(ctx, auditpkg.Event{
		Type:         eventTypeForOperationAction(p.Action),
		Category:     "admin_operation",
		ActorType:    "admin",
		UserID:       p.AdminUserID,
		Username:     p.AdminUsername,
		RequestID:    p.RequestID,
		TraceID:      p.TraceID,
		IP:           p.IPAddress,
		UserAgent:    p.UserAgent,
		Action:       p.Action,
		ResourceType: p.ResourceType,
		ResourceID:   p.ResourceID,
		Result:       "success",
		Before:       rawJSONOrNil(p.OldValue),
		After:        rawJSONOrNil(p.NewValue),
	})
}

// ListOperationLogs 获取操作日志列表
func (r *Repository) ListOperationLogs(ctx context.Context, limit, offset int) ([]AdminOperationLog, int, error) {
	items, total, err := auditpkg.NewRepository(r.db).ListAdminOperations(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("ListOperationLogs: %w", err)
	}
	list := make([]AdminOperationLog, 0, len(items))
	for _, item := range items {
		list = append(list, AdminOperationLog{
			ID:            item.ID,
			AdminUserID:   item.ActorUserID,
			AdminUsername: item.ActorUsername,
			Action:        item.Action,
			ResourceType:  item.ResourceType,
			ResourceID:    item.ResourceID,
			OldValue:      append([]byte(nil), item.BeforeData...),
			NewValue:      append([]byte(nil), item.AfterData...),
			IPAddress:     item.IPAddress,
			UserAgent:     item.UserAgent,
			CreatedAt:     item.CreatedAt,
		})
	}
	return list, total, nil
}

// CleanupOldOperationLogs 清理历史操作日志
func (r *Repository) CleanupOldOperationLogs(ctx context.Context, retentionDays int) (int64, error) {
	rows, err := auditpkg.NewRepository(r.db).CleanupAdminOperations(ctx, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("CleanupOldOperationLogs: %w", err)
	}
	return rows, nil
}

func eventTypeForOperationAction(action string) auditpkg.EventType {
	normalized := strings.ToLower(strings.TrimSpace(action))
	switch {
	case strings.Contains(normalized, "delete"):
		return auditpkg.EventDataDelete
	case strings.Contains(normalized, "create"):
		return auditpkg.EventDataCreate
	default:
		return auditpkg.EventDataUpdate
	}
}

func rawJSONOrNil(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return json.RawMessage(value)
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
	ctx = withDBTable(ctx, "reviews")
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
