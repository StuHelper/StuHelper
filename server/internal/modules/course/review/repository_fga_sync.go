package review

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/outbox"
)

const fgaSyncOutboxStream = outbox.StreamIAMOpenFGATupleSync

var fgaSyncJobTypes = []string{
	fgaSyncJobTypeReviewRelations,
	fgaSyncJobTypeReportRelations,
}

func (r *Repository) UpsertFGASyncJobTx(ctx context.Context, tx pgx.Tx, jobType, dedupeKey string, payload []byte) error {
	if err := outbox.UpsertJobTx(ctx, tx, fgaSyncOutboxStream, jobType, dedupeKey, payload); err != nil {
		return fmt.Errorf("UpsertFGASyncJobTx: %w", err)
	}
	return nil
}

func (r *Repository) ClaimFGASyncJobs(ctx context.Context, limit int, staleAfter time.Duration) ([]FGASyncJob, error) {
	jobs, err := outbox.ClaimJobsByTypes(ctx, r.db, fgaSyncOutboxStream, fgaSyncJobTypes, limit, staleAfter)
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

func (r *Repository) ListReviewRelationProjectionStates(
	ctx context.Context,
	limit int,
) ([]ReviewRelationProjectionState, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT r.id, u.id, r.school_id
		FROM reviews r
		JOIN users u ON u.user_hash = r.user_hash
		WHERE r.status <> 'deleted'
		ORDER BY r.id ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("ListReviewRelationProjectionStates: %w", err)
	}
	defer rows.Close()
	return scanReviewRelationProjectionStates(rows, limit)
}

func (r *Repository) ListReportRelationProjectionStates(
	ctx context.Context,
	limit int,
) ([]ReportRelationProjectionState, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT rr.id, rr.school_id
		FROM review_reports rr
		JOIN reviews r ON r.id = rr.review_id
		WHERE r.status <> 'deleted'
		ORDER BY rr.id ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("ListReportRelationProjectionStates: %w", err)
	}
	defer rows.Close()
	return scanReportRelationProjectionStates(rows, limit)
}

func scanReviewRelationProjectionStates(rows pgx.Rows, limit int) ([]ReviewRelationProjectionState, error) {
	states := make([]ReviewRelationProjectionState, 0, limit)
	for rows.Next() {
		var state ReviewRelationProjectionState
		if err := rows.Scan(&state.ReviewID, &state.AuthorUserID, &state.SchoolID); err != nil {
			return nil, fmt.Errorf("ListReviewRelationProjectionStates scan: %w", err)
		}
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListReviewRelationProjectionStates rows: %w", err)
	}
	return states, nil
}

func scanReportRelationProjectionStates(rows pgx.Rows, limit int) ([]ReportRelationProjectionState, error) {
	states := make([]ReportRelationProjectionState, 0, limit)
	for rows.Next() {
		var state ReportRelationProjectionState
		if err := rows.Scan(&state.ReportID, &state.SchoolID); err != nil {
			return nil, fmt.Errorf("ListReportRelationProjectionStates scan: %w", err)
		}
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListReportRelationProjectionStates rows: %w", err)
	}
	return states, nil
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
