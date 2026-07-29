package review

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
	"github.com/StuHelper/StuHelper/server/internal/pkg/outbox"
)

const (
	fgaSyncJobTypeReviewRelations = "review_relations"
	fgaSyncJobTypeReportRelations = "report_relations"
)

type reviewFGAWriter interface {
	WriteReviewRelations(ctx context.Context, reviewID, authorUserID, schoolID string) error
	WriteReportRelations(ctx context.Context, reportID, schoolID string) error
}

type FGASyncJob struct {
	ID           int64
	JobType      string
	Payload      json.RawMessage
	AttemptCount int
	LockedAt     time.Time
}

type reviewRelationsSyncPayload struct {
	ReviewID     string `json:"reviewID"`
	AuthorUserID string `json:"authorUserID,omitempty"`
	SchoolID     int64  `json:"schoolID"`
}

type reportRelationsSyncPayload struct {
	ReportID string `json:"reportID"`
	SchoolID int64  `json:"schoolID"`
}

func reviewRelationsSyncKey(reviewID string) string {
	return "review-relations:" + reviewID
}

func reportRelationsSyncKey(reportID string) string {
	return "report-relations:" + reportID
}

func formatFGAUserID(userID int64) (string, error) {
	if userID <= 0 {
		return "", ErrUserIdentityRequired
	}
	return strconv.FormatInt(userID, 10), nil
}

func (s *Service) enqueueReviewFGASyncTx(ctx context.Context, tx pgx.Tx, reviewID, authorUserID string, schoolID int64) error {
	payload, err := json.Marshal(reviewRelationsSyncPayload{
		ReviewID:     reviewID,
		AuthorUserID: authorUserID,
		SchoolID:     schoolID,
	})
	if err != nil {
		return fmt.Errorf("marshal review FGA sync payload: %w", err)
	}
	return s.repo.UpsertFGASyncJobTx(ctx, tx, fgaSyncJobTypeReviewRelations, reviewRelationsSyncKey(reviewID), payload)
}

func (s *Service) enqueueReportFGASyncTx(ctx context.Context, tx pgx.Tx, reportID string, schoolID int64) error {
	payload, err := json.Marshal(reportRelationsSyncPayload{
		ReportID: reportID,
		SchoolID: schoolID,
	})
	if err != nil {
		return fmt.Errorf("marshal report FGA sync payload: %w", err)
	}
	return s.repo.UpsertFGASyncJobTx(ctx, tx, fgaSyncJobTypeReportRelations, reportRelationsSyncKey(reportID), payload)
}

func (s *Service) StartBackgroundJobs(ctx context.Context, start func(string, func(context.Context))) {
	if start == nil {
		panic("review.Service.StartBackgroundJobs: starter is required")
	}
	s.asyncCtx = ctx
	s.asyncLaunch = start
	start("review fga sync worker", s.runFGASyncWorker)
	start("review fga sync reconciliation", s.runFGASyncReconciliationLoop)
}

func (s *Service) runFGASyncWorker(ctx context.Context) {
	outbox.RunPollingWorker(
		ctx,
		outbox.IAMWorkerConfig("review FGA sync"),
		s.repo.ClaimFGASyncJobs,
		s.processFGASyncJob,
		s.repo.MarkFGASyncJobDone,
		s.repo.MarkFGASyncJobFailure,
		func(job FGASyncJob) outbox.JobMeta {
			return outbox.JobMeta{ID: job.ID, JobType: job.JobType, AttemptCount: job.AttemptCount, LockedAt: job.LockedAt}
		},
		truncateFGASyncError,
	)
}

func (s *Service) processFGASyncBatch(ctx context.Context) error {
	return outbox.ProcessBatch(
		ctx,
		outbox.IAMWorkerConfig("review FGA sync"),
		s.repo.ClaimFGASyncJobs,
		s.processFGASyncJob,
		s.repo.MarkFGASyncJobDone,
		s.repo.MarkFGASyncJobFailure,
		func(job FGASyncJob) outbox.JobMeta {
			return outbox.JobMeta{ID: job.ID, JobType: job.JobType, AttemptCount: job.AttemptCount, LockedAt: job.LockedAt}
		},
		truncateFGASyncError,
	)
}

func (s *Service) processFGASyncJob(ctx context.Context, job FGASyncJob) error {
	switch job.JobType {
	case fgaSyncJobTypeReviewRelations:
		var payload reviewRelationsSyncPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("decode review relations payload: %w", err)
		}
		writeCtx, cancel := context.WithTimeout(ctx, fga.DefaultWriteTimeout)
		defer cancel()
		return s.fgaWriter.WriteReviewRelations(
			writeCtx,
			payload.ReviewID,
			payload.AuthorUserID,
			strconv.FormatInt(payload.SchoolID, 10),
		)
	case fgaSyncJobTypeReportRelations:
		var payload reportRelationsSyncPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("decode report relations payload: %w", err)
		}
		writeCtx, cancel := context.WithTimeout(ctx, fga.DefaultWriteTimeout)
		defer cancel()
		return s.fgaWriter.WriteReportRelations(
			writeCtx,
			payload.ReportID,
			strconv.FormatInt(payload.SchoolID, 10),
		)
	default:
		return fmt.Errorf("unsupported review FGA sync job type: %s", job.JobType)
	}
}

func truncateFGASyncError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if len(msg) <= 1000 {
		return msg
	}
	return msg[:1000]
}
