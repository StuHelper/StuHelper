package app

import (
	"context"
	"time"

	gozap "go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/modules/openplatform"
	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
)

const auditRetentionCleanupInterval = 24 * time.Hour

func (rt *Runtime) startAuditRetentionCleanup(ctx context.Context, start func(string, func(context.Context))) {
	iamRepo := audit.NewRepository(rt.database)
	start("iam audit retention cleanup", func(ctx context.Context) {
		runAuditRetentionCleanupLoop(ctx, iamRepo)
	})

	openPlatformRepo := openplatform.NewRepository(rt.database)
	start("open platform audit retention cleanup", func(ctx context.Context) {
		runOpenPlatformAuditRetentionCleanupLoop(ctx, openPlatformRepo)
	})
}

func runAuditRetentionCleanupLoop(ctx context.Context, repo *audit.Repository) {
	runAuditRetentionCleanup(ctx, repo)

	ticker := time.NewTicker(auditRetentionCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runAuditRetentionCleanup(ctx, repo)
		}
	}
}

func runAuditRetentionCleanup(ctx context.Context, repo *audit.Repository) {
	deleted, err := repo.CleanupIAMEvents(ctx, audit.IAMRetentionPolicy{})
	if err != nil {
		logger.L().Warn("iam audit retention cleanup failed", gozap.Error(err))
		return
	}
	logger.L().Info("iam audit retention cleanup completed", gozap.Int64("deleted_count", deleted))
}

func runOpenPlatformAuditRetentionCleanupLoop(ctx context.Context, repo *openplatform.Repository) {
	runOpenPlatformAuditRetentionCleanup(ctx, repo)

	ticker := time.NewTicker(auditRetentionCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runOpenPlatformAuditRetentionCleanup(ctx, repo)
		}
	}
}

func runOpenPlatformAuditRetentionCleanup(ctx context.Context, repo *openplatform.Repository) {
	deleted, err := repo.CleanupAuditEvents(ctx, openplatform.AuditRetentionPolicy{})
	if err != nil {
		logger.L().Warn("open platform audit retention cleanup failed", gozap.Error(err))
		return
	}
	logger.L().Info("open platform audit retention cleanup completed", gozap.Int64("deleted_count", deleted))
}
