package review

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/outbox"
)

func reviewLikeNotificationKey(reviewID, voterHash string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(voterHash)))
	return "review-like:" + reviewID + ":" + hex.EncodeToString(sum[:8])
}

func reviewReplyNotificationKey(replyID string) string {
	return "review-reply:" + replyID
}

func (s *Service) enqueueVoteNotificationTx(
	ctx context.Context,
	tx pgx.Tx,
	reviewID string,
	voterHash string,
) error {
	target, err := s.repo.GetReviewNotificationTargetTx(ctx, tx, reviewID)
	if err != nil {
		return err
	}
	if target.UserID <= 0 || target.UserHash == voterHash {
		return nil
	}
	idempotencyKey := reviewLikeNotificationKey(reviewID, voterHash)
	return s.enqueueReviewNotificationTx(ctx, tx, ReviewNotification{
		IdempotencyKey: idempotencyKey,
		UserID:         target.UserID,
		Type:           voteTypeLike,
		Title:          "你的评价获得了一个赞",
		Body:           "有人赞了你的评价",
		SourceModule:   "review",
		SourceID:       reviewID,
		CourseID:       target.CourseID,
	})
}

func (s *Service) enqueueReplyNotificationTx(
	ctx context.Context,
	tx pgx.Tx,
	reviewID string,
	replyID string,
	replierHash string,
) error {
	target, err := s.repo.GetReviewNotificationTargetTx(ctx, tx, reviewID)
	if err != nil {
		return err
	}
	if target.UserID <= 0 || target.UserHash == replierHash {
		return nil
	}
	idempotencyKey := reviewReplyNotificationKey(replyID)
	return s.enqueueReviewNotificationTx(ctx, tx, ReviewNotification{
		IdempotencyKey: idempotencyKey,
		UserID:         target.UserID,
		Type:           "reply",
		Title:          "你的评价收到了新回复",
		Body:           "有人回复了你对课程的评价",
		SourceModule:   "review",
		SourceID:       reviewID,
		CourseID:       target.CourseID,
	})
}

func (s *Service) enqueueReviewNotificationTx(
	ctx context.Context,
	tx pgx.Tx,
	notification ReviewNotification,
) error {
	if strings.TrimSpace(notification.IdempotencyKey) == "" || notification.UserID <= 0 {
		return fmt.Errorf("review notification idempotency key and user id are required")
	}
	payload, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("marshal review notification: %w", err)
	}
	return s.repo.UpsertReviewNotificationJobTx(
		ctx,
		tx,
		notification.IdempotencyKey,
		payload,
	)
}

func (s *Service) runReviewNotificationWorker(ctx context.Context) {
	cfg := outbox.IAMWorkerConfig("review notification")
	outbox.RunPollingWorker(
		ctx,
		cfg,
		s.repo.ClaimReviewNotificationJobs,
		s.processReviewNotificationJob,
		s.repo.MarkReviewNotificationJobDone,
		s.repo.MarkReviewNotificationJobFailure,
		reviewNotificationJobMeta,
		truncateFGASyncError,
	)
}

func (s *Service) processReviewNotificationBatch(ctx context.Context) error {
	cfg := outbox.IAMWorkerConfig("review notification")
	return outbox.ProcessBatch(
		ctx,
		cfg,
		s.repo.ClaimReviewNotificationJobs,
		s.processReviewNotificationJob,
		s.repo.MarkReviewNotificationJobDone,
		s.repo.MarkReviewNotificationJobFailure,
		reviewNotificationJobMeta,
		truncateFGASyncError,
	)
}

func reviewNotificationJobMeta(job ReviewNotificationJob) outbox.JobMeta {
	return outbox.JobMeta{
		ID:           job.ID,
		JobType:      job.JobType,
		AttemptCount: job.AttemptCount,
		LockedAt:     job.LockedAt,
	}
}

func (s *Service) processReviewNotificationJob(
	ctx context.Context,
	job ReviewNotificationJob,
) error {
	if job.JobType != reviewNotificationJobType {
		return fmt.Errorf("unsupported review notification job type %q", job.JobType)
	}
	var notification ReviewNotification
	if err := json.Unmarshal(job.Payload, &notification); err != nil {
		return fmt.Errorf("decode review notification payload: %w", err)
	}
	if strings.TrimSpace(notification.IdempotencyKey) == "" || notification.UserID <= 0 {
		return fmt.Errorf("review notification payload is incomplete")
	}
	return s.notifSender.SendReviewNotification(ctx, notification)
}
