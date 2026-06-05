package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

type testJob struct {
	id           int64
	jobType      string
	attemptCount int
}

func TestProcessBatch_MarksDoneOnSuccess(t *testing.T) {
	t.Parallel()

	var doneID int64
	err := ProcessBatch(
		context.Background(),
		WorkerConfig{Name: "test worker", BatchSize: 10, LockStaleAfter: time.Minute, RetryBaseBackoff: time.Second, MaxBackoff: time.Minute},
		func(context.Context, int, time.Duration) ([]testJob, error) {
			return []testJob{{id: 1, jobType: "sync", attemptCount: 0}}, nil
		},
		func(context.Context, testJob) error { return nil },
		func(_ context.Context, jobID int64, _ time.Time) error {
			doneID = jobID
			return nil
		},
		func(context.Context, int64, time.Time, time.Time, string, bool) error {
			return errors.New("should not retry")
		},
		func(job testJob) JobMeta {
			return JobMeta{ID: job.id, JobType: job.jobType, AttemptCount: job.attemptCount}
		},
		nil,
	)
	require.NoError(t, err)
	assert.EqualValues(t, 1, doneID)
}

func TestProcessBatch_MarkDoneSurvivesParentCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var markDoneErr error
	err := ProcessBatch(
		ctx,
		WorkerConfig{Name: "test worker", BatchSize: 10, LockStaleAfter: time.Minute, RetryBaseBackoff: time.Second, MaxBackoff: time.Minute},
		func(context.Context, int, time.Duration) ([]testJob, error) {
			return []testJob{{id: 1, jobType: "sync", attemptCount: 0}}, nil
		},
		func(context.Context, testJob) error {
			cancel()
			return nil
		},
		func(ctx context.Context, _ int64, _ time.Time) error {
			markDoneErr = ctx.Err()
			return markDoneErr
		},
		func(context.Context, int64, time.Time, time.Time, string, bool) error {
			return errors.New("should not retry")
		},
		func(job testJob) JobMeta {
			return JobMeta{ID: job.id, JobType: job.jobType, AttemptCount: job.attemptCount}
		},
		nil,
	)
	require.NoError(t, err)
	assert.NoError(t, markDoneErr)
}

func TestProcessBatch_MarksRetryOnFailure(t *testing.T) {
	t.Parallel()

	var (
		retryID    int64
		retryError string
	)
	err := ProcessBatch(
		context.Background(),
		WorkerConfig{Name: "test worker", BatchSize: 10, LockStaleAfter: time.Minute, RetryBaseBackoff: time.Second, MaxBackoff: time.Minute},
		func(context.Context, int, time.Duration) ([]testJob, error) {
			return []testJob{{id: 7, jobType: "sync", attemptCount: 2}}, nil
		},
		func(context.Context, testJob) error { return errors.New("boom") },
		func(context.Context, int64, time.Time) error {
			return errors.New("should not mark done")
		},
		func(_ context.Context, jobID int64, _ time.Time, _ time.Time, lastError string, terminal bool) error {
			retryID = jobID
			retryError = lastError
			assert.False(t, terminal)
			return nil
		},
		func(job testJob) JobMeta {
			return JobMeta{ID: job.id, JobType: job.jobType, AttemptCount: job.attemptCount}
		},
		func(err error) string { return "trimmed:" + err.Error() },
	)
	require.NoError(t, err)
	assert.EqualValues(t, 7, retryID)
	assert.Equal(t, "trimmed:boom", retryError)
}

func TestProcessBatch_MarkFailureSurvivesParentCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var markFailureErr error
	err := ProcessBatch(
		ctx,
		WorkerConfig{Name: "test worker", BatchSize: 10, LockStaleAfter: time.Minute, RetryBaseBackoff: time.Second, MaxBackoff: time.Minute},
		func(context.Context, int, time.Duration) ([]testJob, error) {
			return []testJob{{id: 7, jobType: "sync", attemptCount: 0}}, nil
		},
		func(context.Context, testJob) error {
			cancel()
			return errors.New("boom")
		},
		func(context.Context, int64, time.Time) error {
			return errors.New("should not mark done")
		},
		func(ctx context.Context, _ int64, _ time.Time, _ time.Time, _ string, _ bool) error {
			markFailureErr = ctx.Err()
			return markFailureErr
		},
		func(job testJob) JobMeta {
			return JobMeta{ID: job.id, JobType: job.jobType, AttemptCount: job.attemptCount}
		},
		nil,
	)
	require.NoError(t, err)
	assert.NoError(t, markFailureErr)
}

