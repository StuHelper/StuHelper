package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

// Repository 通知数据访问层
type Repository struct {
	db *db.DB
}

// NewRepository 创建通知仓储
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

// CreateParams 创建通知参数
type CreateParams struct {
	UserID       int64
	Type         string
	Title        string
	Body         string
	Payload      json.RawMessage
	SourceModule string
	SourceID     string
	SourceURL    *string
	CourseID     *int64
}

// ListParams 查询通知列表参数
type ListParams struct {
	UserID   int64
	Page     int
	PageSize int
}

// ListResult 通知列表查询结果
type ListResult struct {
	List   []Notification
	Total  int
	Unread int
}

// Create 创建通知
func (r *Repository) Create(ctx context.Context, p CreateParams) (string, error) {
	newID, err := id.New()
	if err != nil {
		return "", fmt.Errorf("notification create generate id: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO notifications (id, user_id, type, title, body, payload, source_module, source_id, source_url, source_course_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, newID, p.UserID, p.Type, p.Title, p.Body, payloadOrEmptyJSON(p.Payload), p.SourceModule, p.SourceID, p.SourceURL, p.CourseID)
	if err != nil {
		return "", fmt.Errorf("notification create: %w", err)
	}
	return newID, nil
}

// List 获取通知列表
func (r *Repository) List(ctx context.Context, p ListParams) (*ListResult, error) {
	offset := httputil.SafeOffset(p.Page, p.PageSize)
	result := &ListResult{List: make([]Notification, 0, p.PageSize)}
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE is_read = false)
		FROM notifications
		WHERE user_id = $1
	`, p.UserID).Scan(&result.Total, &result.Unread); err != nil {
		return nil, fmt.Errorf("notification list count: %w", err)
	}
	if result.Total == 0 {
		return result, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, type, title, body, payload, source_module, source_id, source_url, source_course_id, is_read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, p.UserID, p.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("notification list data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var n Notification
		if err := rows.Scan(
			&n.ID, &n.Type, &n.Title, &n.Body, &n.Payload,
			&n.SourceModule, &n.SourceID, &n.SourceURL, &n.CourseID,
			&n.IsRead, &n.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("notification list scan: %w", err)
		}
		n.Content = n.Body
		result.List = append(result.List, n)
	}
	return result, rows.Err()
}

func payloadOrEmptyJSON(payload json.RawMessage) json.RawMessage {
	if len(payload) == 0 {
		return json.RawMessage(`{}`)
	}
	return payload
}

// CountUnread 统计未读通知数量
func (r *Repository) CountUnread(ctx context.Context, userID int64) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false
	`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("notification count unread: %w", err)
	}
	return count, nil
}

// ResolveInternalUserID 根据 casdoor_subject 查询内部 user_id。
func (r *Repository) ResolveInternalUserID(ctx context.Context, casdoorSubject string) (int64, error) {
	var userID int64
	err := r.db.QueryRow(ctx, `SELECT id FROM users WHERE casdoor_subject = $1`, casdoorSubject).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("notification resolve internal user id: %w", err)
	}
	return userID, nil
}

// MarkRead 标记通知已读
func (r *Repository) MarkRead(ctx context.Context, notifID string, userID int64) error {
	result, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2
	`, notifID, userID)
	if err != nil {
		return fmt.Errorf("notification mark read: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkAllRead 标记所有通知已读
func (r *Repository) MarkAllRead(ctx context.Context, userID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false
	`, userID)
	if err != nil {
		return fmt.Errorf("notification mark all read: %w", err)
	}
	return nil
}
