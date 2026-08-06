package studentverification

import (
	"bytes"
	"context"
	"crypto/rand"
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

	appcrypto "github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/ctxutil"
)

const (
	manualMaterialObjectPrefix   = "student-verification/manual/"
	manualHandoffTokenHashDomain = "student-verification-manual-handoff:v1:"
	manualMaterialCleanupTimeout = 5 * time.Second
)

func (s *Service) UploadManualCameraCapture(
	ctx context.Context,
	input ManualCameraCaptureInput,
) (*ManualReviewCase, error) {
	if s.manualMaterialStore == nil {
		return nil, ErrManualMaterialStoreUnavailable
	}
	reviewCase, config, policy, err := s.loadManualReviewMaterialContext(
		ctx, input.UserID, input.ApplicationID,
	)
	if err != nil {
		return nil, err
	}
	material, content, err := buildManualReviewMaterial(input, reviewCase.ID, policy, s.now())
	if err != nil {
		return nil, err
	}
	if err := s.manualMaterialStore.PutManualReviewMaterial(
		ctx, material.ObjectKey, content, material.ContentType,
	); err != nil {
		return nil, ErrManualMaterialStoreUnavailable
	}
	now := s.now()
	err = s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		lockedCase, err := s.repo.GetManualReviewCaseForUpdateTx(ctx, tx, reviewCase.ID)
		if err != nil {
			return err
		}
		if lockedCase.UserID != input.UserID || lockedCase.ApplicationID != input.ApplicationID ||
			(lockedCase.Status != ManualReviewDraft && lockedCase.Status != ManualReviewSupplementRequired) {
			return ErrManualReviewState
		}
		lockedConfig, err := s.repo.GetMethodConfigTx(
			ctx, tx, lockedCase.SchoolCode, MethodManualMaterialReview,
		)
		if err != nil || !methodHealthy(lockedConfig) ||
			lockedConfig.PrivacyNoticeVersion != config.PrivacyNoticeVersion {
			return ErrMethodUnavailable
		}
		_, lockedPolicy, err := decodeManualReviewConfiguration(lockedConfig)
		if err != nil || lockedPolicy.MaxMaterialBytes != policy.MaxMaterialBytes ||
			lockedPolicy.MaxMaterials != policy.MaxMaterials {
			return ErrMethodUnavailable
		}
		activeHandoff, err := s.repo.GetActiveManualCameraHandoffTx(ctx, tx, lockedCase.ID, now)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if activeHandoff != nil {
			if activeHandoff.Status == ManualHandoffUploaded {
				return ErrManualHandoffState
			}
			if err := s.repo.ExpirePendingManualCameraHandoffTx(
				ctx, tx, activeHandoff.ID, now,
			); err != nil {
				return err
			}
		}
		_, err = s.repo.AddManualReviewMaterialTx(
			ctx, tx, lockedCase.ID, input.UserID, material, lockedPolicy.MaxMaterials, now,
		)
		return err
	})
	if err != nil {
		if cleanupErr := s.cleanupManualReviewMaterial(ctx, material.ObjectKey); cleanupErr != nil {
			return nil, fmt.Errorf("%w: cleanup stored manual material: %v", err, cleanupErr)
		}
		return nil, err
	}
	return s.GetManualReview(ctx, input.UserID, input.ApplicationID)
}

