package review

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

const (
	fgaReconciliationHour        = 3
	fgaReconciliationRepairLimit = 100
)

var ErrFGAReconciliationThresholdExceeded = errors.New("fga relation reconciliation threshold exceeded")

type ReviewRelationProjectionState struct {
	ReviewID     string
	AuthorUserID int64
	CourseID     int64
	SchoolID     int64
}

type ReportRelationProjectionState struct {
	ReportID       string
	ReporterUserID int64
	ReviewID       string
	SchoolID       int64
}

func (s *Service) runFGASyncReconciliationLoop(ctx context.Context) {
	for {
		delay := nextFGASyncReconciliationDelay(time.Now())
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			s.runFGASyncReconciliation(ctx)
		}
	}
}

func nextFGASyncReconciliationDelay(now time.Time) time.Duration {
	next := time.Date(now.Year(), now.Month(), now.Day(), fgaReconciliationHour, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}

func (s *Service) runFGASyncReconciliation(ctx context.Context) {
	requeued, err := s.ReconcileFGARelationProjections(ctx, fgaReconciliationRepairLimit)
	if err != nil {
		logger.L().Warn("review FGA reconciliation failed", zap.Error(err))
		return
	}
	logger.L().Info("review FGA reconciliation completed", zap.Int("requeued_count", requeued))
}

func (s *Service) ReconcileFGARelationProjections(ctx context.Context, repairLimit int) (int, error) {
	if repairLimit <= 0 {
		repairLimit = fgaReconciliationRepairLimit
	}
	reviews, err := s.repo.ListReviewRelationProjectionStates(ctx, repairLimit+1)
	if err != nil {
		return 0, fmt.Errorf("list review relation projection states: %w", err)
	}
	if len(reviews) > repairLimit {
		return 0, fmt.Errorf("%w: limit=%d", ErrFGAReconciliationThresholdExceeded, repairLimit)
	}
	reports, err := s.listReportRelationProjectionStates(ctx, repairLimit, len(reviews))
	if err != nil {
		return 0, err
	}
	if len(reviews) == 0 && len(reports) == 0 {
		return 0, nil
	}
	requeued, err := s.requeueFGARelationProjectionStates(ctx, reviews, reports)
	if err != nil {
		return 0, err
	}
	s.auditFGARelationReconciliation(ctx, len(reviews), len(reports), requeued)
	return requeued, nil
}

func (s *Service) listReportRelationProjectionStates(
	ctx context.Context,
	repairLimit int,
	reviewCount int,
) ([]ReportRelationProjectionState, error) {
	reports, err := s.repo.ListReportRelationProjectionStates(ctx, repairLimit+1-reviewCount)
	if err != nil {
		return nil, fmt.Errorf("list report relation projection states: %w", err)
	}
	if reviewCount+len(reports) > repairLimit {
		return nil, fmt.Errorf("%w: limit=%d", ErrFGAReconciliationThresholdExceeded, repairLimit)
	}
	return reports, nil
}

func (s *Service) requeueFGARelationProjectionStates(
	ctx context.Context,
	reviews []ReviewRelationProjectionState,
	reports []ReportRelationProjectionState,
) (int, error) {
	requeued := 0
	err := s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		for _, state := range reviews {
			authorUserID, err := formatFGAUserID(state.AuthorUserID)
			if err != nil {
				return err
			}
			if err := s.enqueueReviewFGASyncTx(ctx, tx, state.ReviewID, authorUserID, state.CourseID, state.SchoolID); err != nil {
				return err
			}
			requeued++
		}
		for _, state := range reports {
			reporterUserID, err := formatFGAUserID(state.ReporterUserID)
			if err != nil {
				return err
			}
			if err := s.enqueueReportFGASyncTx(ctx, tx, state.ReportID, reporterUserID, state.ReviewID, state.SchoolID); err != nil {
				return err
			}
			requeued++
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("requeue review FGA relation projections: %w", err)
	}
	return requeued, nil
}

func (s *Service) auditFGARelationReconciliation(
	ctx context.Context,
	reviewCount int,
	reportCount int,
	requeued int,
) {
	if reviewCount == 0 && reportCount == 0 {
		return
	}
	audit.Log(audit.EventFromContext(ctx, audit.Event{
		Type:         audit.EventType("iam.drift.reconcile"),
		Category:     "domain_event",
		ActorType:    "system",
		ResourceType: "openfga.relation",
		ResourceID:   "review_report_relations",
		Action:       "requeue",
		Result:       "success",
		Details: map[string]any{
			"review_count":  reviewCount,
			"report_count":  reportCount,
			"requeued_jobs": requeued,
		},
	}))
}
