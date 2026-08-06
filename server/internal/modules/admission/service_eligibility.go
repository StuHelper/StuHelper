package admission

import "context"

type eligibleAdmissionUser struct {
	UserID   int64
	Revision int64
}

func (s *Service) getEligibleAdmissionUserByQQ(
	ctx context.Context,
	qqID string,
	policy *AdmissionPolicy,
) (*eligibleAdmissionUser, error) {
	if s.studentEligibility == nil || policy == nil {
		return nil, ErrAdmissionProjectionUnavailable
	}
	userID, err := s.repo.GetBoundAdmissionUserByQQ(ctx, qqID)
	if err != nil || userID == nil {
		return nil, err
	}
	decision, err := s.studentEligibility.EvaluateStudentEligibility(ctx, *userID, policy.SchoolID)
	if err != nil {
		return nil, ErrAdmissionProjectionUnavailable
	}
	if !studentEligibilityAllowedByPolicy(decision, policy) {
		return nil, nil
	}
	if decision.Revision <= 0 {
		return nil, ErrAdmissionProjectionUnavailable
	}
	return &eligibleAdmissionUser{UserID: *userID, Revision: decision.Revision}, nil
}

func studentEligibilityAllowedByPolicy(
	decision StudentEligibilityDecision,
	policy *AdmissionPolicy,
) bool {
	if !decision.Eligible || policy == nil {
		return false
	}
	switch decision.CredentialClass {
	case "formal_student":
		return true
	case "temporary_freshman":
		return policy.AllowTemporaryFreshman
	default:
		return false
	}
}
