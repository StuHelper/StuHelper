package openplatform

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
)

const (
	clientSecretBytes     = 32
	clientIDBytes         = 16
	consentTokenBytes     = 24
	consentTokenTTL       = 5 * time.Minute
	consentRedisPrefix    = "open_platform:consent:"
	completionTokenTTL    = 10 * time.Minute
	completionRedisPrefix = "open_platform:profile_completion:"

	AuthorizeFlowCasdoor = "casdoor"
	AuthorizeFlowAccount = "account"
)

var resourceIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

type appProvisioner interface {
	GetApplication(ctx context.Context, name string) (ProvisionedApplicationSpec, error)
	EnsureApplication(ctx context.Context, spec ProvisionedApplicationSpec) error
	DeleteApplication(ctx context.Context, name string) error
}

type oidcAuthURLBuilder interface {
	GetAuthURL(clientID string, redirectURI string, scopes []string, state string) string
}

type phoneDecryptor interface {
	Decrypt(ciphertext []byte) (string, error)
}

type resourceRelationClient interface {
	Check(ctx context.Context, user, relation, object string) (bool, error)
	WriteMissingTuples(ctx context.Context, desired []fga.Tuple) error
	DeleteTuples(ctx context.Context, tuples []fga.Tuple) error
	ListObjects(ctx context.Context, user, relation, objectType string) ([]string, error)
}

type Service struct {
	repo               *Repository
	rdb                *redis.Client
	provisioner        appProvisioner
	oidc               oidcAuthURLBuilder
	phoneCipher        phoneDecryptor
	resourceFGA        resourceRelationClient
	consentBaseURL     string
	accountBaseURL     string
	rateLimiter        *disclosureRateLimiter
	replayDetector     *disclosureReplayDetector
	tokenProber        tokenMinimizationRuntimeProber
	tokenProbeRequired bool
}

type ServiceOption func(*Service)

type tokenMinimizationRuntimeProber interface {
	ProbeTokenMinimization(context.Context, ProvisionedApplicationSpec) (RuntimeTokenMinimizationProbeResult, error)
}

func WithAppProvisioner(provisioner appProvisioner) ServiceOption {
	return func(s *Service) {
		s.provisioner = provisioner
	}
}

func WithPhoneDecryptor(cipher phoneDecryptor) ServiceOption {
	return func(s *Service) {
		s.phoneCipher = cipher
	}
}

func WithOIDCAuthURLBuilder(builder oidcAuthURLBuilder) ServiceOption {
	return func(s *Service) {
		s.oidc = builder
	}
}

func WithConsentBaseURL(baseURL string) ServiceOption {
	return func(s *Service) {
		s.consentBaseURL = strings.TrimSpace(baseURL)
	}
}

func WithAccountBaseURL(baseURL string) ServiceOption {
	return func(s *Service) {
		s.accountBaseURL = strings.TrimSpace(baseURL)
	}
}

func WithDisclosureRateLimits(cfg DisclosureRateLimitConfig) ServiceOption {
	return func(s *Service) {
		s.rateLimiter = newDisclosureRateLimiter(s.rdb, cfg)
		s.replayDetector = newDisclosureReplayDetector(s.rdb, cfg)
	}
}

func WithRuntimeTokenProbe(prober tokenMinimizationRuntimeProber, required bool) ServiceOption {
	return func(s *Service) {
		s.tokenProber = prober
		s.tokenProbeRequired = required
	}
}

func WithResourceFGAClient(client resourceRelationClient) ServiceOption {
	return func(s *Service) {
		s.resourceFGA = client
	}
}

func NewService(repo *Repository, rdb *redis.Client, opts ...ServiceOption) (*Service, error) {
	if repo == nil {
		return nil, fmt.Errorf("openplatform.NewService: repo is required")
	}
	if rdb == nil {
		return nil, fmt.Errorf("openplatform.NewService: redis client is required")
	}
	svc := &Service{
		repo:        repo,
		rdb:         rdb,
		rateLimiter: newDisclosureRateLimiter(rdb, defaultDisclosureRateLimitConfig()),
		replayDetector: newDisclosureReplayDetector(
			rdb,
			defaultDisclosureRateLimitConfig(),
		),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(svc)
		}
	}
	return svc, nil
}

