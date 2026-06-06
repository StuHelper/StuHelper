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
	credential, err := s.repo.GetLatestCredentialForUserSchool(ctx, userID, policy.SchoolID, s.now())
	if err != nil {
		return nil, err
	}
	pending, err := s.repo.HasPendingAdmissionProjection(ctx, userID)
	if err != nil {
		return nil, err
	}
	session.ProjectionPending = pending
	return admissionMeFromState(session, credential, pending), nil
}

func admissionMeFromState(
	session *AdmissionSession,
	credential *VerificationCredential,
	pending bool,
) *AdmissionMe {
	me := &AdmissionMe{Status: session.Status, ProjectionPending: pending, Session: session}
	if credential != nil {
		me.CredentialKind = &credential.Kind
		me.ProvisionalExpiresAt = credential.ExpiresAt
	}
	return me
}
