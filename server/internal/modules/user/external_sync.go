package user

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
	externalSyncJobTypeVerifiedStudentRole   = "verified_student_role"
	externalSyncJobTypeUserProfileProjection = "user_profile_projection"
	verifiedStudentRoleName                  = "verified_student"

	externalSyncBatchSize      = 16
	externalSyncPollInterval   = 2 * time.Second
	externalSyncLockStaleAfter = 2 * time.Minute
	externalSyncMaxBackoff     = 5 * time.Minute
	roleSyncTimeout            = 15 * time.Second
)

type externalSyncRepo interface {
	UpsertExternalSyncJobTx(ctx context.Context, tx pgx.Tx, jobType, dedupeKey string, payload []byte) error
	ClaimExternalSyncJobs(ctx context.Context, limit int, staleAfter time.Duration) ([]ExternalSyncJob, error)
	MarkExternalSyncJobDone(ctx context.Context, jobID int64) error
	MarkExternalSyncJobRetry(ctx context.Context, jobID int64, nextAttemptAt time.Time, lastError string) error
}

type ExternalSyncJob struct {
	ID           int64
	JobType      string
	Payload      json.RawMessage
	AttemptCount int
}

type verifiedStudentRoleSyncPayload struct {
	UserID   int64  `json:"userID"`
	Role     string `json:"role"`
	Approved bool   `json:"approved"`
}

type userProfileProjectionPayload struct {
	UserID   int64 `json:"userID"`
	Approved bool  `json:"approved"`
}

func verifiedStudentRoleSyncKey(userID int64) string {
	return fmt.Sprintf("verified-student-role:%d", userID)
}

func userProfileProjectionKey(userID int64) string {
	return fmt.Sprintf("user-profile-projection:%d", userID)
}

func (s *Service) enqueueVerifiedStudentRoleSyncTx(ctx context.Context, tx pgx.Tx, userID int64, approved bool) error {
	repo, ok := s.repo.(externalSyncRepo)
	if !ok {
		return fmt.Errorf("repository does not support external sync outbox")
	}
	payload, err := json.Marshal(verifiedStudentRoleSyncPayload{
		UserID:   userID,
		Role:     verifiedStudentRoleName,
		Approved: approved,
	})
	if err != nil {
		return fmt.Errorf("marshal verified student role payload: %w", err)
	}
	return repo.UpsertExternalSyncJobTx(ctx, tx, externalSyncJobTypeVerifiedStudentRole, verifiedStudentRoleSyncKey(userID), payload)
}

func (s *Service) enqueueUserProfileProjectionTx(ctx context.Context, tx pgx.Tx, userID int64, approved bool) error {
	repo, ok := s.repo.(externalSyncRepo)
	if !ok {
		return fmt.Errorf("repository does not support external sync outbox")
	}
	payload, err := json.Marshal(userProfileProjectionPayload{
		UserID:   userID,
		Approved: approved,
	})
	if err != nil {
		return fmt.Errorf("marshal user profile projection payload: %w", err)
	}
	return repo.UpsertExternalSyncJobTx(ctx, tx, externalSyncJobTypeUserProfileProjection, userProfileProjectionKey(userID), payload)
}

