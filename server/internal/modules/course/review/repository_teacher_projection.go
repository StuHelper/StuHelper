package review

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/outbox"
)

const (
	teacherPublicStatsRefreshJobType   = "teacher_public_stats_refresh"
	teacherPublicStatsRefreshDedupeKey = "teacher_public_stats"
)

var teacherPublicStatsRefreshJobTypes = []string{teacherPublicStatsRefreshJobType}

type TeacherPublicStatsRefreshJob struct {
	ID           int64
	JobType      string
	AttemptCount int
	LockedAt     time.Time
}

func (r *Repository) EnqueueTeacherPublicStatsRefresh(ctx context.Context) error {
	err := r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return outbox.UpsertJobTx(
			ctx,
			tx,
			outbox.StreamReviewProjection,
			teacherPublicStatsRefreshJobType,
			teacherPublicStatsRefreshDedupeKey,
			[]byte(`{}`),
		)
	})
	if err != nil {
		return fmt.Errorf("EnqueueTeacherPublicStatsRefresh: %w", err)
	}
	return nil
}

func (r *Repository) ClaimTeacherPublicStatsRefreshJobs(
	ctx context.Context,
	limit int,
	staleAfter time.Duration,
) ([]TeacherPublicStatsRefreshJob, error) {
	jobs, err := outbox.ClaimJobsByTypes(
		ctx,
		r.db,
		outbox.StreamReviewProjection,
		teacherPublicStatsRefreshJobTypes,
		limit,
		staleAfter,
	)
	if err != nil {
		return nil, fmt.Errorf("ClaimTeacherPublicStatsRefreshJobs: %w", err)
	}
	result := make([]TeacherPublicStatsRefreshJob, 0, len(jobs))
	for _, job := range jobs {
		result = append(result, TeacherPublicStatsRefreshJob{
			ID:           job.ID,
			JobType:      job.JobType,
			AttemptCount: job.AttemptCount,
			LockedAt:     job.LockedAt,
		})
	}
	return result, nil
}

func (r *Repository) MarkTeacherPublicStatsRefreshJobDone(
	ctx context.Context,
	jobID int64,
	lockedAt time.Time,
) error {
	if err := outbox.MarkJobDone(ctx, r.db, jobID, lockedAt); err != nil {
		return fmt.Errorf("MarkTeacherPublicStatsRefreshJobDone: %w", err)
	}
	return nil
}

func (r *Repository) MarkTeacherPublicStatsRefreshJobFailure(
	ctx context.Context,
	jobID int64,
	lockedAt time.Time,
	nextAttemptAt time.Time,
	lastError string,
	terminal bool,
) error {
	if err := outbox.MarkJobFailure(
		ctx,
		r.db,
		jobID,
		lockedAt,
		nextAttemptAt,
		lastError,
		terminal,
	); err != nil {
		return fmt.Errorf("MarkTeacherPublicStatsRefreshJobFailure: %w", err)
	}
	return nil
}
