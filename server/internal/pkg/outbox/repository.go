package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/db"
)

const DomainEventOutboxTable = "domain_event_outbox"

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusDeadLetter = "dead_letter"
)

var ErrJobLockLost = errors.New("outbox job lock lost")

const upsertSQL = `
	INSERT INTO ` + DomainEventOutboxTable + ` AS current_job (
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
		status = CASE
			WHEN current_job.status = 'processing' THEN 'processing'
			ELSE 'pending'
		END,
		attempt_count = 0,
		available_at = NOW(),
		locked_at = CASE
			WHEN current_job.status = 'processing' THEN current_job.locked_at
			ELSE NULL
		END,
		locked_revision = CASE
			WHEN current_job.status = 'processing' THEN current_job.locked_revision
			ELSE NULL
		END,
		revision = current_job.revision + 1,
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
		locked_revision = o.revision,
		updated_at = NOW()
	FROM candidates
	WHERE o.id = candidates.id
	RETURNING o.id, o.job_type, o.payload, o.attempt_count, o.locked_at
`

const claimByTypesSQL = `
	WITH candidates AS (
		SELECT id
		FROM ` + DomainEventOutboxTable + `
		WHERE stream = $1
		  AND job_type = ANY($4::text[])
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
		locked_revision = o.revision,
		updated_at = NOW()
	FROM candidates
	WHERE o.id = candidates.id
	RETURNING o.id, o.job_type, o.payload, o.attempt_count, o.locked_at
`

const markDoneSQL = `
	UPDATE ` + DomainEventOutboxTable + `
	SET status = CASE
			WHEN locked_revision = revision THEN 'completed'
			ELSE 'pending'
		END,
		attempt_count = CASE
			WHEN locked_revision = revision THEN attempt_count
			ELSE 0
		END,
		available_at = CASE
			WHEN locked_revision = revision THEN available_at
			ELSE NOW()
		END,
		locked_at = NULL,
		locked_revision = NULL,
		last_error = NULL,
		updated_at = NOW()
	WHERE id = $1
	  AND status = 'processing'
	  AND locked_at = $2
`

const markRetrySQL = `
	UPDATE ` + DomainEventOutboxTable + `
	SET status = CASE
			WHEN locked_revision = revision THEN '` + StatusFailed + `'
			ELSE '` + StatusPending + `'
		END,
		attempt_count = CASE
			WHEN locked_revision = revision THEN attempt_count + 1
			ELSE 0
		END,
		available_at = CASE
			WHEN locked_revision = revision THEN $3
			ELSE NOW()
		END,
		locked_at = NULL,
		locked_revision = NULL,
		last_error = CASE
			WHEN locked_revision = revision THEN $4
			ELSE NULL
		END,
		updated_at = NOW()
	WHERE id = $1
	  AND status = 'processing'
	  AND locked_at = $2
`

const markDeadLetterSQL = `
	UPDATE ` + DomainEventOutboxTable + `
	SET status = CASE
			WHEN locked_revision = revision THEN '` + StatusDeadLetter + `'
			ELSE '` + StatusPending + `'
		END,
		attempt_count = CASE
			WHEN locked_revision = revision THEN attempt_count + 1
			ELSE 0
		END,
		available_at = NOW(),
		locked_at = NULL,
		locked_revision = NULL,
		last_error = CASE
			WHEN locked_revision = revision THEN $3
			ELSE NULL
		END,
		updated_at = NOW()
	WHERE id = $1
	  AND status = 'processing'
	  AND locked_at = $2
`

const requeueDeadLetterSQL = `
	UPDATE ` + DomainEventOutboxTable + `
	SET status = '` + StatusPending + `',
		attempt_count = 0,
		available_at = NOW(),
		locked_at = NULL,
		locked_revision = NULL,
		last_error = NULL,
		updated_at = NOW()
	WHERE id = $1
	  AND status = '` + StatusDeadLetter + `'
`

const markAbandonedSQL = `
	UPDATE ` + DomainEventOutboxTable + `
	SET status = '` + StatusPending + `',
		available_at = NOW(),
		locked_at = NULL,
		locked_revision = NULL,
		last_error = NULL,
		updated_at = NOW()
	WHERE id = $1
	  AND status = 'processing'
	  AND locked_at = $2
`

