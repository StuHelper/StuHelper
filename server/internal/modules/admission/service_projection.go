package admission

import "context"

func (s *Service) GetAdmissionMe(ctx context.Context, userID int64) (*AdmissionMe, error) {
	session, err := s.repo.GetLatestSessionByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return &AdmissionMe{Status: StatusCancelled}, nil
	}
	credential, err := s.repo.GetLatestCredentialForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	pending, err := s.repo.HasPendingFreshmanProjection(ctx, userID)
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
