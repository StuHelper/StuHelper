package admission

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
)

func (s *Service) ReviewFreshmanApplicationFromBot(
	ctx context.Context,
	input BotFreshmanReviewInput,
) (*FreshmanApplication, error) {
	input = normalizeBotFreshmanReviewInput(input)
	operatorUserID, err := s.authorizeBotFreshmanReviewer(ctx, input)
	if err != nil {
		return nil, err
	}
	return s.reviewFreshmanApplication(ctx, freshmanReviewCommand{
		ApplicationID:  input.ApplicationID,
		Action:         input.Action,
		Reason:         normalizeStringPtr(input.Reason),
		ExpiresInDays:  input.ExpiresInDays,
		ReviewerUserID: &operatorUserID,
		OperatorQQID:   &input.OperatorQQID,
		GuildID:        input.GuildID,
		RawCommand:     input.RawCommand,
	})
}

func (s *Service) ReviewFreshmanApplicationFromAdmin(
	ctx context.Context,
	input AdminFreshmanReviewInput,
) (*FreshmanApplication, error) {
	input.ApplicationID = strings.TrimSpace(input.ApplicationID)
	return s.reviewFreshmanApplication(ctx, freshmanReviewCommand{
		ApplicationID:  input.ApplicationID,
		Action:         input.Action,
		Reason:         normalizeStringPtr(input.Reason),
		ExpiresInDays:  input.ExpiresInDays,
		ReviewerUserID: &input.OperatorUserID,
	})
}

func (s *Service) authorizeBotFreshmanReviewer(
	ctx context.Context,
	input BotFreshmanReviewInput,
) (int64, error) {
	if err := s.ensureManagementGuild(ctx, input); err != nil {
		return 0, err
	}
	userID, err := s.repo.GetUserIDByQQID(ctx, input.OperatorQQID)
	if err != nil {
		return 0, err
	}
	if userID == nil {
		return 0, ErrAdmissionOperatorUnbound
	}
	if err := s.ensureOperatorCapability(ctx, *userID); err != nil {
		return 0, err
	}
	return *userID, nil
}

func (s *Service) ensureManagementGuild(ctx context.Context, input BotFreshmanReviewInput) error {
	app, err := s.repo.GetFreshmanApplicationForReviewPolicy(ctx, input.ApplicationID)
	if err != nil {
		return err
	}
	policy, err := s.policyForFreshmanApplication(ctx, app)
	if err != nil {
		return err
	}
	if !stringInSlice(input.GuildID, policy.ManagementGuildIDs) {
		return ErrAdmissionManagementGuildForbidden
	}
	return nil
}

func (s *Service) ensureOperatorCapability(ctx context.Context, userID int64) error {
	if s.operatorAccess == nil {
		return ErrAdmissionOperatorAccessUnavailable
	}
	allowed, err := s.operatorAccess.UserHasCapability(ctx, userID, capability.AdmissionFreshmanReview)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAdmissionOperatorAccessUnavailable, err)
	}
	if !allowed {
		return ErrAdmissionOperatorForbidden
	}
	return nil
}

func (s *Service) reviewFreshmanApplication(
	ctx context.Context,
	command freshmanReviewCommand,
) (*FreshmanApplication, error) {
	var reviewed *FreshmanApplication
	err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		app, err := s.repo.GetFreshmanApplicationByIDForUpdate(ctx, tx, command.ApplicationID)
		if err != nil {
			return err
		}
		if app.Status != FreshmanApplicationPending {
			return ErrAdmissionInvalidStatus
		}
		if err := s.ensureFreshmanApplicationReviewableTx(ctx, tx, app); err != nil {
			return err
		}
		update, err := s.buildFreshmanReviewUpdate(ctx, app, command)
		if err != nil {
			return err
		}
		reviewed, err = s.repo.ReviewFreshmanApplicationTx(ctx, tx, update)
		if err != nil {
			return err
		}
		approval := freshmanApprovalTxInput{Tx: tx, App: app, Update: update}
		if err := s.applyFreshmanApprovalTx(ctx, approval); err != nil {
			return err
		}
		return s.repo.InsertAuditEventTx(ctx, tx, freshmanReviewAuditEvent(ctx, reviewed, command))
	})
	return reviewed, err
}

func (s *Service) ensureFreshmanApplicationReviewableTx(
	ctx context.Context,
	tx pgx.Tx,
	app *FreshmanApplication,
) error {
	if app.AdmissionSessionID == nil {
		return ErrAdmissionLinkedSessionRequired
	}
	hasMaterial, err := s.repo.FreshmanApplicationHasMaterialTx(ctx, tx, app.ID)
	if err != nil {
		return err
	}
	if !hasMaterial {
		return ErrAdmissionInvalidStatus
	}
	session, err := s.repo.GetSessionByIDForUpdate(ctx, tx, *app.AdmissionSessionID)
	if err != nil {
		return err
	}
	if session.Status != StatusMaterialSubmitted {
		return ErrAdmissionInvalidStatus
	}
	return s.ensureSessionAcceptsVerification(session)
}

