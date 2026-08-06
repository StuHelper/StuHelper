package studentverification

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

var (
	adminAdapterPattern            = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	adminAdapterVersionPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	adminDomainPattern             = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$`)
	adminConnectorOperationPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,127}$`)
)

func (s *Service) ListAdminVerificationSchools(ctx context.Context) ([]AdminVerificationSchoolConfig, error) {
	return s.repo.ListAdminVerificationSchools(ctx)
}

func (s *Service) GetAdminVerificationSchool(ctx context.Context, schoolCode string) (*AdminVerificationSchoolConfig, error) {
	if !schoolCodePattern.MatchString(strings.TrimSpace(schoolCode)) {
		return nil, ErrSchoolNotFound
	}
	profile, err := s.repo.GetAdminVerificationSchool(ctx, schoolCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSchoolNotFound
	}
	return profile, err
}

func (s *Service) CreateAdminVerificationSchool(
	ctx context.Context,
	input CreateAdminVerificationSchoolConfigInput,
) (*AdminVerificationSchoolConfig, error) {
	normalized := UpdateAdminVerificationSchoolConfigInput{
		SchoolCode: input.SchoolCode, ActorUserID: input.ActorUserID,
		AdapterID: input.AdapterID, AdapterVersion: input.AdapterVersion,
		EmailDomains: input.EmailDomains, StudentIDPolicy: input.StudentIDPolicy,
		NameMatchPolicy: input.NameMatchPolicy, EnrollmentPolicy: input.EnrollmentPolicy,
		ManualFormSchema:            input.ManualFormSchema,
		SnapshotSyncIntervalSeconds: input.SnapshotSyncIntervalSeconds,
		SnapshotWarningAfterSeconds: input.SnapshotWarningAfterSeconds,
		SnapshotHardExpirySeconds:   input.SnapshotHardExpirySeconds,
		SnapshotGraceSeconds:        input.SnapshotGraceSeconds,
		SnapshotAutoActivate:        input.SnapshotAutoActivate,
		Reason:                      input.Reason,
	}
	if !normalizeAndValidateAdminSchoolDraft(&normalized, false) {
		return nil, ErrAdminConfigInvalid
	}
	if err := s.repo.CreateAdminVerificationSchool(ctx, normalized); err != nil {
		return nil, err
	}
	return s.GetAdminVerificationSchool(ctx, normalized.SchoolCode)
}

func (s *Service) UpdateAdminVerificationSchool(
	ctx context.Context,
	input UpdateAdminVerificationSchoolConfigInput,
) (*AdminVerificationSchoolConfig, error) {
	if !normalizeAndValidateAdminSchoolDraft(&input, true) {
		return nil, ErrAdminConfigInvalid
	}
	if err := s.repo.UpdateAdminVerificationSchool(ctx, input); err != nil {
		return nil, err
	}
	return s.GetAdminVerificationSchool(ctx, input.SchoolCode)
}

func (s *Service) UpdateAdminVerificationMethod(
	ctx context.Context,
	input UpdateAdminVerificationMethodConfigInput,
) (*AdminVerificationMethodConfig, error) {
	input.SchoolCode = strings.TrimSpace(input.SchoolCode)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Description = strings.TrimSpace(input.Description)
	input.AdapterID = strings.TrimSpace(input.AdapterID)
	input.AdapterVersion = strings.TrimSpace(input.AdapterVersion)
	input.RosterDependency = strings.TrimSpace(input.RosterDependency)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.ConnectorOperationKey != nil {
		value := strings.TrimSpace(*input.ConnectorOperationKey)
		input.ConnectorOperationKey = &value
	}
	if input.PrivacyNoticeVersion != nil {
		value := strings.TrimSpace(*input.PrivacyNoticeVersion)
		input.PrivacyNoticeVersion = &value
	}
	if !validAdminActorReason(input.ActorUserID, input.Reason) || input.ExpectedRevision < 0 ||
		!schoolCodePattern.MatchString(input.SchoolCode) || !validVerificationMethod(input.Method) ||
		utf8.RuneCountInString(input.DisplayName) < 1 || utf8.RuneCountInString(input.DisplayName) > 100 ||
		utf8.RuneCountInString(input.Description) > 500 || containsUnsafeManualText(input.DisplayName) ||
		containsUnsafeManualText(input.Description) || !adminAdapterPattern.MatchString(input.AdapterID) ||
		!adminAdapterVersionPattern.MatchString(input.AdapterVersion) ||
		!validRosterDependency(input.RosterDependency, input.ConditionalPolicy) ||
		!validAdminJSONObjects(input.ConditionalPolicy, input.PublicFormSchema, input.RiskPolicy, input.PrivacyNotice) ||
		!validAdminTTL(input.Method, input.CredentialTTLSeconds) ||
		!validOptionalConnectorOperation(input.ConnectorOperationKey) ||
		!validOptionalNoticeVersion(input.PrivacyNoticeVersion) {
		return nil, ErrAdminConfigInvalid
	}
	if err := s.repo.UpdateAdminVerificationMethod(ctx, input); err != nil {
		return nil, err
	}
	return s.repo.GetAdminVerificationMethod(ctx, input.SchoolCode, input.Method)
}

