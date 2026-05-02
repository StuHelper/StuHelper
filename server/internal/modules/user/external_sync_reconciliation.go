package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

const (
	externalSyncReconciliationHour         = 3
	externalSyncReconciliationRepairLimit  = 100
	externalSyncReconciliationMetricDomain = "user_profile_projection"
)

var ErrExternalSyncReconciliationThresholdExceeded = errors.New("external sync reconciliation threshold exceeded")

type StudentRoleProjectionState struct {
	UserID   int64
	Approved bool
}

func (s *Service) runExternalSyncReconciliationLoop(ctx context.Context) {
	for {
		delay := nextExternalSyncReconciliationDelay(time.Now())
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			s.runExternalSyncReconciliation(ctx)
		}
	}
}

func nextExternalSyncReconciliationDelay(now time.Time) time.Duration {
	next := time.Date(now.Year(), now.Month(), now.Day(), externalSyncReconciliationHour, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}

func (s *Service) runExternalSyncReconciliation(ctx context.Context) {
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.L().Error(
				"user external sync reconciliation panicked",
				zap.Any("panic", recovered),
				zap.Stack("stack"),
			)
		}
	}()

	requeued, err := s.ReconcileUserProfileProjections(ctx, externalSyncReconciliationRepairLimit)
	if err != nil {
		logger.L().Warn("user external sync reconciliation failed", zap.Error(err))
		return
	}
	logger.L().Info("user external sync reconciliation completed", zap.Int("requeued_count", requeued))
}

func (s *Service) ReconcileUserProfileProjections(ctx context.Context, repairLimit int) (int, error) {
	if repairLimit <= 0 {
		repairLimit = externalSyncReconciliationRepairLimit
	}
	states, err := s.repo.ListStudentRoleProjectionStates(ctx, repairLimit+1)
	if err != nil {
		return 0, fmt.Errorf("list student role projection states: %w", err)
	}
	if len(states) > repairLimit {
		return 0, externalSyncReconciliationThresholdExceededError(repairLimit)
	}
	if len(states) == 0 {
		return 0, nil
	}
	requeued, err := s.requeueUserProfileProjectionStates(ctx, states)
	if err != nil {
		return 0, err
	}
	audit.Log(audit.EventFromContext(ctx, audit.Event{
		Type:         audit.EventType("iam.drift.reconcile"),
		Category:     "domain_event",
		ActorType:    "system",
		ResourceType: "user_profile_projection",
		ResourceID:   "user_profiles",
		Action:       "requeue",
		Result:       "success",
		Details:      map[string]any{"profile_count": len(states), "requeued_jobs": requeued},
	}))
	return requeued, nil
}

func externalSyncReconciliationThresholdExceededError(repairLimit int) error {
	metrics.ObserveIAMDriftReconciliationThresholdExceeded(externalSyncReconciliationMetricDomain)
	return fmt.Errorf("%w: limit=%d", ErrExternalSyncReconciliationThresholdExceeded, repairLimit)
}

func (s *Service) requeueUserProfileProjectionStates(ctx context.Context, states []StudentRoleProjectionState) (int, error) {
	requeued := 0
	err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		for _, state := range states {
			if err := s.enqueueUserProfileProjectionTx(ctx, tx, state.UserID, state.Approved); err != nil {
				return err
			}
			requeued++
			if err := s.enqueueVerifiedStudentRoleSyncTx(ctx, tx, state.UserID, state.Approved); err != nil {
				return err
			}
			requeued++
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("requeue user profile projections: %w", err)
	}
	return requeued, nil
}
