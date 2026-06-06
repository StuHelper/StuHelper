package notification

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestNewServicePanicsOnNilDependencies(t *testing.T) {
	tests := []struct {
		name string
		repo *Repository
		hub  *Hub
		rdb  *redis.Client
	}{
		{name: "nil repo", hub: &Hub{}, rdb: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})},
		{name: "nil hub", repo: &Repository{}, rdb: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})},
		{name: "nil redis", repo: &Repository{}, hub: &Hub{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("expected panic")
				}
			}()
			_ = NewService(tt.repo, tt.hub, tt.rdb)
		})
	}
}

func TestServiceRejectsInvalidUserIDBeforeDependencies(t *testing.T) {
	ctx := context.Background()
	service := NewService(
		&Repository{},
		&Hub{},
		redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}),
	)

	err := service.Send(ctx, SendParams{
		UserID: -1,
		Type:   TypeSystem,
		Title:  "系统通知",
		Body:   "body",
	})
	assert.ErrorIs(t, err, ErrInvalidUserID)

	_, err = service.List(ctx, 0, 1, 20)
	assert.ErrorIs(t, err, ErrInvalidUserID)

	_, err = service.CountUnread(ctx, 0)
	assert.ErrorIs(t, err, ErrInvalidUserID)

	err = service.MarkRead(ctx, "550e8400-e29b-41d4-a716-446655440000", 0)
	assert.ErrorIs(t, err, ErrInvalidUserID)

	err = service.MarkAllRead(ctx, -1)
	assert.ErrorIs(t, err, ErrInvalidUserID)
}

func TestServiceSendBatchReturnsInvalidUserID(t *testing.T) {
	ctx := context.Background()
	service := NewService(
		&Repository{},
		&Hub{},
		redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}),
	)

	err := service.SendBatch(ctx, []SendParams{{
		UserID: 0,
		Type:   TypeSystem,
		Title:  "系统通知",
		Body:   "body",
	}})

	assert.ErrorIs(t, err, ErrInvalidUserID)
}
