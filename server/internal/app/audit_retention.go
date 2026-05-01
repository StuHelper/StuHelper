package app

import (
	"context"
	"time"

	gozap "go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

const auditRetentionCleanupInterval = 24 * time.Hour

func (rt *Runtime) startAuditRetentionCleanup(ctx context.Context, start func(string, func(context.Context))) {
	repo := audit.NewRepository(rt.database)
	start("iam audit retention cleanup", func(ctx context.Context) {
		runAuditRetentionCleanupLoop(ctx, repo)
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
