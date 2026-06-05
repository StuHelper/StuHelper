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
		if err := r.lockResourceForDelete(ctx, tx, resourceID, ownerUserID); err != nil {
			return err
		}
		targets, err := r.lockVersionCleanupTargetsForDelete(ctx, tx, resourceID)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `DELETE FROM resource_items WHERE id = $1`, resourceID); err != nil {
			return fmt.Errorf("delete resource item: %w", err)
		}

		for _, target := range targets {
			if err := r.UpsertCleanupJobTx(ctx, tx, resourceID, target); err != nil {
				return err
			}
		}
		return nil
	})
}

type cleanupTarget struct {
	VersionNo int
	MountID   int64
	ObjectKey string
}

func (r *Repository) UpsertCleanupJobTx(ctx context.Context, tx pgx.Tx, resourceID int64, target cleanupTarget) error {
	payload, err := json.Marshal(cleanupPayload{
		MountID:   target.MountID,
		ObjectKey: target.ObjectKey,
	})
	if err != nil {
		return fmt.Errorf("marshal resource cleanup payload: %w", err)
	}
	return outbox.UpsertJobTx(
		ctx,
		tx,
		resourceCleanupOutboxStream,
		resourceCleanupJobType,
		resourceCleanupKey(resourceID, target.VersionNo),
		payload,
	)
}

func (r *Repository) ClaimCleanupJobs(ctx context.Context, limit int, staleAfter time.Duration) ([]cleanupJob, error) {
	jobs, err := outbox.ClaimJobs(ctx, r.db, resourceCleanupOutboxStream, limit, staleAfter)
	if err != nil {
		return nil, fmt.Errorf("claim resource cleanup jobs: %w", err)
	}
	return mapCleanupJobs(jobs), nil
}

func (r *Repository) MarkCleanupJobDone(ctx context.Context, jobID int64, lockedAt time.Time) error {
	if err := outbox.MarkJobDone(ctx, r.db, jobID, lockedAt); err != nil {
		return fmt.Errorf("mark resource cleanup job done: %w", err)
	}
	return nil
}

func (r *Repository) MarkCleanupJobRetry(
	ctx context.Context,
	jobID int64,
	lockedAt time.Time,
	nextAttemptAt time.Time,
	lastError string,
) error {
	if err := outbox.MarkJobRetry(ctx, r.db, jobID, lockedAt, nextAttemptAt, lastError); err != nil {
		return fmt.Errorf("mark resource cleanup job retry: %w", err)
	}
	return nil
}

func (r *Repository) MarkCleanupJobFailure(
	ctx context.Context,
	jobID int64,
	lockedAt time.Time,
	nextAttemptAt time.Time,
	lastError string,
	terminal bool,
) error {
	if err := outbox.MarkJobFailure(ctx, r.db, jobID, lockedAt, nextAttemptAt, lastError, terminal); err != nil {
		return fmt.Errorf("mark resource cleanup job failure: %w", err)
	}
	return nil
}

func (r *Repository) lockResourceForDelete(ctx context.Context, tx pgx.Tx, resourceID int64, ownerUserID string) error {
	var actualOwner string
	err := tx.QueryRow(ctx, `
		SELECT owner_user_id
		FROM resource_items ri
		WHERE ri.id = $1
		FOR UPDATE
	`, resourceID).Scan(&actualOwner)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrResourceNotFound
	}
	if err != nil {
		return fmt.Errorf("load resource delete target: %w", err)
	}
	if actualOwner != ownerUserID {
		return ErrResourceForbidden
	}
	return nil
}

func (r *Repository) lockVersionCleanupTargetsForDelete(ctx context.Context, tx pgx.Tx, resourceID int64) ([]cleanupTarget, error) {
	rows, err := tx.Query(ctx, `
		SELECT version_no, mount_id, object_key
		FROM resource_versions
		WHERE resource_id = $1
		ORDER BY version_no ASC
		FOR UPDATE
	`, resourceID)
	if err != nil {
		return nil, fmt.Errorf("load resource version cleanup targets: %w", err)
	}
	defer rows.Close()

	targets := make([]cleanupTarget, 0)
	for rows.Next() {
		var target cleanupTarget
		if err := rows.Scan(&target.VersionNo, &target.MountID, &target.ObjectKey); err != nil {
			return nil, fmt.Errorf("scan resource version cleanup target: %w", err)
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("resource version cleanup target rows: %w", err)
	}
	return targets, nil
}

func mapCleanupJobs(jobs []outbox.Job) []cleanupJob {
	items := make([]cleanupJob, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, cleanupJob{
			ID:           job.ID,
			JobType:      job.JobType,
			Payload:      append(json.RawMessage(nil), job.Payload...),
			AttemptCount: job.AttemptCount,
			LockedAt:     job.LockedAt,
		})
	}
	return items
}
