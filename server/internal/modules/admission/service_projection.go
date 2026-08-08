package admission

import (
	"context"
	"strings"
)

func (s *Service) GetAdmissionMe(
	ctx context.Context,
	userID int64,
	admissionSessionID string,
) (*AdmissionMe, error) {
	session, err := s.repo.GetLatestSessionByUserID(ctx, userID, admissionSessionID)
	if err != nil {
		return nil, err
	}
	if session == nil && strings.TrimSpace(admissionSessionID) != "" {
		return nil, ErrAdmissionSessionNotFound
	}
	if session == nil {
		return &AdmissionMe{Status: StatusCancelled}, nil
	}
	policy, err := s.loadPolicy(ctx, session.Platform, session.GuildID)
	if err != nil {
		return nil, err
	}
	if s.studentEligibility == nil {
		return nil, ErrAdmissionProjectionUnavailable
	}
	if session.Status == StatusLinked || session.Status == StatusMaterialSubmitted ||
		session.Status == StatusEligible || session.Status == StatusVerified {
		decision, evaluationErr := s.studentEligibility.EvaluateStudentEligibility(ctx, userID, policy.SchoolID)
		if evaluationErr != nil {
			return nil, ErrAdmissionProjectionUnavailable
		}
		if decision.Revision > 0 {
			if err := s.ReevaluateStudentEligibility(
				ctx,
				userID,
				policy.SchoolID,
				decision.Revision,
			); err != nil {
				return nil, err
			}
			session, err = s.repo.GetSessionByID(ctx, session.ID)
			if err != nil {
				return nil, err
			}
		}
	}
	pending, err := s.repo.HasPendingAdmissionProjection(ctx, userID)
	if err != nil {
		return nil, err
	}
	session.ProjectionPending = pending
	return &AdmissionMe{
		Status:            session.Status,
		ProjectionPending: pending,
		Session:           session,
	}, nil
}
