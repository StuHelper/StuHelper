package studentverification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	appcrypto "github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
)

const (
	manualFormDigestDomain           = "student-verification-manual-form:v1:"
	manualEmailHashDomain            = "student-verification-manual-email:v1:"
	manualDefaultMaxBytes      int64 = 10 * 1024 * 1024
	manualDefaultMaxMaterials        = 3
	manualDefaultRetentionDays       = 180
	manualDefaultHandoffTTL          = 30 * time.Minute
	manualDefaultReviewWindow        = 7 * 24 * time.Hour
	manualDefaultMinDimension        = 320
	manualDefaultMaxDimension        = 12000
	manualDefaultMaxPixels     int64 = 40_000_000
)

var manualFieldKeyPattern = regexp.MustCompile(`^[a-z][A-Za-z0-9]{0,63}$`)

type manualReviewConfiguration struct {
	Fields       []VerificationField
	FieldsByKey  map[string]VerificationField
	RequiredKeys []string
}

type manualReviewPolicy struct {
	MaxMaterialBytes                 int64    `json:"maxMaterialBytes"`
	MaxMaterials                     int      `json:"maxMaterials"`
	MaterialRetentionDays            int      `json:"materialRetentionDays"`
	HandoffTTLSeconds                int      `json:"handoffTTLSeconds"`
	ReviewWindowSeconds              int      `json:"reviewWindowSeconds"`
	RequireEmailVerification         *bool    `json:"requireEmailVerification"`
	AllowedMaterialTypes             []string `json:"allowedMaterialTypes"`
	MinimumImageDimension            int      `json:"minimumImageDimension"`
	MaximumImageDimension            int      `json:"maximumImageDimension"`
	MaximumImagePixels               int64    `json:"maximumImagePixels"`
	AdmissionNoticeMaxCredentialDays int      `json:"admissionNoticeMaxCredentialDays"`
}

func decodeManualReviewConfiguration(
	config *MethodConfig,
) (manualReviewConfiguration, manualReviewPolicy, error) {
	if config == nil || config.Method != MethodManualMaterialReview ||
		config.AdapterID != "shared_manual_review" || config.RosterDependency != "independent" ||
		config.CredentialTTL == nil || *config.CredentialTTL <= 0 ||
		config.PrivacyNoticeVersion == "" {
		return manualReviewConfiguration{}, manualReviewPolicy{}, ErrMethodUnavailable
	}
	var form struct {
		Fields []VerificationField `json:"fields"`
	}
	if err := decodeStrictJSON(config.PublicFormSchema, &form); err != nil {
		return manualReviewConfiguration{}, manualReviewPolicy{}, ErrMethodUnavailable
	}
	if len(form.Fields) < 4 || len(form.Fields) > 30 {
		return manualReviewConfiguration{}, manualReviewPolicy{}, ErrMethodUnavailable
	}
	decoded := manualReviewConfiguration{
		Fields: form.Fields, FieldsByKey: make(map[string]VerificationField, len(form.Fields)),
		RequiredKeys: []string{"department", "studentID", "name", "email"},
	}
	for _, field := range form.Fields {
		if err := validateManualFieldDescriptor(field); err != nil {
			return manualReviewConfiguration{}, manualReviewPolicy{}, ErrMethodUnavailable
		}
		if _, exists := decoded.FieldsByKey[field.Key]; exists {
			return manualReviewConfiguration{}, manualReviewPolicy{}, ErrMethodUnavailable
		}
		decoded.FieldsByKey[field.Key] = field
	}
	for _, key := range decoded.RequiredKeys {
		field, exists := decoded.FieldsByKey[key]
		if !exists || !field.Required {
			return manualReviewConfiguration{}, manualReviewPolicy{}, ErrMethodUnavailable
		}
	}
	if decoded.FieldsByKey["email"].InputType != "email" {
		return manualReviewConfiguration{}, manualReviewPolicy{}, ErrMethodUnavailable
	}

	policy := manualReviewPolicy{}
	if len(config.RiskPolicy) > 0 && string(config.RiskPolicy) != "{}" {
		if err := decodeStrictJSON(config.RiskPolicy, &policy); err != nil {
			return manualReviewConfiguration{}, manualReviewPolicy{}, ErrMethodUnavailable
		}
	}
	policy.applyDefaults()
	if !policy.valid() {
		return manualReviewConfiguration{}, manualReviewPolicy{}, ErrMethodUnavailable
	}
	return decoded, policy, nil
}

