package openplatform

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/ctxutil"
)

const casdoorApplicationCleanupTimeout = 5 * time.Second

type ApproveScopeInput struct {
	AppID          int64
	Scope          string
	ReviewerUserID int64
	DecisionNote   string
	RequestID      string
}

type RejectScopeInput struct {
	AppID          int64
	Scope          string
	ReviewerUserID int64
	DecisionNote   string
	RequestID      string
}

type ScopeChangeInput struct {
	AppID       int64
	OwnerUserID int64
	Scopes      []ScopeRequestInput
	RequestID   string
}

type ScopeWithdrawalInput struct {
	AppID       int64
	OwnerUserID int64
	Scope       string
	Reason      string
	RequestID   string
}

type ApproveAppInput struct {
	AppID          int64
	ReviewerUserID int64
	RequestID      string
}

type ApprovedApp struct {
	App          *App
	ClientSecret string
}

type RotateAppSecretInput struct {
	AppID       int64
	ActorUserID int64
	OwnerUserID int64
	ActorType   string
	Reason      string
	RequestID   string
}

type RotatedAppSecret struct {
	App          *App
	ClientSecret string
}

type AppLifecycleActionInput struct {
	AppID       int64
	ActorUserID int64
	Reason      string
	RequestID   string
}

type AppWithdrawalInput struct {
	AppID       int64
	OwnerUserID int64
	Reason      string
	RequestID   string
}

type AppLifecycleResult struct {
	App *App
}

type RedirectURIChangeInput struct {
	AppID        int64
	OwnerUserID  int64
	RedirectURIs []string
	Reason       string
	RequestID    string
}

type RedirectURIReviewInput struct {
	AppID                int64
	RedirectURIRequestID int64
	ReviewerUserID       int64
	DecisionNote         string
	RequestID            string
}

type RedirectURIWithdrawalInput struct {
	AppID                int64
	RedirectURIRequestID int64
	OwnerUserID          int64
	Reason               string
	RequestID            string
}

type ImportCasdoorAppInput struct {
	OwnerUserID            int64
	ReviewerUserID         int64
	CasdoorApplicationName string
	DisplayName            string
	Description            string
	HomepageURL            string
	PrivacyPolicyURL       string
	RedirectURIs           []string
	ClientSecret           string
	Scopes                 []ScopeRequestInput
	RequestID              string
}

type ImportedApp struct {
	App                *App
	ClientSecret       string
	ClientSecretSource string
}

func (s *Service) ApproveScope(ctx context.Context, input ApproveScopeInput) error {
	scope, err := normalizeSingleScope(input.Scope)
	if err != nil {
		return err
	}
	if input.ReviewerUserID <= 0 {
		return fmt.Errorf("open platform reviewer user id is required")
	}
	app, err := s.repo.GetAppByID(ctx, input.AppID)
	if err != nil {
		return err
	}
	if app.Status == AppStatusRevoked {
		return ErrInvalidAppStatus
	}
	if err := s.repo.ApproveScope(ctx, input.AppID, scope, input.ReviewerUserID, input.DecisionNote, input.RequestID); err != nil {
		return err
	}
	return nil
}

func (s *Service) RejectScope(ctx context.Context, input RejectScopeInput) error {
	scope, err := normalizeSingleScope(input.Scope)
	if err != nil {
		return err
	}
	if input.ReviewerUserID <= 0 {
		return fmt.Errorf("open platform reviewer user id is required")
	}
	app, err := s.repo.GetAppByID(ctx, input.AppID)
	if err != nil {
		return err
	}
	if app.Status == AppStatusRevoked {
		return ErrInvalidAppStatus
	}
	return s.repo.RejectScope(ctx, input.AppID, scope, input.ReviewerUserID, input.DecisionNote, input.RequestID)
}