func TestProcessBatch_ContinuesAfterMarkDoneFailure(t *testing.T) {
	t.Parallel()

	doneIDs := make([]int64, 0, 3)
	err := ProcessBatch(
		context.Background(),
		WorkerConfig{Name: "test worker", BatchSize: 10, LockStaleAfter: time.Minute, RetryBaseBackoff: time.Second, MaxBackoff: time.Minute},
		func(context.Context, int, time.Duration) ([]testJob, error) {
			return []testJob{
				{id: 1, jobType: "sync", attemptCount: 0},
				{id: 2, jobType: "sync", attemptCount: 0},
				{id: 3, jobType: "sync", attemptCount: 0},
			}, nil
		},
		func(context.Context, testJob) error { return nil },
		func(_ context.Context, jobID int64, _ time.Time) error {
			doneIDs = append(doneIDs, jobID)
			if jobID == 2 {
				return errors.New("mark done failed")
			}
			return nil
		},
		func(context.Context, int64, time.Time, time.Time, string, bool) error {
			return errors.New("should not retry")
		},
		func(job testJob) JobMeta {
			return JobMeta{ID: job.id, JobType: job.jobType, AttemptCount: job.attemptCount}
		},
		nil,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mark test worker job done")
	assert.Equal(t, []int64{1, 2, 3}, doneIDs)
}

func TestProcessBatch_MarksDeadLetterAfterMaxAttempts(t *testing.T) {
	t.Parallel()

	var (
		nextRetry time.Time
		terminal  bool
	)
	err := ProcessBatch(
		context.Background(),
		WorkerConfig{
			Name:             "test worker",
			BatchSize:        10,
			LockStaleAfter:   time.Minute,
			RetryBaseBackoff: time.Second,
			MaxBackoff:       time.Minute,
			MaxAttempts:      3,
		},
		func(context.Context, int, time.Duration) ([]testJob, error) {
			return []testJob{{id: 7, jobType: "sync", attemptCount: 2}}, nil
		},
		func(context.Context, testJob) error { return errors.New("boom") },
		func(context.Context, int64, time.Time) error { return errors.New("should not mark done") },
		func(_ context.Context, _ int64, _ time.Time, value time.Time, _ string, terminalValue bool) error {
			nextRetry = value
			terminal = terminalValue
			return nil
		},
		func(job testJob) JobMeta {
			return JobMeta{ID: job.id, JobType: job.jobType, AttemptCount: job.attemptCount}
		},
		nil,
	)
	require.NoError(t, err)
	assert.True(t, terminal)
	assert.True(t, nextRetry.IsZero())
}

func TestProcessBatch_RecordsTerminalFailureMetric(t *testing.T) {
	const (
		workerName = "metric terminal worker"
		jobType    = "metric_terminal_sync"
	)
	before := testutil.ToFloat64(metrics.OutboxJobFailuresTotal.WithLabelValues(workerName, jobType, "true"))
	err := ProcessBatch(
		context.Background(),
		WorkerConfig{
			Name:             workerName,
			BatchSize:        10,
			LockStaleAfter:   time.Minute,
			RetryBaseBackoff: time.Second,
			MaxAttempts:      1,
		},
		func(context.Context, int, time.Duration) ([]testJob, error) {
			return []testJob{{id: 9, jobType: jobType, attemptCount: 0}}, nil
		},
		func(context.Context, testJob) error { return errors.New("boom") },
		func(context.Context, int64, time.Time) error { return errors.New("should not mark done") },
		func(context.Context, int64, time.Time, time.Time, string, bool) error { return nil },
		func(job testJob) JobMeta {
			return JobMeta{ID: job.id, JobType: job.jobType, AttemptCount: job.attemptCount}
		},
		nil,
	)
	require.NoError(t, err)
	after := testutil.ToFloat64(metrics.OutboxJobFailuresTotal.WithLabelValues(workerName, jobType, "true"))
	assert.Equal(t, before+1, after)
}

func TestNextAttemptAt_UsesExponentiallyJitteredBackoff(t *testing.T) {
	t.Parallel()

	start := time.Now()
	nextRetry := nextAttemptAt(WorkerConfig{
		RetryBaseBackoff: time.Second,
		MaxBackoff:       10 * time.Second,
	}, 2)
	delay := nextRetry.Sub(start)

	assert.GreaterOrEqual(t, delay, 2*time.Second)
	assert.LessOrEqual(t, delay, 4*time.Second)
}
