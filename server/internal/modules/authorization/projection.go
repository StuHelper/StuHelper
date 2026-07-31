package authorization

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
	"github.com/StuHelper/StuHelper/server/internal/pkg/outbox"
)

type ProjectionClient interface {
	WriteMissingTuples(ctx context.Context, desired []fga.Tuple) error
	TupleExists(ctx context.Context, tuple fga.Tuple) (bool, error)
	DeleteTuplesIgnoringMissing(ctx context.Context, tuples []fga.Tuple) error
}

func (s *Service) StartBackgroundJobs(
	ctx context.Context,
	start func(string, func(context.Context)),
) {
	if start == nil {
		panic("authorization.Service.StartBackgroundJobs: starter is required")
	}
	if s.projection == nil {
		panic("authorization.Service.StartBackgroundJobs: projection client is required")
	}
	start("authorization grant projection worker", s.runProjectionWorker)
}

func (s *Service) runProjectionWorker(ctx context.Context) {
	outbox.RunPollingWorker(
		ctx,
		outbox.IAMWorkerConfig("authorization grant projection"),
		s.repo.ClaimProjectionJobs,
		s.processProjectionJob,
		s.repo.MarkProjectionJobDone,
		s.repo.MarkProjectionJobFailure,
		func(job outbox.Job) outbox.JobMeta {
			return outbox.JobMeta{
				ID:           job.ID,
				JobType:      job.JobType,
				AttemptCount: job.AttemptCount,
				LockedAt:     job.LockedAt,
			}
		},
		truncateProjectionError,
	)
}

func (s *Service) ProcessProjectionBatch(ctx context.Context) error {
	if s.projection == nil {
		return errors.New("authorization projection client is required")
	}
	return outbox.ProcessBatch(
		ctx,
		outbox.IAMWorkerConfig("authorization grant projection"),
		s.repo.ClaimProjectionJobs,
		s.processProjectionJob,
		s.repo.MarkProjectionJobDone,
		s.repo.MarkProjectionJobFailure,
		func(job outbox.Job) outbox.JobMeta {
			return outbox.JobMeta{
				ID:           job.ID,
				JobType:      job.JobType,
				AttemptCount: job.AttemptCount,
				LockedAt:     job.LockedAt,
			}
		},
		truncateProjectionError,
	)
}

func (s *Service) processProjectionJob(ctx context.Context, job outbox.Job) error {
	if job.JobType != ProjectionJobType {
		return fmt.Errorf("%w: unsupported job type %q", ErrProjectionMalformed, job.JobType)
	}
	var payload ProjectionPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("%w: %v", ErrProjectionMalformed, err)
	}
	if payload.GrantID <= 0 || payload.Revision <= 0 ||
		(payload.DesiredState != DesiredGranted && payload.DesiredState != DesiredRevoked) {
		return ErrProjectionMalformed
	}

	grant, err := s.repo.GetGrant(ctx, payload.GrantID)
	if errors.Is(err, ErrGrantNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if grant.Revision != payload.Revision || grant.DesiredState != payload.DesiredState {
		return nil
	}
	tuple, err := tupleForGrant(grant)
	if err != nil {
		return err
	}

	projectionCtx, cancel := context.WithTimeout(ctx, fga.DefaultWriteTimeout)
	defer cancel()
	switch payload.DesiredState {
	case DesiredGranted:
		if err := s.projection.WriteMissingTuples(projectionCtx, []fga.Tuple{tuple}); err != nil {
			return fmt.Errorf("write authorization tuple: %w", err)
		}
		exists, err := s.projection.TupleExists(projectionCtx, tuple)
		if err != nil {
			return fmt.Errorf("verify authorization tuple: %w", err)
		}
		if !exists {
			return errors.New("authorization tuple is absent after write")
		}
	case DesiredRevoked:
		if err := s.projection.DeleteTuplesIgnoringMissing(projectionCtx, []fga.Tuple{tuple}); err != nil {
			return fmt.Errorf("delete authorization tuple: %w", err)
		}
		exists, err := s.projection.TupleExists(projectionCtx, tuple)
		if err != nil {
			return fmt.Errorf("verify authorization tuple removal: %w", err)
		}
		if exists {
			return errors.New("authorization tuple still exists after delete")
		}
	}
	_, err = s.repo.MarkProjectionApplied(ctx, payload)
	return err
}

func tupleForGrant(grant Grant) (fga.Tuple, error) {
	if grant.SubjectUserID <= 0 {
		return fga.Tuple{}, ErrProjectionMalformed
	}
	tuple := fga.Tuple{User: "user:" + strconv.FormatInt(grant.SubjectUserID, 10)}
	switch grant.Role {
	case RoleSuperAdmin:
		if grant.SchoolID != nil || grant.SectionID != nil {
			return fga.Tuple{}, ErrProjectionMalformed
		}
		tuple.Relation = "super_admin"
		tuple.Object = "ecosystem:stuhelper"
	case RoleSchoolAdmin:
		if grant.SchoolID == nil || grant.SectionID != nil {
			return fga.Tuple{}, ErrProjectionMalformed
		}
		tuple.Relation = "admin"
		tuple.Object = "school:" + strconv.FormatInt(*grant.SchoolID, 10)
	case RoleSectionAdmin, RoleSectionModerator, RoleSectionReviewer:
		if grant.SchoolID == nil || grant.SectionID == nil {
			return fga.Tuple{}, ErrProjectionMalformed
		}
		expected := fga.ReviewModerationSectionID(strconv.FormatInt(*grant.SchoolID, 10))
		if *grant.SectionID != expected {
			return fga.Tuple{}, ErrProjectionMalformed
		}
		tuple.Relation = string(grant.Role)
		tuple.Object = "section:" + *grant.SectionID
	default:
		return fga.Tuple{}, ErrProjectionMalformed
	}
	return tuple, nil
}

func truncateProjectionError(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	if len(message) > 1000 {
		return message[:1000]
	}
	return message
}