type RegisterAppInput struct {
	OwnerUserID      int64
	DisplayName      string
	Description      string
	HomepageURL      string
	PrivacyPolicyURL string
	RedirectURIs     []string
	Scopes           []ScopeRequestInput
}

type UpdateAppProfileInput struct {
	AppID            int64
	OwnerUserID      int64
	DisplayName      string
	Description      string
	HomepageURL      string
	PrivacyPolicyURL string
	Reason           string
	RequestID        string
}

type ScopeRequestInput struct {
	Scope  string
	Reason string
}

type RegisteredApp struct {
	App          *App
	ClientSecret string
}

type RevokeConsentInput struct {
	UserID    int64
	AppID     int64
	Scopes    []string
	RequestID string
}

type ListAdminUserConsentsInput struct {
	AppID    int64
	UserID   int64
	Page     int
	PageSize int
}

type AdminRevokeConsentInput struct {
	AppID       int64
	UserID      int64
	ActorUserID int64
	Reason      string
	Scopes      []string
	RequestID   string
}

type ListAppsInput struct {
	OwnerUserID int64
	Status      string
	Page        int
	PageSize    int
}

type ListAuditEventsInput struct {
	AppID     int64
	UserID    int64
	EventType string
	Scope     string
	Page      int
	PageSize  int
}

type ListUserConsentAuditEventsInput struct {
	UserID    int64
	AppID     int64
	EventType string
	Scope     string
	Page      int
	PageSize  int
}

type ListDeveloperAppAuditEventsInput struct {
	OwnerUserID int64
	AppID       int64
	EventType   string
	Scope       string
	Page        int
	PageSize    int
}

type ListTokenProbeEvidenceInput struct {
	AppID          int64
	ReviewerUserID int64
	Result         string
	ClientID       string
	Page           int
	PageSize       int
}

type DisclosureReportInput struct {
	WindowHours int
}

type ResourceGrantInput struct {
	AppID          int64
	ReviewerUserID int64
	ResourceType   string
	ResourceID     string
	Actions        []string
	Reason         string
	RequestID      string
}

type ResourceGrantRevokeInput struct {
	AppID          int64
	ReviewerUserID int64
	ResourceType   string
	ResourceID     string
	Actions        []string
	Reason         string
	RequestID      string
}

type ResourceGrantListInput struct {
	AppID        int64
	ResourceType string
}

type ResourceAccessCheckInput struct {
	ClientID            string
	ClientSecret        string
	AccessTokenClientID string
	AccessTokenScopes   []string
	ResourceType        string
	ResourceID          string
	Action              string
	RequestID           string
}

func (s *Service) RegisterApp(ctx context.Context, input RegisterAppInput) (*RegisteredApp, error) {
	scopes, err := normalizeScopeRequests(input.Scopes)
	if err != nil {
		return nil, err
	}
	app, secret, err := buildNewApp(input)
	if err != nil {
		return nil, err
	}
	requests := make([]ScopeRequest, 0, len(scopes))
	for _, scope := range scopes {
		requests = append(requests, ScopeRequest{
			Scope:  scope.Scope,
			Reason: scope.Reason,
			Status: ScopeStatusPending,
		})
	}
	if err := s.repo.CreateApp(ctx, app, requests); err != nil {
		return nil, fmt.Errorf("RegisterApp create app: %w", err)
	}
	return &RegisteredApp{App: app, ClientSecret: secret}, nil
}

func (s *Service) ListApps(ctx context.Context, input ListAppsInput) (AppListResult, error) {
	if input.OwnerUserID < 0 {
		return AppListResult{}, ErrDisclosureUnavailable
	}
	status, err := normalizeAppStatusFilter(input.Status)
	if err != nil {
		return AppListResult{}, err
	}
	pageSize := httputil.ClampPageSize(input.PageSize)
	return s.repo.ListApps(ctx, appListFilter{
		OwnerUserID: input.OwnerUserID,
		Status:      status,
		Limit:       pageSize,
		Offset:      httputil.SafeOffset(input.Page, pageSize),
	})
}