func (s *Service) ValidateAdminVerificationSchool(
	ctx context.Context,
	input ValidateAdminVerificationConfigInput,
) (*AdminVerificationSchoolConfig, error) {
	input.SchoolCode = strings.TrimSpace(input.SchoolCode)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.Method != nil || !schoolCodePattern.MatchString(input.SchoolCode) ||
		!validAdminActorAndReason(input.ActorUserID, input.ExpectedRevision, input.Reason) {
		return nil, ErrAdminConfigInvalid
	}
	profile, err := s.repo.GetAdminVerificationSchool(ctx, input.SchoolCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSchoolNotFound
	}
	if err != nil {
		return nil, err
	}
	if profile.ConfigRevision != input.ExpectedRevision {
		return nil, ErrAdminConfigRevision
	}
	valid, code := validateAdminSchoolProfile(profile)
	if err := s.repo.ValidateAdminVerificationSchool(ctx, input, valid, code); err != nil {
		return nil, err
	}
	return s.GetAdminVerificationSchool(ctx, input.SchoolCode)
}

func (s *Service) ValidateAdminVerificationMethod(
	ctx context.Context,
	input ValidateAdminVerificationConfigInput,
) (*AdminVerificationMethodConfig, error) {
	input.SchoolCode = strings.TrimSpace(input.SchoolCode)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.Method == nil || !validVerificationMethod(*input.Method) ||
		!schoolCodePattern.MatchString(input.SchoolCode) ||
		!validAdminActorAndReason(input.ActorUserID, input.ExpectedRevision, input.Reason) {
		return nil, ErrAdminConfigInvalid
	}
	profile, err := s.repo.GetAdminVerificationSchool(ctx, input.SchoolCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSchoolNotFound
	}
	if err != nil {
		return nil, err
	}
	method, err := s.repo.GetAdminVerificationMethod(ctx, input.SchoolCode, *input.Method)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMethodUnavailable
	}
	if err != nil {
		return nil, err
	}
	if method.ConfigRevision != input.ExpectedRevision {
		return nil, ErrAdminConfigRevision
	}
	valid, validationCode := s.validateAdminMethodStructure(ctx, profile, method)
	healthStatus, healthCode := "unavailable", "configuration_invalid"
	if valid {
		healthStatus, healthCode = s.evaluateAdminMethodHealth(ctx, profile, method)
	}
	if err := s.repo.ValidateAdminVerificationMethod(
		ctx, input, valid, validationCode, healthStatus, healthCode,
	); err != nil {
		return nil, err
	}
	return s.repo.GetAdminVerificationMethod(ctx, input.SchoolCode, *input.Method)
}

func validateAdminSchoolProfile(profile *AdminVerificationSchoolConfig) (bool, string) {
	if profile == nil || !adminAdapterPattern.MatchString(profile.AdapterID) ||
		!adminAdapterVersionPattern.MatchString(profile.AdapterVersion) {
		return false, "adapter_invalid"
	}
	if profile.SchoolCode == BUAASchoolCode {
		if profile.AdapterID != BUAAAdapterID {
			return false, "school_adapter_mismatch"
		}
	} else if profile.AdapterID != "declarative" {
		return false, "adapter_not_registered"
	}
	if _, ok := normalizeAdminEmailDomains(profile.EmailDomains); !ok {
		return false, "email_domain_invalid"
	}
	if !validAdminStudentIDPolicy(profile.SchoolCode, profile.AdapterID, profile.StudentIDPolicy) {
		return false, "student_id_policy_invalid"
	}
	if !validAdminNamePolicy(profile.SchoolCode, profile.NameMatchPolicy) {
		return false, "name_policy_invalid"
	}
	if _, err := decodeRosterPolicy(profile.enrollmentPolicyRaw); err != nil {
		return false, "enrollment_policy_invalid"
	}
	if profile.SnapshotSyncIntervalSeconds <= 0 ||
		profile.SnapshotWarningAfterSeconds < profile.SnapshotSyncIntervalSeconds ||
		profile.SnapshotHardExpirySeconds <= profile.SnapshotWarningAfterSeconds ||
		profile.SnapshotGraceSeconds < 0 {
		return false, "snapshot_freshness_invalid"
	}
	return true, ""
}

