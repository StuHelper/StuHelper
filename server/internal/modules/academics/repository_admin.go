package academics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
)

var ErrSourceNotFound = errors.New("academic source not found")
var ErrSourceDisabled = errors.New("academic source is disabled")
var ErrOfferingNotFound = errors.New("academic offering not found")

func (r *Repository) ListSources(ctx context.Context) ([]Source, error) {
	ctx = withDBTable(ctx, "academic_sources")
	rows, err := r.db.Query(ctx, `
		SELECT id, key, name, provider, config, enabled
		FROM academic_sources
		ORDER BY key ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list academic sources: %w", err)
	}
	defer rows.Close()
	var sources []Source
	for rows.Next() {
		var item Source
		if err := rows.Scan(&item.ID, &item.Key, &item.Name, &item.Provider, &item.Config, &item.Enabled); err != nil {
			return nil, fmt.Errorf("scan academic source: %w", err)
		}
		sources = append(sources, item)
	}
	return sources, rows.Err()
}

func (r *Repository) GetSourceByKey(ctx context.Context, key string) (*Source, error) {
	ctx = withDBTable(ctx, "academic_sources")
	var item Source
	err := r.db.QueryRow(ctx, `
		SELECT id, key, name, provider, config, enabled
		FROM academic_sources
		WHERE key = $1
	`, key).Scan(&item.ID, &item.Key, &item.Name, &item.Provider, &item.Config, &item.Enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSourceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get academic source by key: %w", err)
	}
	return &item, nil
}

func (r *Repository) CreateImportJob(ctx context.Context, source Source, requestedBy string) (int64, error) {
	ctx = withDBTable(ctx, "academic_import_jobs")
	var requestedByPtr *string
	if requestedBy != "" {
		requestedByPtr = &requestedBy
	}
	var jobID int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO academic_import_jobs (source_id, provider, trigger_mode, status, requested_by_user_id)
		VALUES ($1, $2, 'manual', 'pending', $3)
		RETURNING id
	`, source.ID, source.Provider, requestedByPtr).Scan(&jobID)
	if err != nil {
		return 0, fmt.Errorf("create academic import job: %w", err)
	}
	return jobID, nil
}

func (r *Repository) MarkImportJobRunning(ctx context.Context, jobID int64) error {
	ctx = withDBTable(ctx, "academic_import_jobs")
	_, err := r.db.Exec(ctx, `
		UPDATE academic_import_jobs
		SET status = 'running', started_at = NOW()
		WHERE id = $1
	`, jobID)
	if err != nil {
		return fmt.Errorf("mark academic import job running: %w", err)
	}
	return nil
}

func (r *Repository) MarkImportJobSucceeded(ctx context.Context, jobID int64, stats map[string]any) error {
	ctx = withDBTable(ctx, "academic_import_jobs")
	payload, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("marshal academic import stats: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		UPDATE academic_import_jobs
		SET status = 'succeeded', stats = $2, finished_at = NOW(), error_message = NULL
		WHERE id = $1
	`, jobID, payload)
	if err != nil {
		return fmt.Errorf("mark academic import job succeeded: %w", err)
	}
	return nil
}

func (r *Repository) MarkImportJobFailed(ctx context.Context, jobID int64, message string) error {
	ctx = withDBTable(ctx, "academic_import_jobs")
	_, err := r.db.Exec(ctx, `
		UPDATE academic_import_jobs
		SET status = 'failed', error_message = $2, finished_at = NOW()
		WHERE id = $1
	`, jobID, message)
	if err != nil {
		return fmt.Errorf("mark academic import job failed: %w", err)
	}
	return nil
}

func (r *Repository) ListImportJobs(ctx context.Context, page, pageSize int) ([]ImportJob, int, error) {
	ctx = withDBTable(ctx, "academic_import_jobs")
	offset := httputil.SafeOffset(page, pageSize)
	rows, err := r.db.Query(ctx, `
		SELECT j.id, s.key, s.name, j.provider, j.status, j.trigger_mode, j.requested_by_user_id,
		       j.stats, j.error_message, j.started_at, j.finished_at, j.created_at,
		       COUNT(*) OVER()
		FROM academic_import_jobs j
		JOIN academic_sources s ON s.id = j.source_id
		ORDER BY j.created_at DESC, j.id DESC
		LIMIT $1 OFFSET $2
	`, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list academic import jobs: %w", err)
	}
	defer rows.Close()
	return scanImportJobs(rows)
}

func (r *Repository) GetImportJobByID(ctx context.Context, jobID int64) (*ImportJob, error) {
	ctx = withDBTable(ctx, "academic_import_jobs")
	rows, err := r.db.Query(ctx, `
		SELECT j.id, s.key, s.name, j.provider, j.status, j.trigger_mode, j.requested_by_user_id,
		       j.stats, j.error_message, j.started_at, j.finished_at, j.created_at,
		       COUNT(*) OVER()
		FROM academic_import_jobs j
		JOIN academic_sources s ON s.id = j.source_id
		WHERE j.id = $1
	`, jobID)
	if err != nil {
		return nil, fmt.Errorf("get academic import job: %w", err)
	}
	defer rows.Close()
	list, _, err := scanImportJobs(rows)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("academic import job %d: %w", jobID, pgx.ErrNoRows)
	}
	return &list[0], nil
}

func scanImportJobs(rows pgx.Rows) ([]ImportJob, int, error) {
	jobs := make([]ImportJob, 0)
	total := 0
	for rows.Next() {
		var job ImportJob
		var statsRaw []byte
		var requestedBy *string
		var errorMessage *string
		var startedAt *time.Time
		var finishedAt *time.Time
		var createdAt time.Time
		if err := rows.Scan(
			&job.ID, &job.SourceKey, &job.SourceName, &job.Provider, &job.Status, &job.TriggerMode, &requestedBy,
			&statsRaw, &errorMessage, &startedAt, &finishedAt, &createdAt, &total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan academic import job: %w", err)
		}
		job.RequestedByUserID = requestedBy
		job.ErrorMessage = errorMessage
		job.StartedAt = formatTimePtr(startedAt)
		job.FinishedAt = formatTimePtr(finishedAt)
		job.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		job.Stats = map[string]any{}
		if len(statsRaw) > 0 {
			if err := json.Unmarshal(statsRaw, &job.Stats); err != nil {
				return nil, 0, fmt.Errorf("decode academic import stats: %w", err)
			}
		}
		jobs = append(jobs, job)
	}
	return jobs, total, rows.Err()
}

func formatTimePtr(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}
