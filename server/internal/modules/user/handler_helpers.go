package user

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// resolveCurrentUser 从请求上下文解析当前用户内部 ID
func (h *Handler) resolveCurrentUser(c *gin.Context) (int64, bool) {
	externalID := middleware.GetUserID(c)
	if externalID == "" {
		response.Unauthorized(c, "authentication required", errs.ErrLoginRequired)
		return 0, false
	}

	userID, err := h.service.GetInternalUserID(c.Request.Context(), externalID)
	if err != nil {
		logger.FromGin(c).Error("failed to resolve user ID",
			zap.String("external_id", externalID),
			zap.Error(err),
		)
		response.InternalError(c, "failed to resolve user")
		return 0, false
	}

	return userID, true
}

func identityStatusToJSON(i *IdentityStatus) gin.H {
	return gin.H{
		"userID":          i.UserID,
		"docType":         i.DocType,
		"realName":        i.RealName,
		"verified":        i.Verified,
		"verifyMethod":    i.VerifyMethod,
		"reviewedAt":      i.ReviewedAt,
		"verifiedAt":      i.VerifiedAt,
		"rejectionReason": i.RejectionReason,
		"createdAt":       i.CreatedAt,
		"updatedAt":       i.UpdatedAt,
	}
}

func identityReviewItemToJSON(i *IdentityReviewItem) gin.H {
	return gin.H{
		"userID":          i.UserID,
		"docType":         i.DocType,
		"realName":        i.RealName,
		"verified":        i.Verified,
		"verifyMethod":    i.VerifyMethod,
		"reviewedAt":      i.ReviewedAt,
		"verifiedAt":      i.VerifiedAt,
		"docPhotoFront":   i.DocPhotoFront,
		"docPhotoBack":    i.DocPhotoBack,
		"docPhotoSelfie":  i.DocPhotoSelfie,
		"rejectionReason": i.RejectionReason,
		"createdAt":       i.CreatedAt,
		"updatedAt":       i.UpdatedAt,
	}
}

func profileToJSON(p *Profile) gin.H {
	phone := normalizeMaskedPhone(p.Phone)
	return gin.H{
		"userID":             p.UserID,
		"schoolID":           p.SchoolID,
		"studentIDs":         p.StudentIDs,
		"activeStudentID":    p.ActiveStudentID,
		"verificationStatus": p.VerificationStatus,
		"verificationMethod": p.VerificationMethod,
		"rejectionReason":    p.RejectionReason,
		"reviewedAt":         p.ReviewedAt,
		"phone":              phone,
		"phoneVerified":      p.PhoneVerified,
		"consentGivenAt":     p.ConsentGivenAt,
		"verifiedAt":         p.VerifiedAt,
		"createdAt":          p.CreatedAt,
		"updatedAt":          p.UpdatedAt,
	}
}

func schoolConfigPublicToJSON(s *SchoolConfig) (gin.H, error) {
	manualFormFields, err := decodeManualFieldDescriptors(s.ManualFormFields)
	if err != nil {
		return nil, fmt.Errorf("decode manualFormFields: %w", err)
	}

	result := gin.H{
		"schoolID":           s.SchoolID,
		"schoolName":         s.SchoolName,
		"verificationMethod": s.VerificationMethod,
		"consentText":        s.ConsentText,
		"enabled":            s.Enabled,
	}
	if s.VerificationMethod == VerifyMethodManual {
		result["manualFormFields"] = manualFormFields
	}
	return result, nil
}

func adminSchoolConfigToJSON(s *SchoolConfig) (gin.H, error) {
	ldapConfig, err := buildAdminSchoolLDAPConfig(s.LDAPConfig)
	if err != nil {
		return nil, fmt.Errorf("decode ldapConfig: %w", err)
	}
	manualFormFields, err := decodeManualFieldDescriptors(s.ManualFormFields)
	if err != nil {
		return nil, fmt.Errorf("decode manualFormFields: %w", err)
	}

	result := gin.H{
		"schoolID":           s.SchoolID,
		"schoolName":         s.SchoolName,
		"verificationMethod": s.VerificationMethod,
		"approvalPolicy":     s.ApprovalPolicy,
		"academicDbTable":    s.AcademicDBTable,
		"consentText":        s.ConsentText,
		"manualFormFields":   manualFormFields,
		"enabled":            s.Enabled,
		"createdAt":          s.CreatedAt,
	}
	if ldapConfig != nil {
		result["ldapConfig"] = ldapConfig
	}

	return result, nil
}

func buildAdminSchoolLDAPConfig(raw json.RawMessage) (*SchoolLDAPConfigView, error) {
	settings, err := decodeSchoolLDAPSettings(raw)
	if err != nil {
		return nil, err
	}
	if isEmptySchoolLDAPSettings(settings) {
		return nil, nil
	}

	return &SchoolLDAPConfigView{
		URL:                   optionalTrimmedString(settings.URL),
		BaseDN:                optionalTrimmedString(settings.BaseDN),
		SystemBindDN:          optionalTrimmedString(settings.SystemBindDN),
		UseTLS:                settings.UseTLS,
		InsecureSkipVerify:    settings.InsecureSkipVerify,
		HasSystemBindPassword: strings.TrimSpace(settings.SystemBindPassword) != "",
	}, nil
}

func adminStudentVerificationToJSON(p *Profile) (gin.H, error) {
	manualFormData, err := decodeJSONObject(p.ManualFormData)
	if err != nil {
		return nil, fmt.Errorf("decode manualFormData: %w", err)
	}
	phone := normalizeMaskedPhone(p.Phone)

	return gin.H{
		"userID":             p.UserID,
		"schoolID":           p.SchoolID,
		"studentIDs":         p.StudentIDs,
		"activeStudentID":    p.ActiveStudentID,
		"manualFormData":     manualFormData,
		"verificationStatus": p.VerificationStatus,
		"verificationMethod": p.VerificationMethod,
		"rejectionReason":    p.RejectionReason,
		"reviewedAt":         p.ReviewedAt,
		"phone":              phone,
		"phoneVerified":      p.PhoneVerified,
		"consentGivenAt":     p.ConsentGivenAt,
		"verifiedAt":         p.VerifiedAt,
		"createdAt":          p.CreatedAt,
		"updatedAt":          p.UpdatedAt,
	}, nil
}

func systemConfigToJSON(c *SystemConfig) gin.H {
	return gin.H{
		"key":         c.Key,
		"value":       c.Value,
		"description": c.Description,
		"updatedAt":   c.UpdatedAt,
	}
}

func decodeJSONObject(raw json.RawMessage) (map[string]any, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func normalizeAdminReviewStatus(raw string) (string, bool) {
	switch raw {
	case "", StatusPending:
		return StatusPending, true
	case StatusVerified, StatusRejected:
		return raw, true
	case "all":
		return "", true
	default:
		return "", false
	}
}
