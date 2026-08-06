package studentverification

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
)

const (
	eligibilityOutboxBatchSize    = 50
	eligibilityOutboxMaxAttempts  = 10
	eligibilityOutboxLease        = 30 * time.Second
	eligibilityOutboxPollInterval = 2 * time.Second
)

type EligibilityOutboxEvent struct {
	ID                int64
	EventID           string
	UserID            int64
	SchoolID          int64
	EventType         string
	AggregateRevision int64
	AttemptCount      int
}

func (s *Service) StartEligibilityEventBackgroundJob(
	ctx context.Context,
	start func(string, func(context.Context)),
) {
	if start == nil {
		panic("studentverification.Service.StartEligibilityEventBackgroundJob: starter is required")
	}
	if s.eligibilityEventConsumer == nil {
		return
	}
	start("student verification eligibility event publisher", s.runEligibilityEventPublisher)
}

func (s *Service) runEligibilityEventPublisher(ctx context.Context) {
	owner, err := newID()
	if err != nil {
		logger.FromContext(ctx).Warn("failed to claim student eligibility events", zap.Error(err))
		return
	}
	ticker := time.NewTicker(eligibilityOutboxPollInterval)
	defer ticker.Stop()
	for {
		s.processEligibilityOutboxBatch(ctx, owner)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) processEligibilityOutboxBatch(ctx context.Context, owner string) {
	now := s.now()
	events, err := s.repo.ClaimEligibilityOutboxEvents(
		ctx,
		owner,
		eligibilityOutboxBatchSize,
		eligibilityOutboxLease,
		eligibilityOutboxMaxAttempts,
		now,
	)
	if err != nil {
		return
	}
	for _, event := range events {
		err = s.eligibilityEventConsumer.ReevaluateStudentEligibility(
			ctx,
			event.UserID,
			event.SchoolID,
			event.AggregateRevision,
		)
		if err == nil {
			if completeErr := s.repo.CompleteEligibilityOutboxEvent(
				ctx, event.ID, owner, s.now(),
			); completeErr != nil {
				logger.FromContext(ctx).Warn("failed to complete student eligibility event", zap.Error(completeErr))
			}
			continue
		}
		backoff := time.Duration(1<<min(event.AttemptCount, 8)) * time.Second
		if backoff > 15*time.Minute {
			backoff = 15 * time.Minute
		}
		if retryErr := s.repo.RetryEligibilityOutboxEvent(
			ctx,
			event.ID,
			owner,
			event.AttemptCount,
			eligibilityOutboxMaxAttempts,
			s.now().Add(backoff),
			"consumer_unavailable",
			s.now(),
		); retryErr != nil {
			logger.FromContext(ctx).Warn("failed to reschedule student eligibility event", zap.Error(retryErr))
		}
	}
}

func (r *Repository) ClaimEligibilityOutboxEvents(
	ctx context.Context,
	owner string,
	limit int,
	lease time.Duration,
	maxAttempts int,
	now time.Time,
) ([]EligibilityOutboxEvent, error) {
	if limit <= 0 {
		return []EligibilityOutboxEvent{}, nil
	}
	ctx = withTable(ctx, "student_verification_event_outbox")
	rows, err := r.db.Query(ctx, `
		WITH terminal AS (
			UPDATE student_verification_event_outbox
			SET status = 'dead_letter',
			    claimed_at = NULL,
			    claim_owner = NULL,
			    last_error_code = COALESCE(last_error_code, 'attempts_exhausted'),
			    updated_at = $1
			WHERE status IN ('pending', 'claimed')
			  AND attempts >= $5
			RETURNING id
		), candidates AS (
			SELECT id
			FROM student_verification_event_outbox
			WHERE attempts < $5
			  AND (
			      (status = 'pending' AND available_at <= $1)
			      OR (status = 'claimed' AND claimed_at <= $4)
			  )
			  AND id NOT IN (SELECT id FROM terminal)
			ORDER BY available_at, id
			LIMIT $3
			FOR UPDATE SKIP LOCKED
		)
		UPDATE student_verification_event_outbox AS event
		SET status = 'claimed',
		    claimed_at = $1,
		    claim_owner = $2,
		    attempts = event.attempts + 1,
		    last_error_code = NULL,
		    updated_at = $1
		FROM candidates
		WHERE event.id = candidates.id
		RETURNING event.id, event.event_id, event.user_id, event.school_id,
		          event.event_type, event.aggregate_revision, event.attempts
	`, now, owner, limit, now.Add(-lease), maxAttempts)
	if err != nil {
		return nil, fmt.Errorf("claim student eligibility events: %w", err)
	}
	defer rows.Close()
	events := make([]EligibilityOutboxEvent, 0, limit)
	for rows.Next() {
		var event EligibilityOutboxEvent
		if err := rows.Scan(
			&event.ID,
			&event.EventID,
			&event.UserID,
			&event.SchoolID,
			&event.EventType,
			&event.AggregateRevision,
			&event.AttemptCount,
		); err != nil {
			return nil, fmt.Errorf("scan student eligibility event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate student eligibility events: %w", err)
	}
	return events, nil
}

func (r *Repository) CompleteEligibilityOutboxEvent(
	ctx context.Context,
	eventID int64,
	owner string,
	now time.Time,
) error {
	ctx = withTable(ctx, "student_verification_event_outbox")
	result, err := r.db.Exec(ctx, `
		UPDATE student_verification_event_outbox
		SET status = 'published',
		    claimed_at = NULL,
		    claim_owner = NULL,
		    published_at = $3,
		    last_error_code = NULL,
		    updated_at = $3
		WHERE id = $1 AND status = 'claimed' AND claim_owner = $2
	`, eventID, owner, now)
	if err != nil {
		return fmt.Errorf("complete student eligibility event: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("complete student eligibility event: lease lost")
	}
	return nil
}

func (r *Repository) RetryEligibilityOutboxEvent(
	ctx context.Context,
	eventID int64,
	owner string,
	attemptCount int,
	maxAttempts int,
	availableAt time.Time,
	errorCode string,
	now time.Time,
) error {
	ctx = withTable(ctx, "student_verification_event_outbox")
	status := "pending"
	if attemptCount >= maxAttempts {
		status = "dead_letter"
	}
	result, err := r.db.Exec(ctx, `
		UPDATE student_verification_event_outbox
		SET status = $3,
		    claimed_at = NULL,
		    claim_owner = NULL,
		    available_at = $4,
		    last_error_code = $5,
		    updated_at = $6
		WHERE id = $1 AND status = 'claimed' AND claim_owner = $2
	`, eventID, owner, status, availableAt, errorCode, now)
	if err != nil {
		return fmt.Errorf("retry student eligibility event: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("retry student eligibility event: lease lost")
	}
	return nil
}
