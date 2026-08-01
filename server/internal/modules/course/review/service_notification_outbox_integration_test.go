package review

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestReviewNotificationOutboxCommitsAtomicallyWithBusinessTransaction(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	sender := &recordingNotificationSender{ch: make(chan ReviewNotification, 2)}
	svc := NewService(
		fixture.DB,
		repo,
		sender,
		noopReviewFGAWriter{},
		failClosedReviewAccessReader{},
	)
	ctx := context.Background()
	userID := seedUser(t, fixture, seedUserParams{
		CasdoorSubject: "notification-outbox-owner",
		UserHash:       "notification-outbox-owner-hash",
	})
	notification := ReviewNotification{
		IdempotencyKey: "review-reply:550e8400-e29b-41d4-a716-446655440996",
		UserID:         userID,
		Type:           "reply",
		Title:          "事务通知",
		Body:           "业务回滚时通知任务也必须回滚",
		SourceModule:   "review",
		SourceID:       "550e8400-e29b-41d4-a716-446655440995",
		CourseID:       1,
	}
	rollbackCause := errors.New("rollback business transaction")

	err := fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		require.NoError(t, svc.enqueueReviewNotificationTx(ctx, tx, notification))
		return rollbackCause
	})
	require.ErrorIs(t, err, rollbackCause)

	jobs, err := repo.ClaimReviewNotificationJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	assert.Empty(t, jobs)

	require.NoError(t, fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return svc.enqueueReviewNotificationTx(ctx, tx, notification)
	}))
	require.NoError(t, svc.processReviewNotificationBatch(ctx))

	delivered := waitNotification(t, sender.ch, "reply")
	assert.Equal(t, notification.IdempotencyKey, delivered.IdempotencyKey)
	assert.Equal(t, userID, delivered.UserID)

	jobs, err = repo.ClaimReviewNotificationJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	assert.Empty(t, jobs)
}
