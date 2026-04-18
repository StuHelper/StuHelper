package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
