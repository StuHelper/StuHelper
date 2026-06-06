package user

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	_ "golang.org/x/image/webp"
)

const (
	maxIdentityPhotoSize     = 5 * 1024 * 1024
	maxIdentityRealNameRunes = 100
)

// GetIdentity 获取实名认证状态信息（不含敏感字段）
func (s *Service) GetIdentity(ctx context.Context, userID int64) (*IdentityStatus, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	return s.repo.GetIdentityStatusByUserID(ctx, userID)
}

// SubmitIdentity 提交实名认证
func (s *Service) SubmitIdentity(ctx context.Context, userID int64, req SubmitIdentityRequest) (*IdentityStatus, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetIdentityStatusByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("SubmitIdentity check existing: %w", err)
	}
	if existing != nil {
		if existing.Verified {
			return nil, ErrIdentityAlreadyVerified
		}
		if !canResubmitIdentity(existing) {
			return nil, ErrIdentityAlreadyExists
		}
	}

	req.RealName = strings.TrimSpace(req.RealName)
	req.DocNumber = normalizeIdentityDocNumber(req.DocType, req.DocNumber)
	if req.RealName == "" || utf8.RuneCountInString(req.RealName) > maxIdentityRealNameRunes {
		return nil, ErrIdentityRealNameInvalid
	}
	if !isValidIdentityDocNumber(req.DocType, req.DocNumber) {
		return nil, ErrIdentityDocNumberInvalid
	}

	photos, err := s.validateSubmittedIdentityPhotos(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	req.DocPhotoFront = photos.front
	req.DocPhotoBack = photos.back
	req.DocPhotoSelfie = photos.selfie

	personUID := s.computePersonUID(req.DocType, req.DocNumber)
	docNumberEnc, err := s.docCipher.Encrypt(req.DocNumber)
	if err != nil {
		return nil, fmt.Errorf("SubmitIdentity encrypt doc number: %w", err)
	}

	identity := &IdentityRecord{
		UserID:         userID,
		DocType:        req.DocType,
		DocNumberEnc:   docNumberEnc,
		PersonUID:      personUID,
		RealName:       req.RealName,
		Verified:       false,
		DocPhotoFront:  req.DocPhotoFront,
		DocPhotoBack:   req.DocPhotoBack,
		DocPhotoSelfie: req.DocPhotoSelfie,
	}

	if req.DocType == DocTypeMainlandID {
		matched, err := s.tryAcademicDBMatch(ctx, req.DocNumber, req.RealName)
		if err != nil {
			return nil, fmt.Errorf("SubmitIdentity academic match: %w", err)
		}
		if matched {
			method := VerifyMethodAcademicDB
			now := time.Now()
			identity.Verified = true
			identity.VerifyMethod = &method
			identity.ReviewedAt = &now
			identity.VerifiedAt = &now
		}
	}

	if existing != nil {
		if err := s.repo.UpdateIdentitySubmission(ctx, identity); err != nil {
			return nil, fmt.Errorf("SubmitIdentity resubmit: %w", err)
		}
	} else if err := s.repo.CreateIdentity(ctx, identity); err != nil {
		return nil, fmt.Errorf("SubmitIdentity create: %w", err)
	}

	result, err := s.repo.GetIdentityStatusByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("SubmitIdentity reload: %w", err)
	}
	return result, nil
}

// UploadIdentityPhoto 上传实名认证照片到对象存储。
func (s *Service) UploadIdentityPhoto(ctx context.Context, userID int64, req UploadIdentityPhotoRequest) (string, error) {
	if err := validateUserID(userID); err != nil {
		return "", err
	}
	if s.photoStore == nil {
		return "", ErrIdentityPhotoStoreDisabled
	}
	if !isValidIdentityPhotoSlot(req.Slot) {
		return "", ErrIdentityPhotoInvalidRef
	}

	content, detectedType, err := decodeAndValidateIdentityPhoto(req.ContentType, req.DataBase64)
	if err != nil {
		return "", err
	}

	key := buildIdentityPhotoKey(userID, req.Slot, photoExt(detectedType))
	if err := s.photoStore.Upload(ctx, key, content, detectedType); err != nil {
		return "", fmt.Errorf("UploadIdentityPhoto upload: %w", err)
	}
	return key, nil
}