func (s *Service) UpdateAppProfile(ctx context.Context, input UpdateAppProfileInput) (*AppLifecycleResult, error) {
	if input.AppID <= 0 {
		return nil, ErrAppNotFound
	}
	if input.OwnerUserID <= 0 {
		return nil, ErrDisclosureUnavailable
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return nil, ErrLifecycleReasonRequired
	}
	app, err := s.repo.GetAppByID(ctx, input.AppID)
	if err != nil {
		return nil, err
	}
	if app.OwnerUserID != input.OwnerUserID {
		return nil, ErrAppNotFound
	}
	if app.Status == AppStatusRevoked {
		return nil, ErrInvalidAppStatus
	}

	updated := *app
	updated.DisplayName = strings.TrimSpace(input.DisplayName)
	updated.Description = strings.TrimSpace(input.Description)
	updated.HomepageURL = strings.TrimSpace(input.HomepageURL)
	updated.PrivacyPolicyURL = strings.TrimSpace(input.PrivacyPolicyURL)
	if err := validateAppInput(&updated); err != nil {
		return nil, err
	}

	rollback, err := s.ensureCasdoorApplicationProfileForUpdate(ctx, app, &updated)
	if err != nil {
		return nil, err
	}
	saved, err := s.repo.UpdateAppProfileWithAudit(ctx, &updated, input.OwnerUserID, reason, input.RequestID)
	if err != nil {
		if rollbackErr := s.rollbackCasdoorApplicationProfileUpdate(ctx, rollback); rollbackErr != nil {
			return nil, errors.Join(err, rollbackErr)
		}
		return nil, err
	}
	return &AppLifecycleResult{App: saved}, nil
}

func (s *Service) ensureCasdoorApplicationProfileForUpdate(
	ctx context.Context,
	current *App,
	updated *App,
) (casdoorApplicationSpecRollback, error) {
	if s.provisioner == nil || !casdoorApplicationProfileNeedsSync(current, updated) {
		return casdoorApplicationSpecRollback{}, nil
	}
	name := strings.TrimSpace(current.CasdoorApplicationName)
	if name == "" {
		return casdoorApplicationSpecRollback{}, nil
	}
	previous, err := s.provisioner.GetApplication(ctx, name)
	if err != nil {
		return casdoorApplicationSpecRollback{}, fmt.Errorf("read Casdoor application before open platform profile update: %w", err)
	}
	if strings.TrimSpace(previous.ClientSecret) == "" {
		return casdoorApplicationSpecRollback{}, fmt.Errorf("read Casdoor application before open platform profile update: client secret unavailable")
	}
	desired := previous
	desired.DisplayName = strings.TrimSpace(updated.DisplayName)
	desired.Description = strings.TrimSpace(updated.Description)
	desired.HomepageURL = strings.TrimSpace(updated.HomepageURL)
	if err := s.provisioner.EnsureApplication(ctx, desired); err != nil {
		return casdoorApplicationSpecRollback{}, fmt.Errorf("sync Casdoor application profile before local update: %w", err)
	}
	return casdoorApplicationSpecRollback{spec: previous, ok: true}, nil
}

func casdoorApplicationProfileNeedsSync(current *App, updated *App) bool {
	if current.Status != AppStatusApproved && current.Status != AppStatusSuspended {
		return false
	}
	return strings.TrimSpace(current.DisplayName) != strings.TrimSpace(updated.DisplayName) ||
		strings.TrimSpace(current.Description) != strings.TrimSpace(updated.Description) ||
		strings.TrimSpace(current.HomepageURL) != strings.TrimSpace(updated.HomepageURL)
}

func (s *Service) rollbackCasdoorApplicationProfileUpdate(
	ctx context.Context,
	rollback casdoorApplicationSpecRollback,
) error {
	return s.rollbackCasdoorApplicationSpec(ctx, rollback, "rollback Casdoor application profile after local update failure")
}

func (s *Service) ListAuditEvents(ctx context.Context, input ListAuditEventsInput) (AuditEventListResult, error) {
	if input.AppID < 0 || input.UserID < 0 {
		return AuditEventListResult{}, ErrInvalidAuditFilter
	}
	eventType := strings.TrimSpace(input.EventType)
	scope, err := normalizeAuditScopeFilter(input.Scope)
	if err != nil {
		return AuditEventListResult{}, err
	}
	pageSize := httputil.ClampPageSize(input.PageSize)
	return s.repo.ListAuditEvents(ctx, auditEventListFilter{
		AppID:     input.AppID,
		UserID:    input.UserID,
		EventType: eventType,
		Scope:     scope,
		Limit:     pageSize,
		Offset:    httputil.SafeOffset(input.Page, pageSize),
	})
}

