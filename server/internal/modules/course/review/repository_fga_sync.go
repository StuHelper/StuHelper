package review

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/outbox"
)

const fgaSyncOutboxStream = "review_fga_sync"

func (r *Repository) UpsertFGASyncJobTx(ctx context.Context, tx pgx.Tx, jobType, dedupeKey string, payload []byte) error {
	if err := outbox.UpsertJobTx(ctx, tx, fgaSyncOutboxStream, jobType, dedupeKey, payload); err != nil {
		return fmt.Errorf("UpsertFGASyncJobTx: %w", err)
	}
	return nil
}

func (r *Repository) ClaimFGASyncJobs(ctx context.Context, limit int, staleAfter time.Duration) ([]FGASyncJob, error) {
	jobs, err := outbox.ClaimJobs(ctx, r.db, fgaSyncOutboxStream, limit, staleAfter)
	if err != nil {
		return nil, fmt.Errorf("ClaimFGASyncJobs: %w", err)
	}
	return mapFGASyncJobs(jobs), nil
}

func (r *Repository) MarkFGASyncJobDone(ctx context.Context, jobID int64) error {
	if err := outbox.MarkJobDone(ctx, r.db, jobID); err != nil {
		return fmt.Errorf("MarkFGASyncJobDone: %w", err)
	}
	return nil
}

func (r *Repository) MarkFGASyncJobRetry(ctx context.Context, jobID int64, nextAttemptAt time.Time, lastError string) error {
	if err := outbox.MarkJobRetry(ctx, r.db, jobID, nextAttemptAt, lastError); err != nil {
		return fmt.Errorf("MarkFGASyncJobRetry: %w", err)
	}
	return nil
}

func mapFGASyncJobs(jobs []outbox.Job) []FGASyncJob {
	items := make([]FGASyncJob, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, FGASyncJob{
			ID:           job.ID,
			JobType:      job.JobType,
			Payload:      append([]byte(nil), job.Payload...),
			AttemptCount: job.AttemptCount,
		})
	}
	return items
}
