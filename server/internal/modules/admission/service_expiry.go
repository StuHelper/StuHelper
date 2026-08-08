package admission

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/outbox"
)

func (s *Service) StartBackgroundJobs(ctx context.Context, start func(string, func(context.Context))) {
	if start == nil {
		panic("admission.Service.StartBackgroundJobs: starter is required")
	}
	start("admission member blacklist expiry worker", s.runMemberBlacklistExpiryWorker)
}

func (s *Service) runMemberBlacklistExpiryWorker(ctx context.Context) {
	ticker := time.NewTicker(outbox.IAMWorkerPollInterval)
	defer ticker.Stop()
	for {
		s.runMemberBlacklistExpiryBatch(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) runMemberBlacklistExpiryBatch(ctx context.Context) {
	processed, err := s.ProcessExpiredMemberBlacklists(ctx)
	if err != nil && ctx.Err() == nil {
		logger.L().Warn("admission member blacklist expiry batch failed", zap.Error(err))
		return
	}
	if processed > 0 {
		logger.L().Info("admission member blacklist expiry batch completed", zap.Int("processed_count", processed))
	}
}

func (s *Service) ProcessExpiredMemberBlacklists(ctx context.Context) (int, error) {
	items, err := s.repo.ListExpiredMemberBlacklist(ctx, s.now(), outbox.IAMWorkerBatchSize)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, item := range items {
		updated, err := s.processExpiredMemberBlacklist(ctx, item)
		if err != nil {
			return processed, err
		}
		if updated {
			processed++
		}
	}
	return processed, nil
}

func (s *Service) processExpiredMemberBlacklist(ctx context.Context, item MemberBlacklistEntry) (bool, error) {
	input := MemberBlacklistReleaseInput{
		ID:                item.ID,
		ReleasedByType:    BlacklistActorSystem,
		ReleasedByID:      "system",
		ReleaseReasonCode: BlacklistReleasePolicyExpiredAuto,
	}
	updated := false
	err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		entry, err := s.repo.ReleaseMemberBlacklistByIDTx(ctx, tx, input, s.now())
		if errors.Is(err, ErrMemberBlacklistNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		updated = true
		return s.afterMemberBlacklistReleaseTx(ctx, tx, entry, input.ReleaseReasonCode)
	})
	return updated, err
}
