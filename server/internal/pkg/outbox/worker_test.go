package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		func(_ context.Context, jobID int64) error {
			doneID = jobID
			return nil
		},
		func(context.Context, int64, time.Time, string) error {
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
		func(context.Context, int64) error {
			return errors.New("should not mark done")
		},
		func(_ context.Context, jobID int64, _ time.Time, lastError string) error {
			retryID = jobID
			retryError = lastError
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

func TestProcessBatch_SchedulesLongFailedAfterMaxAttempts(t *testing.T) {
	t.Parallel()

	var nextRetry time.Time
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
		func(context.Context, int64) error { return errors.New("should not mark done") },
		func(_ context.Context, _ int64, value time.Time, _ string) error {
			nextRetry = value
			return nil
		},
		func(job testJob) JobMeta {
			return JobMeta{ID: job.id, JobType: job.jobType, AttemptCount: job.attemptCount}
		},
		nil,
	)
	require.NoError(t, err)
	assert.True(t, nextRetry.After(time.Now().Add(99*365*24*time.Hour)))
}
