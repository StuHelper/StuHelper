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
	start("admission member blacklist expiry worker", s.runMemberBlacklistExpiryWorker)
	start("admission bot action outbox cleanup worker", s.runOutboxCleanupWorker)
}

// 终态 outbox 行的保留期与清理频率：保留 7 天用于排障/审计，每小时清理一次。
const (
	admissionBotActionRetention    = 7 * 24 * time.Hour
	admissionOutboxCleanupInterval = time.Hour
)

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

func (s *Service) runOutboxCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(admissionOutboxCleanupInterval)
	defer ticker.Stop()
	for {
		s.runOutboxCleanupBatch(ctx)
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

func (s *Service) runOutboxCleanupBatch(ctx context.Context) {
	deleted, err := s.PruneTerminalBotActions(ctx)
	if err != nil && ctx.Err() == nil {
		logger.L().Warn("admission outbox cleanup batch failed", zap.Error(err))
		return
	}
	if deleted > 0 {
		logger.L().Info("admission outbox cleanup batch completed", zap.Int64("deleted_count", deleted))
	}
}

// PruneTerminalBotActions 删除保留期之前的终态 outbox 行，返回删除行数。
func (s *Service) PruneTerminalBotActions(ctx context.Context) (int64, error) {
	return s.repo.PruneTerminalBotActions(ctx, s.now().Add(-admissionBotActionRetention))
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
		if err := s.reprojectUserProfileAfterFreshmanExpiryTx(ctx, tx, item.UserID); err != nil {
			return err
		}
		if err := s.projection.EnqueueFreshmanProvisionalRoleSyncTx(ctx, tx, item.UserID, false); err != nil {
			return fmt.Errorf("enqueue freshman provisional role removal: %w", err)
		}
		return s.repo.InsertAuditEventTx(ctx, tx, freshmanExpiryAuditEvent(ctx, item))
	})
}

// reprojectUserProfileAfterFreshmanExpiryTx 在新生临时凭证过期后修正 user_profiles 投影：
// 若用户仍持有其他有效凭证，则以该凭证重投影（保持 verified，刷新来源/学校）；
// 否则降级为 unverified，并入队 approved=false 的投影/角色同步任务。
func (s *Service) reprojectUserProfileAfterFreshmanExpiryTx(
	ctx context.Context,
	tx pgx.Tx,
	userID int64,
) error {
	remaining, err := s.repo.GetLatestCredentialForUserTx(ctx, tx, userID, s.now())
	if err != nil {
		return fmt.Errorf("load remaining credential after freshman expiry: %w", err)
	}
	if remaining != nil {
		return s.repo.ProjectVerifiedUserProfileTx(ctx, tx, *remaining)
	}
	return s.repo.DowngradeVerifiedUserProfileTx(ctx, tx, userID)
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