func (s *Service) RequestScopeChange(ctx context.Context, input ScopeChangeInput) (ScopeChangeResult, error) {
	if input.AppID <= 0 {
		return ScopeChangeResult{}, ErrAppNotFound
	}
	if input.OwnerUserID <= 0 {
		return ScopeChangeResult{}, ErrDisclosureUnavailable
	}
	scopes, err := normalizeScopeChangeRequests(input.Scopes)
	if err != nil {
		return ScopeChangeResult{}, err
	}
	app, err := s.repo.GetAppByID(ctx, input.AppID)
	if err != nil {
		return ScopeChangeResult{}, err
	}
	if app.OwnerUserID != input.OwnerUserID {
		return ScopeChangeResult{}, ErrAppNotFound
	}
	if app.Status == AppStatusRevoked {
		return ScopeChangeResult{}, ErrInvalidAppStatus
	}
	requests := make([]ScopeRequest, 0, len(scopes))
	for _, scope := range scopes {
		requests = append(requests, ScopeRequest{
			AppID:  input.AppID,
			Scope:  scope.Scope,
			Reason: scope.Reason,
			Status: ScopeStatusPending,
		})
	}
	return s.repo.UpsertScopeRequestsWithAudit(ctx, input.AppID, requests, input.OwnerUserID, input.RequestID)
}

func (s *Service) WithdrawScopeRequest(ctx context.Context, input ScopeWithdrawalInput) (ScopeRequest, error) {
	if input.AppID <= 0 {
		return ScopeRequest{}, ErrAppNotFound
	}
	if input.OwnerUserID <= 0 {
		return ScopeRequest{}, ErrDisclosureUnavailable
	}
	scope, err := normalizeSingleScope(input.Scope)
	if err != nil {
		return ScopeRequest{}, err
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return ScopeRequest{}, ErrLifecycleReasonRequired
	}
	app, err := s.repo.GetAppByID(ctx, input.AppID)
	if err != nil {
		return ScopeRequest{}, err
	}
	if app.OwnerUserID != input.OwnerUserID {
		return ScopeRequest{}, ErrAppNotFound
	}
	if app.Status == AppStatusRevoked {
		return ScopeRequest{}, ErrInvalidAppStatus
	}
	return s.repo.WithdrawScopeRequestWithAudit(ctx, input.AppID, scope, input.OwnerUserID, reason, input.RequestID)
}

func (s *Service) ApproveApp(ctx context.Context, appID int64) (*ApprovedApp, error) {
	return s.ApproveAppWithAudit(ctx, ApproveAppInput{AppID: appID})
}

func (s *Service) ApproveAppWithAudit(ctx context.Context, input ApproveAppInput) (*ApprovedApp, error) {
	appID := input.AppID
	app, err := s.repo.GetAppByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if app.Status != AppStatusPending {
		return nil, ErrInvalidAppStatus
	}
	scopes, err := s.repo.ListApprovedScopes(ctx, app.ID)
	if err != nil {
		return nil, err
	}
	if len(scopes) == 0 {
		return nil, ErrScopeNotApproved
	}
	secret, err := randomHex("ids_", clientSecretBytes)
	if err != nil {
		return nil, err
	}
	spec := casdoorApplicationSpecForApprovedApp(app, secret)
	if err := s.ensureCasdoorApplicationReadyForApproval(ctx, app, spec, input.ReviewerUserID, input.RequestID); err != nil {
		return nil, err
	}
	if err := s.repo.MarkAppApproved(ctx, appID, hashClientSecret(secret), input.ReviewerUserID, input.RequestID); err != nil {
		if cleanupErr := s.deleteProvisionedCasdoorApplication(ctx, spec.Name); cleanupErr != nil {
			return nil, errors.Join(err, cleanupErr)
		}
		return nil, err
	}
	app.Status = AppStatusApproved
	return &ApprovedApp{App: app, ClientSecret: secret}, nil
}

func (s *Service) RotateAppSecret(ctx context.Context, input RotateAppSecretInput) (*RotatedAppSecret, error) {
	if input.AppID <= 0 {
		return nil, ErrAppNotFound
	}
	if input.ActorUserID <= 0 {
		return nil, ErrDisclosureUnavailable
	}
	app, err := s.repo.GetAppByID(ctx, input.AppID)
	if err != nil {
		return nil, err
	}
	if input.OwnerUserID > 0 {
		if app.OwnerUserID != input.OwnerUserID {
			return nil, ErrAppNotFound
		}
		if app.Status != AppStatusApproved {
			return nil, ErrAppNotActive
		}
	} else if app.Status != AppStatusApproved && app.Status != AppStatusSuspended {
		return nil, ErrAppNotActive
	}
	secret, err := randomHex("ids_", clientSecretBytes)
	if err != nil {
		return nil, err
	}
	actorType := strings.TrimSpace(input.ActorType)
	if actorType == "" {
		actorType = "admin"
	}
	updated, err := s.repo.RotateAppSecret(
		ctx,
		input.AppID,
		hashClientSecret(secret),
		input.ActorUserID,
		actorType,
		strings.TrimSpace(input.Reason),
		input.RequestID,
	)
	if err != nil {
		return nil, err
	}
	return &RotatedAppSecret{App: updated, ClientSecret: secret}, nil
}

