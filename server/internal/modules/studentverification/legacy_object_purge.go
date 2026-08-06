package studentverification

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
)

const (
	legacyObjectPurgeBatchSize    = 50
	legacyObjectPurgeLease        = 2 * time.Minute
	legacyObjectPurgePollInterval = 30 * time.Second
)

type legacyObjectPurgeItem struct {
	ID        int64
	ObjectKey string
	Attempts  int
}

func (s *Service) runLegacyObjectPurge(ctx context.Context) {
	owner, err := newID()
	if err != nil {
		logger.FromContext(ctx).Warn("failed to claim legacy verification object purge items", zap.Error(err))
		return
	}
	ticker := time.NewTicker(legacyObjectPurgePollInterval)
	defer ticker.Stop()
	for {
		s.processLegacyObjectPurgeBatch(ctx, owner)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) processLegacyObjectPurgeBatch(ctx context.Context, owner string) {
	now := s.now()
	items, err := s.repo.ClaimLegacyObjectPurgeItems(
		ctx,
		owner,
		legacyObjectPurgeBatchSize,
		legacyObjectPurgeLease,
		now,
	)
	if err != nil {
		return
	}
	for _, item := range items {
		if err := s.manualMaterialStore.DeleteManualReviewMaterial(ctx, item.ObjectKey); err == nil {
			if completeErr := s.repo.CompleteLegacyObjectPurgeItem(ctx, item.ID, owner); completeErr != nil {
				logger.FromContext(ctx).Warn("failed to complete legacy verification object purge item", zap.Error(completeErr))
			}
			continue
		}
		backoff := time.Duration(1<<min(item.Attempts, 8)) * time.Second
		if backoff > 15*time.Minute {
			backoff = 15 * time.Minute
		}
		if retryErr := s.repo.RetryLegacyObjectPurgeItem(
			ctx,
			item.ID,
			owner,
			now.Add(backoff),
			"object_delete_unavailable",
			now,
		); retryErr != nil {
			logger.FromContext(ctx).Warn("failed to reschedule legacy verification object purge item", zap.Error(retryErr))
		}
	}
}

func (r *Repository) ClaimLegacyObjectPurgeItems(
	ctx context.Context,
	owner string,
	limit int,
	lease time.Duration,
	now time.Time,
) ([]legacyObjectPurgeItem, error) {
	if limit <= 0 {
		return []legacyObjectPurgeItem{}, nil
	}
	ctx = withTable(ctx, "student_verification_object_purge_queue")
	rows, err := r.db.Query(ctx, `
		WITH candidates AS (
			SELECT id
			FROM student_verification_object_purge_queue
			WHERE (status = 'pending' AND available_at <= $1)
			   OR (status = 'claimed' AND claimed_at <= $4)
			ORDER BY available_at, id
			LIMIT $3
			FOR UPDATE SKIP LOCKED
		)
		UPDATE student_verification_object_purge_queue AS item
		SET status = 'claimed',
		    claimed_at = $1,
		    claim_owner = $2,
		    attempts = item.attempts + 1,
		    last_error_code = NULL,
		    updated_at = $1
		FROM candidates
		WHERE item.id = candidates.id
		RETURNING item.id, item.object_key, item.attempts
	`, now, owner, limit, now.Add(-lease))
	if err != nil {
		return nil, fmt.Errorf("claim legacy student verification objects: %w", err)
	}
	defer rows.Close()
	items := make([]legacyObjectPurgeItem, 0, limit)
	for rows.Next() {
		var item legacyObjectPurgeItem
		if err := rows.Scan(&item.ID, &item.ObjectKey, &item.Attempts); err != nil {
			return nil, fmt.Errorf("scan legacy student verification object: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate legacy student verification objects: %w", err)
	}
	return items, nil
}

func (r *Repository) CompleteLegacyObjectPurgeItem(
	ctx context.Context,
	itemID int64,
	owner string,
) error {
	ctx = withTable(ctx, "student_verification_object_purge_queue")
	result, err := r.db.Exec(ctx, `
		DELETE FROM student_verification_object_purge_queue
		WHERE id = $1 AND status = 'claimed' AND claim_owner = $2
	`, itemID, owner)
	if err != nil {
		return fmt.Errorf("complete legacy student verification object purge: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("complete legacy student verification object purge: lease lost")
	}
	return nil
}

func (r *Repository) RetryLegacyObjectPurgeItem(
	ctx context.Context,
	itemID int64,
	owner string,
	availableAt time.Time,
	errorCode string,
	now time.Time,
) error {
	ctx = withTable(ctx, "student_verification_object_purge_queue")
	result, err := r.db.Exec(ctx, `
		UPDATE student_verification_object_purge_queue
		SET status = 'pending',
		    claimed_at = NULL,
		    claim_owner = NULL,
		    available_at = $3,
		    last_error_code = $4,
		    updated_at = $5
		WHERE id = $1 AND status = 'claimed' AND claim_owner = $2
	`, itemID, owner, availableAt, errorCode, now)
	if err != nil {
		return fmt.Errorf("retry legacy student verification object purge: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("retry legacy student verification object purge: lease lost")
	}
	return nil
}