func (s *Service) enqueueVerificationProjectionTx(ctx context.Context, tx pgx.Tx, userID int64, status string) error {
	approved := status == StatusVerified
	if err := s.enqueueUserProfileProjectionTx(ctx, tx, userID, approved); err != nil {
		return err
	}
	if status == StatusVerified || status == StatusRejected {
		if err := s.enqueueVerifiedStudentRoleSyncTx(ctx, tx, userID, approved); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) StartBackgroundJobs(ctx context.Context) {
	if _, ok := s.repo.(externalSyncRepo); !ok {
		logger.L().Warn("user external sync worker disabled: repository does not support outbox")
		return
	}

	go s.runExternalSyncWorker(ctx)
}

func (s *Service) runExternalSyncWorker(ctx context.Context) {
	ticker := time.NewTicker(externalSyncPollInterval)
	defer ticker.Stop()

	for {
		if err := s.processExternalSyncBatch(ctx); err != nil && ctx.Err() == nil {
			logger.L().Warn("user external sync batch failed", zap.Error(err))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) processExternalSyncBatch(ctx context.Context) error {
	repo, ok := s.repo.(externalSyncRepo)
	if !ok {
		return nil
	}

	jobs, err := repo.ClaimExternalSyncJobs(ctx, externalSyncBatchSize, externalSyncLockStaleAfter)
	if err != nil {
		return fmt.Errorf("claim external sync jobs: %w", err)
	}
	for _, job := range jobs {
		if err := s.processExternalSyncJob(ctx, job); err != nil {
			backoff := time.Duration(job.AttemptCount+1) * 5 * time.Second
			if backoff > externalSyncMaxBackoff {
				backoff = externalSyncMaxBackoff
			}
			nextAttemptAt := time.Now().Add(backoff)
			markErr := repo.MarkExternalSyncJobRetry(ctx, job.ID, nextAttemptAt, truncateExternalSyncError(err))
			if markErr != nil {
				logger.L().Error("failed to mark external sync job retry",
					zap.Int64("job_id", job.ID),
					zap.String("job_type", job.JobType),
					zap.Error(markErr),
				)
			}
			logger.L().Warn("external sync job failed",
				zap.Int64("job_id", job.ID),
				zap.String("job_type", job.JobType),
				zap.Int("attempt", job.AttemptCount+1),
				zap.Time("next_attempt_at", nextAttemptAt),
				zap.Error(err),
			)
			continue
		}

		if err := repo.MarkExternalSyncJobDone(ctx, job.ID); err != nil {
			return fmt.Errorf("mark external sync job done: %w", err)
		}
	}
	return nil
}

func (s *Service) processExternalSyncJob(ctx context.Context, job ExternalSyncJob) error {
	switch job.JobType {
	case externalSyncJobTypeVerifiedStudentRole:
		var payload verifiedStudentRoleSyncPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("decode verified student role payload: %w", err)
		}
		if payload.Role == "" {
			payload.Role = verifiedStudentRoleName
		}
		return s.syncVerifiedStudentRole(ctx, payload.UserID, payload.Role, payload.Approved)
	case externalSyncJobTypeUserProfileProjection:
		var payload userProfileProjectionPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("decode user profile projection payload: %w", err)
		}
		return s.syncUserProfileProjection(ctx, payload.UserID, payload.Approved)
	default:
		return fmt.Errorf("unsupported external sync job type: %s", job.JobType)
	}
}

func truncateExternalSyncError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if len(msg) <= 1000 {
		return msg
	}
	return msg[:1000]
}

func (s *Service) syncVerifiedStudentRole(ctx context.Context, userID int64, role string, approved bool) error {
	if s.onRoleSync == nil {
		return nil
	}

	roleCtx, cancel := context.WithTimeout(ctx, roleSyncTimeout)
	defer cancel()

	if err := s.onRoleSync(roleCtx, userID, role, approved); err != nil {
		return fmt.Errorf("role sync: %w", err)
	}
	return nil
}

func (s *Service) syncUserProfileProjection(ctx context.Context, userID int64, approved bool) error {
	if s.profileFGA == nil {
		return nil
	}

	projectionCtx, cancel := context.WithTimeout(ctx, fga.DefaultWriteTimeout)
	defer cancel()

	externalID, err := s.repo.GetExternalID(projectionCtx, userID)
	if err != nil {
		return fmt.Errorf("get external ID: %w", err)
	}

	profile, err := s.repo.GetProfileByUserID(projectionCtx, userID)
	if err != nil {
		return fmt.Errorf("get profile: %w", err)
	}
	if profile == nil {
		return nil
	}

	profileObj := "user_profile:" + strconv.FormatInt(userID, 10)
	existingSchoolTuples, err := s.profileFGA.ReadTuples(projectionCtx, profileObj, "school")
	if err != nil {
		return fmt.Errorf("read existing school tuples: %w", err)
	}
	if len(existingSchoolTuples) > 0 {
		if err := s.profileFGA.DeleteTuples(projectionCtx, existingSchoolTuples); err != nil {
			return fmt.Errorf("delete existing school tuples: %w", err)
		}
	}

	tuples := []fga.Tuple{{
		User:     "user:" + externalID,
		Relation: "owner",
		Object:   profileObj,
	}}
	if approved {
		if profile.SchoolID == nil {
			return fmt.Errorf("verified profile %d has no school ID", userID)
		}
		tuples = append(tuples, fga.Tuple{
			User:     "school:" + strconv.FormatInt(*profile.SchoolID, 10),
			Relation: "school",
			Object:   profileObj,
		})
	}

	if err := s.profileFGA.WriteTuples(projectionCtx, tuples); err != nil {
		return fmt.Errorf("write projected tuples: %w", err)
	}
	return nil
}
