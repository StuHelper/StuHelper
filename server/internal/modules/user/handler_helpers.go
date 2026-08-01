package user

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/schoolauth"
)

// resolveCurrentUser 从请求上下文解析当前用户内部 ID
func (h *Handler) resolveCurrentUser(c *gin.Context) (int64, bool) {
	return middleware.ResolveRequiredInternalUserID(c, h.service.GetInternalUserID, "failed to resolve user")
}

type messageResponse struct {
	Message string `json:"message"`
}

type pagedListResponse[T any] struct {
	List  []T `json:"list"`
	Total int `json:"total"`
}

type identityStatusResponse struct {
	UserID          int64      `json:"userID"`
	DocType         string     `json:"docType"`
	RealName        string     `json:"realName"`
	Verified        bool       `json:"verified"`
	VerifyMethod    *string    `json:"verifyMethod"`
	ReviewedAt      *time.Time `json:"reviewedAt"`
	VerifiedAt      *time.Time `json:"verifiedAt"`
	RejectionReason *string    `json:"rejectionReason"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type identityReviewItemResponse struct {
	UserID          int64      `json:"userID"`
	DocType         string     `json:"docType"`
	RealName        string     `json:"realName"`
	Verified        bool       `json:"verified"`
	VerifyMethod    *string    `json:"verifyMethod"`
	ReviewedAt      *time.Time `json:"reviewedAt"`
	VerifiedAt      *time.Time `json:"verifiedAt"`
	RejectionReason *string    `json:"rejectionReason"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type identityReviewDetailResponse struct {
	identityReviewItemResponse
	DocPhotoFrontURL  *string `json:"docPhotoFrontURL"`
	DocPhotoBackURL   *string `json:"docPhotoBackURL"`
	DocPhotoSelfieURL *string `json:"docPhotoSelfieURL"`
}

type profileResponse struct {
	UserID             int64      `json:"userID"`
	SchoolID           *int64     `json:"schoolID"`
	StudentIDs         []string   `json:"studentIDs"`
	ActiveStudentID    *string    `json:"activeStudentID"`
	VerificationStatus string     `json:"verificationStatus"`
	VerificationMethod *string    `json:"verificationMethod"`
	RejectionReason    *string    `json:"rejectionReason"`
	ReviewedAt         *time.Time `json:"reviewedAt"`
	Phone              *string    `json:"phone"`
	PhoneVerified      bool       `json:"phoneVerified"`
	ConsentGivenAt     *time.Time `json:"consentGivenAt"`
	VerifiedAt         *time.Time `json:"verifiedAt"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

type schoolConfigPublicResponse struct {
	SchoolID              int64                    `json:"schoolID"`
	SchoolCode            string                   `json:"schoolCode"`
	SchoolName            string                   `json:"schoolName"`
	VerificationMethod    string                   `json:"verificationMethod"`
	ConsentText           *string                  `json:"consentText"`
	Enabled               bool                     `json:"enabled"`
	ManualFormFields      *[]ManualFieldDescriptor `json:"manualFormFields,omitempty"`
	SchoolSSOEnabled      bool                     `json:"schoolSsoEnabled"`
	SchoolEmailOTPEnabled bool                     `json:"schoolEmailOtpEnabled"`
	SchoolEmailPolicy     *SchoolEmailPolicyView   `json:"schoolEmailIdentityPolicy,omitempty"`
}

type adminSchoolConfigResponse struct {
	SchoolID              int64                   `json:"schoolID"`
	SchoolCode            string                  `json:"schoolCode"`
	SchoolName            string                  `json:"schoolName"`
	VerificationMethod    string                  `json:"verificationMethod"`
	ApprovalPolicy        string                  `json:"approvalPolicy"`
	AcademicDBTable       *string                 `json:"academicDbTable"`
	ConsentText           *string                 `json:"consentText"`
	ManualFormFields      []ManualFieldDescriptor `json:"manualFormFields"`
	Enabled               bool                    `json:"enabled"`
	SchoolSSOEnabled      bool                    `json:"schoolSsoEnabled"`
	SchoolEmailOTPEnabled bool                    `json:"schoolEmailOtpEnabled"`
	SchoolEmailPolicy     *SchoolEmailPolicyView  `json:"schoolEmailIdentityPolicy,omitempty"`
	CreatedAt             time.Time               `json:"createdAt"`
	LDAPConfig            *SchoolLDAPConfigView   `json:"ldapConfig,omitempty"`
}

type adminStudentVerificationResponse struct {
	UserID             int64          `json:"userID"`
	SchoolID           *int64         `json:"schoolID"`
	StudentIDs         []string       `json:"studentIDs"`
	ActiveStudentID    *string        `json:"activeStudentID"`
	ManualFormData     map[string]any `json:"manualFormData"`
	VerificationStatus string         `json:"verificationStatus"`
	VerificationMethod *string        `json:"verificationMethod"`
	RejectionReason    *string        `json:"rejectionReason"`
	ReviewedAt         *time.Time     `json:"reviewedAt"`
	Phone              *string        `json:"phone"`
	PhoneVerified      bool           `json:"phoneVerified"`
	ConsentGivenAt     *time.Time     `json:"consentGivenAt"`
	VerifiedAt         *time.Time     `json:"verifiedAt"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
}

type systemConfigResponse struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description *string   `json:"description"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type userSurfaceResponse struct {
	DisplayName        string   `json:"displayName"`
	AvatarURL          string   `json:"avatarURL,omitempty"`
	Phone              *string  `json:"phone,omitempty"`
	IdentityStatus     string   `json:"identityStatus"`
	VerificationStatus string   `json:"verificationStatus"`
	PhoneBound         bool     `json:"phoneBound"`
	Capabilities       []string `json:"capabilities"`
}

type uploadIdentityPhotoResponse struct {
	Key string `json:"key"`
}

type bindPhoneOTPResponse struct {
	Message  string `json:"message"`
	Cooldown int    `json:"cooldown"`
}

type academicInfoResponse struct {
	XH     string  `json:"xh"`
	XM     *string `json:"xm"`
	YXDM   *string `json:"yxdm"`
	ZYDM   *string `json:"zydm"`
	BJDM   *string `json:"bjdm"`
	XZNJ   *string `json:"xznj"`
	RXNJ   *string `json:"rxnj"`
	PYCCDM *string `json:"pyccdm"`
	SJH    *string `json:"sjh"`
	DZXX   *string `json:"dzxx"`
}

type SchoolEmailPolicyView struct {
	Type                 string `json:"type"`
	StudentIDEmailDomain string `json:"studentIDEmailDomain,omitempty"`
	RequireStudentName   bool   `json:"requireStudentName"`
}

func identityStatusToJSON(i *IdentityStatus) identityStatusResponse {
	return identityStatusResponse{
		UserID:          i.UserID,
		DocType:         i.DocType,
		RealName:        i.RealName,
		Verified:        i.Verified,
		VerifyMethod:    i.VerifyMethod,
		ReviewedAt:      i.ReviewedAt,
		VerifiedAt:      i.VerifiedAt,
		RejectionReason: i.RejectionReason,
		CreatedAt:       i.CreatedAt,
		UpdatedAt:       i.UpdatedAt,
	}
}

func identityReviewItemToJSON(i *IdentityReviewItem) identityReviewItemResponse {
	return identityReviewItemResponse{
		UserID:          i.UserID,
		DocType:         i.DocType,
		RealName:        i.RealName,
		Verified:        i.Verified,
		VerifyMethod:    i.VerifyMethod,
		ReviewedAt:      i.ReviewedAt,
		VerifiedAt:      i.VerifiedAt,
		RejectionReason: i.RejectionReason,
		CreatedAt:       i.CreatedAt,
		UpdatedAt:       i.UpdatedAt,
	}
}

func identityReviewDetailToJSON(i *IdentityReviewItem) identityReviewDetailResponse {
	return identityReviewDetailResponse{
		identityReviewItemResponse: identityReviewItemToJSON(i),
		DocPhotoFrontURL:           i.DocPhotoFront,
		DocPhotoBackURL:            i.DocPhotoBack,
		DocPhotoSelfieURL:          i.DocPhotoSelfie,
	}
}

func profileToJSON(p *Profile) profileResponse {
	phone := normalizeMaskedPhone(p.Phone)
	return profileResponse{
		UserID:             p.UserID,
		SchoolID:           p.SchoolID,
		StudentIDs:         nonNilStrings(p.StudentIDs),
		ActiveStudentID:    p.ActiveStudentID,
		VerificationStatus: p.VerificationStatus,
		VerificationMethod: p.VerificationMethod,
		RejectionReason:    p.RejectionReason,
		ReviewedAt:         p.ReviewedAt,
		Phone:              phone,
		PhoneVerified:      p.PhoneVerified,
		ConsentGivenAt:     p.ConsentGivenAt,
		VerifiedAt:         p.VerifiedAt,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}

func schoolConfigPublicToJSON(s *SchoolConfig) (schoolConfigPublicResponse, error) {
	manualFormFields, err := decodeManualFieldDescriptors(s.ManualFormFields)
	if err != nil {
		return schoolConfigPublicResponse{}, fmt.Errorf("decode manualFormFields: %w", err)
	}
	admissionCapabilities := schoolAdmissionCapabilities(s.ManualFormFields)

	result := schoolConfigPublicResponse{
		SchoolID:              s.SchoolID,
		SchoolCode:            s.SchoolCode,
		SchoolName:            s.SchoolName,
		VerificationMethod:    s.VerificationMethod,
		ConsentText:           s.ConsentText,
		Enabled:               s.Enabled,
		SchoolSSOEnabled:      admissionCapabilities.ssoEnabled,
		SchoolEmailOTPEnabled: admissionCapabilities.emailOTPEnabled,
		SchoolEmailPolicy:     schoolEmailPolicyToView(admissionCapabilities.emailIdentityPolicy),
	}
	if s.VerificationMethod == VerifyMethodManual {
		fields := nonNilManualFieldDescriptors(manualFormFields)
		result.ManualFormFields = &fields
	}
	return result, nil
}

func adminSchoolConfigToJSON(s *SchoolConfig) (adminSchoolConfigResponse, error) {
	ldapConfig, err := buildAdminSchoolLDAPConfig(s.LDAPConfig)
	if err != nil {
		return adminSchoolConfigResponse{}, fmt.Errorf("decode ldapConfig: %w", err)
	}
	manualFormFields, err := decodeManualFieldDescriptors(s.ManualFormFields)
	if err != nil {
		return adminSchoolConfigResponse{}, fmt.Errorf("decode manualFormFields: %w", err)
	}
	admissionCapabilities := schoolAdmissionCapabilities(s.ManualFormFields)

	return adminSchoolConfigResponse{
		SchoolID:              s.SchoolID,
		SchoolCode:            s.SchoolCode,
		SchoolName:            s.SchoolName,
		VerificationMethod:    s.VerificationMethod,
		ApprovalPolicy:        s.ApprovalPolicy,
		AcademicDBTable:       s.AcademicDBTable,
		ConsentText:           s.ConsentText,
		ManualFormFields:      nonNilManualFieldDescriptors(manualFormFields),
		Enabled:               s.Enabled,
		SchoolSSOEnabled:      admissionCapabilities.ssoEnabled,
		SchoolEmailOTPEnabled: admissionCapabilities.emailOTPEnabled,
		SchoolEmailPolicy:     schoolEmailPolicyToView(admissionCapabilities.emailIdentityPolicy),
		CreatedAt:             s.CreatedAt,
		LDAPConfig:            ldapConfig,
	}, nil
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
		HasSystemBindPassword: strings.TrimSpace(settings.SystemBindPassword) != "",
	}, nil
}

func adminStudentVerificationToJSON(p *Profile) (adminStudentVerificationResponse, error) {
	manualFormData, err := decodeJSONObject(p.ManualFormData)
	if err != nil {
		return adminStudentVerificationResponse{}, fmt.Errorf("decode manualFormData: %w", err)
	}
	phone := normalizeMaskedPhone(p.Phone)

	return adminStudentVerificationResponse{
		UserID:             p.UserID,
		SchoolID:           p.SchoolID,
		StudentIDs:         nonNilStrings(p.StudentIDs),
		ActiveStudentID:    p.ActiveStudentID,
		ManualFormData:     manualFormData,
		VerificationStatus: p.VerificationStatus,
		VerificationMethod: p.VerificationMethod,
		RejectionReason:    p.RejectionReason,
		ReviewedAt:         p.ReviewedAt,
		Phone:              phone,
		PhoneVerified:      p.PhoneVerified,
		ConsentGivenAt:     p.ConsentGivenAt,
		VerifiedAt:         p.VerifiedAt,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}, nil
}

func systemConfigToJSON(c *SystemConfig) systemConfigResponse {
	return systemConfigResponse{
		Key:         c.Key,
		Value:       c.Value,
		Description: c.Description,
		UpdatedAt:   c.UpdatedAt,
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

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilManualFieldDescriptors(fields []ManualFieldDescriptor) []ManualFieldDescriptor {
	if fields == nil {
		return []ManualFieldDescriptor{}
	}
	return fields
}

type schoolAdmissionCapabilityFlags struct {
	ssoEnabled          bool
	emailOTPEnabled     bool
	emailIdentityPolicy *schoolauth.EmailIdentityPolicy
}

func schoolAdmissionCapabilities(raw json.RawMessage) schoolAdmissionCapabilityFlags {
	settings := schoolauth.ParseAdmissionSettings(raw)
	return schoolAdmissionCapabilityFlags{
		ssoEnabled:          strings.TrimSpace(settings.SSOLoginURL) != "",
		emailOTPEnabled:     hasNonBlankString(settings.EmailDomains),
		emailIdentityPolicy: settings.EmailIdentityPolicy,
	}
}

func schoolEmailPolicyToView(policy *schoolauth.EmailIdentityPolicy) *SchoolEmailPolicyView {
	if policy == nil {
		return nil
	}
	return &SchoolEmailPolicyView{
		Type:                 policy.Type,
		StudentIDEmailDomain: policy.StudentIDEmailDomain,
		RequireStudentName:   policy.RequireStudentName,
	}
}

func hasNonBlankString(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
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