func (s *Service) CreateManualCameraHandoff(
	ctx context.Context,
	userID int64,
	applicationID string,
) (*ManualCameraHandoff, error) {
	reviewCase, _, policy, err := s.loadManualReviewMaterialContext(ctx, userID, applicationID)
	if err != nil {
		return nil, err
	}
	if s.rosterCipher == nil || s.rosterEncryptionKeyVersion <= 0 ||
		s.manualReviewPublicBaseURL == "" {
		return nil, ErrDependencyUnavailable
	}
	now := s.now()
	var stored *storedManualCameraHandoff
	err = s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		lockedCase, err := s.repo.GetManualReviewCaseForUpdateTx(ctx, tx, reviewCase.ID)
		if err != nil {
			return err
		}
		if lockedCase.UserID != userID ||
			(lockedCase.Status != ManualReviewDraft && lockedCase.Status != ManualReviewSupplementRequired) {
			return ErrManualReviewState
		}
		if err := s.repo.ExpireManualCameraHandoffsTx(ctx, tx, lockedCase.ID, now); err != nil {
			return err
		}
		active, err := s.repo.GetActiveManualCameraHandoffTx(ctx, tx, lockedCase.ID, now)
		if err == nil {
			stored = active
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		token, err := generateManualHandoffToken()
		if err != nil {
			return err
		}
		tokenHash, err := s.hashManualHandoffToken(token)
		if err != nil {
			return err
		}
		tokenEnc, err := s.rosterCipher.Encrypt(token)
		if err != nil {
			return err
		}
		handoffID, err := newID()
		if err != nil {
			return err
		}
		handoff := ManualCameraHandoff{
			ID: handoffID, CaseID: lockedCase.ID, ApplicationID: applicationID,
			UserID: userID, Status: ManualHandoffPending,
			ExpiresAt: now.Add(time.Duration(policy.HandoffTTLSeconds) * time.Second),
			CreatedAt: now, MaxMaterialBytes: policy.MaxMaterialBytes,
		}
		if err := s.repo.CreateManualCameraHandoffTx(
			ctx, tx, handoff, tokenHash, tokenEnc, s.rosterEncryptionKeyVersion,
		); err != nil {
			return err
		}
		stored = &storedManualCameraHandoff{
			ManualCameraHandoff: handoff, TokenEnc: tokenEnc,
			EncryptionKeyVersion: &s.rosterEncryptionKeyVersion,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.presentManualCameraHandoff(stored, policy.MaxMaterialBytes)
}

func (s *Service) GetManualCameraHandoff(
	ctx context.Context,
	userID int64,
	applicationID string,
	handoffID string,
) (*ManualCameraHandoff, error) {
	handoff, err := s.repo.GetManualCameraHandoffForUser(ctx, applicationID, handoffID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrManualHandoffNotFound
	}
	if err != nil {
		return nil, err
	}
	_, _, policy, err := s.loadManualReviewMaterialContext(ctx, userID, applicationID)
	if err != nil {
		return nil, err
	}
	if (handoff.Status == ManualHandoffPending || handoff.Status == ManualHandoffUploaded) &&
		!handoff.ExpiresAt.After(s.now()) {
		if err := s.expireManualHandoff(ctx, handoff.CaseID, handoff.ID); err != nil {
			return nil, err
		}
		if handoff.Status == ManualHandoffPending {
			handoff.Status = ManualHandoffExpired
		} else {
			handoff.Status = ManualHandoffLocked
		}
		handoff.TokenEnc = nil
		handoff.EncryptionKeyVersion = nil
	}
	return s.presentManualCameraHandoff(handoff, policy.MaxMaterialBytes)
}

func (s *Service) PreviewManualCameraHandoff(
	ctx context.Context,
	token string,
) (*ManualCameraHandoff, error) {
	tokenHash, err := s.hashManualHandoffToken(token)
	if err != nil {
		return nil, ErrManualHandoffNotFound
	}
	handoff, err := s.repo.GetManualCameraHandoffByTokenHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrManualHandoffNotFound
	}
	if err != nil {
		return nil, err
	}
	reviewCase, config, policy, err := s.loadManualReviewMaterialContext(
		ctx, handoff.UserID, handoff.ApplicationID,
	)
	if err != nil || reviewCase.ID != handoff.CaseID || config == nil {
		return nil, ErrManualHandoffNotFound
	}
	if !handoff.ExpiresAt.After(s.now()) &&
		(handoff.Status == ManualHandoffPending || handoff.Status == ManualHandoffUploaded) {
		if err := s.expireManualHandoff(ctx, handoff.CaseID, handoff.ID); err != nil {
			return nil, err
		}
		return nil, ErrManualHandoffExpired
	}
	view := handoff.ManualCameraHandoff
	view.MobileURL = ""
	view.MaxMaterialBytes = policy.MaxMaterialBytes
	return &view, nil
}

func (s *Service) UploadManualHandoffCameraCapture(
	ctx context.Context,
	input ManualCameraCaptureInput,
) (*ManualCameraHandoff, error) {
	if s.manualMaterialStore == nil {
		return nil, ErrManualMaterialStoreUnavailable
	}
	tokenHash, err := s.hashManualHandoffToken(input.Token)
	if err != nil {
		return nil, ErrManualHandoffNotFound
	}
	handoff, err := s.repo.GetManualCameraHandoffByTokenHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrManualHandoffNotFound
	}
	if err != nil {
		return nil, err
	}
	if handoff.Status == ManualHandoffUploaded || handoff.Status == ManualHandoffLocked {
		return s.PreviewManualCameraHandoff(ctx, input.Token)
	}
	if handoff.Status != ManualHandoffPending || !handoff.ExpiresAt.After(s.now()) {
		return nil, ErrManualHandoffExpired
	}
	reviewCase, _, policy, err := s.loadManualReviewMaterialContext(
		ctx, handoff.UserID, handoff.ApplicationID,
	)
	if err != nil || reviewCase.ID != handoff.CaseID {
		return nil, ErrManualHandoffState
	}
	material, content, err := buildManualReviewMaterial(input, handoff.CaseID, policy, s.now())
	if err != nil {
		return nil, err
	}
	if err := s.manualMaterialStore.PutManualReviewMaterial(
		ctx, material.ObjectKey, content, material.ContentType,
	); err != nil {
		return nil, ErrManualMaterialStoreUnavailable
	}
	now := s.now()
	err = s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		lockedCase, err := s.repo.GetManualReviewCaseForUpdateTx(ctx, tx, handoff.CaseID)
		if err != nil {
			return err
		}
		if lockedCase.Status != ManualReviewDraft && lockedCase.Status != ManualReviewSupplementRequired {
			return ErrManualReviewState
		}
		lockedHandoff, err := s.repo.GetManualCameraHandoffByTokenHashForUpdateTx(ctx, tx, tokenHash)
		if err != nil {
			return err
		}
		if lockedHandoff.CaseID != lockedCase.ID || lockedHandoff.Status != ManualHandoffPending ||
			!lockedHandoff.ExpiresAt.After(now) {
			return ErrManualHandoffState
		}
		if _, err := s.repo.AddManualReviewMaterialTx(
			ctx, tx, lockedCase.ID, lockedCase.UserID, material, policy.MaxMaterials, now,
		); err != nil {
			return err
		}
		return s.repo.MarkManualHandoffUploadedTx(ctx, tx, lockedHandoff.ID, material.ID, now)
	})
	if err != nil {
		if cleanupErr := s.cleanupManualReviewMaterial(ctx, material.ObjectKey); cleanupErr != nil {
			return nil, fmt.Errorf("%w: cleanup stored handoff material: %v", err, cleanupErr)
		}
		return nil, err
	}
	return s.PreviewManualCameraHandoff(ctx, input.Token)
}

