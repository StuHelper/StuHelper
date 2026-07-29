package admission

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/outbox"
)

func (s *Service) StartBackgroundJobs(ctx context.Context, start func(string, func(context.Context))) {
	if start == nil {
		panic("admission.Service.StartBackgroundJobs: starter is required")
	}
	start("admission freshman expiry worker", s.runFreshmanExpiryWorker)
	start("admission member blacklist expiry worker", s.runMemberBlacklistExpiryWorker)
}

func (s *Service) runFreshmanExpiryWorker(ctx context.Context) {
	ticker := time.NewTicker(outbox.IAMWorkerPollInterval)
	defer ticker.Stop()
	for {
		s.runFreshmanExpiryBatch(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
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

func (s *Service) runFreshmanExpiryBatch(ctx context.Context) {
	processed, err := s.ProcessExpiredFreshmanCredentials(ctx)
	if err != nil && ctx.Err() == nil {
		logger.L().Warn("admission freshman expiry batch failed", zap.Error(err))
		return
	}
	if processed > 0 {
		logger.L().Info("admission freshman expiry batch completed", zap.Int("processed_count", processed))
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

func (s *Service) ProcessExpiredFreshmanCredentials(ctx context.Context) (int, error) {
	if s.projection == nil {
		return 0, ErrAdmissionProjectionUnavailable
	}
	items, err := s.repo.ListExpiredFreshmanCredentials(ctx, s.now(), outbox.IAMWorkerBatchSize)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, item := range items {
		if err := s.processExpiredFreshmanCredential(ctx, item); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (s *Service) ProcessExpiredMemberBlacklists(ctx context.Context) (int, error) {
	items, err := s.repo.ListExpiredMemberBlacklist(ctx, s.now(), outbox.IAMWorkerBatchSize)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, item := range items {
		if err := s.processExpiredMemberBlacklist(ctx, item); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (s *Service) processExpiredFreshmanCredential(
	ctx context.Context,
	item ExpiredFreshmanCredential,
) error {
	return s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.MarkFreshmanCredentialExpiryProcessedTx(ctx, tx, item.ID, s.now()); err != nil {
			return err
		}
		if err := s.projection.EnqueueFreshmanProvisionalRoleSyncTx(ctx, tx, item.UserID, false); err != nil {
			return fmt.Errorf("enqueue freshman provisional role removal: %w", err)
		}
		return s.repo.InsertAuditEventTx(ctx, tx, freshmanExpiryAuditEvent(ctx, item))
	})
}

func (s *Service) processExpiredMemberBlacklist(ctx context.Context, item MemberBlacklistEntry) error {
	input := MemberBlacklistReleaseInput{
		ID:                item.ID,
		ReleasedByType:    BlacklistActorSystem,
		ReleasedByID:      "system",
		ReleaseReasonCode: BlacklistReleasePolicyExpiredAuto,
	}
	return s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		entry, err := s.repo.ReleaseMemberBlacklistByIDTx(ctx, tx, input, s.now())
		if err != nil {
			return err
		}
		return s.afterMemberBlacklistReleaseTx(ctx, tx, entry, input.ReleaseReasonCode)
	})
}

func freshmanExpiryAuditEvent(ctx context.Context, item ExpiredFreshmanCredential) audit.Event {
	return audit.EventFromContext(ctx, audit.Event{
		Type:         audit.EventType("admission.freshman.expire"),
		Category:     "domain_event",
		ActorType:    "system",
		ResourceType: "admission.freshman_credential",
		ResourceID:   item.ID,
		Action:       "expire",
		Result:       "success",
		Details:      map[string]any{"user_id": item.UserID, "expires_at": item.ExpiresAt},
	})
}
