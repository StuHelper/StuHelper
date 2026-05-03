package admission

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	_ "golang.org/x/image/webp"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

const freshmanMaterialObjectPrefix = "admission/freshman/"

func (s *Service) CreateFreshmanApplication(
	ctx context.Context,
	input FreshmanApplicationCreateInput,
) (*FreshmanApplication, error) {
	session, err := s.requireLinkedSession(ctx, input.UserID)
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
	if err := s.ensureNoPendingApplication(ctx, input.UserID, input.SchoolID); err != nil {
		return nil, err
	}
	app, err := s.buildFreshmanApplication(input, session.ID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateFreshmanApplication(ctx, app); err != nil {
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
	material, content, err := s.buildFreshmanMaterial(input, policy.MaxMaterialBytes)
	if err != nil {
		return nil, err
	}
	if err := s.materialStore.PutAdmissionMaterial(ctx, material.ObjectKey, content, material.ContentType); err != nil {
		return nil, fmt.Errorf("SubmitCameraCapture store material: %w", err)
	}
	if err := s.repo.CreateFreshmanMaterial(ctx, material); err != nil {
		return nil, err
	}
	if _, err := s.MarkMaterialSubmitted(ctx, session.ID); err != nil {
		return nil, err
	}
	return app, nil
}

func (s *Service) requireLinkedSession(ctx context.Context, userID int64) (*AdmissionSession, error) {
	session, err := s.repo.GetLinkedSessionByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrAdmissionLinkedSessionRequired
	}
	return session, nil
}

func (s *Service) ensureNoPendingApplication(ctx context.Context, userID, schoolID int64) error {
	exists, err := s.repo.HasPendingFreshmanApplication(ctx, userID, schoolID)
	if err != nil {
		return err
	}
	if exists {
		return ErrAdmissionFreshmanPendingExists
	}
	return nil
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
		ObjectKey:     freshmanMaterialObjectPrefix + applicationID + materialExtension(contentType),
		ContentType:   contentType,
		SizeBytes:     int64(len(content)),
		SHA256:        hex.EncodeToString(sum[:]),
	}, nil
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
