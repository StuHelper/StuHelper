package review

import (
	"context"
	"fmt"
)

// NOTE: 通知写入能力已迁移至 internal/modules/notification.Service.Send。
// review 模块的 CreateNotification 已删除（零调用者）。
// reply/vote/report 通知生产者尚无法接入——详见 docs/exec-plans/active/notification-wiring.md。

// ListNotificationsResult 通知列表查询结果
type ListNotificationsResult struct {
	List   []Notification
	Total  int
	Unread int
}

// ListNotifications 获取通知列表
func (r *Repository) ListNotifications(ctx context.Context, userID int64, limit, offset int) (*ListNotificationsResult, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, type, title, body, payload, source_module, source_id, source_url, source_course_id, is_read, created_at,
		       COUNT(*) OVER() AS total,
		       SUM(CASE WHEN is_read = false THEN 1 ELSE 0 END) OVER() AS unread
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("ListNotifications: %w", err)
	}
	defer rows.Close()

	result := &ListNotificationsResult{List: make([]Notification, 0, limit)}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(
			&n.ID, &n.Type, &n.Title, &n.Content,
			&n.Payload, &n.SourceModule, &n.SourceID, &n.SourceURL, &n.CourseID, &n.IsRead, &n.CreatedAt,
			&result.Total, &result.Unread,
		); err != nil {
			return nil, fmt.Errorf("ListNotifications scan: %w", err)
		}
		result.List = append(result.List, n)
	}
	return result, rows.Err()
}

// CountUnreadNotifications 统计未读通知数量
func (r *Repository) CountUnreadNotifications(ctx context.Context, userID int64) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false
	`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("CountUnreadNotifications: %w", err)
	}
	return count, nil
}

// MarkNotificationRead 标记通知已读
func (r *Repository) MarkNotificationRead(ctx context.Context, notifID string, userID int64) error {
	result, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2
	`, notifID, userID)
	if err != nil {
		return fmt.Errorf("MarkNotificationRead: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

// MarkAllNotificationsRead 标记所有通知已读
func (r *Repository) MarkAllNotificationsRead(ctx context.Context, userID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false
	`, userID)
	if err != nil {
		return fmt.Errorf("MarkAllNotificationsRead: %w", err)
	}
	return nil
}
