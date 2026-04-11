package user

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) UpsertExternalSyncJobTx(ctx context.Context, tx pgx.Tx, jobType, dedupeKey string, payload []byte) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO user_external_sync_outbox (
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
		return fmt.Errorf("UpsertExternalSyncJobTx: %w", err)
	}
	return nil
}

func (r *Repository) ClaimExternalSyncJobs(ctx context.Context, limit int, staleAfter time.Duration) ([]ExternalSyncJob, error) {
	if limit <= 0 {
		return nil, nil
	}

	jobs := make([]ExternalSyncJob, 0, limit)
	err := r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			WITH candidates AS (
				SELECT id
				FROM user_external_sync_outbox
				WHERE (
					status = 'pending'
					OR (status = 'failed' AND available_at <= NOW())
					OR (status = 'processing' AND locked_at <= NOW() - $2::interval)
				)
				ORDER BY available_at ASC, id ASC
				LIMIT $1
				FOR UPDATE SKIP LOCKED
			)
			UPDATE user_external_sync_outbox AS o
			SET status = 'processing',
				locked_at = NOW(),
				updated_at = NOW()
			FROM candidates
			WHERE o.id = candidates.id
			RETURNING o.id, o.job_type, o.payload, o.attempt_count
		`, limit, pgInterval(staleAfter))
		if err != nil {
			return fmt.Errorf("ClaimExternalSyncJobs query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var job ExternalSyncJob
			var payload json.RawMessage
			if err := rows.Scan(&job.ID, &job.JobType, &payload, &job.AttemptCount); err != nil {
				return fmt.Errorf("ClaimExternalSyncJobs scan: %w", err)
			}
			job.Payload = append(job.Payload[:0], payload...)
			jobs = append(jobs, job)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("ClaimExternalSyncJobs rows: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *Repository) MarkExternalSyncJobDone(ctx context.Context, jobID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE user_external_sync_outbox
		SET status = 'completed',
			locked_at = NULL,
			last_error = NULL,
			updated_at = NOW()
		WHERE id = $1
	`, jobID)
	if err != nil {
		return fmt.Errorf("MarkExternalSyncJobDone: %w", err)
	}
	return nil
}

func (r *Repository) MarkExternalSyncJobRetry(ctx context.Context, jobID int64, nextAttemptAt time.Time, lastError string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE user_external_sync_outbox
		SET status = 'failed',
			attempt_count = attempt_count + 1,
			available_at = $2,
			locked_at = NULL,
			last_error = $3,
			updated_at = NOW()
		WHERE id = $1
	`, jobID, nextAttemptAt, lastError)
	if err != nil {
		return fmt.Errorf("MarkExternalSyncJobRetry: %w", err)
	}
	return nil
}

func pgInterval(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds <= 0 {
		seconds = 1
	}
	return fmt.Sprintf("%d seconds", seconds)
}
