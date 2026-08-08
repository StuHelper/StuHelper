package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
	"github.com/StuHelper/StuHelper/server/internal/pkg/outbox"
)

const (
	externalSyncJobTypeUserProfileProjection = "user_profile_projection"
	externalSyncJobTypeAdmissionVerification = "admission_verification_projection"
)

type ExternalSyncJob struct {
	ID           int64
	JobType      string
	Payload      json.RawMessage
	AttemptCount int
	LockedAt     time.Time
}

type userProfileProjectionPayload struct {
	UserID   int64 `json:"userID"`
	Approved bool  `json:"approved"`
}

type admissionVerificationProjectionPayload struct {
	UserID   int64 `json:"userID"`
	Approved bool  `json:"approved"`
}

func userProfileProjectionKey(userID int64) string {
	return fmt.Sprintf("user-profile-projection:%d", userID)
}

func admissionVerificationProjectionKey(userID int64) string {
	return fmt.Sprintf("admission-verification-projection:%d", userID)
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

func (s *Service) enqueueAdmissionVerificationProjectionTx(ctx context.Context, tx pgx.Tx, userID int64, approved bool) error {
	payload, err := json.Marshal(admissionVerificationProjectionPayload{
		UserID:   userID,
		Approved: approved,
	})
	if err != nil {
		return fmt.Errorf("marshal admission verification projection payload: %w", err)
	}
	return s.repo.UpsertExternalSyncJobTx(
		ctx,
		tx,
		externalSyncJobTypeAdmissionVerification,
		admissionVerificationProjectionKey(userID),
		payload,
	)
}

func (s *Service) enqueueVerificationProjectionTx(ctx context.Context, tx pgx.Tx, userID int64, status string) error {
	approved := status == StatusVerified
	if err := s.enqueueUserProfileProjectionTx(ctx, tx, userID, approved); err != nil {
		return err
	}
	if status == StatusVerified {
		if err := s.enqueueAdmissionVerificationProjectionTx(ctx, tx, userID, approved); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) StartBackgroundJobs(ctx context.Context, start func(string, func(context.Context))) {
	if start == nil {
		panic("user.Service.StartBackgroundJobs: starter is required")
	}
	start("user external sync worker", s.runExternalSyncWorker)
	start("user external sync reconciliation", s.runExternalSyncReconciliationLoop)
}

func (s *Service) runExternalSyncWorker(ctx context.Context) {
	outbox.RunPollingWorker(
		ctx,
		outbox.IAMWorkerConfig("user external sync"),
		s.repo.ClaimExternalSyncJobs,
		s.processExternalSyncJob,
		s.repo.MarkExternalSyncJobDone,
		s.repo.MarkExternalSyncJobFailure,
		func(job ExternalSyncJob) outbox.JobMeta {
			return outbox.JobMeta{ID: job.ID, JobType: job.JobType, AttemptCount: job.AttemptCount, LockedAt: job.LockedAt}
		},
		truncateExternalSyncError,
	)
}

func (s *Service) processExternalSyncJob(ctx context.Context, job ExternalSyncJob) error {
	switch job.JobType {
	case externalSyncJobTypeUserProfileProjection:
		var payload userProfileProjectionPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("decode user profile projection payload: %w", err)
		}
		return s.syncUserProfileProjection(ctx, payload.UserID)
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

func (s *Service) syncUserProfileProjection(ctx context.Context, userID int64) error {
	if s.profileFGA == nil {
		return errors.New("profile FGA dependency is not configured")
	}

	projectionCtx, cancel := context.WithTimeout(ctx, fga.DefaultWriteTimeout)
	defer cancel()

	if s.verificationStatus == nil {
		return errors.New("student verification status dependency is not configured")
	}
	student, err := s.verificationStatus.GetCurrentStudentStatus(projectionCtx, userID)
	if err != nil {
		return fmt.Errorf("get current student status: %w", err)
	}

	profileObj := "user_profile:" + strconv.FormatInt(userID, 10)
	ownerTuple := fga.Tuple{
		User:     "user:" + strconv.FormatInt(userID, 10),
		Relation: "owner",
		Object:   profileObj,
	}
	desiredOwnerTuples := []fga.Tuple{ownerTuple}
	desiredSchoolTuples := make([]fga.Tuple, 0, 1)
	if student.Eligible {
		if student.SchoolID == nil {
			return fmt.Errorf("eligible student %d has no school ID", userID)
		}
		if err := validateSchoolID(*student.SchoolID); err != nil {
			return fmt.Errorf("eligible student %d has invalid school ID: %w", userID, err)
		}
		desiredSchoolTuples = append(desiredSchoolTuples, fga.Tuple{
			User:     "school:" + strconv.FormatInt(*student.SchoolID, 10),
			Relation: "school",
			Object:   profileObj,
		})
	}

	existingOwnerTuples, err := s.profileFGA.ReadTuples(projectionCtx, profileObj, "owner")
	if err != nil {
		return fmt.Errorf("read existing owner tuples: %w", err)
	}
	existingSchoolTuples, err := s.profileFGA.ReadTuples(projectionCtx, profileObj, "school")
	if err != nil {
		return fmt.Errorf("read existing school tuples: %w", err)
	}

	tuples := fga.MissingTuples(existingOwnerTuples, desiredOwnerTuples)
	tuples = append(tuples, fga.MissingTuples(existingSchoolTuples, desiredSchoolTuples)...)

	staleOwnerTuples := staleFGATuples(existingOwnerTuples, desiredOwnerTuples)
	staleSchoolTuples := staleFGATuples(existingSchoolTuples, desiredSchoolTuples)
	staleTuples := make([]fga.Tuple, 0, len(staleOwnerTuples)+len(staleSchoolTuples))
	staleTuples = append(staleTuples, staleOwnerTuples...)
	staleTuples = append(staleTuples, staleSchoolTuples...)
	if len(staleTuples) > 0 {
		if err := s.profileFGA.DeleteTuples(projectionCtx, staleTuples); err != nil {
			return fmt.Errorf("delete stale projected tuples: %w", err)
		}
	}
	if err := s.profileFGA.WriteTuples(projectionCtx, tuples); err != nil {
		return fmt.Errorf("write projected tuples: %w", err)
	}
	return nil
}

func staleFGATuples(existing, desired []fga.Tuple) []fga.Tuple {
	return fga.MissingTuples(desired, existing)
}