func decodeStrictJSON(raw json.RawMessage, destination any) error {
	if len(raw) == 0 {
		return errors.New("empty JSON configuration")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func validateManualFieldDescriptor(field VerificationField) error {
	if !manualFieldKeyPattern.MatchString(field.Key) || strings.TrimSpace(field.Label) == "" ||
		utf8.RuneCountInString(field.Label) > 100 || utf8.RuneCountInString(field.HelpText) > 300 {
		return ErrMethodUnavailable
	}
	switch field.InputType {
	case "text", "email", "select", "date", "textarea":
	default:
		return ErrMethodUnavailable
	}
	if field.MaxLength != nil && (*field.MaxLength <= 0 || *field.MaxLength > 500) {
		return ErrMethodUnavailable
	}
	if field.InputType == "select" {
		if len(field.Options) == 0 || len(field.Options) > 100 {
			return ErrMethodUnavailable
		}
		seen := make(map[string]struct{}, len(field.Options))
		for _, option := range field.Options {
			if strings.TrimSpace(option.Value) == "" || strings.TrimSpace(option.Label) == "" ||
				utf8.RuneCountInString(option.Value) > 100 || utf8.RuneCountInString(option.Label) > 100 {
				return ErrMethodUnavailable
			}
			if _, exists := seen[option.Value]; exists {
				return ErrMethodUnavailable
			}
			seen[option.Value] = struct{}{}
		}
	} else if len(field.Options) != 0 {
		return ErrMethodUnavailable
	}
	return nil
}

func (policy *manualReviewPolicy) applyDefaults() {
	if policy.MaxMaterialBytes == 0 {
		policy.MaxMaterialBytes = manualDefaultMaxBytes
	}
	if policy.MaxMaterials == 0 {
		policy.MaxMaterials = manualDefaultMaxMaterials
	}
	if policy.MaterialRetentionDays == 0 {
		policy.MaterialRetentionDays = manualDefaultRetentionDays
	}
	if policy.HandoffTTLSeconds == 0 {
		policy.HandoffTTLSeconds = int(manualDefaultHandoffTTL / time.Second)
	}
	if policy.ReviewWindowSeconds == 0 {
		policy.ReviewWindowSeconds = int(manualDefaultReviewWindow / time.Second)
	}
	if policy.RequireEmailVerification == nil {
		required := true
		policy.RequireEmailVerification = &required
	}
	if len(policy.AllowedMaterialTypes) == 0 {
		policy.AllowedMaterialTypes = []string{
			string(ManualMaterialCampusCard), string(ManualMaterialStudentCard),
			string(ManualMaterialAdmissionNotice), string(ManualMaterialOtherApproved),
		}
	}
	if policy.MinimumImageDimension == 0 {
		policy.MinimumImageDimension = manualDefaultMinDimension
	}
	if policy.MaximumImageDimension == 0 {
		policy.MaximumImageDimension = manualDefaultMaxDimension
	}
	if policy.MaximumImagePixels == 0 {
		policy.MaximumImagePixels = manualDefaultMaxPixels
	}
	if policy.AdmissionNoticeMaxCredentialDays == 0 {
		policy.AdmissionNoticeMaxCredentialDays = 180
	}
}

func (policy manualReviewPolicy) valid() bool {
	if policy.MaxMaterialBytes < 64*1024 || policy.MaxMaterialBytes > 20*1024*1024 ||
		policy.MaxMaterials < 1 || policy.MaxMaterials > 5 ||
		policy.MaterialRetentionDays < 1 || policy.MaterialRetentionDays > 365 ||
		policy.HandoffTTLSeconds < 120 || policy.HandoffTTLSeconds > 3600 ||
		policy.ReviewWindowSeconds < 3600 || policy.ReviewWindowSeconds > 30*24*3600 ||
		policy.RequireEmailVerification == nil ||
		policy.MinimumImageDimension < 320 || policy.MinimumImageDimension > 2000 ||
		policy.MaximumImageDimension < policy.MinimumImageDimension || policy.MaximumImageDimension > 12000 ||
		policy.MaximumImagePixels < 1_000_000 || policy.MaximumImagePixels > 40_000_000 ||
		policy.AdmissionNoticeMaxCredentialDays < 1 || policy.AdmissionNoticeMaxCredentialDays > 366 ||
		len(policy.AllowedMaterialTypes) == 0 || len(policy.AllowedMaterialTypes) > 4 {
		return false
	}
	seen := make(map[string]struct{}, len(policy.AllowedMaterialTypes))
	for _, value := range policy.AllowedMaterialTypes {
		switch ManualMaterialType(value) {
		case ManualMaterialCampusCard, ManualMaterialStudentCard,
			ManualMaterialAdmissionNotice, ManualMaterialOtherApproved:
		default:
			return false
		}
		if _, exists := seen[value]; exists {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func (s *Service) UpsertManualReview(
	ctx context.Context,
	input UpsertManualReviewInput,
) (*ManualReviewCase, error) {
	if !input.SensitiveDataConsent {
		return nil, ErrConsentRequired
	}
	if s.manualMaterialStore == nil || s.rosterCipher == nil ||
		s.rosterEncryptionKeyVersion <= 0 || s.manualReviewPublicBaseURL == "" ||
		s.manualMaterialAccessTTL < time.Minute || s.manualMaterialAccessTTL > 15*time.Minute {
		return nil, ErrMethodUnavailable
	}
	application, err := s.GetApplication(ctx, input.UserID, input.ApplicationID)
	if err != nil {
		return nil, err
	}
	config, err := s.repo.GetMethodConfig(ctx, application.School.Code, MethodManualMaterialReview)
	if errors.Is(err, pgx.ErrNoRows) || !methodHealthy(config) {
		return nil, ErrMethodUnavailable
	}
	if err != nil {
		return nil, err
	}
	formConfig, policy, err := decodeManualReviewConfiguration(config)
	if err != nil || config.PrivacyNoticeVersion != input.PrivacyNoticeVersion ||
		!slices.Contains(policy.AllowedMaterialTypes, string(input.MaterialType)) {
		return nil, ErrMethodUnavailable
	}
	formValues, err := s.normalizeManualReviewForm(config, formConfig, input.FormValues)
	if err != nil {
		return nil, err
	}
	reviewCase, err := s.buildManualReviewCase(input, config, formValues)
	if err != nil {
		return nil, err
	}
	now := s.now()
	err = s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		lockedApplication, err := s.repo.GetApplicationForUpdateTx(ctx, tx, input.ApplicationID, input.UserID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrApplicationNotFound
		}
		if err != nil {
			return err
		}
		if !applicationIsNonTerminal(lockedApplication) || !lockedApplication.ExpiresAt.After(now) {
			return ErrApplicationState
		}
		lockedConfig, err := s.repo.GetMethodConfigTx(
			ctx, tx, lockedApplication.SchoolCode, MethodManualMaterialReview,
		)
		if err != nil || !methodHealthy(lockedConfig) ||
			lockedConfig.PrivacyNoticeVersion != input.PrivacyNoticeVersion {
			return ErrMethodUnavailable
		}
		lockedFormConfig, lockedPolicy, err := decodeManualReviewConfiguration(lockedConfig)
		if err != nil || !slices.Contains(lockedPolicy.AllowedMaterialTypes, string(input.MaterialType)) {
			return ErrMethodUnavailable
		}
		lockedValues, err := s.normalizeManualReviewForm(lockedConfig, lockedFormConfig, input.FormValues)
		if err != nil {
			return err
		}
		lockedCase, err := s.buildManualReviewCase(input, lockedConfig, lockedValues)
		if err != nil {
			return err
		}
		lockedCase.ID = reviewCase.ID
		stored, created, err := s.repo.SaveManualReviewCaseTx(ctx, tx, *lockedCase, now)
		if err != nil {
			return err
		}
		if err := s.repo.ProgressApplicationManualDraftTx(
			ctx, tx, input.ApplicationID, input.PrivacyNoticeVersion, now,
		); err != nil {
			return err
		}
		actor := input.UserID
		action := "draft_saved"
		if !created {
			action = "draft_saved"
		}
		return s.repo.InsertManualReviewEventTx(
			ctx, tx, stored.ID, stored.Revision, "applicant", &actor, action, nil, now,
		)
	})
	if err != nil {
		return nil, err
	}
	return s.GetManualReview(ctx, input.UserID, input.ApplicationID)
}

func (s *Service) buildManualReviewCase(
	input UpsertManualReviewInput,
	config *MethodConfig,
	formValues map[string]string,
) (*ManualReviewCase, error) {
	encoded, err := json.Marshal(formValues)
	if err != nil || len(encoded) == 0 || len(encoded) > 64*1024 {
		return nil, ErrManualReviewInvalidForm
	}
	ciphertext, err := s.rosterCipher.Encrypt(string(encoded))
	if err != nil {
		return nil, fmt.Errorf("encrypt manual review form: %w", err)
	}
	digest, err := appcrypto.HMACHashWithKey(
		manualFormDigestDomain+config.SchoolCode+":"+string(encoded), s.hmacKey,
	)
	if err != nil {
		return nil, err
	}
	studentID := formValues["studentID"]
	studentIDHash, err := ComputeRosterBlindIndex(
		s.hmacKey, config.SchoolID, BlindIndexStudentID, studentID,
	)
	if err != nil {
		return nil, err
	}
	email := formValues["email"]
	emailHash, err := appcrypto.HMACHashWithKey(
		manualEmailHashDomain+config.SchoolCode+":"+email, s.hmacKey,
	)
	if err != nil {
		return nil, err
	}
	caseID, err := newID()
	if err != nil {
		return nil, err
	}
	emailMasked := maskEmail(email)
	return &ManualReviewCase{
		ID: caseID, ApplicationID: input.ApplicationID, UserID: input.UserID,
		SchoolID: config.SchoolID, Status: ManualReviewDraft,
		MaterialType: input.MaterialType, FormDataEnc: ciphertext,
		FormDigest: digest, EncryptionKeyVersion: s.rosterEncryptionKeyVersion,
		StudentIDHash: studentIDHash, StudentIDMasked: maskStudentID(studentID),
		ApplicantNameMasked: maskManualReviewName(formValues["name"]),
		EmailHash:           &emailHash, EmailMasked: &emailMasked,
		PrivacyNoticeVersion: input.PrivacyNoticeVersion,
	}, nil
}

func (s *Service) normalizeManualReviewForm(
	config *MethodConfig,
	formConfig manualReviewConfiguration,
	input map[string]string,
) (map[string]string, error) {
	if len(input) == 0 || len(input) > len(formConfig.FieldsByKey) {
		return nil, ErrManualReviewInvalidForm
	}
	result := make(map[string]string, len(input))
	for key, raw := range input {
		field, exists := formConfig.FieldsByKey[key]
		if !exists {
			return nil, ErrManualReviewInvalidForm
		}
		value := strings.TrimSpace(raw)
		if value == "" && field.Required {
			return nil, ErrManualReviewInvalidForm
		}
		if containsUnsafeManualText(value) {
			return nil, ErrManualReviewInvalidForm
		}
		maxLength := 500
		if field.MaxLength != nil {
			maxLength = *field.MaxLength
		}
		if utf8.RuneCountInString(value) > maxLength {
			return nil, ErrManualReviewInvalidForm
		}
		switch field.InputType {
		case "email":
			normalized, ok := normalizeManualReviewEmail(value, config.EmailDomains)
			if !ok {
				return nil, ErrManualReviewInvalidForm
			}
			value = normalized
		case "select":
			if !slices.ContainsFunc(field.Options, func(option VerificationFieldOption) bool {
				return option.Value == value
			}) {
				return nil, ErrManualReviewInvalidForm
			}
		case "date":
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return nil, ErrManualReviewInvalidForm
			}
		}
		result[key] = value
	}
	for _, key := range formConfig.RequiredKeys {
		if strings.TrimSpace(result[key]) == "" {
			return nil, ErrManualReviewInvalidForm
		}
	}
	studentID, ok := s.normalizeManualStudentID(config, result["studentID"])
	if !ok {
		return nil, ErrManualReviewInvalidForm
	}
	result["studentID"] = studentID
	name, ok := s.normalizeManualName(config, result["name"])
	if !ok {
		return nil, ErrManualReviewInvalidForm
	}
	result["name"] = name
	return result, nil
}

func (s *Service) normalizeManualStudentID(config *MethodConfig, value string) (string, bool) {
	if config == nil {
		return "", false
	}
	var policy struct {
		Strategy  string `json:"strategy"`
		Pattern   string `json:"pattern"`
		Transform string `json:"transform"`
	}
	if err := decodeStrictJSON(config.StudentIDPolicy, &policy); err != nil {
		return "", false
	}
	switch policy.Strategy {
	case "adapter":
		if config.SchoolAdapterID != BUAAAdapterID {
			return "", false
		}
		return s.buaa.NormalizeStudentID(value)
	case "regex":
		if len(policy.Pattern) < 3 || len(policy.Pattern) > 256 ||
			!strings.HasPrefix(policy.Pattern, "^") || !strings.HasSuffix(policy.Pattern, "$") {
			return "", false
		}
		normalized := strings.TrimSpace(value)
		switch policy.Transform {
		case "", "none":
		case "lowercase":
			normalized = strings.ToLower(normalized)
		case "uppercase":
			normalized = strings.ToUpper(normalized)
		default:
			return "", false
		}
		pattern, err := regexp.Compile(policy.Pattern)
		if err != nil || !pattern.MatchString(normalized) || containsUnsafeManualText(normalized) {
			return "", false
		}
		return normalized, true
	default:
		return "", false
	}
}

func (s *Service) normalizeManualName(config *MethodConfig, value string) (string, bool) {
	if config != nil && config.SchoolAdapterID == BUAAAdapterID {
		return s.buaa.NormalizeName(value)
	}
	value = strings.TrimSpace(value)
	if value == "" || utf8.RuneCountInString(value) > 100 || containsUnsafeManualText(value) {
		return "", false
	}
	return value, true
}

func containsUnsafeManualText(value string) bool {
	return strings.ContainsAny(value, "\x00\r\n") || strings.ContainsFunc(value, func(r rune) bool {
		return unicode.IsControl(r) || unicode.In(r, unicode.Cf, unicode.Co, unicode.Cs)
	})
}

func normalizeManualReviewEmail(value string, domains []string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || len(value) > 320 || strings.Count(value, "@") != 1 {
		return "", false
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value {
		return "", false
	}
	parts := strings.SplitN(value, "@", 2)
	if parts[0] == "" || parts[1] == "" {
		return "", false
	}
	if len(domains) > 0 && !slices.ContainsFunc(domains, func(domain string) bool {
		return strings.EqualFold(strings.TrimSpace(domain), parts[1])
	}) {
		return "", false
	}
	return value, true
}

func maskManualReviewName(value string) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) == 0 {
		return "*"
	}
	if len(runes) == 1 {
		return string(runes[0]) + "*"
	}
	return string(runes[0]) + strings.Repeat("*", min(len(runes)-1, 4))
}

func (s *Service) GetManualReview(
	ctx context.Context,
	userID int64,
	applicationID string,
) (*ManualReviewCase, error) {
	reviewCase, err := s.repo.GetManualReviewCaseForUser(ctx, applicationID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrManualReviewNotFound
	}
	if err != nil {
		return nil, err
	}
	if config, configErr := s.repo.GetMethodConfig(
		ctx, reviewCase.SchoolCode, MethodManualMaterialReview,
	); configErr == nil {
		if _, policy, policyErr := decodeManualReviewConfiguration(config); policyErr == nil {
			reviewCase.EmailVerificationRequired = *policy.RequireEmailVerification
		}
	}
	s.decorateManualReviewCase(reviewCase, true)
	return reviewCase, nil
}

func (s *Service) decorateManualReviewCase(reviewCase *ManualReviewCase, applicantView bool) {
	if reviewCase == nil {
		return
	}
	reviewCase.EmailVerified = reviewCase.EmailVerifiedAt != nil
	if reviewCase.Materials == nil {
		reviewCase.Materials = []ManualReviewMaterial{}
	}
	actions := make([]string, 0, 4)
	switch reviewCase.Status {
	case ManualReviewDraft, ManualReviewSupplementRequired:
		actions = append(actions, "edit_form", "capture_material")
		if len(reviewCase.Materials) > 0 &&
			(!reviewCase.EmailVerificationRequired || reviewCase.EmailVerified) {
			actions = append(actions, "submit")
		}
	case ManualReviewPending:
		actions = append(actions, "wait_for_review")
	case ManualReviewApproved:
		actions = append(actions, "return_to_consumer")
	}
	if reviewCase.Status == ManualReviewSupplementRequired {
		actions = append(actions, "add_supplement")
	}
	reviewCase.NextActions = actions
	if !applicantView {
		reviewCase.NextActions = []string{}
	}
}

func (s *Service) SubmitManualReview(
	ctx context.Context,
	userID int64,
	applicationID string,
	confirmed bool,
) (*ManualReviewCase, error) {
	if !confirmed {
		return nil, ErrConsentRequired
	}
	now := s.now()
	err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		application, err := s.repo.GetApplicationForUpdateTx(ctx, tx, applicationID, userID)
		if err != nil {
			return err
		}
		if !applicationIsNonTerminal(application) || !application.ExpiresAt.After(now) {
			return ErrApplicationState
		}
		config, err := s.repo.GetMethodConfigTx(ctx, tx, application.SchoolCode, MethodManualMaterialReview)
		if err != nil || !methodHealthy(config) {
			return ErrMethodUnavailable
		}
		_, policy, err := decodeManualReviewConfiguration(config)
		if err != nil {
			return err
		}
		reviewCase, err := s.repo.GetManualReviewCaseForApplicationUpdateTx(ctx, tx, applicationID, userID)
		if err != nil {
			return err
		}
		if reviewCase.PrivacyNoticeVersion != config.PrivacyNoticeVersion {
			return ErrMethodUnavailable
		}
		return s.repo.SubmitManualReviewTx(
			ctx, tx, reviewCase, *policy.RequireEmailVerification,
			now.Add(time.Duration(policy.ReviewWindowSeconds)*time.Second), now,
		)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrManualReviewNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.GetManualReview(ctx, userID, applicationID)
}

func (s *Service) ListAdminManualReviews(
	ctx context.Context,
	schoolCode string,
	status ManualReviewStatus,
	limit int,
	offset int,
) ([]*ManualReviewCase, error) {
	if !schoolCodePattern.MatchString(schoolCode) || limit < 1 || limit > 100 || offset < 0 {
		return nil, ErrManualReviewInvalidForm
	}
	if status != "" {
		switch status {
		case ManualReviewPending, ManualReviewSupplementRequired,
			ManualReviewApproved, ManualReviewRejected:
		default:
			return nil, ErrManualReviewInvalidForm
		}
	}
	cases, err := s.repo.ListManualReviewCases(ctx, schoolCode, status, limit, offset)
	if err != nil {
		return nil, err
	}
	requireEmailVerification := false
	if config, configErr := s.repo.GetMethodConfig(
		ctx, schoolCode, MethodManualMaterialReview,
	); configErr == nil {
		if _, policy, policyErr := decodeManualReviewConfiguration(config); policyErr == nil {
			requireEmailVerification = *policy.RequireEmailVerification
		}
	}
	for _, reviewCase := range cases {
		reviewCase.EmailVerificationRequired = requireEmailVerification
		s.decorateManualReviewCase(reviewCase, false)
	}
	return cases, nil
}

func (s *Service) GetAdminManualReview(
	ctx context.Context,
	caseID string,
) (*AdminManualReviewDetail, error) {
	reviewCase, err := s.repo.GetManualReviewCaseByID(ctx, caseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrManualReviewNotFound
	}
	if err != nil {
		return nil, err
	}
	if config, configErr := s.repo.GetMethodConfig(
		ctx, reviewCase.SchoolCode, MethodManualMaterialReview,
	); configErr == nil {
		if _, policy, policyErr := decodeManualReviewConfiguration(config); policyErr == nil {
			reviewCase.EmailVerificationRequired = *policy.RequireEmailVerification
		}
	}
	formValues, err := s.decryptManualReviewForm(reviewCase)
	if err != nil {
		return nil, err
	}
	s.decorateManualReviewCase(reviewCase, false)
	return &AdminManualReviewDetail{Case: reviewCase, FormValues: formValues}, nil
}

func (s *Service) decryptManualReviewForm(reviewCase *ManualReviewCase) (map[string]string, error) {
	if s.rosterCipher == nil || reviewCase == nil || len(reviewCase.FormDataEnc) == 0 {
		return nil, ErrDependencyUnavailable
	}
	plaintext, err := s.rosterCipher.Decrypt(reviewCase.FormDataEnc)
	if err != nil {
		return nil, ErrDependencyUnavailable
	}
	digest, err := appcrypto.HMACHashWithKey(
		manualFormDigestDomain+reviewCase.SchoolCode+":"+plaintext, s.hmacKey,
	)
	if err != nil || !constantTimeStringEqual(digest, reviewCase.FormDigest) {
		return nil, ErrDependencyUnavailable
	}
	var values map[string]string
	if err := json.Unmarshal([]byte(plaintext), &values); err != nil || len(values) == 0 {
		return nil, ErrDependencyUnavailable
	}
	return values, nil
}

func (s *Service) DecideManualReview(
	ctx context.Context,
	input ManualReviewDecisionInput,
) (*ManualReviewCase, error) {
	if input.ReviewerUserID <= 0 || strings.TrimSpace(input.UserVisibleReason) == "" ||
		utf8.RuneCountInString(input.UserVisibleReason) > 500 ||
		utf8.RuneCountInString(input.InternalRiskNote) > 2000 ||
		containsUnsafeManualText(input.UserVisibleReason) || containsUnsafeManualText(input.InternalRiskNote) {
		return nil, ErrManualReviewInvalidForm
	}
	var internalNoteEnc []byte
	var err error
	if strings.TrimSpace(input.InternalRiskNote) != "" {
		if s.rosterCipher == nil {
			return nil, ErrDependencyUnavailable
		}
		internalNoteEnc, err = s.rosterCipher.Encrypt(strings.TrimSpace(input.InternalRiskNote))
		if err != nil {
			return nil, err
		}
	}
	now := s.now()
	decisionOutcome := manualReviewDecisionOutcome{}
	err = s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		reviewCase, err := s.repo.GetManualReviewCaseForUpdateTx(ctx, tx, input.CaseID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrManualReviewNotFound
		}
		if err != nil {
			return err
		}
		if reviewCase.UserID == input.ReviewerUserID {
			return ErrManualReviewSelfDecision
		}
		if reviewCase.Status != ManualReviewPending {
			return ErrManualReviewState
		}
		application, err := s.repo.GetApplicationForUpdateTx(
			ctx, tx, reviewCase.ApplicationID, reviewCase.UserID,
		)
		if err != nil {
			return err
		}
		if application.Status != ApplicationPendingManualReview || !application.ExpiresAt.After(now) {
			return ErrApplicationExpired
		}
		config, err := s.repo.GetMethodConfigTx(ctx, tx, reviewCase.SchoolCode, MethodManualMaterialReview)
		if err != nil || !methodHealthy(config) {
			return ErrMethodUnavailable
		}
		_, policy, err := decodeManualReviewConfiguration(config)
		if err != nil || reviewCase.PrivacyNoticeVersion != config.PrivacyNoticeVersion {
			return ErrMethodUnavailable
		}
		switch input.Action {
		case ManualDecisionRequestSupplement:
			return s.repo.RequestManualReviewSupplementTx(
				ctx, tx, reviewCase, input.ReviewerUserID,
				strings.TrimSpace(input.UserVisibleReason), internalNoteEnc,
				now.Add(time.Duration(policy.ReviewWindowSeconds)*time.Second), now,
			)
		case ManualDecisionReject:
			return s.repo.RejectManualReviewTx(
				ctx, tx, reviewCase, input.ReviewerUserID,
				strings.TrimSpace(input.UserVisibleReason), internalNoteEnc, config, now,
			)
		case ManualDecisionApprove:
			if input.ExpiresInDays == nil || *input.ExpiresInDays < 1 || *input.ExpiresInDays > 366 {
				return ErrManualReviewInvalidForm
			}
			return s.approveManualReviewTx(
				ctx, tx, reviewCase, input.ReviewerUserID,
				strings.TrimSpace(input.UserVisibleReason), internalNoteEnc,
				*input.ExpiresInDays, config, policy, now, &decisionOutcome,
			)
		default:
			return ErrManualReviewInvalidForm
		}
	})
	if err != nil {
		return nil, err
	}
	if decisionOutcome.SubjectConflict {
		return nil, ErrSubjectConflict
	}
	reviewCase, err := s.repo.GetManualReviewCaseByID(ctx, input.CaseID)
	if err != nil {
		return nil, err
	}
	s.decorateManualReviewCase(reviewCase, false)
	return reviewCase, nil
}