func (s *Service) validateAdminMethodStructure(
	ctx context.Context,
	profile *AdminVerificationSchoolConfig,
	method *AdminVerificationMethodConfig,
) (bool, string) {
	if profile == nil || method == nil || !validVerificationMethod(method.Method) ||
		!adminAdapterPattern.MatchString(method.AdapterID) ||
		!adminAdapterVersionPattern.MatchString(method.AdapterVersion) ||
		!validRosterDependency(method.RosterDependency, method.ConditionalPolicy) ||
		!validAdminTTL(method.Method, method.CredentialTTLSeconds) ||
		!validOptionalConnectorOperation(method.ConnectorOperationKey) ||
		!validOptionalNoticeVersion(method.PrivacyNoticeVersion) {
		return false, "method_fields_invalid"
	}
	if !validPrivacyNotice(method.PrivacyNoticeVersion, method.PrivacyNotice) {
		return false, "privacy_notice_invalid"
	}
	switch method.Method {
	case MethodRealNameIdentityCheck, MethodStudentEmailOutboundOTP, MethodStudentEmailInbound:
		if profile.SchoolCode != BUAASchoolCode || profile.AdapterID != BUAAAdapterID || method.AdapterID != BUAAAdapterID {
			return false, "adapter_method_mismatch"
		}
	case MethodSchoolSSO:
		if profile.SchoolCode != BUAASchoolCode || method.AdapterID != "buaa_ldap_bind" ||
			method.ConnectorOperationKey == nil || *method.ConnectorOperationKey == "" {
			return false, "sso_adapter_invalid"
		}
	case MethodManualMaterialReview:
		if method.AdapterID != "shared_manual_review" {
			return false, "manual_adapter_invalid"
		}
		config, err := s.repo.GetMethodConfig(ctx, profile.SchoolCode, method.Method)
		if err != nil {
			return false, "manual_config_unavailable"
		}
		if _, _, err := decodeManualReviewConfiguration(config); err != nil {
			return false, "manual_config_invalid"
		}
	}
	if method.Method == MethodStudentEmailOutboundOTP {
		if _, ok := parseEmailOTPPolicy(method.riskPolicyRaw); !ok {
			return false, "email_risk_policy_invalid"
		}
	}
	return true, ""
}

func (s *Service) evaluateAdminMethodHealth(
	ctx context.Context,
	profile *AdminVerificationSchoolConfig,
	method *AdminVerificationMethodConfig,
) (string, string) {
	if method.Method == MethodSchoolSSO && method.ConnectorOperationKey != nil &&
		!s.repo.ConnectorOperationAvailable(
			ctx, profile.SchoolID, *method.ConnectorOperationKey,
			method.AdapterID, method.AdapterVersion,
		) {
		return "unavailable", "connector_operation_unavailable"
	}
	capability := MethodCapability{Method: method.Method, adapterID: method.AdapterID}
	school := VerificationSchool{ID: profile.SchoolID, Code: profile.SchoolCode, Name: profile.SchoolName}
	if !s.methodRuntimeAvailable(ctx, school, &capability) {
		return "unavailable", "runtime_dependency_unavailable"
	}
	return "healthy", ""
}

func (s *Service) ListAdminStudentCredentials(
	ctx context.Context,
	schoolCode string,
	status CredentialStatus,
	limit int,
	offset int,
) ([]AdminStudentCredential, error) {
	if !schoolCodePattern.MatchString(strings.TrimSpace(schoolCode)) ||
		limit < 1 || limit > 100 || offset < 0 || (status != "" && !validCredentialStatus(status)) {
		return nil, ErrAdminConfigInvalid
	}
	return s.repo.ListAdminStudentCredentials(ctx, schoolCode, status, limit, offset)
}

