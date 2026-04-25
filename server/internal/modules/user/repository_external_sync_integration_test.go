package user

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestExternalSyncOutbox_UpsertClaimRetryLifecycle(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-hmac-key"))
	ctx := context.Background()

	err := repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return repo.UpsertExternalSyncJobTx(
			ctx,
			tx,
			externalSyncJobTypeVerifiedStudentRole,
			verifiedStudentRoleSyncKey(42),
			[]byte(`{"userID":42,"approved":true}`),
		)
	})
	require.NoError(t, err)

	jobs, err := repo.ClaimExternalSyncJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, externalSyncJobTypeVerifiedStudentRole, jobs[0].JobType)
	assert.JSONEq(t, `{"userID":42,"approved":true}`, string(jobs[0].Payload))
	assert.Equal(t, 0, jobs[0].AttemptCount)

	err = repo.MarkExternalSyncJobRetry(ctx, jobs[0].ID, time.Now().Add(-time.Second), "boom")
	require.NoError(t, err)

	reclaimed, err := repo.ClaimExternalSyncJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1)
	assert.Equal(t, jobs[0].ID, reclaimed[0].ID)
	assert.Equal(t, 1, reclaimed[0].AttemptCount)
}

func TestExternalSyncOutbox_UpsertConflictResetsCompletedJob(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB, []byte("test-hmac-key"))
	ctx := context.Background()
	dedupeKey := userProfileProjectionKey(7)

	upsert := func(payload string) {
		t.Helper()
		err := repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
			return repo.UpsertExternalSyncJobTx(
				ctx,
				tx,
				externalSyncJobTypeUserProfileProjection,
				dedupeKey,
				[]byte(payload),
			)
		})
		require.NoError(t, err)
	}

	upsert(`{"userID":7,"approved":false}`)
	jobs, err := repo.ClaimExternalSyncJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.NoError(t, repo.MarkExternalSyncJobDone(ctx, jobs[0].ID))

	upsert(`{"userID":7,"approved":true}`)
	reclaimed, err := repo.ClaimExternalSyncJobs(ctx, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1)
	assert.Equal(t, jobs[0].ID, reclaimed[0].ID)
	assert.Equal(t, 0, reclaimed[0].AttemptCount)
	assert.JSONEq(t, `{"userID":7,"approved":true}`, string(reclaimed[0].Payload))
}
