package review

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) UpsertFGASyncJobTx(ctx context.Context, tx pgx.Tx, jobType, dedupeKey string, payload []byte) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO review_fga_sync_outbox (
			job_type,
			dedupe_key,
			payload,
			status,
			attempt_count,
			available_at,
			locked_at,
			last_error,
			created_at,
			updated_at
		) VALUES ($1, $2, $3::jsonb, 'pending', 0, NOW(), NULL, NULL, NOW(), NOW())
		ON CONFLICT (dedupe_key)
		DO UPDATE SET
			job_type = EXCLUDED.job_type,
			payload = EXCLUDED.payload,
			status = 'pending',
			attempt_count = 0,
			available_at = NOW(),
			locked_at = NULL,
			last_error = NULL,
			updated_at = NOW()
	`, jobType, dedupeKey, payload)
	if err != nil {
		return fmt.Errorf("UpsertFGASyncJobTx: %w", err)
	}
	return nil
}

func (r *Repository) ClaimFGASyncJobs(ctx context.Context, limit int, staleAfter time.Duration) ([]FGASyncJob, error) {
	if limit <= 0 {
		return nil, nil
	}

	jobs := make([]FGASyncJob, 0, limit)
	err := r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			WITH candidates AS (
				SELECT id
				FROM review_fga_sync_outbox
				WHERE (
					status = 'pending'
					OR (status = 'failed' AND available_at <= NOW())
					OR (status = 'processing' AND locked_at <= NOW() - $2::interval)
				)
				ORDER BY available_at ASC, id ASC
				LIMIT $1
				FOR UPDATE SKIP LOCKED
			)
			UPDATE review_fga_sync_outbox AS o
			SET status = 'processing',
				locked_at = NOW(),
				updated_at = NOW()
			FROM candidates
			WHERE o.id = candidates.id
			RETURNING o.id, o.job_type, o.payload, o.attempt_count
		`, limit, reviewSyncPGInterval(staleAfter))
		if err != nil {
			return fmt.Errorf("ClaimFGASyncJobs query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var job FGASyncJob
			var payload json.RawMessage
			if err := rows.Scan(&job.ID, &job.JobType, &payload, &job.AttemptCount); err != nil {
				return fmt.Errorf("ClaimFGASyncJobs scan: %w", err)
			}
			job.Payload = append(job.Payload[:0], payload...)
			jobs = append(jobs, job)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("ClaimFGASyncJobs rows: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *Repository) MarkFGASyncJobDone(ctx context.Context, jobID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE review_fga_sync_outbox
		SET status = 'completed',
			locked_at = NULL,
			last_error = NULL,
			updated_at = NOW()
		WHERE id = $1
	`, jobID)
	if err != nil {
		return fmt.Errorf("MarkFGASyncJobDone: %w", err)
	}
	return nil
}

func (r *Repository) MarkFGASyncJobRetry(ctx context.Context, jobID int64, nextAttemptAt time.Time, lastError string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE review_fga_sync_outbox
		SET status = 'failed',
			attempt_count = attempt_count + 1,
			available_at = $2,
			locked_at = NULL,
			last_error = $3,
			updated_at = NOW()
		WHERE id = $1
	`, jobID, nextAttemptAt, lastError)
	if err != nil {
		return fmt.Errorf("MarkFGASyncJobRetry: %w", err)
	}
	return nil
}

func reviewSyncPGInterval(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds <= 0 {
		seconds = 1
	}
	return fmt.Sprintf("%d seconds", seconds)
}