func (s *Service) ListUserConsentAuditEvents(
	ctx context.Context,
	input ListUserConsentAuditEventsInput,
) (UserConsentAuditEventListResult, error) {
	if input.UserID <= 0 || input.AppID < 0 {
		return UserConsentAuditEventListResult{}, ErrInvalidAuditFilter
	}
	eventType, err := normalizeUserConsentAuditEventType(input.EventType)
	if err != nil {
		return UserConsentAuditEventListResult{}, err
	}
	scope, err := normalizeAuditScopeFilter(input.Scope)
	if err != nil {
		return UserConsentAuditEventListResult{}, err
	}
	pageSize := httputil.ClampPageSize(input.PageSize)
	return s.repo.ListUserConsentAuditEvents(ctx, userConsentAuditEventListFilter{
		UserID:    input.UserID,
		AppID:     input.AppID,
		EventType: eventType,
		Scope:     scope,
		Limit:     pageSize,
		Offset:    httputil.SafeOffset(input.Page, pageSize),
	})
}

func (s *Service) ListDeveloperAppAuditEvents(
	ctx context.Context,
	input ListDeveloperAppAuditEventsInput,
) (DeveloperAppAuditEventListResult, error) {
	if input.OwnerUserID <= 0 || input.AppID <= 0 {
		return DeveloperAppAuditEventListResult{}, ErrAppNotFound
	}
	app, err := s.repo.GetAppByID(ctx, input.AppID)
	if err != nil {
		return DeveloperAppAuditEventListResult{}, err
	}
	if app.OwnerUserID != input.OwnerUserID {
		return DeveloperAppAuditEventListResult{}, ErrAppNotFound
	}
	eventType, err := normalizeDeveloperAppAuditEventType(input.EventType)
	if err != nil {
		return DeveloperAppAuditEventListResult{}, err
	}
	scope, err := normalizeAuditScopeFilter(input.Scope)
	if err != nil {
		return DeveloperAppAuditEventListResult{}, err
	}
	pageSize := httputil.ClampPageSize(input.PageSize)
	return s.repo.ListDeveloperAppAuditEvents(ctx, developerAppAuditEventListFilter{
		AppID:     input.AppID,
		EventType: eventType,
		Scope:     scope,
		Limit:     pageSize,
		Offset:    httputil.SafeOffset(input.Page, pageSize),
	})
}

func (s *Service) ListTokenProbeEvidence(
	ctx context.Context,
	input ListTokenProbeEvidenceInput,
) (TokenProbeEvidenceListResult, error) {
	if input.AppID < 0 || input.ReviewerUserID < 0 {
		return TokenProbeEvidenceListResult{}, ErrInvalidTokenProbeFilter
	}
	result := strings.TrimSpace(input.Result)
	if result != "" && result != "passed" && result != "failed" {
		return TokenProbeEvidenceListResult{}, ErrInvalidTokenProbeFilter
	}
	pageSize := httputil.ClampPageSize(input.PageSize)
	return s.repo.ListTokenProbeEvidence(ctx, tokenProbeEvidenceListFilter{
		AppID:          input.AppID,
		ReviewerUserID: input.ReviewerUserID,
		Result:         result,
		ClientID:       strings.TrimSpace(input.ClientID),
		Limit:          pageSize,
		Offset:         httputil.SafeOffset(input.Page, pageSize),
	})
}

func (s *Service) DisclosureReport(ctx context.Context, input DisclosureReportInput) (DisclosureReport, error) {
	windowHours := normalizeDisclosureReportWindow(input.WindowHours)
	return s.repo.DisclosureReport(ctx, windowHours)
}

func (s *Service) scopeDefinitionsForApp(ctx context.Context, appID int64, scopes []string) ([]ScopeDefinition, error) {
	reasons, err := s.repo.ListScopeReasons(ctx, appID, scopes)
	if err != nil {
		return nil, err
	}
	return ScopeDefinitionsWithReasons(scopes, reasons), nil
}

func (s *Service) ListUserConsents(ctx context.Context, userID int64) ([]UserAuthorizedApp, error) {
	if userID <= 0 {
		return nil, ErrDisclosureUnavailable
	}
	return s.repo.ListUserConsents(ctx, userID)
}

