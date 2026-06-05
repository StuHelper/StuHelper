package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/outbox"
)

const (
	resourceCleanupOutboxStream = "resource_cleanup"
	resourceCleanupJobType      = "delete_object"

	resourceCleanupBatchSize      = 16
	resourceCleanupPollInterval   = 2 * time.Second
	resourceCleanupLockStaleAfter = 2 * time.Minute
	resourceCleanupMaxBackoff     = 5 * time.Minute
)

type cleanupJob struct {
	ID           int64
	JobType      string
	Payload      json.RawMessage
	AttemptCount int
	LockedAt     time.Time
}

type cleanupPayload struct {
	MountID   int64  `json:"mountID"`
	ObjectKey string `json:"objectKey"`
}

func resourceCleanupKey(resourceID int64) string {
	return fmt.Sprintf("resource-cleanup:%d", resourceID)
}

func (s *Service) StartBackgroundJobs(ctx context.Context, start func(string, func(context.Context))) {
	if start == nil {
		go s.runCleanupWorker(ctx)
		return
	}
	start("resource cleanup worker", s.runCleanupWorker)
}

func (s *Service) runCleanupWorker(ctx context.Context) {
	outbox.RunPollingWorker(
		ctx,
		outbox.WorkerConfig{
			Name:             "resource cleanup",
			BatchSize:        resourceCleanupBatchSize,
			PollInterval:     resourceCleanupPollInterval,
			LockStaleAfter:   resourceCleanupLockStaleAfter,
			RetryBaseBackoff: 5 * time.Second,
			MaxBackoff:       resourceCleanupMaxBackoff,
		},
		s.repo.ClaimCleanupJobs,
		s.processCleanupJob,
		s.repo.MarkCleanupJobDone,
		s.repo.MarkCleanupJobFailure,
		func(job cleanupJob) outbox.JobMeta {
			return outbox.JobMeta{ID: job.ID, JobType: job.JobType, AttemptCount: job.AttemptCount, LockedAt: job.LockedAt}
		},
		truncateCleanupError,
	)
}

func (s *Service) processCleanupBatch(ctx context.Context) error {
	return outbox.ProcessBatch(
		ctx,
		outbox.WorkerConfig{
			Name:             "resource cleanup",
			BatchSize:        resourceCleanupBatchSize,
			LockStaleAfter:   resourceCleanupLockStaleAfter,
			RetryBaseBackoff: 5 * time.Second,
			MaxBackoff:       resourceCleanupMaxBackoff,
		},
		s.repo.ClaimCleanupJobs,
		s.processCleanupJob,
		s.repo.MarkCleanupJobDone,
		s.repo.MarkCleanupJobFailure,
		func(job cleanupJob) outbox.JobMeta {
			return outbox.JobMeta{ID: job.ID, JobType: job.JobType, AttemptCount: job.AttemptCount, LockedAt: job.LockedAt}
		},
		truncateCleanupError,
	)
}

func (s *Service) processCleanupJob(ctx context.Context, job cleanupJob) error {
	if job.JobType != resourceCleanupJobType {
		return fmt.Errorf("unsupported resource cleanup job type: %s", job.JobType)
	}

	var payload cleanupPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("decode resource cleanup payload: %w", err)
	}
	return s.storage.Delete(ctx, payload.MountID, payload.ObjectKey)
}

func truncateCleanupError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if len(msg) <= 1000 {
		return msg
	}
	return msg[:1000]
}