func (s *Service) GetAdminStudentCredential(ctx context.Context, credentialID string) (*AdminStudentCredential, error) {
	credential, err := s.repo.GetAdminStudentCredential(ctx, strings.TrimSpace(credentialID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCredentialNotFound
	}
	return credential, err
}

func (s *Service) RevokeAdminStudentCredential(
	ctx context.Context,
	input AdminCredentialRevokeInput,
) (*AdminStudentCredential, error) {
	input.CredentialID = strings.TrimSpace(input.CredentialID)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.CredentialID == "" || !validAdminActorAndReason(input.ActorUserID, input.ExpectedRevision, input.Reason) {
		return nil, ErrAdminConfigInvalid
	}
	if err := s.repo.RevokeAdminStudentCredential(ctx, input); err != nil {
		return nil, err
	}
	return s.GetAdminStudentCredential(ctx, input.CredentialID)
}

func (s *Service) ListAdminStudentSubjectConflicts(
	ctx context.Context,
	schoolCode string,
	status string,
	limit int,
	offset int,
) ([]AdminStudentSubjectConflict, error) {
	status = strings.TrimSpace(status)
	if !schoolCodePattern.MatchString(strings.TrimSpace(schoolCode)) || limit < 1 || limit > 100 ||
		offset < 0 || (status != "" && status != "open" && status != "under_review" && status != "resolved" && status != "dismissed") {
		return nil, ErrAdminConfigInvalid
	}
	return s.repo.ListAdminStudentSubjectConflicts(ctx, schoolCode, status, limit, offset)
}

func (s *Service) GetAdminStudentSubjectConflict(ctx context.Context, conflictID string) (*AdminStudentSubjectConflict, error) {
	conflict, err := s.repo.GetAdminStudentSubjectConflict(ctx, strings.TrimSpace(conflictID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAdminConflictNotFound
	}
	return conflict, err
}

func (s *Service) DecideAdminStudentSubjectConflict(
	ctx context.Context,
	input AdminSubjectConflictDecisionInput,
) (*AdminStudentSubjectConflict, error) {
	input.ConflictID = strings.TrimSpace(input.ConflictID)
	input.Action = strings.TrimSpace(input.Action)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.ConflictID == "" || input.ActorUserID <= 0 ||
		(input.Action != "start_review" && input.Action != "dismiss_claim" && input.Action != "release_subject_for_reverification") ||
		utf8.RuneCountInString(input.Reason) < 4 || utf8.RuneCountInString(input.Reason) > 500 || containsUnsafeManualText(input.Reason) {
		return nil, ErrAdminConfigInvalid
	}
	if err := s.repo.DecideAdminStudentSubjectConflict(ctx, input); err != nil {
		return nil, err
	}
	return s.GetAdminStudentSubjectConflict(ctx, input.ConflictID)
}

func (s *Service) ListAdminCampusConnectorHealth(
	ctx context.Context,
	schoolCode string,
) ([]AdminCampusConnectorHealth, error) {
	schoolCode = strings.TrimSpace(schoolCode)
	if schoolCode != "" && !schoolCodePattern.MatchString(schoolCode) {
		return nil, ErrAdminConfigInvalid
	}
	return s.repo.ListAdminCampusConnectorHealth(ctx, schoolCode)
}

func validAdminActorAndReason(actorUserID, revision int64, reason string) bool {
	return revision > 0 && validAdminActorReason(actorUserID, reason)
}

func validAdminActorReason(actorUserID int64, reason string) bool {
	return actorUserID > 0 && utf8.RuneCountInString(reason) >= 4 &&
		utf8.RuneCountInString(reason) <= 500 && !containsUnsafeManualText(reason)
}

func normalizeAndValidateAdminSchoolDraft(input *UpdateAdminVerificationSchoolConfigInput, requireRevision bool) bool {
	if input == nil {
		return false
	}
	input.SchoolCode = strings.TrimSpace(input.SchoolCode)
	input.AdapterID = strings.TrimSpace(input.AdapterID)
	input.AdapterVersion = strings.TrimSpace(input.AdapterVersion)
	input.Reason = strings.TrimSpace(input.Reason)
	if !validAdminActorReason(input.ActorUserID, input.Reason) ||
		(requireRevision && input.ExpectedRevision <= 0) ||
		!schoolCodePattern.MatchString(input.SchoolCode) ||
		!adminAdapterPattern.MatchString(input.AdapterID) ||
		!adminAdapterVersionPattern.MatchString(input.AdapterVersion) ||
		!validSnapshotFreshnessInput(*input) || !validAdminJSONObjects(
		input.StudentIDPolicy, input.NameMatchPolicy, input.EnrollmentPolicy, input.ManualFormSchema,
	) {
		return false
	}
	domains, ok := normalizeAdminEmailDomains(input.EmailDomains)
	if !ok {
		return false
	}
	input.EmailDomains = domains
	return true
}

func validSnapshotFreshnessInput(input UpdateAdminVerificationSchoolConfigInput) bool {
	return input.SnapshotSyncIntervalSeconds > 0 &&
		input.SnapshotWarningAfterSeconds >= input.SnapshotSyncIntervalSeconds &&
		input.SnapshotHardExpirySeconds > input.SnapshotWarningAfterSeconds &&
		input.SnapshotGraceSeconds >= 0
}

func validAdminJSONObjects(values ...map[string]any) bool {
	for _, value := range values {
		if value == nil {
			return false
		}
		encoded, err := json.Marshal(value)
		if err != nil || len(encoded) > 64*1024 {
			return false
		}
	}
	return true
}

func normalizeAdminEmailDomains(values []string) ([]string, bool) {
	if len(values) > 20 {
		return nil, false
	}
	unique := make(map[string]struct{}, len(values))
	for _, value := range values {
		domain := strings.ToLower(strings.TrimSpace(value))
		if len(domain) > 253 || !adminDomainPattern.MatchString(domain) || net.ParseIP(domain) != nil {
			return nil, false
		}
		unique[domain] = struct{}{}
	}
	result := make([]string, 0, len(unique))
	for value := range unique {
		result = append(result, value)
	}
	sort.Strings(result)
	return result, true
}

func validAdminStudentIDPolicy(schoolCode, adapterID string, policy map[string]any) bool {
	strategy, ok := policy["strategy"].(string)
	if !ok {
		return false
	}
	if strategy == "adapter" {
		return schoolCode == BUAASchoolCode && adapterID == BUAAAdapterID
	}
	if strategy != "regex" {
		return false
	}
	pattern, ok := policy["pattern"].(string)
	if !ok {
		return false
	}
	transform := ""
	if rawTransform, exists := policy["transform"]; exists {
		transform, ok = rawTransform.(string)
		if !ok {
			return false
		}
	}
	if len(pattern) < 3 || len(pattern) > 256 || !strings.HasPrefix(pattern, "^") ||
		!strings.HasSuffix(pattern, "$") || (transform != "" && transform != "none" &&
		transform != "lowercase" && transform != "uppercase") {
		return false
	}
	_, err := regexp.Compile(pattern)
	return err == nil
}

func validAdminNamePolicy(schoolCode string, policy map[string]any) bool {
	strategy, ok := policy["strategy"].(string)
	if !ok {
		return false
	}
	if schoolCode == BUAASchoolCode {
		return strategy == "adapter"
	}
	return strategy == "exact_trimmed" || strategy == "exact"
}

func validVerificationMethod(method Method) bool {
	switch method {
	case MethodRealNameIdentityCheck, MethodSchoolSSO, MethodStudentEmailOutboundOTP,
		MethodStudentEmailInbound, MethodManualMaterialReview:
		return true
	default:
		return false
	}
}

func validRosterDependency(value string, policy map[string]any) bool {
	if value != "required" && value != "independent" && value != "conditional" {
		return false
	}
	return value != "conditional" || len(policy) > 0
}

func validAdminTTL(method Method, value *int) bool {
	if value != nil && (*value < 60 || *value > 157680000) {
		return false
	}
	return method != MethodManualMaterialReview || value != nil
}

func validOptionalConnectorOperation(value *string) bool {
	if value == nil {
		return true
	}
	return adminConnectorOperationPattern.MatchString(*value)
}

func validOptionalNoticeVersion(value *string) bool {
	return value == nil || (*value != "" && len(*value) <= 100 && !containsUnsafeManualText(*value))
}

func validPrivacyNotice(version *string, notice map[string]any) bool {
	if version == nil || *version == "" || len(notice) == 0 {
		return false
	}
	title, titleOK := notice["title"].(string)
	summary, summaryOK := notice["summary"].(string)
	retention, retentionOK := notice["retentionSummary"].(string)
	categories, categoriesOK := notice["dataCategories"].([]any)
	if !categoriesOK {
		if typed, ok := notice["dataCategories"].([]string); ok {
			categories = make([]any, len(typed))
			for index := range typed {
				categories[index] = typed[index]
			}
			categoriesOK = true
		}
	}
	return titleOK && summaryOK && retentionOK && categoriesOK && title != "" && summary != "" &&
		retention != "" && len(categories) > 0 && len(categories) <= 20
}

func validCredentialStatus(status CredentialStatus) bool {
	switch status {
	case CredentialPending, CredentialActive, CredentialReviewRequired,
		CredentialExpired, CredentialRevoked, CredentialRejected:
		return true
	default:
		return false
	}
}
