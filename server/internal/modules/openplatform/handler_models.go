package openplatform

import (
	"strconv"
	"time"
)

type authorizeResponse struct {
	RedirectURL          string                   `json:"redirectURL,omitempty"`
	ConsentURL           string                   `json:"consentURL,omitempty"`
	ProfileCompletionURL string                   `json:"profileCompletionURL,omitempty"`
	Scopes               []ScopeDefinition        `json:"scopes,omitempty"`
	MissingFields        []ProfileCompletionField `json:"missingFields,omitempty"`
}

type consentAppResponse struct {
	ID               int64  `json:"id"`
	ClientID         string `json:"clientID"`
	DisplayName      string `json:"displayName"`
	Description      string `json:"description"`
	HomepageURL      string `json:"homepageURL"`
	PrivacyPolicyURL string `json:"privacyPolicyURL"`
}

type consentPageResponse struct {
	Token       string             `json:"token"`
	App         consentAppResponse `json:"app"`
	Scopes      []ScopeDefinition  `json:"scopes"`
	RedirectURI string             `json:"redirectURI"`
	ExpiresAt   time.Time          `json:"expiresAt"`
}

type profileCompletionPageResponse struct {
	Token         string                   `json:"token"`
	App           consentAppResponse       `json:"app"`
	Scopes        []ScopeDefinition        `json:"scopes"`
	MissingFields []ProfileCompletionField `json:"missingFields"`
	RedirectURI   string                   `json:"redirectURI"`
	ExpiresAt     time.Time                `json:"expiresAt"`
}

type consentDecisionRequest struct {
	Token string `json:"token" binding:"required"`
}

type redirectResponse struct {
	RedirectURL string `json:"redirectURL"`
}

type registerAppRequest struct {
	DisplayName      string              `json:"displayName" binding:"required"`
	Description      string              `json:"description"`
	HomepageURL      string              `json:"homepageURL" binding:"required"`
	PrivacyPolicyURL string              `json:"privacyPolicyURL" binding:"required"`
	RedirectURIs     []string            `json:"redirectURIs" binding:"required"`
	Scopes           []scopeRequestInput `json:"scopes" binding:"required"`
}

type scopeRequestInput struct {
	Scope  string `json:"scope" binding:"required"`
	Reason string `json:"reason"`
}

type appResponse struct {
	ID               int64     `json:"id"`
	ClientID         string    `json:"clientID"`
	DisplayName      string    `json:"displayName"`
	Description      string    `json:"description"`
	HomepageURL      string    `json:"homepageURL"`
	PrivacyPolicyURL string    `json:"privacyPolicyURL"`
	RedirectURIs     []string  `json:"redirectURIs"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type registeredAppResponse struct {
	App          appResponse `json:"app"`
	ClientSecret string      `json:"clientSecret,omitempty"`
}

type approveScopeRequest struct {
	DecisionNote string `json:"decisionNote"`
}

type importCasdoorAppRequest struct {
	CasdoorApplicationName string              `json:"casdoorApplicationName" binding:"required"`
	DisplayName            string              `json:"displayName"`
	Description            string              `json:"description"`
	HomepageURL            string              `json:"homepageURL"`
	PrivacyPolicyURL       string              `json:"privacyPolicyURL" binding:"required"`
	RedirectURIs           []string            `json:"redirectURIs"`
	ClientSecret           string              `json:"clientSecret"`
	Scopes                 []scopeRequestInput `json:"scopes" binding:"required"`
}

type approvedAppResponse struct {
	App          appResponse `json:"app"`
	ClientSecret string      `json:"clientSecret"`
}

type importedAppResponse struct {
	App                appResponse `json:"app"`
	ClientSecret       string      `json:"clientSecret,omitempty"`
	ClientSecretSource string      `json:"clientSecretSource"`
}

type messageResponse struct {
	Message string `json:"message"`
}

func consentPageToJSON(page *ConsentPage) consentPageResponse {
	return consentPageResponse{
		Token:       page.Token,
		App:         consentAppToJSON(page.App),
		Scopes:      page.Scopes,
		RedirectURI: page.RedirectURI,
		ExpiresAt:   page.ExpiresAt,
	}
}

func profileCompletionPageToJSON(page *ProfileCompletionPage) profileCompletionPageResponse {
	return profileCompletionPageResponse{
		Token:         page.Token,
		App:           consentAppToJSON(page.App),
		Scopes:        page.Scopes,
		MissingFields: page.MissingFields,
		RedirectURI:   page.RedirectURI,
		ExpiresAt:     page.ExpiresAt,
	}
}

func consentAppToJSON(app ConsentApp) consentAppResponse {
	return consentAppResponse(app)
}

func registerAppInput(ownerUserID int64, req registerAppRequest) RegisterAppInput {
	scopes := make([]ScopeRequestInput, 0, len(req.Scopes))
	for _, scope := range req.Scopes {
		scopes = append(scopes, ScopeRequestInput(scope))
	}
	return RegisterAppInput{
		OwnerUserID:      ownerUserID,
		DisplayName:      req.DisplayName,
		Description:      req.Description,
		HomepageURL:      req.HomepageURL,
		PrivacyPolicyURL: req.PrivacyPolicyURL,
		RedirectURIs:     req.RedirectURIs,
		Scopes:           scopes,
	}
}

func importCasdoorAppInput(importerUserID int64, req importCasdoorAppRequest) ImportCasdoorAppInput {
	scopes := make([]ScopeRequestInput, 0, len(req.Scopes))
	for _, scope := range req.Scopes {
		scopes = append(scopes, ScopeRequestInput(scope))
	}
	return ImportCasdoorAppInput{
		OwnerUserID:            importerUserID,
		ReviewerUserID:         importerUserID,
		CasdoorApplicationName: req.CasdoorApplicationName,
		DisplayName:            req.DisplayName,
		Description:            req.Description,
		HomepageURL:            req.HomepageURL,
		PrivacyPolicyURL:       req.PrivacyPolicyURL,
		RedirectURIs:           req.RedirectURIs,
		ClientSecret:           req.ClientSecret,
		Scopes:                 scopes,
	}
}

func registeredAppToJSON(registered *RegisteredApp) registeredAppResponse {
	return registeredAppResponse{
		App:          appToJSON(registered.App),
		ClientSecret: registered.ClientSecret,
	}
}

func approvedAppToJSON(approved *ApprovedApp) approvedAppResponse {
	return approvedAppResponse{
		App:          appToJSON(approved.App),
		ClientSecret: approved.ClientSecret,
	}
}

func importedAppToJSON(imported *ImportedApp) importedAppResponse {
	return importedAppResponse{
		App:                appToJSON(imported.App),
		ClientSecret:       imported.ClientSecret,
		ClientSecretSource: imported.ClientSecretSource,
	}
}

func appToJSON(app *App) appResponse {
	return appResponse{
		ID:               app.ID,
		ClientID:         app.ClientID,
		DisplayName:      app.DisplayName,
		Description:      app.Description,
		HomepageURL:      app.HomepageURL,
		PrivacyPolicyURL: app.PrivacyPolicyURL,
		RedirectURIs:     append([]string(nil), app.RedirectURIs...),
		Status:           app.Status,
		CreatedAt:        app.CreatedAt,
		UpdatedAt:        app.UpdatedAt,
	}
}

func parseInt64(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}
