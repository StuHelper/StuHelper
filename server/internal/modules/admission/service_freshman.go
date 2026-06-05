package admission

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	_ "golang.org/x/image/webp"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

const freshmanMaterialObjectPrefix = "admission/freshman/"
const freshmanCameraHandoffTTL = 30 * time.Minute
const freshmanMaterialCleanupTimeout = 5 * time.Second

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

func (s *Service) SubmitCameraCapture(ctx context.Context, input CameraCaptureInput) (*FreshmanApplication, error) {
	if s.materialStore == nil {
		return nil, ErrAdmissionMaterialStoreUnavailable
	}
	app, err := s.repo.GetFreshmanApplicationForUser(ctx, input.ApplicationID, input.UserID)
	if err != nil {
		return nil, err
	}
	session, policy, err := s.loadApplicationSessionPolicy(ctx, app)
	if err != nil {
		return nil, err
	}
	if err := s.ensureFreshmanDesktopCaptureAllowed(ctx, app.ID); err != nil {
		return nil, err
	}
	if err := s.ensureLinkedSessionAcceptsSubmission(session); err != nil {
		return nil, err
	}
	material, content, err := s.buildFreshmanMaterial(input, policy.MaxMaterialBytes)
	if err != nil {
		return nil, err
	}
	if err := s.materialStore.PutAdmissionMaterial(ctx, material.ObjectKey, content, material.ContentType); err != nil {
		return nil, fmt.Errorf("SubmitCameraCapture store material: %w", err)
	}
	if err := s.createFreshmanMaterialAndSubmitSession(ctx, material, session, policy); err != nil {
		if cleanupErr := s.cleanupFreshmanMaterial(ctx, material.ObjectKey); cleanupErr != nil {
			return nil, fmt.Errorf("SubmitCameraCapture create material: %w; cleanup: %v", err, cleanupErr)
		}
		if isFreshmanMaterialApplicationUniqueViolation(err) {
			return nil, ErrAdmissionCameraHandoffLocked
		}
		return nil, err
	}
	return app, nil
}

func (s *Service) CreateFreshmanCameraHandoff(
	ctx context.Context,
	input FreshmanCameraHandoffCreateInput,
) (*FreshmanCameraHandoff, error) {
	app, err := s.repo.GetFreshmanApplicationForUser(ctx, input.ApplicationID, input.UserID)
	if err != nil {
		return nil, err
	}
	session, _, err := s.loadApplicationSessionPolicy(ctx, app)
	if err != nil {
		return nil, err
	}
	hasMaterial, err := s.repo.FreshmanApplicationHasMaterial(ctx, app.ID)
	if err != nil {
		return nil, err
	}
	if hasMaterial {
		return nil, ErrAdmissionCameraHandoffLocked
	}
	if err := s.ensureLinkedSessionAcceptsSubmission(session); err != nil {
		return nil, err
	}
	active, err := s.repo.GetActiveFreshmanCameraHandoff(ctx, app.ID, s.now())
	if err != nil {
		return nil, err
	}
	if active != nil && active.Status == FreshmanCameraHandoffUploaded {
		return s.hydrateFreshmanCameraHandoff(ctx, active)
	}
	if err := s.repo.ExpirePendingFreshmanCameraHandoffs(ctx, app.ID, s.now()); err != nil {
		return nil, err
	}
	if s.beforeFreshmanCameraHandoffCreate != nil {
		s.beforeFreshmanCameraHandoffCreate()
	}
	token, err := s.generateToken()
	if err != nil {
		return nil, err
	}
	handoffID, err := id.New()
	if err != nil {
		return nil, err
	}
	handoff := FreshmanCameraHandoff{
		ID:            handoffID,
		ApplicationID: app.ID,
		UserID:        input.UserID,
		Status:        FreshmanCameraHandoffPending,
		MobileURL:     s.buildFreshmanCameraMobileURL(token),
		ExpiresAt:     s.now().Add(freshmanCameraHandoffTTL),
		CreatedAt:     s.now(),
	}
	if err := s.repo.CreateFreshmanCameraHandoff(ctx, handoff, s.hashToken(token)); err != nil {
		if isFreshmanCameraHandoffActiveUniqueViolation(err) {
			active, getErr := s.repo.GetActiveFreshmanCameraHandoff(ctx, app.ID, s.now())
			if getErr != nil {
				return nil, getErr
			}
			if active != nil {
				return s.hydrateFreshmanCameraHandoff(ctx, active)
			}
			return nil, ErrAdmissionCameraHandoffLocked
		}
		return nil, err
	}
	return &handoff, nil
}

