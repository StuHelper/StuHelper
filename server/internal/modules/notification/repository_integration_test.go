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
