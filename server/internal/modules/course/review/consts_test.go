package review

import "testing"

func TestReviewConstValidators(t *testing.T) {
	for _, status := range []string{StatusPublished, StatusPendingReview, StatusHidden, StatusDeleted, StatusAll} {
		if !isValidReviewStatus(status) {
			t.Fatalf("expected valid review status: %s", status)
		}
	}
	for _, status := range []string{"", "oops"} {
		if isValidReviewStatus(status) {
			t.Fatalf("expected invalid review status: %s", status)
		}
	}

	for _, status := range []string{ReportStatusPending, ReportStatusResolved, ReportStatusRejected, StatusAll} {
		if !isValidReportStatus(status) {
			t.Fatalf("expected valid report status: %s", status)
		}
	}
	for _, status := range []string{"bad", "closed"} {
		if isValidReportStatus(status) {
			t.Fatalf("expected invalid report status: %s", status)
		}
	}

	for _, sort := range []string{SortTime, SortLikes, SortRating} {
		if !isValidSort(sort) {
			t.Fatalf("expected valid sort: %s", sort)
		}
	}
	for _, sort := range []string{"score", "recent"} {
		if isValidSort(sort) {
			t.Fatalf("expected invalid sort: %s", sort)
		}
	}
}
