package admission

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/outbox"
)

func (s *Service) StartBackgroundJobs(ctx context.Context, start func(string, func(context.Context))) {
	if start == nil {
		panic("admission.Service.StartBackgroundJobs: starter is required")
	}
	start("admission freshman expiry worker", s.runFreshmanExpiryWorker)
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
	if processed > 0 {
		auditFreshmanExpiry(ctx, processed)
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
		return nil
	})
}

func auditFreshmanExpiry(ctx context.Context, processed int) {
	audit.Log(audit.EventFromContext(ctx, audit.Event{
		Type:         audit.EventType("admission.freshman.expire"),
		Category:     "domain_event",
		ActorType:    "system",
		ResourceType: "admission.freshman_credential",
		ResourceID:   "freshman_provisional",
		Action:       "expire",
		Result:       "success",
		Details:      map[string]any{"processed_count": processed},
	}))
}
