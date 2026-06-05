package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/course/review"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/notification"
)

type recordingReviewNotificationSender struct {
	params notification.SendParams
}

func (s *recordingReviewNotificationSender) Send(_ context.Context, params notification.SendParams) error {
	s.params = params
	return nil
}

func (s *recordingReviewNotificationSender) SendBatch(context.Context, []notification.SendParams) error {
	return nil
}

func TestReviewNotificationAdapterMapsReviewNotification(t *testing.T) {
	sender := &recordingReviewNotificationSender{}
	adapter := newReviewNotificationAdapter(sender)

	err := adapter.SendReviewNotification(context.Background(), review.ReviewNotification{
		UserID:       42,
		Type:         "reply",
		Title:        "title",
		Body:         "body",
		SourceModule: "review",
		SourceID:     "review-1",
		CourseID:     7,
	})

	require.NoError(t, err)
	require.Equal(t, notification.SendParams{
		UserID:       42,
		Type:         "reply",
		Title:        "title",
		Body:         "body",
		SourceModule: "review",
		SourceID:     "review-1",
		CourseID:     7,
	}, sender.params)
}
