package resource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/outbox"
)

func (r *Repository) DeleteResource(ctx context.Context, resourceID int64, ownerUserID string) error {
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		mountID, objectKey, err := r.lockLatestVersionForDelete(ctx, tx, resourceID, ownerUserID)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `DELETE FROM resource_items WHERE id = $1`, resourceID); err != nil {
			return fmt.Errorf("delete resource item: %w", err)
		}

		return r.UpsertCleanupJobTx(ctx, tx, resourceID, mountID, objectKey)
	})
}

func (r *Repository) UpsertCleanupJobTx(ctx context.Context, tx pgx.Tx, resourceID, mountID int64, objectKey string) error {
	payload, err := json.Marshal(cleanupPayload{
		MountID:   mountID,
		ObjectKey: objectKey,
	})
	if err != nil {
		return fmt.Errorf("marshal resource cleanup payload: %w", err)
	}
	return outbox.UpsertJobTx(ctx, tx, resourceCleanupOutboxStream, resourceCleanupJobType, resourceCleanupKey(resourceID), payload)
}

func (r *Repository) ClaimCleanupJobs(ctx context.Context, limit int, staleAfter time.Duration) ([]cleanupJob, error) {
	jobs, err := outbox.ClaimJobs(ctx, r.db, resourceCleanupOutboxStream, limit, staleAfter)
	if err != nil {
		return nil, fmt.Errorf("claim resource cleanup jobs: %w", err)
	}
	return mapCleanupJobs(jobs), nil
}

func (r *Repository) MarkCleanupJobDone(ctx context.Context, jobID int64) error {
	if err := outbox.MarkJobDone(ctx, r.db, jobID); err != nil {
		return fmt.Errorf("mark resource cleanup job done: %w", err)
	}
	return nil
}

func (r *Repository) MarkCleanupJobRetry(ctx context.Context, jobID int64, nextAttemptAt time.Time, lastError string) error {
	if err := outbox.MarkJobRetry(ctx, r.db, jobID, nextAttemptAt, lastError); err != nil {
		return fmt.Errorf("mark resource cleanup job retry: %w", err)
	}
	return nil
}

func (r *Repository) lockLatestVersionForDelete(ctx context.Context, tx pgx.Tx, resourceID int64, ownerUserID string) (int64, string, error) {
	var (
		actualOwner string
		mountID     int64
		objectKey   string
	)
	err := tx.QueryRow(ctx, `
		SELECT ri.owner_user_id, rv.mount_id, rv.object_key
		FROM resource_items ri
		JOIN LATERAL (
			SELECT mount_id, object_key
			FROM resource_versions
			WHERE resource_id = ri.id
			ORDER BY version_no DESC
			LIMIT 1
		) rv ON TRUE
		WHERE ri.id = $1
		FOR UPDATE
	`, resourceID).Scan(&actualOwner, &mountID, &objectKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", ErrResourceNotFound
	}
	if err != nil {
		return 0, "", fmt.Errorf("load resource delete target: %w", err)
	}
	if actualOwner != ownerUserID {
		return 0, "", ErrResourceForbidden
	}
	return mountID, objectKey, nil
}

func mapCleanupJobs(jobs []outbox.Job) []cleanupJob {
	items := make([]cleanupJob, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, cleanupJob{
			ID:           job.ID,
			JobType:      job.JobType,
			Payload:      append(json.RawMessage(nil), job.Payload...),
			AttemptCount: job.AttemptCount,
		})
	}
	return items
}
