package review

import (
	"context"
	"fmt"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

// CreateNotificationParams 创建通知参数
type CreateNotificationParams struct {
	UserHash    string
	Type        string
	Title       string
	Content     string
	RelatedType string
	RelatedID   string
}

// ListNotificationsResult 通知列表查询结果
type ListNotificationsResult struct {
	List   []Notification
	Total  int
	Unread int
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

// ListNotifications 获取通知列表
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
