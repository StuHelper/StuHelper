package outbox

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

type WorkerConfig struct {
	Name             string
	BatchSize        int
	PollInterval     time.Duration
	LockStaleAfter   time.Duration
	RetryBaseBackoff time.Duration
	MaxBackoff       time.Duration
	MaxAttempts      int
}

type JobMeta struct {
	ID           int64
	JobType      string
	AttemptCount int
	LockedAt     time.Time
}

type ClaimFunc[T any] func(ctx context.Context, limit int, staleAfter time.Duration) ([]T, error)
type ProcessFunc[T any] func(ctx context.Context, job T) error
type MarkDoneFunc func(ctx context.Context, jobID int64, lockedAt time.Time) error
type MarkFailureFunc func(
	ctx context.Context,
	jobID int64,
	lockedAt time.Time,
	nextAttemptAt time.Time,
	lastError string,
	terminal bool,
) error
type MetaFunc[T any] func(job T) JobMeta

func RunPollingWorker[T any](
	ctx context.Context,
	cfg WorkerConfig,
	claim ClaimFunc[T],
	process ProcessFunc[T],
	markDone MarkDoneFunc,
	markFailure MarkFailureFunc,
	meta MetaFunc[T],
	truncateError func(error) string,
) {
	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		if err := ProcessBatch(ctx, cfg, claim, process, markDone, markFailure, meta, truncateError); err != nil && ctx.Err() == nil {
			logger.L().Warn(cfg.Name+" batch failed", zap.Error(err))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func ProcessBatch[T any](
	ctx context.Context,
	cfg WorkerConfig,
	claim ClaimFunc[T],
	process ProcessFunc[T],
	markDone MarkDoneFunc,
	markFailure MarkFailureFunc,
	meta MetaFunc[T],
	truncateError func(error) string,
) error {
	jobs, err := claim(ctx, cfg.BatchSize, cfg.LockStaleAfter)
	if err != nil {
		return fmt.Errorf("claim %s jobs: %w", cfg.Name, err)
	}

	var batchErr error
	for _, job := range jobs {
		if err := process(ctx, job); err != nil {
			jobMeta := meta(job)
			terminalFailed := reachedMaxAttempts(cfg, jobMeta.AttemptCount)
			var nextAttempt time.Time
			if !terminalFailed {
				nextAttempt = nextAttemptAt(cfg, jobMeta.AttemptCount)
			}
			if retryErr := markFailure(ctx, jobMeta.ID, jobMeta.LockedAt, nextAttempt, truncate(truncateError, err), terminalFailed); retryErr != nil {
				logger.L().Error("failed to mark "+cfg.Name+" job failure",
					zap.Int64("job_id", jobMeta.ID),
					zap.String("job_type", jobMeta.JobType),
					zap.Bool("terminal_failed", terminalFailed),
					zap.Error(retryErr),
				)
			}
			metrics.ObserveOutboxJobFailure(cfg.Name, jobMeta.JobType, terminalFailed)
			fields := []zap.Field{
				zap.Int64("job_id", jobMeta.ID),
				zap.String("job_type", jobMeta.JobType),
				zap.Int("attempt", jobMeta.AttemptCount+1),
				zap.Bool("terminal_failed", terminalFailed),
				zap.Error(err),
			}
			if !terminalFailed {
				fields = append(fields, zap.Time("next_attempt_at", nextAttempt))
			}
			logger.L().Warn(cfg.Name+" job failed", fields...)
			continue
		}

		jobMeta := meta(job)
		if err := markDone(ctx, jobMeta.ID, jobMeta.LockedAt); err != nil {
			batchErr = errors.Join(batchErr, fmt.Errorf("mark %s job done: %w", cfg.Name, err))
			metrics.ObserveOutboxJobFailure(cfg.Name, jobMeta.JobType, false)
			logger.L().Error("failed to mark "+cfg.Name+" job done",
				zap.Int64("job_id", jobMeta.ID),
				zap.String("job_type", jobMeta.JobType),
				zap.Error(err),
			)
		}
	}
	return batchErr
}

func nextAttemptAt(cfg WorkerConfig, attemptCount int) time.Time {
	return time.Now().Add(jitteredBackoff(nextBackoffDuration(cfg, attemptCount)))
}

func nextBackoffDuration(cfg WorkerConfig, attemptCount int) time.Duration {
	backoff := cfg.RetryBaseBackoff
	if backoff <= 0 {
		return 0
	}
	for retry := 0; retry < attemptCount; retry++ {
		if cfg.MaxBackoff > 0 && backoff >= cfg.MaxBackoff {
			return cfg.MaxBackoff
		}
		backoff *= 2
	}
	if cfg.MaxBackoff > 0 && backoff > cfg.MaxBackoff {
		return cfg.MaxBackoff
	}
	return backoff
}

func jitteredBackoff(backoff time.Duration) time.Duration {
	if backoff <= 1 {
		return backoff
	}
	minDelay := backoff / 2
	jitterWindow := backoff - minDelay
	return minDelay + time.Duration(rand.Float64()*float64(jitterWindow)) // #nosec G404 -- worker backoff jitter is not security-sensitive randomness.
}

func reachedMaxAttempts(cfg WorkerConfig, attemptCount int) bool {
	return cfg.MaxAttempts > 0 && attemptCount+1 >= cfg.MaxAttempts
}

func truncate(truncateError func(error) string, err error) string {
	if err == nil {
		return ""
	}
	if truncateError == nil {
		return err.Error()
	}
	return truncateError(err)
}
