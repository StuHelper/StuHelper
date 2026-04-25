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
	"strings"
	"time"

	_ "golang.org/x/image/webp"
)

const maxIdentityPhotoSize = 5 * 1024 * 1024

// GetIdentity 获取实名认证状态信息（不含敏感字段）
func (s *Service) GetIdentity(ctx context.Context, userID int64) (*IdentityStatus, error) {
	return s.repo.GetIdentityStatusByUserID(ctx, userID)
}

// SubmitIdentity 提交实名认证
func (s *Service) SubmitIdentity(ctx context.Context, userID int64, req SubmitIdentityRequest) (*IdentityStatus, error) {
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

	if req.DocType != DocTypeMainlandID {
		if req.DocPhotoFront == nil || *req.DocPhotoFront == "" {
			return nil, ErrPhotoRequired
		}
		if err := validateSubmittedIdentityPhotoRef(userID, req.DocPhotoFront); err != nil {
			return nil, err
		}
		if err := validateSubmittedIdentityPhotoRef(userID, req.DocPhotoBack); err != nil {
			return nil, err
		}
		if err := validateSubmittedIdentityPhotoRef(userID, req.DocPhotoSelfie); err != nil {
			return nil, err
		}
	}

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
	if provided := strings.TrimSpace(contentType); provided != "" && provided != detectedType {
		return nil, "", ErrIdentityPhotoInvalidType
	}
	if _, _, err := image.DecodeConfig(bytes.NewReader(content)); err != nil {
		return nil, "", ErrIdentityPhotoInvalidData
	}
	return content, detectedType, nil
}

func validateSubmittedIdentityPhotoRef(userID int64, value *string) error {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	raw := strings.TrimSpace(*value)
	if !strings.HasPrefix(raw, identityPhotoPrefix(userID)) {
		return ErrIdentityPhotoInvalidRef
	}
	return nil
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