func (s *Service) applyFreshmanApprovalTx(
	ctx context.Context,
	input freshmanApprovalTxInput,
) error {
	if input.Update.Status != FreshmanApplicationApproved {
		return nil
	}
	if s.projection == nil {
		return ErrAdmissionProjectionUnavailable
	}
	if input.Update.ProvisionalExpiresAt == nil {
		return ErrAdmissionInvalidStatus
	}
	credential := s.newFreshmanMaterialCredential(input.App, *input.Update.ProvisionalExpiresAt)
	if err := s.repo.CreateVerificationCredentialTx(ctx, input.Tx, credential); err != nil {
		return err
	}
	if err := s.repo.ProjectVerifiedUserProfileTx(ctx, input.Tx, credential); err != nil {
		return err
	}
	if err := s.repo.MarkUserLinkedSessionsVerifiedTx(ctx, input.Tx, input.App.UserID, input.App.SchoolID, s.now()); err != nil {
		return err
	}
	if err := s.queueVerifiedUserReleaseActionsTx(ctx, input.Tx, input.App.UserID, input.App.SchoolID, s.now()); err != nil {
		return err
	}
	return s.projection.EnqueueFreshmanProvisionalRoleSyncTx(ctx, input.Tx, input.App.UserID, true)
}

type freshmanApprovalTxInput struct {
	Tx     pgx.Tx
	App    *FreshmanApplication
	Update freshmanApplicationReviewUpdate
}

func (s *Service) newFreshmanMaterialCredential(
	app *FreshmanApplication,
	expiresAt time.Time,
) VerificationCredential {
	sourceID := app.ID
	return VerificationCredential{
		UserID:              app.UserID,
		SchoolID:            app.SchoolID,
		Kind:                CredentialFreshmanMaterialManual,
		SubjectHash:         s.hashCredentialSubject(app.SchoolID, "freshman:"+app.ID),
		SubjectDisplay:      "freshman material " + app.ApplicantNameMasked,
		SourceApplicationID: &sourceID,
		ExpiresAt:           &expiresAt,
		VerifiedAt:          s.now(),
	}
}

func (s *Service) buildFreshmanReviewUpdate(
	ctx context.Context,
	app *FreshmanApplication,
	command freshmanReviewCommand,
) (freshmanApplicationReviewUpdate, error) {
	status, expiresAt, err := s.resolveFreshmanReviewOutcome(ctx, app, command)
	if err != nil {
		return freshmanApplicationReviewUpdate{}, err
	}
	return freshmanApplicationReviewUpdate{
		ApplicationID:        app.ID,
		Status:               status,
		Reason:               command.Reason,
		ReviewerUserID:       command.ReviewerUserID,
		OperatorQQID:         command.OperatorQQID,
		ProvisionalExpiresAt: expiresAt,
		ReviewedAt:           s.now(),
	}, nil
}

func (s *Service) resolveFreshmanReviewOutcome(
	ctx context.Context,
	app *FreshmanApplication,
	command freshmanReviewCommand,
) (FreshmanApplicationStatus, *time.Time, error) {
	switch command.Action {
	case FreshmanReviewApprove:
		expiresAt, err := s.resolveFreshmanCredentialExpiry(ctx, app, command.ExpiresInDays)
		return FreshmanApplicationApproved, expiresAt, err
	case FreshmanReviewReject:
		return FreshmanApplicationRejected, nil, nil
	default:
		return "", nil, ErrAdmissionInvalidStatus
	}
}

func (s *Service) resolveFreshmanCredentialExpiry(
	ctx context.Context,
	app *FreshmanApplication,
	expiresInDays *int,
) (*time.Time, error) {
	policy, err := s.policyForFreshmanApplication(ctx, app)
	if err != nil {
		return nil, err
	}
	if expiresInDays == nil {
		expires := policy.FreshmanDefaultExpiresAt
		return &expires, nil
	}
	if *expiresInDays <= 0 {
		return nil, ErrAdmissionInvalidInput
	}
	if *expiresInDays > policy.MaxExtensionDays {
		return nil, ErrAdmissionReviewExtensionTooLong
	}
	expires := s.now().Add(time.Duration(*expiresInDays) * admissionDay)
	return &expires, nil
}

func (s *Service) policyForFreshmanApplication(
	ctx context.Context,
	app *FreshmanApplication,
) (*AdmissionPolicy, error) {
	if app.AdmissionSessionID == nil {
		return nil, ErrAdmissionLinkedSessionRequired
	}
	session, err := s.repo.GetSessionByID(ctx, *app.AdmissionSessionID)
	if err != nil {
		return nil, err
	}
	return s.loadPolicy(ctx, session.Platform, session.GuildID)
}

type freshmanReviewCommand struct {
	ApplicationID  string
	Action         FreshmanReviewAction
	Reason         *string
	ExpiresInDays  *int
	ReviewerUserID *int64
	OperatorQQID   *string
	GuildID        string
	RawCommand     string
}

const admissionDay = 24 * time.Hour

func normalizeBotFreshmanReviewInput(input BotFreshmanReviewInput) BotFreshmanReviewInput {
	input.ApplicationID = strings.TrimSpace(input.ApplicationID)
	input.Action = FreshmanReviewAction(strings.ToLower(strings.TrimSpace(string(input.Action))))
	input.OperatorQQID = strings.TrimSpace(input.OperatorQQID)
	input.GuildID = strings.TrimSpace(input.GuildID)
	input.ChannelID = normalizeStringPtr(input.ChannelID)
	input.RawCommand = strings.TrimSpace(input.RawCommand)
	input.Reason = normalizeStringPtr(input.Reason)
	return input
}

func stringInSlice(value string, list []string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
