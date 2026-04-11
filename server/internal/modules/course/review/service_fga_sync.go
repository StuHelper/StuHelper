package review

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

const (
	fgaSyncJobTypeReviewRelations = "review_relations"
	fgaSyncJobTypeReportRelations = "report_relations"

	fgaSyncBatchSize      = 16
	fgaSyncPollInterval   = 2 * time.Second
	fgaSyncLockStaleAfter = 2 * time.Minute
	fgaSyncMaxBackoff     = 5 * time.Minute
)

type reviewFGAWriter interface {
	WriteReviewRelations(ctx context.Context, reviewID, authorUserID, courseID, schoolID string) error
	WriteReportRelations(ctx context.Context, reportID, reporterUserID, reviewID, schoolID string) error
}

type reviewFGASyncRepo interface {
	UpsertFGASyncJobTx(ctx context.Context, tx pgx.Tx, jobType, dedupeKey string, payload []byte) error
	ClaimFGASyncJobs(ctx context.Context, limit int, staleAfter time.Duration) ([]FGASyncJob, error)
	MarkFGASyncJobDone(ctx context.Context, jobID int64) error
	MarkFGASyncJobRetry(ctx context.Context, jobID int64, nextAttemptAt time.Time, lastError string) error
}

type FGASyncJob struct {
	ID           int64
	JobType      string
	Payload      json.RawMessage
	AttemptCount int
}

type reviewRelationsSyncPayload struct {
	ReviewID     string `json:"reviewID"`
	AuthorUserID string `json:"authorUserID"`
	CourseID     int64  `json:"courseID"`
	SchoolID     int64  `json:"schoolID"`
}

type reportRelationsSyncPayload struct {
	ReportID       string `json:"reportID"`
	ReporterUserID string `json:"reporterUserID"`
	ReviewID       string `json:"reviewID"`
	SchoolID       int64  `json:"schoolID"`
}

func reviewRelationsSyncKey(reviewID string) string {
	return "review-relations:" + reviewID
}

func reportRelationsSyncKey(reportID string) string {
	return "report-relations:" + reportID
}

func (s *Service) SetFGAWriter(writer reviewFGAWriter) {
	s.fgaWriter = writer
}

func (s *Service) enqueueReviewFGASyncTx(ctx context.Context, tx pgx.Tx, reviewID, authorUserID string, courseID, schoolID int64) error {
	if s.fgaWriter == nil {
		return nil
	}
	repo, ok := any(s.repo).(reviewFGASyncRepo)
	if !ok {
		return fmt.Errorf("repository does not support review FGA sync outbox")
	}
	payload, err := json.Marshal(reviewRelationsSyncPayload{
		ReviewID:     reviewID,
		AuthorUserID: authorUserID,
		CourseID:     courseID,
		SchoolID:     schoolID,
	})
	if err != nil {
		return fmt.Errorf("marshal review FGA sync payload: %w", err)
	}
	return repo.UpsertFGASyncJobTx(ctx, tx, fgaSyncJobTypeReviewRelations, reviewRelationsSyncKey(reviewID), payload)
}

func (s *Service) enqueueReportFGASyncTx(ctx context.Context, tx pgx.Tx, reportID, reporterUserID, reviewID string, schoolID int64) error {
	if s.fgaWriter == nil {
		return nil
	}
	repo, ok := any(s.repo).(reviewFGASyncRepo)
	if !ok {
		return fmt.Errorf("repository does not support review FGA sync outbox")
	}
	payload, err := json.Marshal(reportRelationsSyncPayload{
		ReportID:       reportID,
		ReporterUserID: reporterUserID,
		ReviewID:       reviewID,
		SchoolID:       schoolID,
	})
	if err != nil {
		return fmt.Errorf("marshal report FGA sync payload: %w", err)
	}
	return repo.UpsertFGASyncJobTx(ctx, tx, fgaSyncJobTypeReportRelations, reportRelationsSyncKey(reportID), payload)
}

func (s *Service) StartBackgroundJobs(ctx context.Context) {
	if s.fgaWriter == nil {
		return
	}
	if _, ok := any(s.repo).(reviewFGASyncRepo); !ok {
		logger.L().Warn("review FGA sync worker disabled: repository does not support outbox")
		return
	}

	go s.runFGASyncWorker(ctx)
}

func (s *Service) runFGASyncWorker(ctx context.Context) {
	ticker := time.NewTicker(fgaSyncPollInterval)
	defer ticker.Stop()

	for {
		if err := s.processFGASyncBatch(ctx); err != nil && ctx.Err() == nil {
			logger.L().Warn("review FGA sync batch failed", zap.Error(err))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) processFGASyncBatch(ctx context.Context) error {
	repo, ok := any(s.repo).(reviewFGASyncRepo)
	if !ok {
		return nil
	}

	jobs, err := repo.ClaimFGASyncJobs(ctx, fgaSyncBatchSize, fgaSyncLockStaleAfter)
	if err != nil {
		return fmt.Errorf("claim review FGA sync jobs: %w", err)
	}

	for _, job := range jobs {
		if err := s.processFGASyncJob(ctx, job); err != nil {
			backoff := time.Duration(job.AttemptCount+1) * 5 * time.Second
			if backoff > fgaSyncMaxBackoff {
				backoff = fgaSyncMaxBackoff
			}
			nextAttemptAt := time.Now().Add(backoff)
			markErr := repo.MarkFGASyncJobRetry(ctx, job.ID, nextAttemptAt, truncateFGASyncError(err))
			if markErr != nil {
				logger.L().Error("failed to mark review FGA sync job retry",
					zap.Int64("job_id", job.ID),
					zap.String("job_type", job.JobType),
					zap.Error(markErr),
				)
			}
			logger.L().Warn("review FGA sync job failed",
				zap.Int64("job_id", job.ID),
				zap.String("job_type", job.JobType),
				zap.Int("attempt", job.AttemptCount+1),
				zap.Time("next_attempt_at", nextAttemptAt),
				zap.Error(err),
			)
			continue
		}

		if err := repo.MarkFGASyncJobDone(ctx, job.ID); err != nil {
			return fmt.Errorf("mark review FGA sync job done: %w", err)
		}
	}

	return nil
}

func (s *Service) processFGASyncJob(ctx context.Context, job FGASyncJob) error {
	if s.fgaWriter == nil {
		return nil
	}

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
			strconv.FormatInt(payload.CourseID, 10),
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
			payload.ReporterUserID,
			payload.ReviewID,
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