// ResolveIdentityReviewItemAssets 将对象存储 key 解析为签名 URL，兼容历史 data URL / 外链。
func (s *Service) ResolveIdentityReviewItemAssets(ctx context.Context, item *IdentityReviewItem) (*IdentityReviewItem, error) {
	if item == nil {
		return nil, nil
	}
	resolved := *item
	var err error
	resolved.DocPhotoFront, err = s.resolveIdentityPhotoValue(ctx, item.DocPhotoFront)
	if err != nil {
		return nil, err
	}
	resolved.DocPhotoBack, err = s.resolveIdentityPhotoValue(ctx, item.DocPhotoBack)
	if err != nil {
		return nil, err
	}
	resolved.DocPhotoSelfie, err = s.resolveIdentityPhotoValue(ctx, item.DocPhotoSelfie)
	if err != nil {
		return nil, err
	}
	return &resolved, nil
}

// tryAcademicDBMatch 尝试通过学籍数据库匹配进行自动实名验证
func (s *Service) tryAcademicDBMatch(ctx context.Context, docNumber, realName string) (bool, error) {
	schools, err := s.repo.ListSchoolConfigs(ctx)
	if err != nil {
		return false, err
	}
	if len(schools) == 0 {
		return false, nil
	}

	trimmedRealName := strings.TrimSpace(realName)
	visitedTables := make(map[string]struct{}, len(schools))

	for i := range schools {
		school := &schools[i]
		tableName, err := s.ensureAcademicTableConfigured(school)
		if err != nil {
			return false, fmt.Errorf("school %d academic table config: %w", school.SchoolID, err)
		}
		if _, ok := visitedTables[tableName]; ok {
			continue
		}
		visitedTables[tableName] = struct{}{}

		students, err := s.findAcademicStudentsByPersonUID(ctx, DocTypeMainlandID, docNumber, tableName)
		if err != nil {
			return false, fmt.Errorf("academic DB auto-match query failed for school %d table %s: %w", school.SchoolID, tableName, err)
		}

		for _, stu := range students {
			if stu.XM != nil && strings.EqualFold(strings.TrimSpace(*stu.XM), trimmedRealName) {
				return true, nil
			}
		}
	}
	return false, nil
}

func decodeAndValidateIdentityPhoto(contentType, dataBase64 string) ([]byte, string, error) {
	raw := strings.TrimSpace(dataBase64)
	if raw == "" {
		return nil, "", ErrIdentityPhotoInvalidData
	}
	if idx := strings.Index(raw, ","); strings.HasPrefix(raw, "data:") && idx > 0 {
		raw = raw[idx+1:]
	}

	content, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, "", ErrIdentityPhotoInvalidData
	}
	if len(content) == 0 {
		return nil, "", ErrIdentityPhotoInvalidData
	}
	if len(content) > maxIdentityPhotoSize {
		return nil, "", ErrIdentityPhotoTooLarge
	}

	detectedType := http.DetectContentType(content)
	if !isAllowedIdentityPhotoType(detectedType) {
		return nil, "", ErrIdentityPhotoInvalidType
	}
	provided := strings.TrimSpace(contentType)
	if !isAllowedIdentityPhotoType(provided) || provided != detectedType {
		return nil, "", ErrIdentityPhotoInvalidType
	}
	if _, _, err := image.DecodeConfig(bytes.NewReader(content)); err != nil {
		return nil, "", ErrIdentityPhotoInvalidData
	}
	return content, detectedType, nil
}

type submittedIdentityPhotos struct {
	front  *string
	back   *string
	selfie *string
}

