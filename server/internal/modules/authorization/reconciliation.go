package authorization

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/metrics"
)

const (
	projectionReconciliationHour         = 3
	projectionReconciliationMinute       = 20
	projectionReconciliationPageSize     = 200
	projectionReconciliationRepairLimit  = 100
	projectionPendingStaleAfter          = 10 * time.Minute
	projectionReconciliationMetricDomain = "authorization_grant_projection"
)

func (s *Service) runProjectionReconciliationLoop(ctx context.Context) {
	for {
		timer := time.NewTimer(nextProjectionReconciliationDelay(time.Now()))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			s.runProjectionReconciliation(ctx)
		}
	}
}

func nextProjectionReconciliationDelay(now time.Time) time.Duration {
	next := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		projectionReconciliationHour,
		projectionReconciliationMinute,
		0,
		0,
		now.Location(),
	)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}

func (s *Service) runProjectionReconciliation(ctx context.Context) {
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.L().Error(
				"authorization projection reconciliation panicked",
				zap.Any("panic", recovered),
				zap.Stack("stack"),
			)
		}
	}()

	queued, err := s.ReconcileProjectionDrift(
		ctx,
		projectionReconciliationRepairLimit,
		time.Now(),
	)
	if err != nil {
		logger.L().Warn("authorization projection reconciliation failed", zap.Error(err))
		return
	}
	logger.L().Info(
		"authorization projection reconciliation completed",
		zap.Int("requeued_count", queued),
	)
}

// ReconcileProjectionDrift compares every managed grant with its exact direct
// OpenFGA tuple. It only requeues stale/failed/mismatched grants; unknown tuples
// are never imported into PostgreSQL or deleted by this path.
func (s *Service) ReconcileProjectionDrift(
	ctx context.Context,
	repairLimit int,
	now time.Time,
) (int, error) {
	if s.projection == nil {
		return 0, fmt.Errorf("authorization projection client is required")
	}
	if repairLimit <= 0 {
		repairLimit = projectionReconciliationRepairLimit
	}
	driftedIDs := make([]int64, 0)
	afterID := int64(0)
	for {
		grants, err := s.repo.ListGrantsForReconciliation(
			ctx,
			afterID,
			projectionReconciliationPageSize,
		)
		if err != nil {
			return 0, err
		}
		for _, grant := range grants {
			afterID = grant.ID
			drifted, err := s.authorizationGrantNeedsReconciliation(ctx, grant, now)
			if err != nil {
				return 0, err
			}
			if !drifted {
				continue
			}
			driftedIDs = append(driftedIDs, grant.ID)
			if len(driftedIDs) > repairLimit {
				metrics.ObserveIAMDriftReconciliationThresholdExceeded(
					projectionReconciliationMetricDomain,
				)
				return 0, fmt.Errorf(
					"%w: limit=%d",
					ErrReconciliationLimit,
					repairLimit,
				)
			}
		}
		if len(grants) < projectionReconciliationPageSize {
			break
		}
	}
	if len(driftedIDs) == 0 {
		return 0, nil
	}
	queued, err := s.repo.ReconcileGrantsAsSystem(
		ctx,
		driftedIDs,
		"scheduled OpenFGA authorization projection reconciliation",
	)
	if err != nil {
		return 0, fmt.Errorf("requeue drifted authorization grants: %w", err)
	}
	return queued, nil
}

func (s *Service) authorizationGrantNeedsReconciliation(
	ctx context.Context,
	grant Grant,
	now time.Time,
) (bool, error) {
	switch grant.ProjectionStatus {
	case ProjectionFailed:
		return true, nil
	case ProjectionPending:
		return !grant.UpdatedAt.IsZero() &&
			!now.Before(grant.UpdatedAt.Add(projectionPendingStaleAfter)), nil
	}

	tuple, err := tupleForGrant(grant)
	if err != nil {
		return false, err
	}
	readCtx, cancel := context.WithTimeout(ctx, fga.DefaultWriteTimeout)
	defer cancel()
	exists, err := s.projection.TupleExists(readCtx, tuple)
	if err != nil {
		return false, fmt.Errorf(
			"read authorization tuple for grant %d reconciliation: %w",
			grant.ID,
			err,
		)
	}
	wantExists := grant.DesiredState == DesiredGranted
	return exists != wantExists, nil
}