func (s *Service) RequestRedirectURIChange(ctx context.Context, input RedirectURIChangeInput) (RedirectURIRequest, error) {
	if input.AppID <= 0 {
		return RedirectURIRequest{}, ErrAppNotFound
	}
	if input.OwnerUserID <= 0 {
		return RedirectURIRequest{}, ErrDisclosureUnavailable
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return RedirectURIRequest{}, ErrRedirectURIReasonRequired
	}
	redirects, err := normalizeAndValidateRedirectURIs(input.RedirectURIs)
	if err != nil {
		return RedirectURIRequest{}, err
	}
	app, err := s.repo.GetAppByID(ctx, input.AppID)
	if err != nil {
		return RedirectURIRequest{}, err
	}
	if app.OwnerUserID != input.OwnerUserID {
		return RedirectURIRequest{}, ErrAppNotFound
	}
	if app.Status != AppStatusApproved && app.Status != AppStatusSuspended {
		return RedirectURIRequest{}, ErrInvalidAppStatus
	}
	return s.repo.UpsertRedirectURIRequestWithAudit(ctx, RedirectURIRequest{
		AppID:        input.AppID,
		RedirectURIs: redirects,
		Reason:       reason,
		Status:       ScopeStatusPending,
	}, input.OwnerUserID, input.RequestID)
}

func (s *Service) ApproveRedirectURIRequest(ctx context.Context, input RedirectURIReviewInput) (RedirectURIRequest, error) {
	return s.reviewRedirectURIRequest(
		ctx,
		input,
		ScopeStatusApproved,
		"open_platform.app.redirect_uris.approved",
	)
}

func (s *Service) RejectRedirectURIRequest(ctx context.Context, input RedirectURIReviewInput) (RedirectURIRequest, error) {
	return s.reviewRedirectURIRequest(
		ctx,
		input,
		ScopeStatusRejected,
		"open_platform.app.redirect_uris.rejected",
	)
}

func (s *Service) WithdrawRedirectURIRequest(
	ctx context.Context,
	input RedirectURIWithdrawalInput,
) (RedirectURIRequest, error) {
	if input.AppID <= 0 {
		return RedirectURIRequest{}, ErrAppNotFound
	}
	if input.RedirectURIRequestID <= 0 {
		return RedirectURIRequest{}, ErrRedirectURIRequestNotFound
	}
	if input.OwnerUserID <= 0 {
		return RedirectURIRequest{}, ErrDisclosureUnavailable
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return RedirectURIRequest{}, ErrLifecycleReasonRequired
	}
	app, err := s.repo.GetAppByID(ctx, input.AppID)
	if err != nil {
		return RedirectURIRequest{}, err
	}
	if app.OwnerUserID != input.OwnerUserID {
		return RedirectURIRequest{}, ErrAppNotFound
	}
	if app.Status != AppStatusApproved && app.Status != AppStatusSuspended {
		return RedirectURIRequest{}, ErrInvalidAppStatus
	}
	return s.repo.WithdrawRedirectURIRequestWithAudit(
		ctx,
		input.AppID,
		input.RedirectURIRequestID,
		input.OwnerUserID,
		reason,
		input.RequestID,
	)
}

func (s *Service) reviewRedirectURIRequest(
	ctx context.Context,
	input RedirectURIReviewInput,
	status string,
	eventType string,
) (RedirectURIRequest, error) {
	if input.AppID <= 0 {
		return RedirectURIRequest{}, ErrAppNotFound
	}
	if input.RedirectURIRequestID <= 0 {
		return RedirectURIRequest{}, ErrRedirectURIRequestNotFound
	}
	if input.ReviewerUserID <= 0 {
		return RedirectURIRequest{}, ErrDisclosureUnavailable
	}
	app, err := s.repo.GetAppByID(ctx, input.AppID)
	if err != nil {
		return RedirectURIRequest{}, err
	}
	if app.Status != AppStatusApproved && app.Status != AppStatusSuspended {
		return RedirectURIRequest{}, ErrInvalidAppStatus
	}
	return s.repo.ReviewRedirectURIRequestWithAudit(
		ctx,
		input.AppID,
		input.RedirectURIRequestID,
		input.ReviewerUserID,
		status,
		strings.TrimSpace(input.DecisionNote),
		eventType,
		input.RequestID,
	)
}

