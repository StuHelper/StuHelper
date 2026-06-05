package review

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestNewService_PanicsOnMissingDeps(t *testing.T) {
	assert.Panics(t, func() {
		NewService(nil, nil, nil, nil, nil)
	})
	assert.Panics(t, func() {
		NewService(nil, &Repository{}, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	})
	assert.Panics(t, func() {
		NewService(nil, nil, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	})
	assert.Panics(t, func() {
		NewService(nil, &Repository{}, nil, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	})
	assert.Panics(t, func() {
		NewService(nil, &Repository{}, noopNotificationSender{}, nil, failClosedReviewAccessReader{})
	})
	assert.Panics(t, func() {
		NewService(nil, &Repository{}, noopNotificationSender{}, noopReviewFGAWriter{}, nil)
	})
}

func TestNewService_InitialCacheContextHonorsCancellation(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)

	warmed := NewService(
		fixture.DB,
		repo,
		noopNotificationSender{},
		noopReviewFGAWriter{},
		failClosedReviewAccessReader{},
		WithInitialCacheContext(context.Background()),
	)
	assert.Contains(t, warmed.getDimensionNames(), "teaching")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	canceled := NewService(
		fixture.DB,
		repo,
		noopNotificationSender{},
		noopReviewFGAWriter{},
		failClosedReviewAccessReader{},
		WithInitialCacheContext(ctx),
	)
	assert.Nil(t, canceled.getDimensionNames())
}