type manualReviewDecisionOutcome struct {
	SubjectConflict bool
}

func (s *Service) approveManualReviewTx(
	ctx context.Context,
	tx pgx.Tx,
	reviewCase *ManualReviewCase,
	reviewerUserID int64,
	userVisibleReason string,
	internalNoteEnc []byte,
	expiresInDays int,
	config *MethodConfig,
	policy manualReviewPolicy,
	now time.Time,
	outcome *manualReviewDecisionOutcome,
) error {
	if config.CredentialTTL == nil {
		return ErrMethodUnavailable
	}
	maximumExpiry := int(config.CredentialTTL.Hours() / 24)
	credentialClass := "formal_student"
	if reviewCase.MaterialType == ManualMaterialAdmissionNotice {
		credentialClass = "temporary_freshman"
		maximumExpiry = min(maximumExpiry, policy.AdmissionNoticeMaxCredentialDays)
	}
	if expiresInDays > maximumExpiry || maximumExpiry < 1 {
		return ErrManualReviewInvalidForm
	}
	formValues, err := s.decryptManualReviewForm(reviewCase)
	if err != nil {
		return err
	}
	studentID := formValues["studentID"]
	subjectHash, err := ComputeRosterBlindIndex(
		s.hmacKey, reviewCase.SchoolID, BlindIndexSubject, studentID,
	)
	if err != nil {
		return err
	}
	if err := s.repo.LockEnrollmentSubjectTx(ctx, tx, reviewCase.SchoolID, subjectHash); err != nil {
		return err
	}
	subject, err := s.repo.GetActiveEnrollmentSubjectTx(ctx, tx, reviewCase.SchoolID, subjectHash)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		subjectID, idErr := newID()
		if idErr != nil {
			return idErr
		}
		subject = &EnrollmentSubject{
			ID: subjectID, UserID: reviewCase.UserID, SchoolID: reviewCase.SchoolID,
			SubjectHash: subjectHash, StudentIDHash: reviewCase.StudentIDHash,
			StudentDisplay: reviewCase.StudentIDMasked, BindingStatus: "active",
		}
		if err := s.repo.CreateEnrollmentSubjectTx(
			ctx, tx, *subject, nil, MethodManualMaterialReview, nil, nil, now,
		); err != nil {
			return err
		}
	case err != nil:
		return err
	case subject.UserID != reviewCase.UserID:
		application := &Application{
			ID: reviewCase.ApplicationID, UserID: reviewCase.UserID,
			SchoolID: reviewCase.SchoolID,
		}
		if err := s.repo.CreateSubjectConflictTx(
			ctx, tx, application, subjectHash, subject.UserID, now,
		); err != nil {
			return err
		}
		outcome.SubjectConflict = true
		return nil
	}
	credentialID, err := newID()
	if err != nil {
		return err
	}
	expiresAt := now.AddDate(0, 0, expiresInDays)
	credential := Credential{
		ID: credentialID, UserID: reviewCase.UserID, SchoolID: reviewCase.SchoolID,
		Method: MethodManualMaterialReview, Status: CredentialActive,
		CredentialClass: credentialClass, SubjectHash: subjectHash,
		SubjectDisplay: reviewCase.StudentIDMasked, EnrollmentID: &subject.ID,
		RosterDependency: "independent", Assurance: "reviewed", VerifiedAt: now,
		ExpiresAt: &expiresAt, Revision: 1,
	}
	metadata := json.RawMessage(`{"evidence_path":"manual_material_review","reviewed":true}`)
	if err := s.repo.CreateCredentialTx(
		ctx, tx, credential, reviewCase.ApplicationID, config.AdapterID,
		config.AdapterVersion, nil, nil, metadata, now,
	); err != nil {
		return err
	}
	if err := s.repo.ApproveManualReviewCaseTx(
		ctx, tx, reviewCase, reviewerUserID, userVisibleReason,
		internalNoteEnc, credential, config, now,
	); err != nil {
		return err
	}
	return s.repo.BumpEligibilityRevisionTx(
		ctx, tx, reviewCase.UserID, reviewCase.SchoolID, "credential_activated", now,
	)
}

