package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/outbox"
)

const (
	externalSyncJobTypeVerifiedStudentRole   = "verified_student_role"
	externalSyncJobTypeUserProfileProjection = "user_profile_projection"
	verifiedStudentRoleName                  = "verified_student"

	externalSyncBatchSize = outbox.IAMWorkerBatchSize
	roleSyncTimeout       = 15 * time.Second
)

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
	payload, err := json.Marshal(verifiedStudentRoleSyncPayload{
		UserID:   userID,
		Role:     verifiedStudentRoleName,
		Approved: approved,
	})
	if err != nil {
		return fmt.Errorf("marshal verified student role payload: %w", err)
	}
	return s.repo.UpsertExternalSyncJobTx(ctx, tx, externalSyncJobTypeVerifiedStudentRole, verifiedStudentRoleSyncKey(userID), payload)
}

func (s *Service) enqueueUserProfileProjectionTx(ctx context.Context, tx pgx.Tx, userID int64, approved bool) error {
	payload, err := json.Marshal(userProfileProjectionPayload{
		UserID:   userID,
		Approved: approved,
	})
	if err != nil {
		return fmt.Errorf("marshal user profile projection payload: %w", err)
	}
	return s.repo.UpsertExternalSyncJobTx(ctx, tx, externalSyncJobTypeUserProfileProjection, userProfileProjectionKey(userID), payload)
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

func (s *Service) StartBackgroundJobs(ctx context.Context, start func(string, func(context.Context))) {
	if start == nil {
		go s.runExternalSyncWorker(ctx)
		return
	}
	start("user external sync worker", s.runExternalSyncWorker)
}

func (s *Service) runExternalSyncWorker(ctx context.Context) {
	outbox.RunPollingWorker(
		ctx,
		outbox.IAMWorkerConfig("user external sync"),
		s.repo.ClaimExternalSyncJobs,
		s.processExternalSyncJob,
		s.repo.MarkExternalSyncJobDone,
		s.repo.MarkExternalSyncJobRetry,
		func(job ExternalSyncJob) outbox.JobMeta {
			return outbox.JobMeta{ID: job.ID, JobType: job.JobType, AttemptCount: job.AttemptCount}
		},
		truncateExternalSyncError,
	)
}

func (s *Service) processExternalSyncBatch(ctx context.Context) error {
	return outbox.ProcessBatch(
		ctx,
		outbox.IAMWorkerConfig("user external sync"),
		s.repo.ClaimExternalSyncJobs,
		s.processExternalSyncJob,
		s.repo.MarkExternalSyncJobDone,
		s.repo.MarkExternalSyncJobRetry,
		func(job ExternalSyncJob) outbox.JobMeta {
			return outbox.JobMeta{ID: job.ID, JobType: job.JobType, AttemptCount: job.AttemptCount}
		},
		truncateExternalSyncError,
	)
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
		return errors.New("role sync dependency is not configured")
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
		return errors.New("profile FGA dependency is not configured")
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