func (s *Service) SuspendApp(ctx context.Context, input AppLifecycleActionInput) (*AppLifecycleResult, error) {
	return s.updateAppLifecycleStatus(ctx, input, AppStatusSuspended, "open_platform.app.suspended", map[string]struct{}{
		AppStatusApproved: {},
	})
}

func (s *Service) ResumeApp(ctx context.Context, input AppLifecycleActionInput) (*AppLifecycleResult, error) {
	return s.updateAppLifecycleStatus(ctx, input, AppStatusApproved, "open_platform.app.resumed", map[string]struct{}{
		AppStatusSuspended: {},
	})
}

func (s *Service) RevokeApp(ctx context.Context, input AppLifecycleActionInput) (*AppLifecycleResult, error) {
	return s.updateAppLifecycleStatus(ctx, input, AppStatusRevoked, "open_platform.app.revoked", map[string]struct{}{
		AppStatusPending:   {},
		AppStatusApproved:  {},
		AppStatusSuspended: {},
	})
}

func (s *Service) WithdrawApp(ctx context.Context, input AppWithdrawalInput) (*AppLifecycleResult, error) {
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
	if app.Status != AppStatusPending {
		return nil, ErrInvalidAppStatus
	}
	updated, err := s.repo.UpdateAppStatusWithAudit(
		ctx,
		input.AppID,
		AppStatusRevoked,
		[]string{AppStatusPending},
		input.OwnerUserID,
		"open_platform.app.withdrawn",
		reason,
		input.RequestID,
	)
	if err != nil {
		return nil, err
	}
	return &AppLifecycleResult{App: updated}, nil
}

