package openplatform

import (
	"context"
	"fmt"
	"strings"
)

const (
	defaultIdentityPublicSmokeClientID   = "identity-public-smoke"
	defaultIdentityPublicSmokeDisplay    = "Identity Public Smoke"
	defaultIdentityPublicSmokeRequestID  = "identity-public-smoke-bootstrap"
	defaultIdentityPublicSmokeScope      = ScopeResourceRead
	identityPublicSmokeCasdoorNamePrefix = "identity-public-smoke-"
)

type IdentityPublicSmokeClientBootstrapInput struct {
	OwnerUserID        int64
	ReviewerUserID     int64
	ClientID           string
	ClientSecret       string
	DisplayName        string
	Description        string
	HomepageURL        string
	PrivacyPolicyURL   string
	RedirectURI        string
	ClientScopes       []string
	RequestID          string
	AllowRevokedRepair bool
}

type IdentityPublicSmokeClientBootstrapResult struct {
	App          *App
	ClientSecret string
	ClientScopes []string
}

func BootstrapIdentityPublicSmokeClient(
	ctx context.Context,
	repo *Repository,
	input IdentityPublicSmokeClientBootstrapInput,
) (*IdentityPublicSmokeClientBootstrapResult, error) {
	if repo == nil {
		return nil, fmt.Errorf("open platform repository is required")
	}
	normalized, err := normalizeIdentityPublicSmokeBootstrapInput(input)
	if err != nil {
		return nil, err
	}
	app := &App{
		CasdoorApplicationName: identityPublicSmokeCasdoorName(normalized.ClientID),
		OwnerUserID:            normalized.OwnerUserID,
		ClientID:               normalized.ClientID,
		ClientSecretHash:       hashClientSecret(normalized.ClientSecret),
		DisplayName:            normalized.DisplayName,
		Description:            normalized.Description,
		HomepageURL:            normalized.HomepageURL,
		PrivacyPolicyURL:       normalized.PrivacyPolicyURL,
		RedirectURIs:           []string{normalized.RedirectURI},
		Status:                 AppStatusApproved,
	}
	if err := validateAppInput(app); err != nil {
		return nil, err
	}
	requests := make([]ScopeRequest, 0, len(normalized.ClientScopes))
	for _, scope := range normalized.ClientScopes {
		requests = append(requests, ScopeRequest{
			Scope:  scope,
			Reason: "Production public Identity smoke client_credentials probe",
			Status: ScopeStatusApproved,
		})
	}
	ensured, err := repo.EnsureApprovedApp(ctx, app, requests, EnsureApprovedAppOptions{
		ReviewerUserID:     normalized.ReviewerUserID,
		RequestID:          normalized.RequestID,
		AllowRevokedRepair: normalized.AllowRevokedRepair,
		AuditEventType:     "open_platform.app.identity_public_smoke_bootstrapped",
	})
	if err != nil {
		return nil, err
	}
	return &IdentityPublicSmokeClientBootstrapResult{
		App:          ensured,
		ClientSecret: normalized.ClientSecret,
		ClientScopes: normalized.ClientScopes,
	}, nil
}

func normalizeIdentityPublicSmokeBootstrapInput(input IdentityPublicSmokeClientBootstrapInput) (IdentityPublicSmokeClientBootstrapInput, error) {
	if input.OwnerUserID <= 0 {
		return input, fmt.Errorf("identity public smoke owner user id is required")
	}
	if input.ReviewerUserID <= 0 {
		return input, fmt.Errorf("identity public smoke reviewer user id is required")
	}
	input.ClientID = firstNonBlank(input.ClientID, defaultIdentityPublicSmokeClientID)
	input.DisplayName = firstNonBlank(input.DisplayName, defaultIdentityPublicSmokeDisplay)
	input.Description = firstNonBlank(input.Description, "Dedicated approved client for production public Identity smoke checks.")
	input.RequestID = firstNonBlank(input.RequestID, defaultIdentityPublicSmokeRequestID)
	input.HomepageURL = strings.TrimSpace(input.HomepageURL)
	input.PrivacyPolicyURL = strings.TrimSpace(input.PrivacyPolicyURL)
	input.RedirectURI = strings.TrimSpace(input.RedirectURI)
	if input.HomepageURL == "" {
		return input, fmt.Errorf("identity public smoke homepage URL is required")
	}
	if input.PrivacyPolicyURL == "" {
		return input, fmt.Errorf("identity public smoke privacy policy URL is required")
	}
	if input.RedirectURI == "" {
		return input, fmt.Errorf("identity public smoke redirect URI is required")
	}
	if strings.TrimSpace(input.ClientSecret) == "" {
		secret, err := randomHex("ids_", clientSecretBytes)
		if err != nil {
			return input, err
		}
		input.ClientSecret = secret
	} else {
		input.ClientSecret = strings.TrimSpace(input.ClientSecret)
	}
	rawScopes := input.ClientScopes
	if len(rawScopes) == 0 {
		rawScopes = []string{defaultIdentityPublicSmokeScope}
	}
	scopes, err := NormalizeScopes(rawScopes)
	if err != nil {
		return input, err
	}
	for _, scope := range scopes {
		if scope != ScopeResourceRead && scope != ScopeResourceWrite {
			return input, fmt.Errorf("%w: identity public smoke client credentials scope must be resource.read or resource.write", ErrInvalidScope)
		}
	}
	input.ClientScopes = scopes
	return input, nil
}

func identityPublicSmokeCasdoorName(clientID string) string {
	normalized := strings.NewReplacer(" ", "-", "_", "-", ".", "-").Replace(strings.ToLower(strings.TrimSpace(clientID)))
	if normalized == "" {
		normalized = defaultIdentityPublicSmokeClientID
	}
	return identityPublicSmokeCasdoorNamePrefix + normalized
}
