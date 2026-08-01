package outbox

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestRepository_MarkJobFailureTerminalWritesDeadLetter(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	job := upsertAndClaimOutboxJob(t, fixture, ctx, "dead-letter-1")

	err := MarkJobFailure(ctx, fixture.DB, job.ID, job.LockedAt, time.Time{}, "permanent failure", true)
	require.NoError(t, err)

	var status string
	var attemptCount int
	var lastError string
	err = fixture.Pool.QueryRow(ctx, `
		SELECT status, attempt_count, last_error
		FROM domain_event_outbox
		WHERE id = $1
	`, job.ID).Scan(&status, &attemptCount, &lastError)
	require.NoError(t, err)
	assert.Equal(t, StatusDeadLetter, status)
	assert.Equal(t, 1, attemptCount)
	assert.Equal(t, "permanent failure", lastError)

	reclaimed, err := ClaimJobs(ctx, fixture.DB, "test_outbox", 10, time.Minute)
	require.NoError(t, err)
	assert.Empty(t, reclaimed)
}

func TestProcessBatch_PanicDeadLettersOnlyPoisonWithRepository(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	require.NoError(t, fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := UpsertJobTx(
			ctx,
			tx,
			"panic_isolation",
			"poison",
			"panic-poison",
			[]byte(`{"version":1}`),
		); err != nil {
			return err
		}
		return UpsertJobTx(
			ctx,
			tx,
			"panic_isolation",
			"healthy",
			"panic-healthy",
			[]byte(`{"version":1}`),
		)
	}))

	var processedTypes []string
	err := ProcessBatch(
		ctx,
		WorkerConfig{
			Name:           "panic isolation",
			BatchSize:      10,
			LockStaleAfter: time.Minute,
			MaxAttempts:    1,
		},
		func(ctx context.Context, limit int, staleAfter time.Duration) ([]Job, error) {
			return ClaimJobs(ctx, fixture.DB, "panic_isolation", limit, staleAfter)
		},
		func(_ context.Context, job Job) error {
			processedTypes = append(processedTypes, job.JobType)
			if job.JobType == "poison" {
				panic("repository poison")
			}
			return nil
		},
		func(ctx context.Context, jobID int64, lockedAt time.Time) error {
			return MarkJobDone(ctx, fixture.DB, jobID, lockedAt)
		},
		func(
			ctx context.Context,
			jobID int64,
			lockedAt time.Time,
			nextAttemptAt time.Time,
			lastError string,
			terminal bool,
		) error {
			return MarkJobFailure(
				ctx,
				fixture.DB,
				jobID,
				lockedAt,
				nextAttemptAt,
				lastError,
				terminal,
			)
		},
		func(job Job) JobMeta {
			return JobMeta{
				ID:           job.ID,
				JobType:      job.JobType,
				AttemptCount: job.AttemptCount,
				LockedAt:     job.LockedAt,
			}
		},
		nil,
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"poison", "healthy"}, processedTypes)

	var (
		poisonStatus      string
		poisonAttempts    int
		poisonLastError   string
		healthyStatus     string
		healthyLastError  *string
		healthyLockedAt   *time.Time
		healthyAttemptCnt int
	)
	err = fixture.Pool.QueryRow(ctx, `
		SELECT status, attempt_count, last_error
		FROM domain_event_outbox
		WHERE stream = 'panic_isolation' AND job_type = 'poison'
	`).Scan(&poisonStatus, &poisonAttempts, &poisonLastError)
	require.NoError(t, err)
	err = fixture.Pool.QueryRow(ctx, `
		SELECT status, attempt_count, last_error, locked_at
		FROM domain_event_outbox
		WHERE stream = 'panic_isolation' AND job_type = 'healthy'
	`).Scan(&healthyStatus, &healthyAttemptCnt, &healthyLastError, &healthyLockedAt)
	require.NoError(t, err)

	assert.Equal(t, StatusDeadLetter, poisonStatus)
	assert.Equal(t, 1, poisonAttempts)
	assert.Contains(t, poisonLastError, "job handler panicked: repository poison")
	assert.Contains(t, poisonLastError, "runtime/debug.Stack")
	assert.Equal(t, StatusCompleted, healthyStatus)
	assert.Equal(t, 0, healthyAttemptCnt)
	assert.Nil(t, healthyLastError)
	assert.Nil(t, healthyLockedAt)
}

func TestRepository_UpsertResetsDeadLetterJob(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	job := upsertAndClaimOutboxJob(t, fixture, ctx, "dead-letter-reset")
	require.NoError(t, MarkJobFailure(ctx, fixture.DB, job.ID, job.LockedAt, time.Time{}, "permanent failure", true))

	require.NoError(t, fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return UpsertJobTx(ctx, tx, "test_outbox", "sync", "dead-letter-reset", []byte(`{"version":2}`))
	}))

	reclaimed, err := ClaimJobs(ctx, fixture.DB, "test_outbox", 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1)
	assert.Equal(t, job.ID, reclaimed[0].ID)
	assert.Equal(t, 0, reclaimed[0].AttemptCount)
	assert.JSONEq(t, `{"version":2}`, string(reclaimed[0].Payload))
}