func (s *Service) updateAppLifecycleStatus(
	ctx context.Context,
	input AppLifecycleActionInput,
	status string,
	eventType string,
	allowedFrom map[string]struct{},
) (*AppLifecycleResult, error) {
	if input.AppID <= 0 {
		return nil, ErrAppNotFound
	}
	if input.ActorUserID <= 0 {
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
	if status == AppStatusRevoked && app.Status == AppStatusRevoked {
		cleaned, err := s.revokeAllResourceAccessForRevokedApp(ctx, input.AppID, input.ActorUserID, input.RequestID, reason)
		casdoorCleaned, casdoorErr := s.deleteCasdoorApplicationForRevokedApp(ctx, app.CasdoorApplicationName)
		if err != nil || casdoorErr != nil {
			return nil, errors.Join(err, casdoorErr)
		}
		if cleaned == 0 && !casdoorCleaned {
			return nil, ErrInvalidAppStatus
		}
		return &AppLifecycleResult{App: app}, nil
	}
	if _, ok := allowedFrom[app.Status]; !ok {
		return nil, ErrInvalidAppStatus
	}
	updated, err := s.repo.UpdateAppStatusWithAudit(
		ctx,
		input.AppID,
		status,
		appStatusSetToSlice(allowedFrom),
		input.ActorUserID,
		eventType,
		reason,
		input.RequestID,
	)
	if err != nil {
		return nil, err
	}
	if status == AppStatusRevoked {
		var revokeErr error
		if _, err := s.revokeAllResourceAccessForRevokedApp(ctx, input.AppID, input.ActorUserID, input.RequestID, reason); err != nil {
			revokeErr = err
		}
		if shouldDeleteCasdoorApplicationAfterRevoke(app.Status) {
			if _, err := s.deleteCasdoorApplicationForRevokedApp(ctx, app.CasdoorApplicationName); err != nil {
				revokeErr = errors.Join(revokeErr, err)
			}
		}
		if revokeErr != nil {
			return nil, revokeErr
		}
	}
	return &AppLifecycleResult{App: updated}, nil
}

func shouldDeleteCasdoorApplicationAfterRevoke(previousStatus string) bool {
	return previousStatus == AppStatusApproved || previousStatus == AppStatusSuspended || previousStatus == AppStatusRevoked
}

func (s *Service) deleteCasdoorApplicationForRevokedApp(ctx context.Context, name string) (bool, error) {
	if s.provisioner == nil {
		return false, nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return false, nil
	}
	cleanupCtx, cancel := casdoorApplicationCleanupContext(ctx)
	defer cancel()
	if err := s.provisioner.DeleteApplication(cleanupCtx, name); err != nil {
		return true, fmt.Errorf("delete Casdoor application after open platform app revoke: %w", err)
	}
	return true, nil
}

func appStatusSetToSlice(statuses map[string]struct{}) []string {
	values := make([]string, 0, len(statuses))
	for status := range statuses {
		values = append(values, status)
	}
	return values
}

func (s *Service) ImportCasdoorApp(ctx context.Context, input ImportCasdoorAppInput) (*ImportedApp, error) {
	if s.provisioner == nil {
		return nil, fmt.Errorf("open platform Casdoor application reader is not configured")
	}
	if input.OwnerUserID <= 0 || input.ReviewerUserID <= 0 {
		return nil, fmt.Errorf("open platform importer and owner user id are required")
	}
	scopes, err := normalizeScopeRequests(input.Scopes)
	if err != nil {
		return nil, err
	}
	spec, err := s.provisioner.GetApplication(ctx, strings.TrimSpace(input.CasdoorApplicationName))
	if err != nil {
		return nil, fmt.Errorf("ImportCasdoorApp read Casdoor application: %w", err)
	}
	if err := s.ensureCasdoorTokenMinimized(ctx, nil, spec, input.ReviewerUserID, input.RequestID); err != nil {
		return nil, err
	}
	secret, source, err := importedClientSecret(input.ClientSecret, spec.ClientSecret)
	if err != nil {
		return nil, err
	}
	app := &App{
		CasdoorApplicationName: strings.TrimSpace(spec.Name),
		OwnerUserID:            input.OwnerUserID,
		ClientID:               strings.TrimSpace(spec.ClientID),
		ClientSecretHash:       hashClientSecret(secret),
		DisplayName:            firstNonBlank(input.DisplayName, spec.DisplayName, spec.Name),
		Description:            firstNonBlank(input.Description, spec.Description),
		HomepageURL:            firstNonBlank(input.HomepageURL, spec.HomepageURL),
		PrivacyPolicyURL:       strings.TrimSpace(input.PrivacyPolicyURL),
		RedirectURIs:           importedRedirectURIs(input.RedirectURIs, spec.RedirectURIs),
		Status:                 AppStatusApproved,
	}
	if err := validateAppInput(app); err != nil {
		return nil, err
	}
	requests := make([]ScopeRequest, 0, len(scopes))
	for _, scope := range scopes {
		requests = append(requests, ScopeRequest{
			Scope:          scope.Scope,
			Reason:         scope.Reason,
			Status:         ScopeStatusApproved,
			ReviewerUserID: &input.ReviewerUserID,
		})
	}
	if err := s.repo.ImportApprovedApp(ctx, app, requests, input.ReviewerUserID); err != nil {
		return nil, err
	}
	return &ImportedApp{
		App:                app,
		ClientSecret:       secretForImportResponse(secret, source),
		ClientSecretSource: source,
	}, nil
}

func (s *Service) ensureCasdoorApplicationReadyForApproval(
	ctx context.Context,
	app *App,
	spec ProvisionedApplicationSpec,
	reviewerUserID int64,
	requestID string,
) error {
	if s.provisioner == nil {
		if !s.tokenProbeRequired {
			return nil
		}
		err := fmt.Errorf("%w: Casdoor application provisioning is not configured", ErrTokenMinimizationProbe)
		if auditErr := s.recordRuntimeTokenProbeResult(
			ctx,
			app,
			spec,
			reviewerUserID,
			requestID,
			"failed",
			RuntimeTokenMinimizationProbeResult{},
			err,
		); auditErr != nil {
			return auditErr
		}
		return err
	}
	if err := s.provisioner.EnsureApplication(ctx, spec); err != nil {
		return fmt.Errorf("ensure Casdoor application before open platform approval: %w", err)
	}
	if err := s.ensureCasdoorTokenMinimized(ctx, app, spec, reviewerUserID, requestID); err != nil {
		if cleanupErr := s.deleteProvisionedCasdoorApplication(ctx, spec.Name); cleanupErr != nil {
			return errors.Join(err, cleanupErr)
		}
		return err
	}
	if err := s.ensureCasdoorRuntimeTokenMinimized(ctx, app, spec, reviewerUserID, requestID); err != nil {
		if cleanupErr := s.deleteProvisionedCasdoorApplication(ctx, spec.Name); cleanupErr != nil {
			return errors.Join(err, cleanupErr)
		}
		return err
	}
	return nil
}

func (s *Service) deleteProvisionedCasdoorApplication(ctx context.Context, name string) error {
	if s.provisioner == nil {
		return nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	cleanupCtx, cancel := casdoorApplicationCleanupContext(ctx)
	defer cancel()
	if err := s.provisioner.DeleteApplication(cleanupCtx, name); err != nil {
		return fmt.Errorf("rollback Casdoor application after open platform approval failure: %w", err)
	}
	return nil
}

func casdoorApplicationCleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctxutil.DetachedTimeout(ctx, casdoorApplicationCleanupTimeout)
}

func casdoorApplicationSpecForApprovedApp(app *App, clientSecret string) ProvisionedApplicationSpec {
	return ProvisionedApplicationSpec{
		Name:                 strings.TrimSpace(app.CasdoorApplicationName),
		DisplayName:          strings.TrimSpace(app.DisplayName),
		HomepageURL:          strings.TrimSpace(app.HomepageURL),
		Description:          strings.TrimSpace(app.Description),
		ClientID:             strings.TrimSpace(app.ClientID),
		ClientSecret:         strings.TrimSpace(clientSecret),
		RedirectURIs:         append([]string(nil), app.RedirectURIs...),
		GrantTypes:           []string{"authorization_code"},
		TokenFormat:          "JWT-Custom",
		TokenFields:          []string{},
		ExpireInHours:        1,
		RefreshExpireInHours: 0,
	}
}

func (s *Service) ensureCasdoorTokenMinimized(
	ctx context.Context,
	app *App,
	spec ProvisionedApplicationSpec,
	reviewerUserID int64,
	requestID string,
) error {
	result, err := probeApplicationSpecTokenMinimization(spec)
	if err == nil {
		return s.recordTokenProbeAudit(ctx, app, spec, reviewerUserID, requestID, "passed", result, nil)
	}
	if auditErr := s.recordTokenProbeAudit(ctx, app, spec, reviewerUserID, requestID, "failed", result, err); auditErr != nil {
		return auditErr
	}
	return fmt.Errorf("%w: %v", ErrTokenMinimizationProbe, err)
}

func (s *Service) ensureCasdoorRuntimeTokenMinimized(
	ctx context.Context,
	app *App,
	spec ProvisionedApplicationSpec,
	reviewerUserID int64,
	requestID string,
) error {
	if s.tokenProber == nil {
		if !s.tokenProbeRequired {
			return nil
		}
		err := fmt.Errorf("%w: runtime code-flow probe runner is not configured", ErrTokenMinimizationProbe)
		if auditErr := s.recordRuntimeTokenProbeResult(
			ctx,
			app,
			spec,
			reviewerUserID,
			requestID,
			"failed",
			RuntimeTokenMinimizationProbeResult{},
			err,
		); auditErr != nil {
			return auditErr
		}
		return err
	}
	result, err := s.tokenProber.ProbeTokenMinimization(ctx, spec)
	if err == nil {
		return s.recordRuntimeTokenProbeResult(ctx, app, spec, reviewerUserID, requestID, "passed", result, nil)
	}
	if auditErr := s.recordRuntimeTokenProbeResult(ctx, app, spec, reviewerUserID, requestID, "failed", result, err); auditErr != nil {
		return auditErr
	}
	return fmt.Errorf("%w: runtime code-flow probe failed: %v", ErrTokenMinimizationProbe, err)
}

func (s *Service) recordRuntimeTokenProbeResult(
	ctx context.Context,
	app *App,
	spec ProvisionedApplicationSpec,
	reviewerUserID int64,
	requestID string,
	result string,
	probeResult RuntimeTokenMinimizationProbeResult,
	cause error,
) error {
	appID := int64(0)
	if app != nil {
		appID = app.ID
	}
	redirectURI := ""
	if len(spec.RedirectURIs) > 0 {
		redirectURI = spec.RedirectURIs[0]
	}
	evidence := TokenProbeEvidence{
		AppID:                  appID,
		ReviewerUserID:         reviewerUserID,
		RequestID:              requestID,
		CasdoorApplicationName: strings.TrimSpace(spec.Name),
		ClientID:               strings.TrimSpace(spec.ClientID),
		RedirectURI:            redirectURI,
		ProbeMethod:            firstNonBlank(probeResult.Method, "authorization_code"),
		Result:                 result,
		InspectedClaims:        append([]string(nil), probeResult.InspectedClaims...),
		BusinessClaims:         append([]string(nil), probeResult.BusinessClaims...),
		TokenClaims:            copyTokenClaims(probeResult.TokenClaims),
		Metadata:               sanitizeTokenProbeMetadata(probeResult.Metadata),
	}
	if cause != nil {
		evidence.Error = cause.Error()
		evidence.Metadata["error"] = cause.Error()
	}
	if err := s.repo.RecordTokenProbeEvidence(ctx, evidence); err != nil {
		return fmt.Errorf("%w: runtime token probe evidence unavailable", ErrDisclosureUnavailable)
	}
	metadata := map[string]any{
		"casdoorApplicationName": evidence.CasdoorApplicationName,
		"clientID":               evidence.ClientID,
		"redirectURI":            evidence.RedirectURI,
		"result":                 result,
		"probeType":              "runtime_code_flow",
		"probeMethod":            evidence.ProbeMethod,
		"inspectedClaims":        evidence.InspectedClaims,
		"businessClaims":         evidence.BusinessClaims,
		"tokenClaims":            evidence.TokenClaims,
	}
	if cause != nil {
		metadata["error"] = cause.Error()
	}
	if err := s.repo.RecordAuditEvent(ctx, auditEvent{
		AppID:     appID,
		UserID:    reviewerUserID,
		EventType: "open_platform.app.token_probe.runtime." + result,
		RequestID: requestID,
		Metadata:  metadata,
	}); err != nil {
		return fmt.Errorf("%w: runtime token probe audit unavailable", ErrDisclosureUnavailable)
	}
	return nil
}

func (s *Service) recordTokenProbeAudit(
	ctx context.Context,
	app *App,
	spec ProvisionedApplicationSpec,
	reviewerUserID int64,
	requestID string,
	result string,
	probeResult TokenMinimizationProbeResult,
	cause error,
) error {
	appID := int64(0)
	if app != nil {
		appID = app.ID
	}
	metadata := map[string]any{
		"casdoorApplicationName": strings.TrimSpace(spec.Name),
		"clientID":               strings.TrimSpace(spec.ClientID),
		"result":                 result,
		"probeType":              "static_token_fields",
		"inspectedClaims":        append([]string(nil), probeResult.InspectedClaims...),
		"businessClaims":         append([]string(nil), probeResult.BusinessClaims...),
	}
	if cause != nil {
		metadata["error"] = cause.Error()
	}
	if err := s.repo.RecordAuditEvent(ctx, auditEvent{
		AppID:     appID,
		UserID:    reviewerUserID,
		EventType: "open_platform.app.token_probe." + result,
		RequestID: requestID,
		Metadata:  metadata,
	}); err != nil {
		return fmt.Errorf("%w: token minimization probe audit unavailable", ErrDisclosureUnavailable)
	}
	return nil
}

func copyTokenClaims(input map[string][]string) map[string][]string {
	output := make(map[string][]string, len(input))
	for key, value := range input {
		output[key] = append([]string(nil), value...)
	}
	return output
}

func sanitizeTokenProbeMetadata(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		if tokenProbeMetadataKeyAllowed(key) {
			output[key] = value
		}
	}
	return output
}

func tokenProbeMetadataKeyAllowed(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if normalized == "" {
		return false
	}
	for _, forbidden := range []string{"token", "secret", "password", "auth_code", "authorization_code", "code_verifier"} {
		if strings.Contains(normalized, forbidden) {
			return false
		}
	}
	return true
}

func normalizeSingleScope(raw string) (string, error) {
	scopes, err := NormalizeScopes([]string{raw})
	if err != nil {
		return "", err
	}
	return scopes[0], nil
}

func importedClientSecret(provided, casdoorSecret string) (string, string, error) {
	if secret := strings.TrimSpace(provided); secret != "" {
		return secret, "provided", nil
	}
	if secret := strings.TrimSpace(casdoorSecret); secret != "" {
		return secret, "casdoor", nil
	}
	secret, err := randomHex("ids_", clientSecretBytes)
	if err != nil {
		return "", "", err
	}
	return secret, "generated", nil
}

func importedRedirectURIs(overrides, casdoorRedirects []string) []string {
	if len(overrides) > 0 {
		return normalizeRedirects(overrides)
	}
	return normalizeRedirects(casdoorRedirects)
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func secretForImportResponse(secret, source string) string {
	if source == "provided" {
		return ""
	}
	return secret
}