func (s *Service) CreateSchoolVerificationSuggestion(
	ctx context.Context,
	userID int64,
	schoolName string,
	schoolLocation string,
) (*SchoolVerificationSuggestion, error) {
	schoolName = strings.TrimSpace(schoolName)
	schoolLocation = strings.TrimSpace(schoolLocation)
	if userID <= 0 || utf8.RuneCountInString(schoolName) < 2 ||
		utf8.RuneCountInString(schoolName) > 100 || utf8.RuneCountInString(schoolLocation) > 100 ||
		containsUnsafeManualText(schoolName) || containsUnsafeManualText(schoolLocation) {
		return nil, ErrManualReviewInvalidForm
	}
	var location *string
	if schoolLocation != "" {
		if utf8.RuneCountInString(schoolLocation) < 2 {
			return nil, ErrManualReviewInvalidForm
		}
		location = &schoolLocation
	}
	id, err := newID()
	if err != nil {
		return nil, err
	}
	now := s.now()
	if err := s.repo.CreateSchoolVerificationSuggestion(
		ctx, userID, id, schoolName, location, now,
	); err != nil {
		return nil, err
	}
	return &SchoolVerificationSuggestion{ID: id, Status: "pending", CreatedAt: now}, nil
}

func manualReviewEmailFromForm(values map[string]string) string {
	return strings.ToLower(strings.TrimSpace(values["email"]))
}