func (s *Service) ChooseManualCameraContinuation(
	ctx context.Context,
	token string,
	continueOn string,
) (*ManualCameraHandoff, error) {
	if continueOn != "desktop" && continueOn != "mobile" {
		return nil, ErrManualHandoffChoice
	}
	tokenHash, err := s.hashManualHandoffToken(token)
	if err != nil {
		return nil, ErrManualHandoffNotFound
	}
	initial, err := s.repo.GetManualCameraHandoffByTokenHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrManualHandoffNotFound
	}
	if err != nil {
		return nil, err
	}
	now := s.now()
	var result *storedManualCameraHandoff
	err = s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := s.repo.GetManualReviewCaseForUpdateTx(ctx, tx, initial.CaseID); err != nil {
			return err
		}
		locked, err := s.repo.GetManualCameraHandoffByTokenHashForUpdateTx(ctx, tx, tokenHash)
		if err != nil {
			return err
		}
		if !locked.ExpiresAt.After(now) {
			return ErrManualHandoffExpired
		}
		if locked.Status == ManualHandoffLocked && locked.ContinueOn != nil && *locked.ContinueOn == continueOn {
			result = locked
			return nil
		}
		if locked.Status != ManualHandoffUploaded {
			return ErrManualHandoffState
		}
		if err := s.repo.ChooseManualHandoffContinuationTx(
			ctx, tx, locked.ID, continueOn, now,
		); err != nil {
			return err
		}
		locked.Status = ManualHandoffLocked
		locked.ContinueOn = &continueOn
		locked.ChosenAt = &now
		locked.TokenEnc = nil
		locked.EncryptionKeyVersion = nil
		result = locked
		return nil
	})
	if err != nil {
		return nil, err
	}
	view := result.ManualCameraHandoff
	view.MobileURL = ""
	return &view, nil
}