func (s *Service) GetFreshmanCameraHandoffForUser(
	ctx context.Context,
	userID int64,
	handoffID string,
) (*FreshmanCameraHandoff, error) {
	handoff, err := s.repo.GetFreshmanCameraHandoffForUser(ctx, handoffID, userID)
	if err != nil {
		return nil, err
	}
	return s.hydrateFreshmanCameraHandoff(ctx, handoff)
}

func (s *Service) PreviewFreshmanCameraHandoff(
	ctx context.Context,
	token string,
) (*FreshmanCameraHandoff, error) {
	handoff, err := s.repo.GetFreshmanCameraHandoffByTokenHash(ctx, s.hashToken(token))
	if err != nil {
		return nil, err
	}
	return s.hydrateFreshmanCameraHandoff(ctx, handoff)
}

func (s *Service) SubmitFreshmanCameraHandoffCapture(
	ctx context.Context,
	input FreshmanCameraHandoffCaptureInput,
) (*FreshmanCameraHandoff, error) {
	if s.materialStore == nil {
		return nil, ErrAdmissionMaterialStoreUnavailable
	}
	handoff, err := s.repo.GetFreshmanCameraHandoffByTokenHash(ctx, s.hashToken(input.Token))
	if err != nil {
		return nil, err
	}
	if err := s.ensureFreshmanCameraHandoffUsable(handoff); err != nil {
		if errors.Is(err, ErrAdmissionCameraHandoffLocked) {
			return s.recoverFreshmanCameraHandoffUpload(ctx, handoff)
		}
		return nil, err
	}
	app, err := s.repo.GetFreshmanApplicationByID(ctx, handoff.ApplicationID)
	if err != nil {
		return nil, err
	}
	session, policy, err := s.loadApplicationSessionPolicy(ctx, app)
	if err != nil {
		return nil, err
	}
	if err := s.ensureLinkedSessionAcceptsSubmission(session); err != nil {
		return nil, err
	}
	material, content, err := s.buildFreshmanMaterial(CameraCaptureInput{
		ApplicationID: app.ID,
		ContentType:   input.ContentType,
		ImageBase64:   input.ImageBase64,
	}, policy.MaxMaterialBytes)
	if err != nil {
		return nil, err
	}
	if err := s.materialStore.PutAdmissionMaterial(ctx, material.ObjectKey, content, material.ContentType); err != nil {
		return nil, fmt.Errorf("SubmitFreshmanCameraHandoffCapture store material: %w", err)
	}
	if err := s.createFreshmanMaterialAndSubmitSession(ctx, material, session, policy); err != nil {
		if cleanupErr := s.cleanupFreshmanMaterial(ctx, material.ObjectKey); cleanupErr != nil {
			return nil, fmt.Errorf("SubmitFreshmanCameraHandoffCapture create material: %w; cleanup: %v", err, cleanupErr)
		}
		if isFreshmanMaterialApplicationUniqueViolation(err) {
			return s.recoverFreshmanCameraHandoffUpload(ctx, handoff)
		}
		return nil, err
	}
	now := s.now()
	if s.beforeFreshmanCameraHandoffMarkUploaded != nil {
		s.beforeFreshmanCameraHandoffMarkUploaded()
	}
	if err := s.repo.MarkFreshmanCameraHandoffUploaded(ctx, handoff.ID, now); err != nil {
		if errors.Is(err, ErrAdmissionCameraHandoffLocked) {
			return s.recoverFreshmanCameraHandoffUpload(ctx, handoff)
		}
		return nil, err
	}
	handoff.Status = FreshmanCameraHandoffUploaded
	handoff.UploadedAt = &now
	return handoff, nil
}

func (s *Service) ChooseFreshmanCameraHandoffContinuation(
	ctx context.Context,
	input FreshmanCameraHandoffContinuationInput,
) (*FreshmanCameraHandoff, error) {
	switch input.ContinueOn {
	case FreshmanCameraContinueDesktop, FreshmanCameraContinueMobile:
	default:
		return nil, ErrAdmissionCameraHandoffInvalidChoice
	}
	handoff, err := s.repo.GetFreshmanCameraHandoffByTokenHash(ctx, s.hashToken(input.Token))
	if err != nil {
		return nil, err
	}
	if s.now().After(handoff.ExpiresAt) {
		return nil, ErrAdmissionCameraHandoffExpired
	}
	if handoff.Status != FreshmanCameraHandoffUploaded {
		return nil, ErrAdmissionCameraHandoffLocked
	}
	if s.beforeFreshmanCameraHandoffContinuationChoose != nil {
		s.beforeFreshmanCameraHandoffContinuationChoose()
	}
	updated, err := s.repo.ChooseFreshmanCameraHandoffContinuation(ctx, handoff.ID, input.ContinueOn, s.now())
	if err != nil {
		if errors.Is(err, ErrAdmissionCameraHandoffNotFound) {
			return nil, ErrAdmissionCameraHandoffLocked
		}
		return nil, err
	}
	return updated, nil
}

