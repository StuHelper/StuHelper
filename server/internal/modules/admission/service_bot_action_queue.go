package admission

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Service) queueInitialBotActionsTx(
	ctx context.Context,
	tx pgx.Tx,
	session *AdmissionSession,
	now time.Time,
) error {
	if session == nil {
		return nil
	}
	switch session.Status {
	case StatusJoinedMuted:
		return s.queueBotActionTx(ctx, tx, session, BotActionKick, session.LinkWaitDeadlineAt, now)
	case StatusLinked:
		return s.queueBotActionTx(ctx, tx, session, BotActionKick, session.SubmissionWaitDeadlineAt, now)
	case StatusMaterialSubmitted:
		if session.ManualReviewDeadlineAt == nil {
			return nil
		}
		return s.queueBotActionTx(ctx, tx, session, BotActionKick, *session.ManualReviewDeadlineAt, now)
	default:
		return nil
	}
}

func (s *Service) queueNextReminderTx(
	ctx context.Context,
	tx pgx.Tx,
	session *AdmissionSession,
	policy *AdmissionPolicy,
	now time.Time,
) error {
	if session == nil || policy == nil {
		return nil
	}
	next := now.Add(time.Duration(policy.ReminderIntervalSeconds) * time.Second)
	_, deadline := resolvePendingAction(session, now)
	if !next.Before(deadline) {
		return nil
	}
	return s.queueBotActionTx(ctx, tx, session, BotActionRemind, next, now)
}

func (s *Service) queueBotActionTx(
	ctx context.Context,
	tx pgx.Tx,
	session *AdmissionSession,
	action BotAction,
	scheduledAt time.Time,
	now time.Time,
) error {
	return s.repo.QueueBotActionTx(ctx, tx, AdmissionBotActionQueueInput{
		Session:     session,
		Action:      action,
		ScheduledAt: scheduledAt,
		Now:         now,
	})
}

func (s *Service) queueVerifiedUserReleaseActionsTx(
	ctx context.Context,
	tx pgx.Tx,
	userID int64,
	schoolID int64,
	now time.Time,
) error {
	sessions, err := s.repo.ListVerifiedUnreleasedSessionsByUserSchoolTx(ctx, tx, userID, schoolID)
	if err != nil {
		return err
	}
	for i := range sessions {
		if err := s.queueBotActionTx(ctx, tx, &sessions[i], BotActionRelease, now, now); err != nil {
			return err
		}
	}
	return nil
}