func (s *Service) validateSubmittedIdentityPhotos(
	ctx context.Context,
	userID int64,
	req SubmitIdentityRequest,
) (submittedIdentityPhotos, error) {
	front, err := normalizeSubmittedIdentityPhotoRef(userID, IdentityPhotoSlotFront, req.DocPhotoFront)
	if err != nil {
		return submittedIdentityPhotos{}, err
	}
	back, err := normalizeSubmittedIdentityPhotoRef(userID, IdentityPhotoSlotBack, req.DocPhotoBack)
	if err != nil {
		return submittedIdentityPhotos{}, err
	}
	selfie, err := normalizeSubmittedIdentityPhotoRef(userID, IdentityPhotoSlotSelfie, req.DocPhotoSelfie)
	if err != nil {
		return submittedIdentityPhotos{}, err
	}
	if req.DocType != DocTypeMainlandID && front == nil {
		return submittedIdentityPhotos{}, ErrPhotoRequired
	}

	photos := submittedIdentityPhotos{front: front, back: back, selfie: selfie}
	if !photos.hasAny() {
		return photos, nil
	}
	if s.photoStore == nil {
		return submittedIdentityPhotos{}, ErrIdentityPhotoStoreDisabled
	}
	for _, key := range photos.keys() {
		if _, err := s.photoStore.PresignGetURL(ctx, key); err != nil {
			return submittedIdentityPhotos{}, fmt.Errorf("verify identity photo reference: %w", err)
		}
	}
	return photos, nil
}

func (p submittedIdentityPhotos) hasAny() bool {
	return p.front != nil || p.back != nil || p.selfie != nil
}

func (p submittedIdentityPhotos) keys() []string {
	keys := make([]string, 0, 3)
	for _, ref := range []*string{p.front, p.back, p.selfie} {
		if ref != nil {
			keys = append(keys, *ref)
		}
	}
	return keys
}

func normalizeSubmittedIdentityPhotoRef(userID int64, slot string, value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	key := normalizeIdentityPhotoKey(*value)
	if key == "" {
		return nil, nil
	}
	if !strings.HasPrefix(key, identityPhotoPrefix(userID)) || !identityPhotoKeyMatchesSlot(key, slot) {
		return nil, ErrIdentityPhotoInvalidRef
	}
	return &key, nil
}

func normalizeIdentityPhotoKey(value string) string {
	key := path.Clean("/" + strings.TrimSpace(value))
	return strings.Trim(key, "/")
}

func identityPhotoKeyMatchesSlot(key, slot string) bool {
	segments := strings.Split(key, "/")
	if len(segments) != 5 ||
		segments[0] != "identities" ||
		!isFixedWidthDecimalString(segments[2], 4) ||
		!isFixedWidthDecimalString(segments[3], 2) {
		return false
	}
	filename := path.Base(key)
	ext := strings.ToLower(path.Ext(filename))
	if !isAllowedIdentityPhotoExt(ext) {
		return false
	}
	name := strings.TrimSuffix(filename, ext)
	hyphen := strings.LastIndexByte(name, '-')
	if hyphen <= 0 || name[hyphen+1:] != slot {
		return false
	}
	return isDecimalString(name[:hyphen])
}

func isAllowedIdentityPhotoExt(ext string) bool {
	switch ext {
	case ".jpg", ".png", ".webp":
		return true
	default:
		return false
	}
}

func isDecimalString(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func isFixedWidthDecimalString(value string, width int) bool {
	return len(value) == width && isDecimalString(value)
}

func canResubmitIdentity(existing *IdentityStatus) bool {
	return existing != nil && !existing.Verified && existing.ReviewedAt != nil
}

func buildIdentityPhotoKey(userID int64, slot, ext string) string {
	now := time.Now()
	return fmt.Sprintf("identities/%d/%04d/%02d/%d-%s%s", userID, now.Year(), now.Month(), now.UnixNano(), slot, ext)
}

func identityPhotoPrefix(userID int64) string {
	return fmt.Sprintf("identities/%d/", userID)
}

func photoExt(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}

func isValidIdentityPhotoSlot(slot string) bool {
	switch slot {
	case IdentityPhotoSlotFront, IdentityPhotoSlotBack, IdentityPhotoSlotSelfie:
		return true
	default:
		return false
	}
}

func isAllowedIdentityPhotoType(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func looksLikeLegacyPhotoValue(value string) bool {
	raw := strings.TrimSpace(value)
	return strings.HasPrefix(raw, "data:") ||
		strings.HasPrefix(raw, "http://") ||
		strings.HasPrefix(raw, "https://")
}

func (s *Service) resolveIdentityPhotoValue(ctx context.Context, value *string) (*string, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	raw := strings.TrimSpace(*value)
	if s.photoStore == nil || looksLikeLegacyPhotoValue(raw) {
		return &raw, nil
	}
	url, err := s.photoStore.PresignGetURL(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("presign identity photo: %w", err)
	}
	return &url, nil
}
