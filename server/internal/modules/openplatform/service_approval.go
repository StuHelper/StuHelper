package openplatform

import (
	"context"
	"fmt"
	"strings"
)

type ApproveScopeInput struct {
	AppID          int64
	Scope          string
	ReviewerUserID int64
	DecisionNote   string
}

type ApprovedApp struct {
	App          *App
	ClientSecret string
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
	if err := s.repo.ApproveScope(ctx, input.AppID, scope, input.ReviewerUserID, input.DecisionNote); err != nil {
		return err
	}
	return nil
}

func (s *Service) ApproveApp(ctx context.Context, appID int64) (*ApprovedApp, error) {
	app, err := s.repo.GetAppByID(ctx, appID)
	if err != nil {
		return nil, err
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
	if err := s.repo.MarkAppApproved(ctx, appID, hashClientSecret(secret)); err != nil {
		return nil, err
	}
	app.Status = AppStatusApproved
	return &ApprovedApp{App: app, ClientSecret: secret}, nil
}

func (s *Service) ImportCasdoorApp(ctx context.Context, input ImportCasdoorAppInput) (*ImportedApp, error) {
	if s.provisioner == nil {
		return nil, fmt.Errorf("open platform Casdoor application reader is not configured")
	}
	if input.OwnerUserID <= 0 || input.ReviewerUserID <= 0 {
		return nil, fmt.Errorf("open platform importer and owner user id are required")
	}
	spec, err := s.provisioner.GetApplication(ctx, strings.TrimSpace(input.CasdoorApplicationName))
	if err != nil {
		return nil, fmt.Errorf("ImportCasdoorApp read Casdoor application: %w", err)
	}
	scopes, err := normalizeScopeRequests(input.Scopes)
	if err != nil {
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
			Reason:         firstNonBlank(scope.Reason, "Imported from legacy Casdoor application"),
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
