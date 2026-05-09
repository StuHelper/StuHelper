package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

const DomainEventOutboxTable = "domain_event_outbox"

const upsertSQL = `
	INSERT INTO ` + DomainEventOutboxTable + ` (
		stream,
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
	) VALUES ($1, $2, $3, $4::jsonb, 'pending', 0, NOW(), NULL, NULL, NOW(), NOW())
	ON CONFLICT (stream, dedupe_key)
	DO UPDATE SET
		job_type = EXCLUDED.job_type,
		payload = EXCLUDED.payload,
		status = 'pending',
		attempt_count = 0,
		available_at = NOW(),
		locked_at = NULL,
		last_error = NULL,
		updated_at = NOW()
`

const claimSQL = `
	WITH candidates AS (
		SELECT id
		FROM ` + DomainEventOutboxTable + `
		WHERE stream = $1
		  AND (
			status = 'pending'
			OR (status = 'failed' AND available_at <= NOW())
			OR (status = 'processing' AND locked_at <= NOW() - $3::interval)
		)
		ORDER BY available_at ASC, id ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	)
	UPDATE ` + DomainEventOutboxTable + ` AS o
	SET status = 'processing',
		locked_at = NOW(),
		updated_at = NOW()
	FROM candidates
	WHERE o.id = candidates.id
	RETURNING o.id, o.job_type, o.payload, o.attempt_count
`

const markDoneSQL = `
	UPDATE ` + DomainEventOutboxTable + `
	SET status = 'completed',
		locked_at = NULL,
		last_error = NULL,
		updated_at = NOW()
	WHERE id = $1
`

const markRetrySQL = `
	UPDATE ` + DomainEventOutboxTable + `
	SET status = 'failed',
		attempt_count = attempt_count + 1,
		available_at = $2,
		locked_at = NULL,
		last_error = $3,
		updated_at = NOW()
	WHERE id = $1
`

var streamNamePattern = regexp.MustCompile(`^[a-z_]+$`)

type Job struct {
	ID           int64
	JobType      string
	Payload      json.RawMessage
	AttemptCount int
}

func UpsertJobTx(ctx context.Context, tx pgx.Tx, stream, jobType, dedupeKey string, payload []byte) error {
	stream, err := safeStreamName(stream)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, upsertSQL, stream, jobType, dedupeKey, payload)
	if err != nil {
		return fmt.Errorf("upsert outbox job: %w", err)
	}
	return nil
}

func ClaimJobs(ctx context.Context, database *db.DB, stream string, limit int, staleAfter time.Duration) ([]Job, error) {
	if limit <= 0 {
		return nil, nil
	}
	stream, err := safeStreamName(stream)
	if err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, limit)
	err = database.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, claimSQL, stream, limit, intervalLiteral(staleAfter))
		if err != nil {
			return fmt.Errorf("claim outbox jobs query: %w", err)
		}
		defer rows.Close()
		return scanJobs(rows, &jobs)
	})
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func MarkJobDone(ctx context.Context, database *db.DB, jobID int64) error {
	_, err := database.Exec(ctx, markDoneSQL, jobID)
	if err != nil {
		return fmt.Errorf("mark outbox job done: %w", err)
	}
	return nil
}

func MarkJobRetry(ctx context.Context, database *db.DB, jobID int64, nextAttemptAt time.Time, lastError string) error {
	_, err := database.Exec(ctx, markRetrySQL, jobID, nextAttemptAt, lastError)
	if err != nil {
		return fmt.Errorf("mark outbox job retry: %w", err)
	}
	return nil
}

func scanJobs(rows pgx.Rows, jobs *[]Job) error {
	for rows.Next() {
		var job Job
		var payload json.RawMessage
		if err := rows.Scan(&job.ID, &job.JobType, &payload, &job.AttemptCount); err != nil {
			return fmt.Errorf("scan outbox job: %w", err)
		}
		job.Payload = append(job.Payload[:0], payload...)
		*jobs = append(*jobs, job)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("claim outbox jobs rows: %w", err)
	}
	return nil
}

func safeStreamName(stream string) (string, error) {
	if !streamNamePattern.MatchString(stream) {
		return "", fmt.Errorf("invalid outbox stream %q", stream)
	}
	return stream, nil
}

func intervalLiteral(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds <= 0 {
		seconds = 1
	}
	return fmt.Sprintf("%d seconds", seconds)
}
