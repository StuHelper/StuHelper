package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
	"github.com/StuHelper/StuHelper/server/internal/pkg/outbox"
)

const (
	externalSyncJobTypeVerifiedStudentRole     = "verified_student_role"
	externalSyncJobTypeFreshmanProvisionalRole = "freshman_provisional_role"
	externalSyncJobTypeUserProfileProjection   = "user_profile_projection"
	externalSyncJobTypeAdmissionVerification   = "admission_verification_projection"
	verifiedStudentRoleName                    = "verified_student"
	freshmanProvisionalRoleName                = "freshman_provisional"

	externalSyncBatchSize = outbox.IAMWorkerBatchSize
	roleSyncTimeout       = 15 * time.Second
)

type ExternalSyncJob struct {
	ID           int64
	JobType      string
	Payload      json.RawMessage
	AttemptCount int
	LockedAt     time.Time
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

type admissionVerificationProjectionPayload struct {
	UserID   int64 `json:"userID"`
	Approved bool  `json:"approved"`
}

func roleSyncKey(role string, userID int64) string {
	return fmt.Sprintf("%s-role:%d", strings.ReplaceAll(role, "_", "-"), userID)
}

func verifiedStudentRoleSyncKey(userID int64) string {
	return roleSyncKey(verifiedStudentRoleName, userID)
}

func freshmanProvisionalRoleSyncKey(userID int64) string {
	return roleSyncKey(freshmanProvisionalRoleName, userID)
}

func userProfileProjectionKey(userID int64) string {
	return fmt.Sprintf("user-profile-projection:%d", userID)
}

func admissionVerificationProjectionKey(userID int64) string {
	return fmt.Sprintf("admission-verification-projection:%d", userID)
}

func (s *Service) enqueueVerifiedStudentRoleSyncTx(ctx context.Context, tx pgx.Tx, userID int64, approved bool) error {
	return s.enqueueRoleSyncTx(ctx, roleSyncInput{
		Tx: tx, UserID: userID, Role: verifiedStudentRoleName, Approved: approved,
	})
}

func (s *Service) EnqueueFreshmanProvisionalRoleSyncTx(
	ctx context.Context,
	tx pgx.Tx,
	userID int64,
	approved bool,
) error {
	return s.enqueueRoleSyncTx(ctx, roleSyncInput{
		Tx: tx, UserID: userID, Role: freshmanProvisionalRoleName, Approved: approved,
	})
}

func (s *Service) enqueueRoleSyncTx(ctx context.Context, input roleSyncInput) error {
	payload, err := json.Marshal(verifiedStudentRoleSyncPayload{
		UserID:   input.UserID,
		Role:     input.Role,
		Approved: input.Approved,
	})
	if err != nil {
		return fmt.Errorf("marshal role sync payload: %w", err)
	}
	jobType, key := roleSyncJobTypeAndKey(input.Role, input.UserID)
	return s.repo.UpsertExternalSyncJobTx(ctx, input.Tx, jobType, key, payload)
}

type roleSyncInput struct {
	Tx       pgx.Tx
	UserID   int64
	Role     string
	Approved bool
}

func roleSyncJobTypeAndKey(role string, userID int64) (string, string) {
	if role == freshmanProvisionalRoleName {
		return externalSyncJobTypeFreshmanProvisionalRole, freshmanProvisionalRoleSyncKey(userID)
	}
	return externalSyncJobTypeVerifiedStudentRole, verifiedStudentRoleSyncKey(userID)
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
	if status == StatusVerified || status == StatusRejected {
		if err := s.enqueueVerifiedStudentRoleSyncTx(ctx, tx, userID, approved); err != nil {
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

func (s *Service) processExternalSyncBatch(ctx context.Context) error {
	return outbox.ProcessBatch(
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
	case externalSyncJobTypeVerifiedStudentRole:
		var payload verifiedStudentRoleSyncPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("decode role sync payload: %w", err)
		}
		payload.Role = defaultRoleSyncPayloadRole(job.JobType, payload.Role)
		approved, err := s.currentProfileApproved(ctx, payload.UserID)
		if err != nil {
			return err
		}
		return s.syncVerifiedStudentRole(ctx, payload.UserID, payload.Role, approved)
	case externalSyncJobTypeFreshmanProvisionalRole:
		var payload verifiedStudentRoleSyncPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("decode role sync payload: %w", err)
		}
		payload.Role = defaultRoleSyncPayloadRole(job.JobType, payload.Role)
		return s.syncVerifiedStudentRole(ctx, payload.UserID, payload.Role, payload.Approved)
	case externalSyncJobTypeUserProfileProjection:
		var payload userProfileProjectionPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("decode user profile projection payload: %w", err)
		}
		return s.syncUserProfileProjection(ctx, payload.UserID)
	case externalSyncJobTypeAdmissionVerification:
		var payload admissionVerificationProjectionPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("decode admission verification projection payload: %w", err)
		}
		return s.syncAdmissionVerificationProjection(ctx, payload.UserID)
	default:
		return fmt.Errorf("unsupported external sync job type: %s", job.JobType)
	}
}

