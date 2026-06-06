package notification

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// 业务错误
var (
	ErrNotFound      = errors.New("notification not found")
	ErrInvalidUserID = errors.New("notification user id is invalid")
)

// Service 通知服务
type Service struct {
	repo     *Repository
	realtime RealtimePublisher
}

var _ Sender = (*Service)(nil)

// NewService 创建通知服务
func NewService(repo *Repository, realtime RealtimePublisher) *Service {
	if repo == nil {
		panic("notification.NewService: repo must not be nil")
	}
	if realtime == nil {
		panic("notification.NewService: realtime must not be nil")
	}
	return &Service{repo: repo, realtime: realtime}
}

// Send 发送通知（实现 Sender 接口）
func (s *Service) Send(ctx context.Context, params SendParams) error {
	if err := validateNotificationUserID(params.UserID); err != nil {
		return err
	}
	body := params.Body
	if body == "" {
		body = params.Content
	}
	notifID, err := s.repo.Create(ctx, CreateParams{
		UserID:       params.UserID,
		Type:         params.Type,
		Title:        params.Title,
		Body:         body,
		Payload:      params.Payload,
		SourceModule: params.SourceModule,
		SourceID:     params.SourceID,
		SourceURL:    nilIfEmpty(params.SourceURL),
		CourseID:     courseIDOrNil(params.CourseID),
	})
	if err != nil {
		return fmt.Errorf("notification send: %w", err)
	}

	s.publishRealtime(ctx, params.UserID, SSEEvent{
		Event: "notification",
		Data:  buildRealtimeNotificationPayload(notifID, params, time.Now().UTC()),
	})

	return nil
}

// SendBatch 批量发送通知（实现 Sender 接口）
// 使用 errgroup 并发发送，限制并发数避免打爆 DB
func (s *Service) SendBatch(ctx context.Context, params []SendParams) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10)

	var mu sync.Mutex
	var errs []error

	for _, p := range params {
		g.Go(func() error {
			if err := s.Send(ctx, p); err != nil {
				logger.L().Warn("batch notification send failed",
					zap.Int64("user_id", p.UserID),
					zap.Error(err),
				)
				mu.Lock()
				errs = append(errs, fmt.Errorf("user %d: %w", p.UserID, err))
				mu.Unlock()
			}
			return nil // 不中断其他发送
		})
	}
	waitErr := g.Wait()
	return errors.Join(waitErr, errors.Join(errs...))
}

// List 获取通知列表
func (s *Service) List(ctx context.Context, userID int64, page, pageSize int) (*ListResult, error) {
	if err := validateNotificationUserID(userID); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, ListParams{UserID: userID, Page: page, PageSize: pageSize})
}

// CountUnread 获取未读通知数量
func (s *Service) CountUnread(ctx context.Context, userID int64) (int, error) {
	if err := validateNotificationUserID(userID); err != nil {
		return 0, err
	}
	return s.repo.CountUnread(ctx, userID)
}

// MarkRead 标记通知已读
func (s *Service) MarkRead(ctx context.Context, notifID string, userID int64) error {
	if err := validateNotificationUserID(userID); err != nil {
		return err
	}
	marked, err := s.repo.MarkRead(ctx, notifID, userID)
	if err != nil {
		return err
	}
	if !marked {
		return nil
	}

	s.publishRealtime(ctx, userID, SSEEvent{
		Event: "notification_read",
		Data: map[string]any{
			"id":     notifID,
			"userId": userID,
			"isRead": true,
		},
	})

	return nil
}

// MarkAllRead 标记所有通知已读
func (s *Service) MarkAllRead(ctx context.Context, userID int64) error {
	if err := validateNotificationUserID(userID); err != nil {
		return err
	}
	if err := s.repo.MarkAllRead(ctx, userID); err != nil {
		return err
	}

	s.publishRealtime(ctx, userID, SSEEvent{
		Event: "notification_read_all",
		Data: map[string]any{
			"userId": userID,
			"isRead": true,
		},
	})

	// 推送未读数更新
	s.publishRealtime(ctx, userID, SSEEvent{
		Event: "unread_count",
		Data:  map[string]int{"count": 0},
	})

	return nil
}

func validateNotificationUserID(userID int64) error {
	if userID <= 0 {
		return ErrInvalidUserID
	}
	return nil
}

func (s *Service) publishRealtime(ctx context.Context, userID int64, event SSEEvent) {
	if err := s.realtime.Publish(ctx, userID, event); err != nil {
		logger.L().Warn("failed to publish notification realtime event",
			zap.Int64("user_id", userID),
			zap.String("event", event.Event),
			zap.Error(err),
		)
	}
}

func buildRealtimeNotificationPayload(notifID string, params SendParams, createdAt time.Time) Notification {
	body := params.Body
	if body == "" {
		body = params.Content
	}

	return Notification{
		ID:           notifID,
		UserID:       params.UserID,
		Type:         params.Type,
		Title:        params.Title,
		Body:         body,
		Content:      body,
		Payload:      payloadOrEmptyJSON(params.Payload),
		SourceModule: params.SourceModule,
		SourceID:     params.SourceID,
		SourceURL:    nilIfEmpty(params.SourceURL),
		CourseID:     courseIDOrNil(params.CourseID),
		IsRead:       false,
		CreatedAt:    createdAt.Format(time.RFC3339Nano),
	}
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func courseIDOrNil(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
}
