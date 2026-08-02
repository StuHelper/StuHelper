package review

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportReview_PreflightValidation(t *testing.T) {
	svc := &Service{}

	base := ReportReviewParams{
		UserHash:               "u-reporter",
		ReporterInternalUserID: 1,
	}

	_, err := svc.ReportReview(context.Background(), ReportReviewParams{
		UserHash:               base.UserHash,
		ReporterInternalUserID: base.ReporterInternalUserID,
		Reason:                 "   ",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidAction)

	_, err = svc.ReportReview(context.Background(), ReportReviewParams{
		UserHash:               base.UserHash,
		ReporterInternalUserID: base.ReporterInternalUserID,
		Reason:                 "abuse",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidAction)

	_, err = svc.ReportReview(context.Background(), ReportReviewParams{
		UserHash:               base.UserHash,
		ReporterInternalUserID: base.ReporterInternalUserID,
		Reason:                 "spam",
		Description:            `<script>alert(1)</script>`,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDangerousContent)

	_, err = svc.ReportReview(context.Background(), ReportReviewParams{
		UserHash:               base.UserHash,
		ReporterInternalUserID: base.ReporterInternalUserID,
		Reason:                 "spam",
		Description:            strings.Repeat("界", maxReportDescriptionRunes+1),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrReportDescriptionTooLong)
}

func TestReportReviewRejectsMissingUserIdentityBeforeDependencies(t *testing.T) {
	svc := &Service{}

	_, err := svc.ReportReview(context.Background(), ReportReviewParams{
		ReporterInternalUserID: 1,
		Reason:                 "spam",
	})
	require.ErrorIs(t, err, ErrUserIdentityRequired)

	_, err = svc.ReportReview(context.Background(), ReportReviewParams{
		UserHash: "u-reporter",
		Reason:   "spam",
	})
	require.ErrorIs(t, err, ErrUserIdentityRequired)
}

func TestProcessReport_InvalidAction(t *testing.T) {
	svc := &Service{}
	err := svc.ProcessReport(context.Background(), ProcessReportParams{Action: "archive", ResolvedBy: "admin-1"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidAction)
}

func TestProcessReportRejectsMissingAdminIdentityBeforeDependencies(t *testing.T) {
	svc := &Service{}

	err := svc.ProcessReport(context.Background(), ProcessReportParams{
		Action:     "reject",
		ResolvedBy: "   ",
	})
	require.ErrorIs(t, err, ErrAdminIdentityRequired)
}