func defaultRoleSyncPayloadRole(jobType string, role string) string {
	if strings.TrimSpace(role) != "" {
		return role
	}
	if jobType == externalSyncJobTypeFreshmanProvisionalRole {
		return freshmanProvisionalRoleName
	}
	return verifiedStudentRoleName
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

func (s *Service) currentProfileApproved(ctx context.Context, userID int64) (bool, error) {
	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("get profile: %w", err)
	}
	return profile != nil && profile.VerificationStatus == StatusVerified, nil
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

func (s *Service) syncUserProfileProjection(ctx context.Context, userID int64) error {
	if s.profileFGA == nil {
		return errors.New("profile FGA dependency is not configured")
	}

	projectionCtx, cancel := context.WithTimeout(ctx, fga.DefaultWriteTimeout)
	defer cancel()

	profile, err := s.repo.GetProfileByUserID(projectionCtx, userID)
	if err != nil {
		return fmt.Errorf("get profile: %w", err)
	}
	if profile == nil {
		return nil
	}

	profileObj := "user_profile:" + strconv.FormatInt(userID, 10)
	ownerTuple := fga.Tuple{
		User:     "user:" + strconv.FormatInt(userID, 10),
		Relation: "owner",
		Object:   profileObj,
	}
	desiredOwnerTuples := []fga.Tuple{ownerTuple}
	desiredSchoolTuples := make([]fga.Tuple, 0, 1)
	if profile.VerificationStatus == StatusVerified {
		if profile.SchoolID == nil {
			return fmt.Errorf("verified profile %d has no school ID", userID)
		}
		if err := validateSchoolID(*profile.SchoolID); err != nil {
			return fmt.Errorf("verified profile %d has invalid school ID: %w", userID, err)
		}
		desiredSchoolTuples = append(desiredSchoolTuples, fga.Tuple{
			User:     "school:" + strconv.FormatInt(*profile.SchoolID, 10),
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

var errAdmissionProjectionProfileNotVerified = errors.New("verified profile is not available for admission projection")

func (s *Service) syncAdmissionVerificationProjection(ctx context.Context, userID int64) error {
	approved, err := s.currentProfileApproved(ctx, userID)
	if err != nil {
		return err
	}
	if !approved {
		return nil
	}
	if s.admissionProjection == nil {
		return errors.New("admission verification projection dependency is not configured")
	}
	schoolID, err := s.ensureVerifiedProfileCredential(ctx, userID)
	if errors.Is(err, errAdmissionProjectionProfileNotVerified) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("ensure admission verification credential: %w", err)
	}
	if err := s.admissionProjection.ProjectStudentVerification(ctx, userID, schoolID, true); err != nil {
		return fmt.Errorf("project admission student verification: %w", err)
	}
	return nil
}

func (s *Service) ensureVerifiedProfileCredential(ctx context.Context, userID int64) (int64, error) {
	var schoolID int64
	err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		profile, err := s.repo.GetProfileByUserIDForUpdateTx(ctx, tx, userID)
		if err != nil {
			return fmt.Errorf("get profile tx: %w", err)
		}
		if profile == nil || profile.VerificationStatus != StatusVerified {
			return errAdmissionProjectionProfileNotVerified
		}
		if profile.SchoolID == nil || *profile.SchoolID <= 0 {
			return fmt.Errorf("verified profile %d has no school ID for admission projection", userID)
		}
		schoolID = *profile.SchoolID
		return s.ensureProfileVerificationCredentialTx(ctx, tx, profile)
	})
	return schoolID, err
}