func TestRepository_SupersededProcessingJobFinishesBeforeLatestPayloadRuns(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	first := upsertAndClaimOutboxJob(t, fixture, ctx, "superseded-processing")

	require.NoError(t, fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return UpsertJobTx(
			ctx,
			tx,
			"test_outbox",
			"sync",
			"superseded-processing",
			[]byte(`{"version":2}`),
		)
	}))

	concurrent, err := ClaimJobs(ctx, fixture.DB, "test_outbox", 10, time.Minute)
	require.NoError(t, err)
	assert.Empty(t, concurrent, "new payload must not run concurrently with an active older revision")

	require.NoError(t, MarkJobDone(ctx, fixture.DB, first.ID, first.LockedAt))
	assertOutboxStatus(t, fixture, first.ID, StatusPending)

	latest, err := ClaimJobs(ctx, fixture.DB, "test_outbox", 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, latest, 1)
	assert.JSONEq(t, `{"version":2}`, string(latest[0].Payload))
	assert.Equal(t, 0, latest[0].AttemptCount)

	require.NoError(t, MarkJobDone(ctx, fixture.DB, latest[0].ID, latest[0].LockedAt))
	assertOutboxStatus(t, fixture, first.ID, StatusCompleted)
}

func TestRepository_SupersededProcessingFailureDoesNotDeadLetterLatestPayload(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	first := upsertAndClaimOutboxJob(t, fixture, ctx, "superseded-failure")

	require.NoError(t, fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return UpsertJobTx(
			ctx,
			tx,
			"test_outbox",
			"sync",
			"superseded-failure",
			[]byte(`{"version":2}`),
		)
	}))
	require.NoError(t, MarkJobFailure(
		ctx,
		fixture.DB,
		first.ID,
		first.LockedAt,
		time.Time{},
		"old revision failed",
		true,
	))

	var status string
	var attemptCount int
	var lastError *string
	err := fixture.Pool.QueryRow(ctx, `
		SELECT status, attempt_count, last_error
		FROM domain_event_outbox
		WHERE id = $1
	`, first.ID).Scan(&status, &attemptCount, &lastError)
	require.NoError(t, err)
	assert.Equal(t, StatusPending, status)
	assert.Equal(t, 0, attemptCount)
	assert.Nil(t, lastError)

	latest, err := ClaimJobs(ctx, fixture.DB, "test_outbox", 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, latest, 1)
	assert.JSONEq(t, `{"version":2}`, string(latest[0].Payload))
}

func TestRepository_RequeueDeadLetterJob(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	job := upsertAndClaimOutboxJob(t, fixture, ctx, "dead-letter-requeue")
	require.NoError(t, MarkJobFailure(ctx, fixture.DB, job.ID, job.LockedAt, time.Time{}, "permanent failure", true))

	requeued, err := RequeueDeadLetterJob(ctx, fixture.DB, job.ID)
	require.NoError(t, err)
	assert.True(t, requeued)

	jobs, err := ClaimJobs(ctx, fixture.DB, "test_outbox", 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, job.ID, jobs[0].ID)
	assert.Equal(t, 0, jobs[0].AttemptCount)

	requeued, err = RequeueDeadLetterJob(ctx, fixture.DB, job.ID)
	require.NoError(t, err)
	assert.False(t, requeued)
}

func TestRepository_AbandonRequeuesWithoutConsumingAttempt(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	job := upsertAndClaimOutboxJob(t, fixture, ctx, "graceful-shutdown")

	require.NoError(t, MarkJobFailure(
		ctx,
		fixture.DB,
		job.ID,
		job.LockedAt,
		time.Time{},
		"",
		false,
	))
	assertOutboxStatus(t, fixture, job.ID, StatusPending)

	reclaimed, err := ClaimJobs(ctx, fixture.DB, "test_outbox", 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1)
	assert.Equal(t, 0, reclaimed[0].AttemptCount)
}

func TestRepository_MarkDoneRejectsStaleLease(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	first := upsertAndClaimOutboxJob(t, fixture, ctx, "stale-lease")
	require.NotZero(t, first.LockedAt)
	_, err := fixture.Pool.Exec(ctx, `
		UPDATE domain_event_outbox
		SET locked_at = NOW() - INTERVAL '10 minutes'
		WHERE id = $1
	`, first.ID)
	require.NoError(t, err)

	reclaimed, err := ClaimJobs(ctx, fixture.DB, "test_outbox", 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1)
	second := reclaimed[0]
	require.Equal(t, first.ID, second.ID)
	require.NotEqual(t, first.LockedAt, second.LockedAt)

	err = MarkJobDone(ctx, fixture.DB, first.ID, first.LockedAt)
	require.ErrorIs(t, err, ErrJobLockLost)
	assertOutboxStatus(t, fixture, first.ID, StatusProcessing)

	require.NoError(t, MarkJobDone(ctx, fixture.DB, second.ID, second.LockedAt))
	assertOutboxStatus(t, fixture, first.ID, StatusCompleted)
}

func upsertAndClaimOutboxJob(t *testing.T, fixture *postgresfixture.Fixture, ctx context.Context, dedupeKey string) Job {
	t.Helper()
	require.NoError(t, fixture.DB.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return UpsertJobTx(ctx, tx, "test_outbox", "sync", dedupeKey, []byte(`{"version":1}`))
	}))
	jobs, err := ClaimJobs(ctx, fixture.DB, "test_outbox", 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	return jobs[0]
}

func assertOutboxStatus(t *testing.T, fixture *postgresfixture.Fixture, jobID int64, expected string) {
	t.Helper()
	var status string
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT status
		FROM domain_event_outbox
		WHERE id = $1
	`, jobID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, expected, status)
}
