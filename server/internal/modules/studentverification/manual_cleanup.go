package studentverification

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
)

const (
	manualMaterialCleanupInterval = time.Hour
	manualMaterialCleanupBatch    = 100
)

func (s *Service) StartManualReviewBackgroundJobs(
	ctx context.Context,
	start func(string, func(context.Context)),
) {
	if start == nil {
		panic("studentverification.Service.StartManualReviewBackgroundJobs: starter is required")
	}
	if s.manualMaterialStore == nil {
		return
	}
	start("student verification manual material retention", s.runManualMaterialCleanup)
	start("legacy student verification object purge", s.runLegacyObjectPurge)
}

func (s *Service) runManualMaterialCleanup(ctx context.Context) {
	ticker := time.NewTicker(manualMaterialCleanupInterval)
	defer ticker.Stop()
	for {
		if _, err := s.CleanupExpiredManualReviewMaterials(ctx, manualMaterialCleanupBatch); err != nil {
			logger.FromContext(ctx).Warn("failed to clean expired student review materials", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) CleanupExpiredManualReviewMaterials(
	ctx context.Context,
	limit int,
) (int, error) {
	if s.manualMaterialStore == nil {
		return 0, ErrManualMaterialStoreUnavailable
	}
	if limit < 1 || limit > 500 {
		return 0, ErrManualReviewInvalidForm
	}
	materials, err := s.repo.ListExpiredManualReviewMaterials(ctx, s.now(), limit)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, material := range materials {
		if err := s.manualMaterialStore.DeleteManualReviewMaterial(
			ctx, material.ObjectKey,
		); err != nil {
			return deleted, ErrManualMaterialStoreUnavailable
		}
		if err := s.repo.MarkManualReviewMaterialDeleted(ctx, material, s.now()); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}
