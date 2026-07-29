package notification

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestRepositoryListFormatsCreatedAt(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	userID := seedNotificationUser(t, fixture, "notif-list-created-at-user")

	notifID, err := repo.Create(ctx, CreateParams{
		UserID:       userID,
		Type:         "reply",
		Title:        "通知时间覆盖",
		Body:         "created_at should be returned as RFC3339",
		SourceModule: "review",
		SourceID:     "review-1",
	})
	require.NoError(t, err)

	result, err := repo.List(ctx, ListParams{UserID: userID, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, result.List, 1)

	got := result.List[0]
	assert.Equal(t, notifID, got.ID)
	assert.Equal(t, got.Body, got.Content)
	_, err = time.Parse(time.RFC3339Nano, got.CreatedAt)
	require.NoError(t, err)
}

func TestRepositoryListClampsInvalidPagination(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	userID := seedNotificationUser(t, fixture, "notif-list-pagination-user")

	notifID, err := repo.Create(ctx, CreateParams{
		UserID:       userID,
		Type:         "reply",
		Title:        "通知分页覆盖",
		Body:         "invalid page size should be clamped",
		SourceModule: "review",
		SourceID:     "review-1",
	})
	require.NoError(t, err)

	result, err := repo.List(ctx, ListParams{UserID: userID, Page: 0, PageSize: -1})
	require.NoError(t, err)
	require.Len(t, result.List, 1)
	assert.Equal(t, notifID, result.List[0].ID)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, 1, result.Unread)
}

func TestServiceSendIsIdempotentAndSuppressesDuplicateRealtimeDelivery(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := postgresfixture.Start(t)
	publisher := &recordingRealtimePublisher{}
	service := NewService(NewRepository(fixture.DB), publisher)
	userID := seedNotificationUser(t, fixture, "notif-idempotent-user")
	params := SendParams{
		IdempotencyKey: "review-reply:550e8400-e29b-41d4-a716-446655440998",
		UserID:         userID,
		Type:           "reply",
		Title:          "幂等通知",
		Body:           "同一业务事件只能形成一条通知",
		SourceModule:   "review",
		SourceID:       "550e8400-e29b-41d4-a716-446655440997",
	}

	require.NoError(t, service.Send(ctx, params))
	require.NoError(t, service.Send(ctx, params))

	result, err := service.List(ctx, userID, 1, 10)
	require.NoError(t, err)
	require.Len(t, result.List, 1)
	assert.Len(t, publisher.snapshot(), 1)
}