func (s *Service) ListAdminUserConsents(ctx context.Context, input ListAdminUserConsentsInput) (AdminUserConsentListResult, error) {
	if input.AppID < 0 || input.UserID < 0 {
		return AdminUserConsentListResult{}, ErrInvalidAuditFilter
	}
	if input.AppID == 0 && input.UserID == 0 {
		return AdminUserConsentListResult{}, ErrInvalidAuditFilter
	}
	pageSize := httputil.ClampPageSize(input.PageSize)
	return s.repo.ListAdminUserConsents(ctx, adminUserConsentListFilter{
		AppID:  input.AppID,
		UserID: input.UserID,
		Limit:  pageSize,
		Offset: httputil.SafeOffset(input.Page, pageSize),
	})
}

func (s *Service) RevokeUserConsent(ctx context.Context, input RevokeConsentInput) error {
	if input.UserID <= 0 {
		return ErrDisclosureUnavailable
	}
	if input.AppID <= 0 {
		return ErrAppNotFound
	}
	scopes, err := normalizeOptionalScopes(input.Scopes)
	if err != nil {
		return err
	}
	return s.repo.RevokeAppConsents(ctx, input.AppID, input.UserID, scopes, input.RequestID)
}

func (s *Service) RevokeAdminUserConsent(ctx context.Context, input AdminRevokeConsentInput) error {
	if input.ActorUserID <= 0 {
		return ErrDisclosureUnavailable
	}
	if input.UserID <= 0 {
		return ErrInvalidAuditFilter
	}
	if input.AppID <= 0 {
		return ErrAppNotFound
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return ErrLifecycleReasonRequired
	}
	scopes, err := normalizeOptionalScopes(input.Scopes)
	if err != nil {
		return err
	}
	return s.repo.RevokeAppConsentsWithAuditMetadata(ctx, input.AppID, input.UserID, scopes, input.RequestID, map[string]any{
		"actor":       "admin",
		"actorUserID": input.ActorUserID,
		"reason":      reason,
		"source":      "admin_console",
	})
}

func (s *Service) VerifyClientSecret(ctx context.Context, clientID, clientSecret string) (*App, error) {
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	if clientID == "" || clientSecret == "" {
		return nil, ErrAppNotFound
	}
	app, err := s.repo.VerifyClientSecret(ctx, clientID, clientSecret)
	if err != nil {
		return nil, err
	}
	if app.Status != AppStatusApproved {
		return nil, ErrAppNotActive
	}
	return app, nil
}

func (s *Service) AppByID(ctx context.Context, appID int64) (*App, error) {
	if appID <= 0 {
		return nil, ErrAppNotFound
	}
	return s.repo.GetAppByID(ctx, appID)
}

func (s *Service) UserProjection(ctx context.Context, userID int64) (*UserProjection, error) {
	if userID <= 0 {
		return nil, ErrDisclosureUnavailable
	}
	return s.repo.GetUserProjection(ctx, userID)
}

func buildNewApp(input RegisterAppInput) (*App, string, error) {
	name, err := randomHex("third-party-", clientIDBytes)
	if err != nil {
		return nil, "", err
	}
	clientID, err := randomHex("op_", clientIDBytes)
	if err != nil {
		return nil, "", err
	}
	app := &App{
		CasdoorApplicationName: name,
		OwnerUserID:            input.OwnerUserID,
		ClientID:               clientID,
		ClientSecretHash:       "",
		DisplayName:            strings.TrimSpace(input.DisplayName),
		Description:            strings.TrimSpace(input.Description),
		HomepageURL:            strings.TrimSpace(input.HomepageURL),
		PrivacyPolicyURL:       strings.TrimSpace(input.PrivacyPolicyURL),
		RedirectURIs:           normalizeRedirects(input.RedirectURIs),
		Status:                 AppStatusPending,
	}
	if err := validateAppInput(app); err != nil {
		return nil, "", err
	}
	return app, "", nil
}

func hashClientSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func normalizeScopeRequests(inputs []ScopeRequestInput) ([]ScopeRequestInput, error) {
	raw := make([]string, 0, len(inputs))
	reasons := make(map[string]string, len(inputs))
	for _, input := range inputs {
		scope := strings.TrimSpace(input.Scope)
		raw = append(raw, scope)
		reason := strings.TrimSpace(input.Reason)
		if existing, ok := reasons[scope]; ok && (existing != "" || reason == "") {
			continue
		}
		reasons[scope] = reason
	}
	scopes, err := NormalizeScopes(raw)
	if err != nil {
		return nil, err
	}
	result := make([]ScopeRequestInput, 0, len(scopes))
	for _, scope := range scopes {
		if reasons[scope] == "" {
			return nil, ErrScopeReasonRequired
		}
		result = append(result, ScopeRequestInput{Scope: scope, Reason: reasons[scope]})
	}
	return result, nil
}

func normalizeScopeChangeRequests(inputs []ScopeRequestInput) ([]ScopeRequestInput, error) {
	scopes, err := normalizeScopeRequests(inputs)
	if err != nil {
		return nil, err
	}
	if len(scopes) == 0 {
		return nil, ErrInvalidScope
	}
	return scopes, nil
}

func normalizeOptionalScopes(scopes []string) ([]string, error) {
	if len(scopes) == 0 {
		return nil, nil
	}
	return NormalizeScopes(scopes)
}

func normalizeAuditScopeFilter(raw string) (string, error) {
	scope := strings.TrimSpace(raw)
	if scope == "" {
		return "", nil
	}
	return normalizeSingleScope(scope)
}

func normalizeUserConsentAuditEventType(raw string) (string, error) {
	eventType := strings.TrimSpace(raw)
	if eventType == "" {
		return "", nil
	}
	switch eventType {
	case "open_platform.consent.granted",
		"open_platform.consent.denied",
		"open_platform.consent.revoked",
		"open_platform.disclosure.granted",
		"open_platform.disclosure.denied",
		"open_platform.disclosure.replay_detected":
		return eventType, nil
	default:
		return "", ErrInvalidAuditFilter
	}
}

func normalizeDeveloperAppAuditEventType(raw string) (string, error) {
	eventType := strings.TrimSpace(raw)
	if eventType == "" {
		return "", nil
	}
	for _, allowed := range developerAppAuditEventTypes {
		if eventType == allowed {
			return eventType, nil
		}
	}
	return "", ErrInvalidAuditFilter
}

func normalizeDisclosureReportWindow(windowHours int) int {
	if windowHours <= 0 {
		return 24
	}
	if windowHours > 168 {
		return 168
	}
	return windowHours
}

func normalizeAppStatusFilter(raw string) (string, error) {
	status := strings.TrimSpace(raw)
	if status == "" || status == "all" {
		return "", nil
	}
	switch status {
	case AppStatusPending, AppStatusApproved, AppStatusSuspended, AppStatusRevoked:
		return status, nil
	default:
		return "", ErrInvalidAppStatus
	}
}

func validateAppInput(app *App) error {
	if app.OwnerUserID <= 0 || app.DisplayName == "" {
		return fmt.Errorf("%w: owner and display name are required", ErrInvalidAppProfile)
	}
	if err := validateHTTPSURL("homepage URL", app.HomepageURL); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidAppProfile, err)
	}
	if err := validateHTTPSURL("privacy policy URL", app.PrivacyPolicyURL); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidAppProfile, err)
	}
	redirects, err := normalizeAndValidateRedirectURIs(app.RedirectURIs)
	if err != nil {
		return err
	}
	app.RedirectURIs = redirects
	return nil
}

func validateHTTPSURL(label, value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.Fragment != "" {
		return fmt.Errorf("open platform %s must be an https URL without fragment", label)
	}
	return nil
}

func validateRedirectURI(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.Fragment != "" {
		return ErrRedirectURINotAllowed
	}
	if strings.Contains(value, "*") {
		return ErrRedirectURINotAllowed
	}
	return nil
}

func normalizeRedirects(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func normalizeAndValidateRedirectURIs(values []string) ([]string, error) {
	redirects := normalizeRedirects(values)
	if len(redirects) == 0 {
		return nil, ErrRedirectURINotAllowed
	}
	for _, redirect := range redirects {
		if err := validateRedirectURI(redirect); err != nil {
			return nil, err
		}
	}
	return redirects, nil
}

func randomHex(prefix string, size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("open platform random: %w", err)
	}
	return prefix + hex.EncodeToString(bytes), nil
}
