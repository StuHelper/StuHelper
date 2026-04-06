package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// 业务错误
var (
	ErrNotFound = errors.New("notification not found")
)

// Service 通知服务
type Service struct {
	repo *Repository
	hub  *Hub
	rdb  *redis.Client
}

// NewService 创建通知服务
func NewService(repo *Repository, hub *Hub, rdb *redis.Client) *Service {
	return &Service{repo: repo, hub: hub, rdb: rdb}
}

// Send 发送通知（实现 Sender 接口）
func (s *Service) Send(ctx context.Context, params SendParams) error {
	notifID, err := s.repo.Create(ctx, CreateParams{
		UserID:       params.UserID,
		Type:         params.Type,
		Title:        params.Title,
		Body:         params.Body,
		SourceModule: params.SourceModule,
		SourceID:     params.SourceID,
		SourceURL:    nilIfEmpty(params.SourceURL),
		CourseID:     courseIDOrNil(params.CourseID),
	})
	if err != nil {
		return fmt.Errorf("notification send: %w", err)
	}

	// 通过 Redis Pub/Sub 广播给 SSE 连接
	s.publishToRedis(ctx, params.UserID, notifID, params)

	return nil
}

// SendBatch 批量发送通知（实现 Sender 接口）
func (s *Service) SendBatch(ctx context.Context, params []SendParams) error {
	var errs []error
	for _, p := range params {
		if err := s.Send(ctx, p); err != nil {
			logger.L().Warn("batch notification send failed",
				zap.Int64("user_id", p.UserID),
				zap.Error(err),
			)
			errs = append(errs, fmt.Errorf("user %d: %w", p.UserID, err))
		}
	}
	return errors.Join(errs...)
}

// List 获取通知列表
func (s *Service) List(ctx context.Context, userID int64, page, pageSize int) (*ListResult, error) {
	return s.repo.List(ctx, ListParams{UserID: userID, Page: page, PageSize: pageSize})
}

// CountUnread 获取未读通知数量
func (s *Service) CountUnread(ctx context.Context, userID int64) (int, error) {
	return s.repo.CountUnread(ctx, userID)
}

// MarkRead 标记通知已读
func (s *Service) MarkRead(ctx context.Context, notifID string, userID int64) error {
	return s.repo.MarkRead(ctx, notifID, userID)
}

// MarkAllRead 标记所有通知已读
func (s *Service) MarkAllRead(ctx context.Context, userID int64) error {
	if err := s.repo.MarkAllRead(ctx, userID); err != nil {
		return err
	}

	// 推送未读数更新
	s.hub.Broadcast(userID, SSEEvent{
		Event: "unread_count",
		Data:  map[string]int{"count": 0},
	})

	return nil
}

// ResolveInternalUserID 根据 external_id 解析内部用户 ID。
func (s *Service) ResolveInternalUserID(ctx context.Context, externalID string) (int64, error) {
	return s.repo.ResolveInternalUserID(ctx, externalID)
}

// publishToRedis 通过 Redis Pub/Sub 广播通知
func (s *Service) publishToRedis(ctx context.Context, userID int64, notifID string, params SendParams) {
	payload := map[string]any{
		"id":           notifID,
		"type":         params.Type,
		"title":        params.Title,
		"body":         params.Body,
		"sourceModule": params.SourceModule,
		"sourceId":     params.SourceID,
	}
	if params.CourseID != 0 {
		payload["courseID"] = params.CourseID
	}
	data, err := json.Marshal(payload)
	if err != nil {
		logger.L().Warn("failed to marshal notification payload", zap.Error(err))
		return
	}

	channel := fmt.Sprintf("notify:%d", userID)
	if err := s.rdb.Publish(ctx, channel, data).Err(); err != nil {
		logger.L().Warn("failed to publish notification",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
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
