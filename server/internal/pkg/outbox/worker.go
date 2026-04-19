package outbox

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

type WorkerConfig struct {
	Name             string
	BatchSize        int
	PollInterval     time.Duration
	LockStaleAfter   time.Duration
	RetryBaseBackoff time.Duration
	MaxBackoff       time.Duration
}

type JobMeta struct {
	ID           int64
	JobType      string
	AttemptCount int
}

type ClaimFunc[T any] func(ctx context.Context, limit int, staleAfter time.Duration) ([]T, error)
type ProcessFunc[T any] func(ctx context.Context, job T) error
type MarkDoneFunc func(ctx context.Context, jobID int64) error
type MarkRetryFunc func(ctx context.Context, jobID int64, nextAttemptAt time.Time, lastError string) error
type MetaFunc[T any] func(job T) JobMeta

func RunPollingWorker[T any](
	ctx context.Context,
	cfg WorkerConfig,
	claim ClaimFunc[T],
	process ProcessFunc[T],
	markDone MarkDoneFunc,
	markRetry MarkRetryFunc,
	meta MetaFunc[T],
	truncateError func(error) string,
) {
	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		if err := ProcessBatch(ctx, cfg, claim, process, markDone, markRetry, meta, truncateError); err != nil && ctx.Err() == nil {
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
	markRetry MarkRetryFunc,
	meta MetaFunc[T],
	truncateError func(error) string,
) error {
	jobs, err := claim(ctx, cfg.BatchSize, cfg.LockStaleAfter)
	if err != nil {
		return fmt.Errorf("claim %s jobs: %w", cfg.Name, err)
	}

	for _, job := range jobs {
		if err := process(ctx, job); err != nil {
			jobMeta := meta(job)
			nextAttempt := nextAttemptAt(cfg, jobMeta.AttemptCount)
			if retryErr := markRetry(ctx, jobMeta.ID, nextAttempt, truncate(truncateError, err)); retryErr != nil {
				logger.L().Error("failed to mark "+cfg.Name+" job retry",
					zap.Int64("job_id", jobMeta.ID),
					zap.String("job_type", jobMeta.JobType),
					zap.Error(retryErr),
				)
			}
			logger.L().Warn(cfg.Name+" job failed",
				zap.Int64("job_id", jobMeta.ID),
				zap.String("job_type", jobMeta.JobType),
				zap.Int("attempt", jobMeta.AttemptCount+1),
				zap.Time("next_attempt_at", nextAttempt),
				zap.Error(err),
			)
			continue
		}

		jobMeta := meta(job)
		if err := markDone(ctx, jobMeta.ID); err != nil {
			return fmt.Errorf("mark %s job done: %w", cfg.Name, err)
		}
	}
	return nil
}

func nextAttemptAt(cfg WorkerConfig, attemptCount int) time.Time {
	backoff := time.Duration(attemptCount+1) * cfg.RetryBaseBackoff
	if cfg.MaxBackoff > 0 && backoff > cfg.MaxBackoff {
		backoff = cfg.MaxBackoff
	}
	return time.Now().Add(backoff)
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
