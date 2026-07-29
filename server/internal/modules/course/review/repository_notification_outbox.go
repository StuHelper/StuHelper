package review

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/outbox"
)

const reviewNotificationJobType = "send"

var reviewNotificationJobTypes = []string{reviewNotificationJobType}

type ReviewNotificationJob struct {
	ID           int64
	JobType      string
	Payload      json.RawMessage
	AttemptCount int
	LockedAt     time.Time
}

type ReviewNotificationTarget struct {
	UserID   int64
	UserHash string
	CourseID int64
}

func (r *Repository) GetReviewNotificationTargetTx(
	ctx context.Context,
	tx pgx.Tx,
	reviewID string,
) (ReviewNotificationTarget, error) {
	var target ReviewNotificationTarget
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(u.id, 0), r.user_hash, r.course_id
		FROM reviews r
		LEFT JOIN users u ON u.user_hash = r.user_hash
		WHERE r.id = $1
	`, reviewID).Scan(&target.UserID, &target.UserHash, &target.CourseID)
	if err != nil {
		return ReviewNotificationTarget{}, fmt.Errorf("GetReviewNotificationTargetTx: %w", err)
	}
	return target, nil
}

func (r *Repository) UpsertReviewNotificationJobTx(
	ctx context.Context,
	tx pgx.Tx,
	dedupeKey string,
	payload []byte,
) error {
	if err := outbox.UpsertJobTx(
		ctx,
		tx,
		outbox.StreamReviewNotification,
		reviewNotificationJobType,
		dedupeKey,
		payload,
	); err != nil {
		return fmt.Errorf("UpsertReviewNotificationJobTx: %w", err)
	}
	return nil
}

func (r *Repository) ClaimReviewNotificationJobs(
	ctx context.Context,
	limit int,
	staleAfter time.Duration,
) ([]ReviewNotificationJob, error) {
	jobs, err := outbox.ClaimJobsByTypes(
		ctx,
		r.db,
		outbox.StreamReviewNotification,
		reviewNotificationJobTypes,
		limit,
		staleAfter,
	)
	if err != nil {
		return nil, fmt.Errorf("ClaimReviewNotificationJobs: %w", err)
	}
	result := make([]ReviewNotificationJob, 0, len(jobs))
	for _, job := range jobs {
		result = append(result, ReviewNotificationJob{
			ID:           job.ID,
			JobType:      job.JobType,
			Payload:      job.Payload,
			AttemptCount: job.AttemptCount,
			LockedAt:     job.LockedAt,
		})
	}
	return result, nil
}

func (r *Repository) MarkReviewNotificationJobDone(
	ctx context.Context,
	jobID int64,
	lockedAt time.Time,
) error {
	if err := outbox.MarkJobDone(ctx, r.db, jobID, lockedAt); err != nil {
		return fmt.Errorf("MarkReviewNotificationJobDone: %w", err)
	}
	return nil
}

func (r *Repository) MarkReviewNotificationJobFailure(
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
		return fmt.Errorf("MarkReviewNotificationJobFailure: %w", err)
	}
	return nil
}
