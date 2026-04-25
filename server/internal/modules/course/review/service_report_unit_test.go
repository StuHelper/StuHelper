package review

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportReview_PreflightValidation(t *testing.T) {
	svc := &Service{}

	_, err := svc.ReportReview(context.Background(), ReportReviewParams{Reason: "   "})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidAction)

	_, err = svc.ReportReview(context.Background(), ReportReviewParams{Reason: "spam", Description: `<script>alert(1)</script>`})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDangerousContent)
}

func TestProcessReport_InvalidAction(t *testing.T) {
	svc := &Service{}
	err := svc.ProcessReport(context.Background(), ProcessReportParams{Action: "archive"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidAction)
}
