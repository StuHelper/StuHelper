package notification

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubRealtimePublisher struct {
	err error
}

func (p stubRealtimePublisher) Publish(context.Context, int64, SSEEvent) error {
	if p.err != nil {
		return p.err
	}
	return nil
}

func TestNewServicePanicsOnNilDependencies(t *testing.T) {
	tests := []struct {
		name     string
		repo     *Repository
		realtime RealtimePublisher
	}{
		{name: "nil repo", realtime: stubRealtimePublisher{}},
		{name: "nil realtime", repo: &Repository{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("expected panic")
				}
			}()
			_ = NewService(tt.repo, tt.realtime)
		})
	}
}

func TestServiceRejectsInvalidUserIDBeforeDependencies(t *testing.T) {
	ctx := context.Background()
	service := NewService(
		&Repository{},
		stubRealtimePublisher{},
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
		stubRealtimePublisher{},
	)

	err := service.SendBatch(ctx, []SendParams{{
		UserID: 0,
		Type:   TypeSystem,
		Title:  "系统通知",
		Body:   "body",
	}})

	assert.ErrorIs(t, err, ErrInvalidUserID)
}

func TestServicePublishRealtimeSwallowsPublisherError(t *testing.T) {
	t.Parallel()

	service := NewService(
		&Repository{},
		stubRealtimePublisher{err: fmt.Errorf("redis unavailable")},
	)

	assert.NotPanics(t, func() {
		service.publishRealtime(context.Background(), 1, SSEEvent{Event: "unread_count"})
	})
}