func (s *Service) requireLinkedSession(
	ctx context.Context,
	userID int64,
	admissionSessionID string,
) (*AdmissionSession, error) {
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

func (s *Service) ensureFreshmanCameraHandoffUsable(handoff *FreshmanCameraHandoff) error {
	if handoff == nil {
		return ErrAdmissionCameraHandoffNotFound
	}
	if s.now().After(handoff.ExpiresAt) {
		return ErrAdmissionCameraHandoffExpired
	}
	if handoff.Status != FreshmanCameraHandoffPending {
		return ErrAdmissionCameraHandoffLocked
	}
	return nil
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

func (s *Service) cleanupFreshmanMaterial(ctx context.Context, objectKey string) error {
	cleanupCtx, cancel := freshmanMaterialCleanupContext(ctx)
	defer cancel()
	return s.materialStore.DeleteAdmissionMaterial(cleanupCtx, objectKey)
}

func freshmanMaterialCleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), freshmanMaterialCleanupTimeout)
}

func (s *Service) createFreshmanMaterialAndSubmitSession(
	ctx context.Context,
	material FreshmanMaterialRecord,
	session *AdmissionSession,
	policy *AdmissionPolicy,
) error {
	return s.submitFreshmanSessionWithPolicy(ctx, session, policy, func(ctx context.Context, tx pgx.Tx) error {
		return s.repo.CreateFreshmanMaterialTx(ctx, tx, material)
	})
}

func (s *Service) submitFreshmanSessionWithPolicy(
	ctx context.Context,
	session *AdmissionSession,
	policy *AdmissionPolicy,
	beforeSubmit func(context.Context, pgx.Tx) error,
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
		if beforeSubmit != nil {
			if err := beforeSubmit(ctx, tx); err != nil {
				return err
			}
		}
		submitted, err := s.repo.MarkMaterialSubmittedTx(ctx, tx, session.ID, deadline)
		if err != nil {
			return err
		}
		return s.queueInitialBotActionsTx(ctx, tx, submitted, now)
	})
}

func (s *Service) ensureFreshmanDesktopCaptureAllowed(ctx context.Context, applicationID string) error {
	hasMaterial, err := s.repo.FreshmanApplicationHasMaterial(ctx, applicationID)
	if err != nil {
		return err
	}
	if hasMaterial {
		return ErrAdmissionCameraHandoffLocked
	}
	active, err := s.repo.GetActiveFreshmanCameraHandoff(ctx, applicationID, s.now())
	if err != nil {
		return err
	}
	if active == nil {
		return nil
	}
	if active.Status == FreshmanCameraHandoffPending {
		return s.repo.ExpirePendingFreshmanCameraHandoffs(ctx, applicationID, s.now())
	}
	if active.Status == FreshmanCameraHandoffUploaded {
		return ErrAdmissionCameraHandoffLocked
	}
	return nil
}

func (s *Service) hydrateFreshmanCameraHandoff(
	_ context.Context,
	handoff *FreshmanCameraHandoff,
) (*FreshmanCameraHandoff, error) {
	if handoff == nil {
		return nil, ErrAdmissionCameraHandoffNotFound
	}
	if s.now().After(handoff.ExpiresAt) && handoff.Status == FreshmanCameraHandoffPending {
		handoff.Status = FreshmanCameraHandoffExpired
	}
	return handoff, nil
}