// ResumeManualCameraHandoff lets the authenticated owner continue on the
// phone after the public handoff has been atomically locked to that device.
// The same token remains only a lookup capability: account ownership is still
// required, and mismatches are intentionally indistinguishable from absence.
func (s *Service) ResumeManualCameraHandoff(
	ctx context.Context,
	userID int64,
	token string,
) (*ApplicationView, error) {
	if userID <= 0 {
		return nil, ErrManualHandoffNotFound
	}
	tokenHash, err := s.hashManualHandoffToken(token)
	if err != nil {
		return nil, ErrManualHandoffNotFound
	}
	handoff, err := s.repo.GetManualCameraHandoffByTokenHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && handoff.UserID != userID) {
		return nil, ErrManualHandoffNotFound
	}
	if err != nil {
		return nil, err
	}
	if handoff.Status != ManualHandoffLocked || handoff.ContinueOn == nil || *handoff.ContinueOn != "mobile" {
		return nil, ErrManualHandoffState
	}
	return s.GetApplication(ctx, userID, handoff.ApplicationID)
}

func (s *Service) loadManualReviewMaterialContext(
	ctx context.Context,
	userID int64,
	applicationID string,
) (*ManualReviewCase, *MethodConfig, manualReviewPolicy, error) {
	reviewCase, err := s.repo.GetManualReviewCaseForUser(ctx, applicationID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, manualReviewPolicy{}, ErrManualReviewNotFound
	}
	if err != nil {
		return nil, nil, manualReviewPolicy{}, err
	}
	if reviewCase.Status != ManualReviewDraft && reviewCase.Status != ManualReviewSupplementRequired {
		return nil, nil, manualReviewPolicy{}, ErrManualReviewState
	}
	config, err := s.repo.GetMethodConfig(ctx, reviewCase.SchoolCode, MethodManualMaterialReview)
	if err != nil || !methodHealthy(config) ||
		config.PrivacyNoticeVersion != reviewCase.PrivacyNoticeVersion {
		return nil, nil, manualReviewPolicy{}, ErrMethodUnavailable
	}
	_, policy, err := decodeManualReviewConfiguration(config)
	if err != nil {
		return nil, nil, manualReviewPolicy{}, err
	}
	return reviewCase, config, policy, nil
}

