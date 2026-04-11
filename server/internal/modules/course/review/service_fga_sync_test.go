package review

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type fakeReviewFGAWriter struct {
	reviewArgs []string
	reportArgs []string
	err        error
}

func (f *fakeReviewFGAWriter) WriteReviewRelations(_ context.Context, reviewID, authorUserID, courseID, schoolID string) error {
	f.reviewArgs = []string{reviewID, authorUserID, courseID, schoolID}
	return f.err
}

func (f *fakeReviewFGAWriter) WriteReportRelations(_ context.Context, reportID, reporterUserID, reviewID, schoolID string) error {
	f.reportArgs = []string{reportID, reporterUserID, reviewID, schoolID}
	return f.err
}

func TestProcessFGASyncJob_ReviewRelations(t *testing.T) {
	payload, err := json.Marshal(reviewRelationsSyncPayload{
		ReviewID:     "review-1",
		AuthorUserID: "user-1",
		CourseID:     42,
		SchoolID:     10006,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	writer := &fakeReviewFGAWriter{}
	service := &Service{fgaWriter: writer}
	err = service.processFGASyncJob(context.Background(), FGASyncJob{
		JobType: fgaSyncJobTypeReviewRelations,
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("processFGASyncJob returned error: %v", err)
	}

	want := []string{"review-1", "user-1", "42", "10006"}
	for i := range want {
		if writer.reviewArgs[i] != want[i] {
			t.Fatalf("review arg %d = %q, want %q", i, writer.reviewArgs[i], want[i])
		}
	}
}

func TestProcessFGASyncJob_ReportRelations(t *testing.T) {
	payload, err := json.Marshal(reportRelationsSyncPayload{
		ReportID:       "report-1",
		ReporterUserID: "user-9",
		ReviewID:       "review-1",
		SchoolID:       10006,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	writer := &fakeReviewFGAWriter{}
	service := &Service{fgaWriter: writer}
	err = service.processFGASyncJob(context.Background(), FGASyncJob{
		JobType: fgaSyncJobTypeReportRelations,
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("processFGASyncJob returned error: %v", err)
	}

	want := []string{"report-1", "user-9", "review-1", "10006"}
	for i := range want {
		if writer.reportArgs[i] != want[i] {
			t.Fatalf("report arg %d = %q, want %q", i, writer.reportArgs[i], want[i])
		}
	}
}

func TestProcessFGASyncJob_PropagatesWriterError(t *testing.T) {
	wantErr := errors.New("boom")
	payload, err := json.Marshal(reviewRelationsSyncPayload{
		ReviewID:     "review-1",
		AuthorUserID: "user-1",
		CourseID:     42,
		SchoolID:     10006,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	service := &Service{fgaWriter: &fakeReviewFGAWriter{err: wantErr}}
	err = service.processFGASyncJob(context.Background(), FGASyncJob{
		JobType: fgaSyncJobTypeReviewRelations,
		Payload: payload,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}
