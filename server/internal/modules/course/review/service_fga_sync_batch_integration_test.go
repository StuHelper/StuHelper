package review

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/outbox"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

type flakyReviewFGAWriter struct {
	failures int
	calls    int
}

func (f *flakyReviewFGAWriter) WriteReviewRelations(_ context.Context, _, _, _ string) error {
	f.calls++
	if f.calls <= f.failures {
		return errors.New("transient review fga failure")
	}
	return nil
}

func (f *flakyReviewFGAWriter) WriteReportRelations(_ context.Context, _, _ string) error {
	f.calls++
	if f.calls <= f.failures {
		return errors.New("transient report fga failure")
	}
	return nil
}

func TestReviewService_ProcessFGASyncBatchLifecycle(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	writer := &flakyReviewFGAWriter{failures: 1}
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, writer, failClosedReviewAccessReader{})
	ctx := context.Background()

	require.NoError(t, fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		payload, err := json.Marshal(reviewRelationsSyncPayload{
			ReviewID:     "review-sync-1",
			AuthorUserID: "user-sync-1",
			SchoolID:     4111010006,
		})
		if err != nil {
			return err
		}
		return repo.UpsertFGASyncJobTx(ctx, tx, fgaSyncJobTypeReviewRelations, reviewRelationsSyncKey("review-sync-1"), payload)
	}))

	require.NoError(t, svc.processFGASyncBatch(ctx))

	var attemptCount int
	var lastError string
	err := fixture.Pool.QueryRow(ctx, `SELECT attempt_count, last_error FROM domain_event_outbox WHERE stream = $1 AND dedupe_key = $2`, outbox.StreamIAMOpenFGATupleSync, reviewRelationsSyncKey("review-sync-1")).Scan(&attemptCount, &lastError)
	require.NoError(t, err)
	assert.Equal(t, 1, attemptCount)
	assert.Contains(t, lastError, "transient review fga failure")

	_, err = fixture.Pool.Exec(ctx, `UPDATE domain_event_outbox SET available_at = NOW() - INTERVAL '1 second' WHERE stream = $1 AND dedupe_key = $2`, outbox.StreamIAMOpenFGATupleSync, reviewRelationsSyncKey("review-sync-1"))
	require.NoError(t, err)

	require.NoError(t, svc.processFGASyncBatch(ctx))

	var remaining int
	err = fixture.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM domain_event_outbox WHERE stream = $1 AND dedupe_key = $2 AND status <> 'completed'`, outbox.StreamIAMOpenFGATupleSync, reviewRelationsSyncKey("review-sync-1")).Scan(&remaining)
	require.NoError(t, err)
	assert.Equal(t, 0, remaining)
	assert.Equal(t, 2, writer.calls)
}

func TestReviewService_ReconcileFGARelationProjectionsRequeuesOutbox(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	ctx := context.Background()

	schoolID := int64(4111010006)
	departmentID := seedDepartment(t, fixture, schoolID, "FGA 对账学院")
	teacherID := seedTeacher(t, fixture, schoolID, "FGA 对账老师", departmentID)
	courseID := seedCourse(t, fixture, schoolID, departmentID, "FGA 对账课程")
	authorID := seedUser(t, fixture, seedUserParams{CasdoorSubject: "fga-author", UserHash: "hash-fga-author"})
	reviewID := "review-fga-reconcile-1"
	reportID := "report-fga-reconcile-1"
	seedReviewWithRatings(t, fixture, reviewID, courseID, teacherID, "hash-fga-author", 4.5, StatusPublished, ReviewRatings{"teaching": 5}, "FGA 标题", "FGA 内容")
	_, err := fixture.Pool.Exec(ctx, `
		INSERT INTO review_reports (id, review_id, school_id, reporter_hash, reason, description, status, created_at)
		VALUES ($1, $2, $3, $4, 'spam', '需要处理', 'pending', NOW())
	`, reportID, reviewID, schoolID, "hash-fga-reporter")
	require.NoError(t, err)

	requeued, err := svc.ReconcileFGARelationProjections(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, requeued)
	assertFGASyncPayload(t, fixture, reviewRelationsSyncKey(reviewID), fgaSyncJobTypeReviewRelations, map[string]any{
		"reviewID":     reviewID,
		"authorUserID": fmt.Sprint(authorID),
		"schoolID":     json.Number(fmt.Sprint(schoolID)),
	})
	assertFGASyncPayload(t, fixture, reportRelationsSyncKey(reportID), fgaSyncJobTypeReportRelations, map[string]any{
		"reportID": reportID,
		"schoolID": json.Number(fmt.Sprint(schoolID)),
	})
}

func TestReviewService_ReconcileFGARelationProjectionsRequeuesReviewWithoutAuthor(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	ctx := context.Background()

	schoolID := int64(4111010006)
	departmentID := seedDepartment(t, fixture, schoolID, "FGA 历史学院")
	teacherID := seedTeacher(t, fixture, schoolID, "FGA 历史老师", departmentID)
	courseID := seedCourse(t, fixture, schoolID, departmentID, "FGA 历史课程")
	reviewID := "review-fga-orphan-author"
	seedReviewWithRatings(t, fixture, reviewID, courseID, teacherID, "missing-author-hash", 4.5, StatusPublished, ReviewRatings{"teaching": 5}, "历史标题", "历史内容")

	requeued, err := svc.ReconcileFGARelationProjections(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, requeued)
	assertFGASyncPayload(t, fixture, reviewRelationsSyncKey(reviewID), fgaSyncJobTypeReviewRelations, map[string]any{
		"reviewID": reviewID,
		"schoolID": json.Number(fmt.Sprint(schoolID)),
	})
}

func TestReviewService_ReconcileFGARelationProjectionsStopsAboveThreshold(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	ctx := context.Background()
	before := testutil.ToFloat64(metrics.IAMDriftReconciliationThresholdExceededTotal.WithLabelValues("openfga_relation"))

	schoolID := int64(4111010006)
	departmentID := seedDepartment(t, fixture, schoolID, "FGA 阈值学院")
	teacherID := seedTeacher(t, fixture, schoolID, "FGA 阈值老师", departmentID)
	courseID := seedCourse(t, fixture, schoolID, departmentID, "FGA 阈值课程")
	seedUser(t, fixture, seedUserParams{CasdoorSubject: "fga-threshold-a", UserHash: "hash-threshold-a"})
	seedUser(t, fixture, seedUserParams{CasdoorSubject: "fga-threshold-b", UserHash: "hash-threshold-b"})
	seedReviewWithRatings(t, fixture, "review-threshold-a", courseID, teacherID, "hash-threshold-a", 4.5, StatusPublished, ReviewRatings{"teaching": 5}, "阈值 A", "阈值内容 A")
	seedReviewWithRatings(t, fixture, "review-threshold-b", courseID, teacherID, "hash-threshold-b", 4.5, StatusPublished, ReviewRatings{"teaching": 5}, "阈值 B", "阈值内容 B")

	requeued, err := svc.ReconcileFGARelationProjections(ctx, 1)
	require.ErrorIs(t, err, ErrFGAReconciliationThresholdExceeded)
	assert.Equal(t, 0, requeued)
	var queued int
	err = fixture.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM domain_event_outbox WHERE stream = $1`, outbox.StreamIAMOpenFGATupleSync).Scan(&queued)
	require.NoError(t, err)
	assert.Equal(t, 0, queued)
	after := testutil.ToFloat64(metrics.IAMDriftReconciliationThresholdExceededTotal.WithLabelValues("openfga_relation"))
	assert.Equal(t, before+1, after)
}

func TestNextFGASyncReconciliationDelay(t *testing.T) {
	location := time.FixedZone("test", 8*60*60)
	cases := []struct {
		name string
		now  time.Time
		want time.Duration
	}{
		{name: "before window", now: time.Date(2026, 5, 2, 2, 30, 0, 0, location), want: 30 * time.Minute},
		{name: "at window", now: time.Date(2026, 5, 2, 3, 0, 0, 0, location), want: 24 * time.Hour},
		{name: "after window", now: time.Date(2026, 5, 2, 4, 0, 0, 0, location), want: 23 * time.Hour},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, nextFGASyncReconciliationDelay(tc.now))
		})
	}
}

func assertFGASyncPayload(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	dedupeKey string,
	jobType string,
	want map[string]any,
) {
	t.Helper()
	var gotType string
	var payload []byte
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT job_type, payload
		FROM domain_event_outbox
		WHERE stream = $1 AND dedupe_key = $2
	`, outbox.StreamIAMOpenFGATupleSync, dedupeKey).Scan(&gotType, &payload)
	require.NoError(t, err)
	assert.Equal(t, jobType, gotType)
	var got map[string]any
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	require.NoError(t, decoder.Decode(&got))
	assert.Equal(t, want, got)
}