func (s *Service) GetManualMaterialAccess(
	ctx context.Context,
	caseID string,
	materialID string,
	actorUserID int64,
) (*ManualMaterialAccess, string, error) {
	if s.manualMaterialStore == nil || s.manualMaterialAccessTTL < time.Minute ||
		s.manualMaterialAccessTTL > 15*time.Minute {
		return nil, "", ErrManualMaterialStoreUnavailable
	}
	material, _, schoolCode, err := s.repo.GetManualMaterialForCase(ctx, caseID, materialID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrManualMaterialNotFound
	}
	if err != nil {
		return nil, "", err
	}
	now := s.now()
	requestedReason := "short_lived_read_url"
	if err := s.repo.RecordManualMaterialAccessEvent(
		ctx, caseID, actorUserID, "material_access_requested", &requestedReason, now,
	); err != nil {
		return nil, "", err
	}
	url, err := s.manualMaterialStore.GetManualReviewMaterialURL(ctx, material.ObjectKey)
	if err != nil {
		failedReason := "material_store_unavailable"
		auditErr := s.repo.RecordManualMaterialAccessEvent(
			context.WithoutCancel(ctx), caseID, actorUserID,
			"material_access_failed", &failedReason, s.now(),
		)
		return nil, "", errors.Join(ErrManualMaterialStoreUnavailable, auditErr)
	}
	accessedReason := "short_lived_read_url_issued"
	if err := s.repo.RecordManualMaterialAccessEvent(
		ctx, caseID, actorUserID, "material_accessed", &accessedReason, now,
	); err != nil {
		return nil, "", err
	}
	return &ManualMaterialAccess{URL: url, ExpiresAt: now.Add(s.manualMaterialAccessTTL)}, schoolCode, nil
}

func (s *Service) RecordManualMaterialAccessDenied(
	ctx context.Context,
	caseID string,
	actorUserID int64,
) error {
	reason := "school_scope_denied"
	return s.repo.RecordManualMaterialAccessEvent(
		ctx, caseID, actorUserID, "material_access_denied", &reason, s.now(),
	)
}

func (s *Service) ManualReviewCaseSchoolCode(ctx context.Context, caseID string) (string, error) {
	schoolCode, err := s.repo.GetManualReviewCaseSchoolCode(ctx, caseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrManualReviewNotFound
	}
	if err != nil {
		return "", err
	}
	return schoolCode, nil
}
