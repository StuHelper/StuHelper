package review

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

type flakyReviewFGAWriter struct {
	failures int
	calls    int
}

func (f *flakyReviewFGAWriter) WriteReviewRelations(_ context.Context, _, _, _, _ string) error {
	f.calls++
	if f.calls <= f.failures {
		return errors.New("transient review fga failure")
	}
	return nil
}

func (f *flakyReviewFGAWriter) WriteReportRelations(_ context.Context, _, _, _, _ string) error {
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
			CourseID:     42,
			SchoolID:     10006,
		})
		if err != nil {
			return err
		}
		return repo.UpsertFGASyncJobTx(ctx, tx, fgaSyncJobTypeReviewRelations, reviewRelationsSyncKey("review-sync-1"), payload)
	}))

	require.NoError(t, svc.processFGASyncBatch(ctx))

	var attemptCount int
	var lastError string
	err := fixture.Pool.QueryRow(ctx, `SELECT attempt_count, last_error FROM review_fga_sync_outbox WHERE dedupe_key = $1`, reviewRelationsSyncKey("review-sync-1")).Scan(&attemptCount, &lastError)
	require.NoError(t, err)
	assert.Equal(t, 1, attemptCount)
	assert.Contains(t, lastError, "transient review fga failure")

	_, err = fixture.Pool.Exec(ctx, `UPDATE review_fga_sync_outbox SET available_at = NOW() - INTERVAL '1 second' WHERE dedupe_key = $1`, reviewRelationsSyncKey("review-sync-1"))
	require.NoError(t, err)

	require.NoError(t, svc.processFGASyncBatch(ctx))

	var remaining int
	err = fixture.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM review_fga_sync_outbox WHERE dedupe_key = $1 AND status <> 'completed'`, reviewRelationsSyncKey("review-sync-1")).Scan(&remaining)
	require.NoError(t, err)
	assert.Equal(t, 0, remaining)
	assert.Equal(t, 2, writer.calls)
}
