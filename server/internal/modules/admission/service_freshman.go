package admission

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/id"
)

func (s *Service) CreateFreshmanApplication(
	ctx context.Context,
	input FreshmanApplicationCreateInput,
) (*FreshmanApplication, error) {
	session, err := s.requireLinkedSession(ctx, input.UserID, input.AdmissionSessionID)
	if err != nil {
		return nil, err
	}
	policy, err := s.loadPolicy(ctx, session.Platform, session.GuildID)
	if err != nil {
		return nil, err
	}
	if !policy.FreshmanChannelEnabled || s.now().After(policy.FreshmanChannelClosesAt) {
		return nil, ErrAdmissionFreshmanChannelClosed
	}
	existing, err := s.repo.GetPendingFreshmanApplication(ctx, input.UserID, input.SchoolID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return s.usePendingFreshmanApplicationForSession(ctx, existing, session)
	}
	if s.beforeFreshmanApplicationCreate != nil {
		s.beforeFreshmanApplicationCreate()
	}
	app, err := s.buildFreshmanApplication(input, session.ID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateFreshmanApplication(ctx, app); err != nil {
		if isFreshmanApplicationPendingUniqueViolation(err) {
			existing, getErr := s.repo.GetPendingFreshmanApplication(ctx, input.UserID, input.SchoolID)
			if getErr != nil {
				return nil, getErr
			}
			if existing != nil {
				return s.usePendingFreshmanApplicationForSession(ctx, existing, session)
			}
		}
		return nil, err
	}
	return app, nil
}

func (s *Service) usePendingFreshmanApplicationForSession(
	ctx context.Context,
	app *FreshmanApplication,
	session *AdmissionSession,
) (*FreshmanApplication, error) {
	if app == nil {
		return nil, ErrAdmissionApplicationNotFound
	}
	if session == nil {
		return nil, ErrAdmissionLinkedSessionRequired
	}
	if app.AdmissionSessionID == nil || *app.AdmissionSessionID != session.ID {
		reassigned, err := s.repo.ReassignPendingFreshmanApplicationSession(ctx, app.ID, session.ID)
		if err != nil {
			return nil, err
		}
		app = reassigned
	}
	if err := s.syncFreshmanApplicationMaterialSubmission(ctx, app, session); err != nil {
		return nil, err
	}
	return app, nil
}

func (s *Service) requireLinkedSession(
	ctx context.Context,
	userID int64,
	admissionSessionID string,
) (*AdmissionSession, error) {
	if err := validateAdmissionUserID(userID); err != nil {
		return nil, err
	}
	session, err := s.repo.GetLinkedSessionByUserID(ctx, userID, s.now(), admissionSessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrAdmissionLinkedSessionRequired
	}
	if err := s.ensureLinkedSessionAcceptsSubmission(session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) ensureLinkedSessionAcceptsSubmission(session *AdmissionSession) error {
	if session == nil {
		return ErrAdmissionLinkedSessionRequired
	}
	if session.Status != StatusLinked {
		return ErrAdmissionInvalidStatus
	}
	if s.now().After(session.SubmissionWaitDeadlineAt) {
		return ErrAdmissionTokenExpired
	}
	return nil
}

func (s *Service) ensureSessionAcceptsVerification(session *AdmissionSession) error {
	if session == nil {
		return ErrAdmissionSessionNotFound
	}
	switch session.Status {
	case StatusLinked:
		return s.ensureLinkedSessionAcceptsSubmission(session)
	case StatusMaterialSubmitted:
		if session.ManualReviewDeadlineAt != nil && s.now().After(*session.ManualReviewDeadlineAt) {
			return ErrAdmissionInvalidStatus
		}
		return nil
	default:
		return ErrAdmissionInvalidStatus
	}
}

func (s *Service) syncFreshmanApplicationMaterialSubmission(
	ctx context.Context,
	app *FreshmanApplication,
	session *AdmissionSession,
) error {
	hasMaterial, err := s.repo.FreshmanApplicationHasMaterial(ctx, app.ID)
	if err != nil {
		return err
	}
	if !hasMaterial {
		return nil
	}
	return s.ensureFreshmanMaterialSubmitted(ctx, session)
}

func (s *Service) submitFreshmanSessionWithPolicy(
	ctx context.Context,
	session *AdmissionSession,
	policy *AdmissionPolicy,
) error {
	if policy == nil {
		return ErrAdmissionPolicyNotFound
	}
	if err := s.ensureLinkedSessionAcceptsSubmission(session); err != nil {
		return err
	}
	now := s.now()
	deadline := now.Add(time.Duration(policy.ManualReviewTimeoutSeconds) * time.Second)
	return s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		submitted, err := s.repo.MarkMaterialSubmittedTx(ctx, tx, session.ID, deadline)
		if err != nil {
			return err
		}
		return s.queueInitialBotActionsTx(ctx, tx, submitted, now)
	})
}

func (s *Service) ensureFreshmanMaterialSubmitted(ctx context.Context, session *AdmissionSession) error {
	switch session.Status {
	case StatusLinked:
		if err := s.ensureLinkedSessionAcceptsSubmission(session); err != nil {
			return err
		}
		policy, err := s.loadPolicy(ctx, session.Platform, session.GuildID)
		if err != nil {
			return err
		}
		return s.submitFreshmanSessionWithPolicy(ctx, session, policy)
	case StatusMaterialSubmitted, StatusVerified:
		return nil
	default:
		return ErrAdmissionInvalidStatus
	}
}

func (s *Service) buildFreshmanApplication(
	input FreshmanApplicationCreateInput,
	sessionID string,
) (*FreshmanApplication, error) {
	appID, err := id.New()
	if err != nil {
		return nil, err
	}
	linkedSessionID := sessionID
	return &FreshmanApplication{
		ID:                  appID,
		UserID:              input.UserID,
		SchoolID:            input.SchoolID,
		AdmissionSessionID:  &linkedSessionID,
		Status:              FreshmanApplicationPending,
		ApplicantName:       strings.TrimSpace(input.ApplicantName),
		ApplicantNameMasked: maskAdmissionName(input.ApplicantName),
		DepartmentOrMajor:   normalizeStringPtr(input.DepartmentOrMajor),
		MaterialType:        input.MaterialType,
	}, nil
}

func maskAdmissionName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) == 1 {
		return string(runes[0]) + "*"
	}
	return string(runes[0]) + "***"
}
