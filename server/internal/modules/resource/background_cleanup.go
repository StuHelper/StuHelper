package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

func resourceCleanupKey(resourceID int64, versionNo int) string {
	return fmt.Sprintf("resource-cleanup:%d:%d", resourceID, versionNo)
}

func (s *Service) StartBackgroundJobs(ctx context.Context, start func(string, func(context.Context))) {
	if start == nil {
		panic("resource.Service.StartBackgroundJobs: starter is required")
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
	payload, err := normalizeCleanupPayload(payload)
	if err != nil {
		return err
	}
	return s.storage.Delete(ctx, payload.MountID, payload.ObjectKey)
}

func normalizeCleanupPayload(payload cleanupPayload) (cleanupPayload, error) {
	if payload.MountID <= 0 {
		return cleanupPayload{}, fmt.Errorf("invalid resource cleanup payload: mountID must be positive")
	}
	payload.ObjectKey = strings.TrimSpace(payload.ObjectKey)
	if payload.ObjectKey == "" {
		return cleanupPayload{}, fmt.Errorf("invalid resource cleanup payload: objectKey is required")
	}
	return payload, nil
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
