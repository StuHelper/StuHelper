package outbox

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestRepository_MarkJobFailureTerminalWritesDeadLetter(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	jobID := upsertAndClaimOutboxJob(t, fixture, ctx, "dead-letter-1")

	err := MarkJobFailure(ctx, fixture.DB, jobID, time.Time{}, "permanent failure", true)
	require.NoError(t, err)

	var status string
	var attemptCount int
	var lastError string
	err = fixture.Pool.QueryRow(ctx, `
		SELECT status, attempt_count, last_error
		FROM domain_event_outbox
		WHERE id = $1
	`, jobID).Scan(&status, &attemptCount, &lastError)
	require.NoError(t, err)
	assert.Equal(t, StatusDeadLetter, status)
	assert.Equal(t, 1, attemptCount)
	assert.Equal(t, "permanent failure", lastError)

	reclaimed, err := ClaimJobs(ctx, fixture.DB, "test_outbox", 10, time.Minute)
	require.NoError(t, err)
	assert.Empty(t, reclaimed)
}

func TestRepository_UpsertResetsDeadLetterJob(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	jobID := upsertAndClaimOutboxJob(t, fixture, ctx, "dead-letter-reset")
	require.NoError(t, MarkJobFailure(ctx, fixture.DB, jobID, time.Time{}, "permanent failure", true))

	require.NoError(t, fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return UpsertJobTx(ctx, tx, "test_outbox", "sync", "dead-letter-reset", []byte(`{"version":2}`))
	}))

	reclaimed, err := ClaimJobs(ctx, fixture.DB, "test_outbox", 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1)
	assert.Equal(t, jobID, reclaimed[0].ID)
	assert.Equal(t, 0, reclaimed[0].AttemptCount)
	assert.JSONEq(t, `{"version":2}`, string(reclaimed[0].Payload))
}

func TestRepository_RequeueDeadLetterJob(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	jobID := upsertAndClaimOutboxJob(t, fixture, ctx, "dead-letter-requeue")
	require.NoError(t, MarkJobFailure(ctx, fixture.DB, jobID, time.Time{}, "permanent failure", true))

	requeued, err := RequeueDeadLetterJob(ctx, fixture.DB, jobID)
	require.NoError(t, err)
	assert.True(t, requeued)

	jobs, err := ClaimJobs(ctx, fixture.DB, "test_outbox", 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, jobID, jobs[0].ID)
	assert.Equal(t, 0, jobs[0].AttemptCount)

	requeued, err = RequeueDeadLetterJob(ctx, fixture.DB, jobID)
	require.NoError(t, err)
	assert.False(t, requeued)
}

func upsertAndClaimOutboxJob(t *testing.T, fixture *postgresfixture.Fixture, ctx context.Context, dedupeKey string) int64 {
	t.Helper()
	require.NoError(t, fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return UpsertJobTx(ctx, tx, "test_outbox", "sync", dedupeKey, []byte(`{"version":1}`))
	}))
	jobs, err := ClaimJobs(ctx, fixture.DB, "test_outbox", 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	return jobs[0].ID
}