func (s *Service) recoverFreshmanCameraHandoffUpload(
	ctx context.Context,
	handoff *FreshmanCameraHandoff,
) (*FreshmanCameraHandoff, error) {
	hasMaterial, err := s.repo.FreshmanApplicationHasMaterial(ctx, handoff.ApplicationID)
	if err != nil {
		return nil, err
	}
	if !hasMaterial {
		return nil, ErrAdmissionCameraHandoffLocked
	}
	app, err := s.repo.GetFreshmanApplicationByID(ctx, handoff.ApplicationID)
	if err != nil {
		return nil, err
	}
	session, _, err := s.loadApplicationSessionPolicy(ctx, app)
	if err != nil {
		return nil, err
	}
	if err := s.ensureFreshmanMaterialSubmitted(ctx, session); err != nil {
		return nil, err
	}
	if handoff.Status == FreshmanCameraHandoffPending {
		if err := s.repo.MarkFreshmanCameraHandoffUploaded(ctx, handoff.ID, s.now()); err != nil &&
			!errors.Is(err, ErrAdmissionCameraHandoffLocked) {
			return nil, err
		}
	}
	current, err := s.repo.GetFreshmanCameraHandoffByID(ctx, handoff.ID)
	if err != nil {
		return nil, err
	}
	return s.hydrateFreshmanCameraHandoff(ctx, current)
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
		return s.submitFreshmanSessionWithPolicy(ctx, session, policy, nil)
	case StatusMaterialSubmitted, StatusVerified:
		return nil
	default:
		return ErrAdmissionInvalidStatus
	}
}

func (s *Service) buildFreshmanCameraMobileURL(token string) string {
	return s.returnURLOrigin + "/admission/freshman/camera/" + url.PathEscape(token)
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

func (s *Service) loadApplicationSessionPolicy(
	ctx context.Context,
	app *FreshmanApplication,
) (*AdmissionSession, *AdmissionPolicy, error) {
	if app.AdmissionSessionID == nil {
		return nil, nil, ErrAdmissionLinkedSessionRequired
	}
	session, err := s.repo.GetSessionByID(ctx, *app.AdmissionSessionID)
	if err != nil {
		return nil, nil, err
	}
	policy, err := s.loadPolicy(ctx, session.Platform, session.GuildID)
	if err != nil {
		return nil, nil, err
	}
	return session, policy, nil
}

func (s *Service) buildFreshmanMaterial(
	input CameraCaptureInput,
	maxBytes int64,
) (FreshmanMaterialRecord, []byte, error) {
	contentType := normalizeMaterialContentType(input.ContentType)
	if contentType == "" {
		return FreshmanMaterialRecord{}, nil, ErrAdmissionMaterialInvalidType
	}
	content, err := decodeCameraImage(input.ImageBase64, maxBytes)
	if err != nil {
		return FreshmanMaterialRecord{}, nil, err
	}
	if err := validateCameraImage(content, contentType); err != nil {
		return FreshmanMaterialRecord{}, nil, err
	}
	material, err := newFreshmanMaterialRecord(input.ApplicationID, content, contentType)
	return material, content, err
}

func decodeCameraImage(encoded string, maxBytes int64) ([]byte, error) {
	content, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, ErrAdmissionMaterialInvalidData
	}
	if int64(len(content)) > maxBytes {
		return nil, ErrAdmissionMaterialTooLarge
	}
	return content, nil
}

func validateCameraImage(content []byte, contentType string) error {
	format, err := decodeImageFormat(content)
	if err != nil {
		return err
	}
	if imageFormatContentType(format) != contentType {
		return ErrAdmissionMaterialInvalidData
	}
	return nil
}

func decodeImageFormat(content []byte) (string, error) {
	_, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		return "", ErrAdmissionMaterialInvalidData
	}
	return format, nil
}

func newFreshmanMaterialRecord(
	applicationID string,
	content []byte,
	contentType string,
) (FreshmanMaterialRecord, error) {
	materialID, err := id.New()
	if err != nil {
		return FreshmanMaterialRecord{}, err
	}
	sum := sha256.Sum256(content)
	return FreshmanMaterialRecord{
		ID:            materialID,
		ApplicationID: applicationID,
		ObjectKey:     freshmanMaterialObjectKey(applicationID, materialID, contentType),
		ContentType:   contentType,
		SizeBytes:     int64(len(content)),
		SHA256:        hex.EncodeToString(sum[:]),
	}, nil
}

func freshmanMaterialObjectKey(applicationID, materialID string, contentType string) string {
	return freshmanMaterialObjectPrefix + applicationID + "/" + materialID + materialExtension(contentType)
}

func normalizeMaterialContentType(value string) string {
	contentType := strings.ToLower(strings.TrimSpace(value))
	switch contentType {
	case "image/jpeg", "image/png", "image/webp":
		return contentType
	default:
		return ""
	}
}

func imageFormatContentType(format string) string {
	switch format {
	case "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	default:
		return ""
	}
}

func materialExtension(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".img"
	}
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
