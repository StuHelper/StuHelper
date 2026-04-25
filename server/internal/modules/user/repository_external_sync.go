package user

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/outbox"
)

const externalSyncOutboxStream = "user_external_sync"

func (r *Repository) UpsertExternalSyncJobTx(ctx context.Context, tx pgx.Tx, jobType, dedupeKey string, payload []byte) error {
	if err := outbox.UpsertJobTx(ctx, tx, externalSyncOutboxStream, jobType, dedupeKey, payload); err != nil {
		return fmt.Errorf("UpsertExternalSyncJobTx: %w", err)
	}
	return nil
}

func (r *Repository) ClaimExternalSyncJobs(ctx context.Context, limit int, staleAfter time.Duration) ([]ExternalSyncJob, error) {
	jobs, err := outbox.ClaimJobs(ctx, r.db, externalSyncOutboxStream, limit, staleAfter)
	if err != nil {
		return nil, fmt.Errorf("ClaimExternalSyncJobs: %w", err)
	}
	return mapExternalSyncJobs(jobs), nil
}

func (r *Repository) MarkExternalSyncJobDone(ctx context.Context, jobID int64) error {
	if err := outbox.MarkJobDone(ctx, r.db, jobID); err != nil {
		return fmt.Errorf("MarkExternalSyncJobDone: %w", err)
	}
	return nil
}

func (r *Repository) MarkExternalSyncJobRetry(ctx context.Context, jobID int64, nextAttemptAt time.Time, lastError string) error {
	if err := outbox.MarkJobRetry(ctx, r.db, jobID, nextAttemptAt, lastError); err != nil {
		return fmt.Errorf("MarkExternalSyncJobRetry: %w", err)
	}
	return nil
}

func mapExternalSyncJobs(jobs []outbox.Job) []ExternalSyncJob {
	items := make([]ExternalSyncJob, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, ExternalSyncJob{
			ID:           job.ID,
			JobType:      job.JobType,
			Payload:      append([]byte(nil), job.Payload...),
			AttemptCount: job.AttemptCount,
		})
	}
	return items
}
