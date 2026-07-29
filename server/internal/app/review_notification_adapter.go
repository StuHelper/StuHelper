package app

import (
	"context"

	"github.com/StuHelper/StuHelper/server/internal/modules/course/review"
	"github.com/StuHelper/StuHelper/server/internal/modules/notification"
)

type reviewNotificationAdapter struct {
	sender notification.Sender
}

func newReviewNotificationAdapter(sender notification.Sender) reviewNotificationAdapter {
	if sender == nil {
		panic("app.newReviewNotificationAdapter: sender must not be nil")
	}
	return reviewNotificationAdapter{sender: sender}
}

func (a reviewNotificationAdapter) SendReviewNotification(ctx context.Context, params review.ReviewNotification) error {
	return a.sender.Send(ctx, notification.SendParams{
		UserID:       params.UserID,
		Type:         params.Type,
		Title:        params.Title,
		Body:         params.Body,
		SourceModule: params.SourceModule,
		SourceID:     params.SourceID,
		CourseID:     params.CourseID,
	})
}
