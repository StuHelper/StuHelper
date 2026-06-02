package review

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/outbox"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestFGASyncOutbox_UpsertClaimRetryLifecycle(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	err := fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return repo.UpsertFGASyncJobTx(
			ctx,
			tx,
			fgaSyncJobTypeReviewRelations,
			reviewRelationsSyncKey("review-1"),
			[]byte(`{"reviewID":"review-1","authorUserID":"u-1","schoolID":4111010006}`),
		)
	})
	require.NoError(t, err)

	jobs, err := repo.ClaimFGASyncJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, fgaSyncJobTypeReviewRelations, jobs[0].JobType)
	assert.JSONEq(t, `{"reviewID":"review-1","authorUserID":"u-1","schoolID":4111010006}`, string(jobs[0].Payload))
	assert.Equal(t, 0, jobs[0].AttemptCount)

	err = repo.MarkFGASyncJobRetry(ctx, jobs[0].ID, time.Now().Add(-time.Second), "boom")
	require.NoError(t, err)

	reclaimed, err := repo.ClaimFGASyncJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1)
	assert.Equal(t, jobs[0].ID, reclaimed[0].ID)
	assert.Equal(t, 1, reclaimed[0].AttemptCount)
}

func TestFGASyncOutbox_UpsertConflictResetsCompletedJob(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()
	dedupeKey := reportRelationsSyncKey("report-9")

	upsert := func(payload string) {
		t.Helper()
		err := fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
			return repo.UpsertFGASyncJobTx(
				ctx,
				tx,
				fgaSyncJobTypeReportRelations,
				dedupeKey,
				[]byte(payload),
			)
		})
		require.NoError(t, err)
	}

	upsert(`{"reportID":"report-9","schoolID":4111010006}`)
	jobs, err := repo.ClaimFGASyncJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.NoError(t, repo.MarkFGASyncJobDone(ctx, jobs[0].ID))

	upsert(`{"reportID":"report-9","schoolID":4111010007}`)
	reclaimed, err := repo.ClaimFGASyncJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1)
	assert.Equal(t, jobs[0].ID, reclaimed[0].ID)
	assert.Equal(t, 0, reclaimed[0].AttemptCount)
	assert.JSONEq(t, `{"reportID":"report-9","schoolID":4111010007}`, string(reclaimed[0].Payload))
}

func TestFGASyncOutbox_ClaimIgnoresUserProfileProjectionJobs(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	err := fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := outbox.UpsertJobTx(
			ctx,
			tx,
			outbox.StreamIAMOpenFGATupleSync,
			"user_profile_projection",
			"user-profile-projection:7",
			[]byte(`{"userID":7,"approved":true}`),
		); err != nil {
			return err
		}
		return repo.UpsertFGASyncJobTx(
			ctx,
			tx,
			fgaSyncJobTypeReviewRelations,
			reviewRelationsSyncKey("review-7"),
			[]byte(`{"reviewID":"review-7","authorUserID":"u-7","schoolID":4111010006}`),
		)
	})
	require.NoError(t, err)

	jobs, err := repo.ClaimFGASyncJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, fgaSyncJobTypeReviewRelations, jobs[0].JobType)
	assertOutboxStatus(t, fixture, "user-profile-projection:7", "pending")
}

func assertOutboxStatus(t *testing.T, fixture *postgresfixture.Fixture, dedupeKey string, want string) {
	t.Helper()
	var got string
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT status FROM domain_event_outbox WHERE dedupe_key = $1
	`, dedupeKey).Scan(&got)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
