package review

import (
	"context"
	"fmt"

	"github.com/StuHelper/StuHelper/server/internal/pkg/outbox"
)

func (h *Handler) runTeacherPublicStatsRefreshWorker(ctx context.Context) {
	cfg := teacherPublicStatsRefreshWorkerConfig()
	outbox.RunPollingWorker(
		ctx,
		cfg,
		h.service.repo.ClaimTeacherPublicStatsRefreshJobs,
		h.processTeacherPublicStatsRefreshJob,
		h.service.repo.MarkTeacherPublicStatsRefreshJobDone,
		h.service.repo.MarkTeacherPublicStatsRefreshJobFailure,
		teacherPublicStatsRefreshJobMeta,
		truncateFGASyncError,
	)
}

func (h *Handler) processTeacherPublicStatsRefreshBatch(ctx context.Context) error {
	cfg := teacherPublicStatsRefreshWorkerConfig()
	return outbox.ProcessBatch(
		ctx,
		cfg,
		h.service.repo.ClaimTeacherPublicStatsRefreshJobs,
		h.processTeacherPublicStatsRefreshJob,
		h.service.repo.MarkTeacherPublicStatsRefreshJobDone,
		h.service.repo.MarkTeacherPublicStatsRefreshJobFailure,
		teacherPublicStatsRefreshJobMeta,
		truncateFGASyncError,
	)
}

func teacherPublicStatsRefreshWorkerConfig() outbox.WorkerConfig {
	cfg := outbox.IAMWorkerConfig("review teacher public stats projection")
	cfg.BatchSize = 1
	return cfg
}

func teacherPublicStatsRefreshJobMeta(job TeacherPublicStatsRefreshJob) outbox.JobMeta {
	return outbox.JobMeta{
		ID:           job.ID,
		JobType:      job.JobType,
		AttemptCount: job.AttemptCount,
		LockedAt:     job.LockedAt,
	}
}

func (h *Handler) processTeacherPublicStatsRefreshJob(
	ctx context.Context,
	job TeacherPublicStatsRefreshJob,
) error {
	if job.JobType != teacherPublicStatsRefreshJobType {
		return fmt.Errorf("unsupported teacher public stats projection job type %q", job.JobType)
	}
	return h.RefreshTeacherPublicStats(ctx)
}
