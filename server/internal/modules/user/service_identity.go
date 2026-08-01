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
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	_ "golang.org/x/image/webp"

	"github.com/StuHelper/StuHelper/server/internal/pkg/schoolauth"
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
	if !isValidIdentityRealName(req.RealName) {
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

	var academicMatch *academicIdentityMatch
	if req.DocType == DocTypeMainlandID {
		academicMatch, err = s.tryAcademicDBMatch(ctx, userID, req.DocNumber, req.RealName)
		if err != nil {
			return nil, fmt.Errorf("SubmitIdentity academic match: %w", err)
		}
	}
	// 大陆身份证先尝试使用当前账号已绑定的学籍凭据自动核验；无法形成
	// 自动核验凭据时，必须补齐证件正面和本人手持证件照后才能进入人工审核。
	if academicMatch == nil && !photos.hasManualEvidence() {
		return nil, ErrPhotoRequired
	}

	if err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		txExisting, err := s.repo.GetIdentityStatusByUserIDTx(ctx, tx, userID)
		if err != nil {
			return fmt.Errorf("SubmitIdentity check existing tx: %w", err)
		}
		if txExisting != nil {
			if txExisting.Verified {
				return ErrIdentityAlreadyVerified
			}
			if !canResubmitIdentity(txExisting) {
				return ErrIdentityAlreadyExists
			}
		}

		autoVerified := false
		if academicMatch != nil {
			stillValid, err := s.academicDBMatchStillValidTx(ctx, tx, userID, *academicMatch)
			if err != nil {
				return fmt.Errorf("SubmitIdentity revalidate academic match: %w", err)
			}
			if stillValid {
				method := VerifyMethodAcademicDB
				now := time.Now()
				identity.Verified = true
				identity.VerifyMethod = &method
				identity.ReviewedAt = &now
				identity.VerifiedAt = &now
				autoVerified = true
			}
		}
		// 学籍绑定可能在事务开始前发生变化。自动核验凭据失效时不能静默
		// 创建一条没有审核材料的 pending 记录。
		if !autoVerified && !photos.hasManualEvidence() {
			return ErrPhotoRequired
		}

		if txExisting != nil {
			if err := s.repo.UpdateIdentitySubmissionTx(ctx, tx, identity); err != nil {
				return fmt.Errorf("SubmitIdentity resubmit tx: %w", err)
			}
		} else if err := s.repo.CreateIdentityTx(ctx, tx, identity); err != nil {
			return fmt.Errorf("SubmitIdentity create tx: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	result, err := s.repo.GetIdentityStatusByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("SubmitIdentity reload: %w", err)
	}
	return result, nil
}

func isValidIdentityRealName(realName string) bool {
	return realName != "" &&
		utf8.ValidString(realName) &&
		utf8.RuneCountInString(realName) <= maxIdentityRealNameRunes &&
		!strings.ContainsFunc(realName, func(r rune) bool {
			return unicode.IsControl(r) || unicode.In(r, unicode.Cf, unicode.Co, unicode.Cs)
		})
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

// ResolveIdentityReviewItemAssets 将属于当前用户和照片槽位的对象存储 key
// 解析为短期签名 URL。历史 data URL、外链及跨用户/跨槽位 key 均拒绝返回。
func (s *Service) ResolveIdentityReviewItemAssets(ctx context.Context, item *IdentityReviewItem) (*IdentityReviewItem, error) {
	if item == nil {
		return nil, nil
	}
	resolved := *item
	var err error
	resolved.DocPhotoFront, err = s.resolveIdentityReviewPhoto(
		ctx,
		item.UserID,
		IdentityPhotoSlotFront,
		item.DocPhotoFront,
	)
	if err != nil {
		return nil, err
	}
	resolved.DocPhotoBack, err = s.resolveIdentityReviewPhoto(
		ctx,
		item.UserID,
		IdentityPhotoSlotBack,
		item.DocPhotoBack,
	)
	if err != nil {
		return nil, err
	}
	resolved.DocPhotoSelfie, err = s.resolveIdentityReviewPhoto(
		ctx,
		item.UserID,
		IdentityPhotoSlotSelfie,
		item.DocPhotoSelfie,
	)
	if err != nil {
		return nil, err
	}
	return &resolved, nil
}

type academicIdentityMatch struct {
	schoolID  int64
	studentID string
	tableName string
}

// tryAcademicDBMatch 仅在当前账号已经完成学生认证，且证件记录、姓名、
// 学校和账号绑定学号全部一致时，生成可供事务内二次校验的自动实名凭据。
func (s *Service) tryAcademicDBMatch(
	ctx context.Context,
	userID int64,
	docNumber string,
	realName string,
) (*academicIdentityMatch, error) {
	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load verified profile: %w", err)
	}
	if profile == nil || profile.VerificationStatus != StatusVerified || profile.SchoolID == nil {
		return nil, nil
	}

	school, err := s.repo.GetSchoolConfig(ctx, *profile.SchoolID)
	if err != nil {
		return nil, fmt.Errorf("load school %d: %w", *profile.SchoolID, err)
	}
	if school == nil || !school.Enabled {
		return nil, nil
	}
	tableName, err := s.ensureAcademicTableConfigured(school)
	if err != nil {
		return nil, fmt.Errorf("school %d academic table config: %w", school.SchoolID, err)
	}

	boundStudentIDs := profileStudentIDSet(profile)
	if len(boundStudentIDs) == 0 {
		return nil, nil
	}
	students, err := s.findAcademicStudentsByPersonUID(ctx, DocTypeMainlandID, docNumber, tableName)
	if err != nil {
		return nil, fmt.Errorf(
			"academic DB auto-match query failed for school %d table %s: %w",
			school.SchoolID,
			tableName,
			err,
		)
	}

	normalizedRealName := schoolauth.NormalizeAcademicName(realName)
	for _, student := range students {
		studentID := schoolauth.NormalizeStudentID(student.XH)
		if _, ok := boundStudentIDs[studentID]; !ok || student.XM == nil {
			continue
		}
		academicName := schoolauth.NormalizeAcademicName(*student.XM)
		if academicName != "" && strings.EqualFold(academicName, normalizedRealName) {
			return &academicIdentityMatch{
				schoolID:  school.SchoolID,
				studentID: studentID,
				tableName: tableName,
			}, nil
		}
	}
	return nil, nil
}

func (s *Service) academicDBMatchStillValidTx(
	ctx context.Context,
	tx pgx.Tx,
	userID int64,
	match academicIdentityMatch,
) (bool, error) {
	profile, err := s.repo.GetProfileByUserIDForUpdateTx(ctx, tx, userID)
	if err != nil {
		return false, fmt.Errorf("lock verified profile: %w", err)
	}
	if profile == nil || profile.VerificationStatus != StatusVerified || profile.SchoolID == nil ||
		*profile.SchoolID != match.schoolID {
		return false, nil
	}
	if _, ok := profileStudentIDSet(profile)[match.studentID]; !ok {
		return false, nil
	}

	school, err := s.repo.GetSchoolConfigForUpdateTx(ctx, tx, match.schoolID)
	if err != nil {
		return false, fmt.Errorf("lock school %d: %w", match.schoolID, err)
	}
	if school == nil || !school.Enabled {
		return false, nil
	}
	tableName, err := s.ensureAcademicTableConfigured(school)
	if err != nil || tableName != match.tableName {
		return false, nil
	}
	return true, nil
}

func profileStudentIDSet(profile *Profile) map[string]struct{} {
	if profile == nil {
		return nil
	}
	studentIDs := make(map[string]struct{}, len(profile.StudentIDs)+1)
	for _, rawID := range profile.StudentIDs {
		studentID := schoolauth.NormalizeStudentID(rawID)
		if schoolauth.IsValidStudentID(studentID) {
			studentIDs[studentID] = struct{}{}
		}
	}
	if profile.ActiveStudentID != nil {
		studentID := schoolauth.NormalizeStudentID(*profile.ActiveStudentID)
		if schoolauth.IsValidStudentID(studentID) {
			studentIDs[studentID] = struct{}{}
		}
	}
	return studentIDs
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
	if req.DocType != DocTypeMainlandID && (front == nil || selfie == nil) {
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

func (p submittedIdentityPhotos) hasManualEvidence() bool {
	return p.front != nil && p.selfie != nil
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

func (s *Service) resolveIdentityReviewPhoto(
	ctx context.Context,
	userID int64,
	slot string,
	value *string,
) (*string, error) {
	key, err := normalizeSubmittedIdentityPhotoRef(userID, slot, value)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, nil
	}
	if s.photoStore == nil {
		return nil, ErrIdentityPhotoStoreDisabled
	}
	url, err := s.photoStore.PresignGetURL(ctx, *key)
	if err != nil {
		return nil, fmt.Errorf("presign identity photo: %w", err)
	}
	if strings.TrimSpace(url) == "" {
		return nil, ErrIdentityPhotoStorageUnavailable
	}
	return &url, nil
}