var streamNamePattern = regexp.MustCompile(`^[a-z_]+$`)

type Job struct {
	ID           int64
	JobType      string
	Payload      json.RawMessage
	AttemptCount int
	LockedAt     time.Time
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

func ClaimJobsByTypes(
	ctx context.Context,
	database *db.DB,
	stream string,
	jobTypes []string,
	limit int,
	staleAfter time.Duration,
) ([]Job, error) {
	if limit <= 0 || len(jobTypes) == 0 {
		return nil, nil
	}
	stream, err := safeStreamName(stream)
	if err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, limit)
	err = database.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, claimByTypesSQL, stream, limit, intervalLiteral(staleAfter), jobTypes)
		if err != nil {
			return fmt.Errorf("claim outbox jobs by types query: %w", err)
		}
		defer rows.Close()
		return scanJobs(rows, &jobs)
	})
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func MarkJobDone(ctx context.Context, database *db.DB, jobID int64, lockedAt time.Time) error {
	ctx = db.WithTableHint(ctx, DomainEventOutboxTable)
	result, err := database.Exec(ctx, markDoneSQL, jobID, lockedAt)
	if err != nil {
		return fmt.Errorf("mark outbox job done: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrJobLockLost
	}
	return nil
}

func MarkJobRetry(
	ctx context.Context,
	database *db.DB,
	jobID int64,
	lockedAt time.Time,
	nextAttemptAt time.Time,
	lastError string,
) error {
	ctx = db.WithTableHint(ctx, DomainEventOutboxTable)
	result, err := database.Exec(ctx, markRetrySQL, jobID, lockedAt, nextAttemptAt, lastError)
	if err != nil {
		return fmt.Errorf("mark outbox job retry: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrJobLockLost
	}
	return nil
}

func MarkJobDeadLetter(ctx context.Context, database *db.DB, jobID int64, lockedAt time.Time, lastError string) error {
	ctx = db.WithTableHint(ctx, DomainEventOutboxTable)
	result, err := database.Exec(ctx, markDeadLetterSQL, jobID, lockedAt, lastError)
	if err != nil {
		return fmt.Errorf("mark outbox job dead letter: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrJobLockLost
	}
	return nil
}

func MarkJobFailure(
	ctx context.Context,
	database *db.DB,
	jobID int64,
	lockedAt time.Time,
	nextAttemptAt time.Time,
	lastError string,
	terminal bool,
) error {
	if !terminal && nextAttemptAt.IsZero() && lastError == "" {
		return MarkJobAbandoned(ctx, database, jobID, lockedAt)
	}
	if terminal {
		return MarkJobDeadLetter(ctx, database, jobID, lockedAt, lastError)
	}
	return MarkJobRetry(ctx, database, jobID, lockedAt, nextAttemptAt, lastError)
}

// MarkJobAbandoned releases a claimed job without consuming a retry attempt.
// Workers use this when their parent context is canceled during graceful
// shutdown; the same or a newer revision becomes immediately claimable.
func MarkJobAbandoned(ctx context.Context, database *db.DB, jobID int64, lockedAt time.Time) error {
	ctx = db.WithTableHint(ctx, DomainEventOutboxTable)
	result, err := database.Exec(ctx, markAbandonedSQL, jobID, lockedAt)
	if err != nil {
		return fmt.Errorf("mark outbox job abandoned: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrJobLockLost
	}
	return nil
}

func RequeueDeadLetterJob(ctx context.Context, database *db.DB, jobID int64) (bool, error) {
	ctx = db.WithTableHint(ctx, DomainEventOutboxTable)
	result, err := database.Exec(ctx, requeueDeadLetterSQL, jobID)
	if err != nil {
		return false, fmt.Errorf("requeue dead-letter outbox job: %w", err)
	}
	return result.RowsAffected() > 0, nil
}

func scanJobs(rows pgx.Rows, jobs *[]Job) error {
	for rows.Next() {
		var job Job
		var payload json.RawMessage
		if err := rows.Scan(&job.ID, &job.JobType, &payload, &job.AttemptCount, &job.LockedAt); err != nil {
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