func buildManualReviewMaterial(
	input ManualCameraCaptureInput,
	caseID string,
	policy manualReviewPolicy,
	now time.Time,
) (ManualReviewMaterial, []byte, error) {
	if input.CaptureSource != "web_camera" {
		return ManualReviewMaterial{}, nil, ErrManualMaterialInvalidData
	}
	facingMode := input.RequestedFacingMode
	if facingMode == "" {
		facingMode = "environment"
	}
	if facingMode != "environment" && facingMode != "unknown" {
		return ManualReviewMaterial{}, nil, ErrManualMaterialInvalidData
	}
	contentType := normalizeManualMaterialContentType(input.ContentType)
	if contentType == "" {
		return ManualReviewMaterial{}, nil, ErrManualMaterialInvalidType
	}
	content, err := base64.StdEncoding.DecodeString(strings.TrimSpace(input.ImageBase64))
	if err != nil || len(content) == 0 {
		return ManualReviewMaterial{}, nil, ErrManualMaterialInvalidData
	}
	if int64(len(content)) > policy.MaxMaterialBytes {
		return ManualReviewMaterial{}, nil, ErrManualMaterialTooLarge
	}
	imageConfig, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil || manualImageFormatContentType(format) != contentType {
		return ManualReviewMaterial{}, nil, ErrManualMaterialInvalidData
	}
	if imageConfig.Width < policy.MinimumImageDimension || imageConfig.Height < policy.MinimumImageDimension ||
		imageConfig.Width > policy.MaximumImageDimension || imageConfig.Height > policy.MaximumImageDimension ||
		int64(imageConfig.Width)*int64(imageConfig.Height) > policy.MaximumImagePixels {
		return ManualReviewMaterial{}, nil, ErrManualMaterialPixelBounds
	}
	materialID, err := newID()
	if err != nil {
		return ManualReviewMaterial{}, nil, err
	}
	sum := sha256.Sum256(content)
	material := ManualReviewMaterial{
		ID: materialID, ContentType: contentType, SizeBytes: int64(len(content)),
		SHA256: hex.EncodeToString(sum[:]), Width: imageConfig.Width, Height: imageConfig.Height,
		CaptureSource: "web_camera", FacingMode: facingMode,
		RetentionAt: now.AddDate(0, 0, policy.MaterialRetentionDays), CapturedAt: now,
	}
	material.ObjectKey = manualMaterialObjectKey(caseID, materialID, contentType)
	return material, content, nil
}

func normalizeManualMaterialContentType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "image/jpeg":
		return "image/jpeg"
	case "image/png":
		return "image/png"
	case "image/webp":
		return "image/webp"
	default:
		return ""
	}
}

func manualImageFormatContentType(format string) string {
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

func manualMaterialObjectKey(caseID string, materialID string, contentType string) string {
	extension := ".img"
	switch contentType {
	case "image/jpeg":
		extension = ".jpg"
	case "image/png":
		extension = ".png"
	case "image/webp":
		extension = ".webp"
	}
	return manualMaterialObjectPrefix + caseID + "/" + materialID + extension
}

func generateManualHandoffToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (s *Service) hashManualHandoffToken(token string) (string, error) {
	token = strings.TrimSpace(token)
	if len(token) < 32 || len(token) > 256 {
		return "", ErrManualHandoffNotFound
	}
	return appcrypto.HMACHashWithKey(manualHandoffTokenHashDomain+token, s.hmacKey)
}

func (s *Service) presentManualCameraHandoff(
	stored *storedManualCameraHandoff,
	maxMaterialBytes int64,
) (*ManualCameraHandoff, error) {
	if stored == nil {
		return nil, ErrManualHandoffNotFound
	}
	view := stored.ManualCameraHandoff
	view.MaxMaterialBytes = maxMaterialBytes
	if (stored.Status == ManualHandoffPending || stored.Status == ManualHandoffUploaded) &&
		len(stored.TokenEnc) > 0 {
		if s.rosterCipher == nil {
			return nil, ErrDependencyUnavailable
		}
		token, err := s.rosterCipher.Decrypt(stored.TokenEnc)
		if err != nil {
			return nil, ErrDependencyUnavailable
		}
		view.MobileURL = s.manualReviewPublicBaseURL +
			"/student-verification/manual-camera/" + url.PathEscape(token)
	}
	return &view, nil
}

func (s *Service) expireManualHandoff(ctx context.Context, caseID string, handoffID string) error {
	return s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := s.repo.GetManualReviewCaseForUpdateTx(ctx, tx, caseID); err != nil {
			return err
		}
		return s.repo.CloseExpiredManualCameraHandoffTx(ctx, tx, handoffID, s.now())
	})
}

func (s *Service) cleanupManualReviewMaterial(ctx context.Context, objectKey string) error {
	cleanupCtx, cancel := ctxutil.DetachedTimeout(ctx, manualMaterialCleanupTimeout)
	defer cancel()
	return s.manualMaterialStore.DeleteManualReviewMaterial(cleanupCtx, objectKey)
}
