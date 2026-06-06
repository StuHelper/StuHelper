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

type updateAppProfileRequest struct {
	DisplayName      string `json:"displayName" binding:"required"`
	Description      string `json:"description"`
	HomepageURL      string `json:"homepageURL" binding:"required"`
	PrivacyPolicyURL string `json:"privacyPolicyURL" binding:"required"`
	Reason           string `json:"reason" binding:"required"`
}

type scopeChangeRequest struct {
	Scopes []scopeRequestInput `json:"scopes" binding:"required"`
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

type scopeRequestResponse struct {
	ID             int64      `json:"id"`
	Scope          string     `json:"scope"`
	DisplayName    string     `json:"displayName"`
	Sensitivity    string     `json:"sensitivity"`
	Fields         []string   `json:"fields"`
	Reason         string     `json:"reason"`
	Status         string     `json:"status"`
	ReviewerUserID *int64     `json:"reviewerUserID"`
	ReviewedAt     *time.Time `json:"reviewedAt"`
	DecisionNote   *string    `json:"decisionNote"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type appWithScopesResponse struct {
	App                 appResponse                  `json:"app"`
	Scopes              []scopeRequestResponse       `json:"scopes"`
	RedirectURIRequests []redirectURIRequestResponse `json:"redirectURIRequests"`
}

type appListResponse struct {
	List  []appWithScopesResponse `json:"list"`
	Total int                     `json:"total"`
}

type scopeChangeResponse struct {
	Scopes []scopeRequestResponse `json:"scopes"`
}

type auditEventResponse struct {
	ID        int64          `json:"id"`
	AppID     *int64         `json:"appID"`
	UserID    *int64         `json:"userID"`
	EventType string         `json:"eventType"`
	Scope     *string        `json:"scope"`
	RequestID *string        `json:"requestID"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"createdAt"`
}

type auditEventListResponse struct {
	List  []auditEventResponse `json:"list"`
	Total int                  `json:"total"`
}

type userConsentAuditEventResponse struct {
	ID             int64          `json:"id"`
	AppID          *int64         `json:"appID"`
	AppDisplayName *string        `json:"appDisplayName"`
	ClientID       *string        `json:"clientID"`
	EventType      string         `json:"eventType"`
	Scope          *string        `json:"scope"`
	Scopes         []string       `json:"scopes"`
	Endpoint       *string        `json:"endpoint"`
	Result         *string        `json:"result"`
	RequestID      *string        `json:"requestID"`
	Details        map[string]any `json:"details"`
	CreatedAt      time.Time      `json:"createdAt"`
}

type userConsentAuditEventListResponse struct {
	List  []userConsentAuditEventResponse `json:"list"`
	Total int                             `json:"total"`
}

type adminUserAuthorizedAppResponse struct {
	UserID int64                      `json:"userID"`
	App    appResponse                `json:"app"`
	Scopes []userConsentScopeResponse `json:"scopes"`
}

type adminUserConsentListResponse struct {
	List  []adminUserAuthorizedAppResponse `json:"list"`
	Total int                              `json:"total"`
}

type developerAppAuditEventResponse struct {
	ID        int64          `json:"id"`
	EventType string         `json:"eventType"`
	Scope     *string        `json:"scope"`
	Scopes    []string       `json:"scopes"`
	Endpoint  *string        `json:"endpoint"`
	Result    *string        `json:"result"`
	RequestID *string        `json:"requestID"`
	Details   map[string]any `json:"details"`
	CreatedAt time.Time      `json:"createdAt"`
}

type developerAppAuditEventListResponse struct {
	List  []developerAppAuditEventResponse `json:"list"`
	Total int                              `json:"total"`
}

type tokenProbeEvidenceResponse struct {
	ID                     int64               `json:"id"`
	AppID                  int64               `json:"appID"`
	ReviewerUserID         *int64              `json:"reviewerUserID"`
	RequestID              *string             `json:"requestID"`
	CasdoorApplicationName string              `json:"casdoorApplicationName"`
	ClientID               string              `json:"clientID"`
	RedirectURI            string              `json:"redirectURI"`
	ProbeMethod            string              `json:"probeMethod"`
	Result                 string              `json:"result"`
	InspectedClaims        []string            `json:"inspectedClaims"`
	BusinessClaims         []string            `json:"businessClaims"`
	TokenClaims            map[string][]string `json:"tokenClaims"`
	Metadata               map[string]any      `json:"metadata"`
	Error                  string              `json:"error"`
	CreatedAt              time.Time           `json:"createdAt"`
}

type tokenProbeEvidenceListResponse struct {
	List  []tokenProbeEvidenceResponse `json:"list"`
	Total int                          `json:"total"`
}

type resourceGrantRequest struct {
	ResourceType string   `json:"resourceType" binding:"required"`
	ResourceID   string   `json:"resourceID" binding:"required"`
	Actions      []string `json:"actions" binding:"required"`
	Reason       string   `json:"reason" binding:"required"`
}

type resourceAccessCheckRequest struct {
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	ResourceType string `json:"resourceType" binding:"required"`
	ResourceID   string `json:"resourceID" binding:"required"`
	Action       string `json:"action" binding:"required"`
}

type resourceGrantResponse struct {
	AppID        int64  `json:"appID"`
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceID"`
	Action       string `json:"action"`
	Relation     string `json:"relation"`
}

type resourceGrantResultResponse struct {
	App    appResponse             `json:"app"`
	Grants []resourceGrantResponse `json:"grants"`
}

type resourceAccessDecisionResponse struct {
	Allowed      bool   `json:"allowed"`
	AppID        int64  `json:"appID"`
	ClientID     string `json:"clientID"`
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceID"`
	Action       string `json:"action"`
	Relation     string `json:"relation"`
	Reason       string `json:"reason"`
}

type disclosureReportSummaryResponse struct {
	WindowHours    int `json:"windowHours"`
	Total          int `json:"total"`
	Granted        int `json:"granted"`
	Denied         int `json:"denied"`
	RateLimited    int `json:"rateLimited"`
	ReplayDetected int `json:"replayDetected"`
}

type disclosureEndpointStatsResponse struct {
	Endpoint       string `json:"endpoint"`
	Total          int    `json:"total"`
	Granted        int    `json:"granted"`
	Denied         int    `json:"denied"`
	RateLimited    int    `json:"rateLimited"`
	ReplayDetected int    `json:"replayDetected"`
}

type disclosureReasonStatsResponse struct {
	Reason string `json:"reason"`
	Total  int    `json:"total"`
}

type disclosureRateLimitStatsResponse struct {
	Dimension string `json:"dimension"`
	Total     int    `json:"total"`
}

type disclosureReplayEventResponse struct {
	ID         int64          `json:"id"`
	AppID      *int64         `json:"appID"`
	UserID     *int64         `json:"userID"`
	Endpoint   string         `json:"endpoint"`
	Result     string         `json:"result"`
	Count      int            `json:"count"`
	Scopes     []string       `json:"scopes"`
	RequestID  *string        `json:"requestID"`
	Metadata   map[string]any `json:"metadata"`
	DetectedAt time.Time      `json:"detectedAt"`
}

type disclosureReportResponse struct {
	Summary             disclosureReportSummaryResponse    `json:"summary"`
	Endpoints           []disclosureEndpointStatsResponse  `json:"endpoints"`
	DenialReasons       []disclosureReasonStatsResponse    `json:"denialReasons"`
	RateLimitDimensions []disclosureRateLimitStatsResponse `json:"rateLimitDimensions"`
	RecentReplayEvents  []disclosureReplayEventResponse    `json:"recentReplayEvents"`
}

type userConsentScopeResponse struct {
	Scope       string     `json:"scope"`
	DisplayName string     `json:"displayName"`
	Sensitivity string     `json:"sensitivity"`
	Fields      []string   `json:"fields"`
	GrantedAt   time.Time  `json:"grantedAt"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
	GrantSource string     `json:"grantSource"`
	Reason      string     `json:"reason"`
}

type userAuthorizedAppResponse struct {
	App    appResponse                `json:"app"`
	Scopes []userConsentScopeResponse `json:"scopes"`
}

type userConsentsResponse struct {
	Apps []userAuthorizedAppResponse `json:"apps"`
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

type secretRotationRequest struct {
	Reason string `json:"reason"`
}

type redirectURIChangeRequest struct {
	RedirectURIs []string `json:"redirectURIs" binding:"required"`
	Reason       string   `json:"reason" binding:"required"`
}

type redirectURIReviewRequest struct {
	DecisionNote string `json:"decisionNote"`
}

type redirectURIRequestResponse struct {
	ID             int64      `json:"id"`
	RedirectURIs   []string   `json:"redirectURIs"`
	Reason         string     `json:"reason"`
	Status         string     `json:"status"`
	ReviewerUserID *int64     `json:"reviewerUserID"`
	ReviewedAt     *time.Time `json:"reviewedAt"`
	DecisionNote   *string    `json:"decisionNote"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type rotatedSecretResponse struct {
	App          appResponse `json:"app"`
	ClientSecret string      `json:"clientSecret"`
}

type lifecycleActionRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type adminConsentRevokeRequest struct {
	UserID int64    `json:"userID" binding:"required"`
	Scopes []string `json:"scopes"`
	Reason string   `json:"reason" binding:"required"`
}

type appLifecycleResponse struct {
	App appResponse `json:"app"`
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

func updateAppProfileInput(
	appID int64,
	ownerUserID int64,
	requestID string,
	req updateAppProfileRequest,
) UpdateAppProfileInput {
	return UpdateAppProfileInput{
		AppID:            appID,
		OwnerUserID:      ownerUserID,
		DisplayName:      req.DisplayName,
		Description:      req.Description,
		HomepageURL:      req.HomepageURL,
		PrivacyPolicyURL: req.PrivacyPolicyURL,
		Reason:           req.Reason,
		RequestID:        requestID,
	}
}

func scopeChangeInput(req scopeChangeRequest) []ScopeRequestInput {
	scopes := make([]ScopeRequestInput, 0, len(req.Scopes))
	for _, scope := range req.Scopes {
		scopes = append(scopes, ScopeRequestInput(scope))
	}
	return scopes
}

func importCasdoorAppInput(importerUserID int64, requestID string, req importCasdoorAppRequest) ImportCasdoorAppInput {
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
		RequestID:              requestID,
	}
}

func registeredAppToJSON(registered *RegisteredApp) registeredAppResponse {
	return registeredAppResponse{
		App:          appToJSON(registered.App),
		ClientSecret: registered.ClientSecret,
	}
}

func appListToJSON(result AppListResult) appListResponse {
	list := make([]appWithScopesResponse, 0, len(result.List))
	for _, item := range result.List {
		scopes := make([]scopeRequestResponse, 0, len(item.Scopes))
		for _, scope := range item.Scopes {
			scopes = append(scopes, scopeRequestToJSON(scope))
		}
		redirects := make([]redirectURIRequestResponse, 0, len(item.RedirectURIRequests))
		for _, redirect := range item.RedirectURIRequests {
			redirects = append(redirects, redirectURIRequestToJSON(redirect))
		}
		list = append(list, appWithScopesResponse{
			App:                 appToJSON(item.App),
			Scopes:              scopes,
			RedirectURIRequests: redirects,
		})
	}
	return appListResponse{List: list, Total: result.Total}
}

func scopeChangeToJSON(result ScopeChangeResult) scopeChangeResponse {
	scopes := make([]scopeRequestResponse, 0, len(result.Scopes))
	for _, scope := range result.Scopes {
		scopes = append(scopes, scopeRequestToJSON(scope))
	}
	return scopeChangeResponse{Scopes: scopes}
}

func auditEventListToJSON(result AuditEventListResult) auditEventListResponse {
	list := make([]auditEventResponse, 0, len(result.List))
	for _, event := range result.List {
		list = append(list, auditEventToJSON(event))
	}
	return auditEventListResponse{List: list, Total: result.Total}
}

func auditEventToJSON(event AuditEvent) auditEventResponse {
	metadata := event.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	return auditEventResponse{
		ID:        event.ID,
		AppID:     event.AppID,
		UserID:    event.UserID,
		EventType: event.EventType,
		Scope:     event.Scope,
		RequestID: event.RequestID,
		Metadata:  metadata,
		CreatedAt: event.CreatedAt,
	}
}

func userConsentAuditEventListToJSON(
	result UserConsentAuditEventListResult,
) userConsentAuditEventListResponse {
	list := make([]userConsentAuditEventResponse, 0, len(result.List))
	for _, event := range result.List {
		list = append(list, userConsentAuditEventToJSON(event))
	}
	return userConsentAuditEventListResponse{List: list, Total: result.Total}
}

func userConsentAuditEventToJSON(event UserConsentAuditEvent) userConsentAuditEventResponse {
	details := event.Details
	if details == nil {
		details = map[string]any{}
	}
	scopes := event.Scopes
	if scopes == nil {
		scopes = []string{}
	}
	return userConsentAuditEventResponse{
		ID:             event.ID,
		AppID:          event.AppID,
		AppDisplayName: event.AppDisplayName,
		ClientID:       event.ClientID,
		EventType:      event.EventType,
		Scope:          event.Scope,
		Scopes:         scopes,
		Endpoint:       event.Endpoint,
		Result:         event.Result,
		RequestID:      event.RequestID,
		Details:        details,
		CreatedAt:      event.CreatedAt,
	}
}

func developerAppAuditEventListToJSON(
	result DeveloperAppAuditEventListResult,
) developerAppAuditEventListResponse {
	list := make([]developerAppAuditEventResponse, 0, len(result.List))
	for _, event := range result.List {
		list = append(list, developerAppAuditEventToJSON(event))
	}
	return developerAppAuditEventListResponse{List: list, Total: result.Total}
}

func developerAppAuditEventToJSON(event DeveloperAppAuditEvent) developerAppAuditEventResponse {
	details := event.Details
	if details == nil {
		details = map[string]any{}
	}
	scopes := event.Scopes
	if scopes == nil {
		scopes = []string{}
	}
	return developerAppAuditEventResponse{
		ID:        event.ID,
		EventType: event.EventType,
		Scope:     event.Scope,
		Scopes:    scopes,
		Endpoint:  event.Endpoint,
		Result:    event.Result,
		RequestID: event.RequestID,
		Details:   details,
		CreatedAt: event.CreatedAt,
	}
}

func tokenProbeEvidenceListToJSON(
	result TokenProbeEvidenceListResult,
) tokenProbeEvidenceListResponse {
	list := make([]tokenProbeEvidenceResponse, 0, len(result.List))
	for _, record := range result.List {
		list = append(list, tokenProbeEvidenceToJSON(record))
	}
	return tokenProbeEvidenceListResponse{List: list, Total: result.Total}
}

func tokenProbeEvidenceToJSON(record TokenProbeEvidenceRecord) tokenProbeEvidenceResponse {
	tokenClaims := record.TokenClaims
	if tokenClaims == nil {
		tokenClaims = map[string][]string{}
	}
	metadata := record.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	return tokenProbeEvidenceResponse{
		ID:                     record.ID,
		AppID:                  record.AppID,
		ReviewerUserID:         record.ReviewerUserID,
		RequestID:              record.RequestID,
		CasdoorApplicationName: record.CasdoorApplicationName,
		ClientID:               record.ClientID,
		RedirectURI:            record.RedirectURI,
		ProbeMethod:            record.ProbeMethod,
		Result:                 record.Result,
		InspectedClaims:        append([]string(nil), record.InspectedClaims...),
		BusinessClaims:         append([]string(nil), record.BusinessClaims...),
		TokenClaims:            tokenClaims,
		Metadata:               metadata,
		Error:                  record.Error,
		CreatedAt:              record.CreatedAt,
	}
}

func disclosureReportToJSON(report DisclosureReport) disclosureReportResponse {
	endpoints := make([]disclosureEndpointStatsResponse, 0, len(report.Endpoints))
	for _, endpoint := range report.Endpoints {
		endpoints = append(endpoints, disclosureEndpointStatsResponse(endpoint))
	}
	reasons := make([]disclosureReasonStatsResponse, 0, len(report.DenialReasons))
	for _, reason := range report.DenialReasons {
		reasons = append(reasons, disclosureReasonStatsResponse(reason))
	}
	dimensions := make([]disclosureRateLimitStatsResponse, 0, len(report.RateLimitDimensions))
	for _, dimension := range report.RateLimitDimensions {
		dimensions = append(dimensions, disclosureRateLimitStatsResponse(dimension))
	}
	replayEvents := make([]disclosureReplayEventResponse, 0, len(report.RecentReplayEvents))
	for _, event := range report.RecentReplayEvents {
		metadata := event.Metadata
		if metadata == nil {
			metadata = map[string]any{}
		}
		replayEvents = append(replayEvents, disclosureReplayEventResponse{
			ID:         event.ID,
			AppID:      event.AppID,
			UserID:     event.UserID,
			Endpoint:   event.Endpoint,
			Result:     event.Result,
			Count:      event.Count,
			Scopes:     append([]string(nil), event.Scopes...),
			RequestID:  event.RequestID,
			Metadata:   metadata,
			DetectedAt: event.DetectedAt,
		})
	}
	return disclosureReportResponse{
		Summary:             disclosureReportSummaryResponse(report.Summary),
		Endpoints:           endpoints,
		DenialReasons:       reasons,
		RateLimitDimensions: dimensions,
		RecentReplayEvents:  replayEvents,
	}
}

func resourceGrantResultToJSON(result *ResourceGrantResult) resourceGrantResultResponse {
	grants := make([]resourceGrantResponse, 0, len(result.Grants))
	for _, grant := range result.Grants {
		grants = append(grants, resourceGrantResponse(grant))
	}
	return resourceGrantResultResponse{
		App:    appToJSON(result.App),
		Grants: grants,
	}
}

func resourceAccessDecisionToJSON(decision ResourceAccessDecision) resourceAccessDecisionResponse {
	return resourceAccessDecisionResponse(decision)
}

func scopeRequestToJSON(scope ScopeRequest) scopeRequestResponse {
	response := scopeRequestResponse{
		ID:             scope.ID,
		Scope:          scope.Scope,
		Reason:         scope.Reason,
		Status:         scope.Status,
		ReviewerUserID: scope.ReviewerUserID,
		ReviewedAt:     scope.ReviewedAt,
		DecisionNote:   scope.DecisionNote,
		CreatedAt:      scope.CreatedAt,
		UpdatedAt:      scope.UpdatedAt,
	}
	if definition, ok := scopeDefinitionForResponse(scope.Scope); ok {
		response.DisplayName = definition.DisplayName
		response.Sensitivity = definition.Sensitivity
		response.Fields = definition.Fields
	}
	return response
}

func redirectURIRequestToJSON(req RedirectURIRequest) redirectURIRequestResponse {
	return redirectURIRequestResponse{
		ID:             req.ID,
		RedirectURIs:   append([]string(nil), req.RedirectURIs...),
		Reason:         req.Reason,
		Status:         req.Status,
		ReviewerUserID: req.ReviewerUserID,
		ReviewedAt:     req.ReviewedAt,
		DecisionNote:   req.DecisionNote,
		CreatedAt:      req.CreatedAt,
		UpdatedAt:      req.UpdatedAt,
	}
}

func userConsentsToJSON(consents []UserAuthorizedApp) userConsentsResponse {
	apps := make([]userAuthorizedAppResponse, 0, len(consents))
	for _, consent := range consents {
		scopes := make([]userConsentScopeResponse, 0, len(consent.Scopes))
		for _, scope := range consent.Scopes {
			scopeResponse := userConsentScopeResponse{
				Scope:       scope.Scope,
				GrantedAt:   scope.GrantedAt,
				LastUsedAt:  scope.LastUsedAt,
				GrantSource: scope.GrantSource,
				Reason:      scope.Reason,
			}
			if definition, ok := scopeDefinitionForResponse(scope.Scope); ok {
				scopeResponse.DisplayName = definition.DisplayName
				scopeResponse.Sensitivity = definition.Sensitivity
				scopeResponse.Fields = definition.Fields
			}
			scopes = append(scopes, scopeResponse)
		}
		apps = append(apps, userAuthorizedAppResponse{
			App:    appToJSON(consent.App),
			Scopes: scopes,
		})
	}
	return userConsentsResponse{Apps: apps}
}

func adminUserConsentListToJSON(result AdminUserConsentListResult) adminUserConsentListResponse {
	list := make([]adminUserAuthorizedAppResponse, 0, len(result.List))
	for _, consent := range result.List {
		scopes := make([]userConsentScopeResponse, 0, len(consent.Scopes))
		for _, scope := range consent.Scopes {
			scopeResponse := userConsentScopeResponse{
				Scope:       scope.Scope,
				GrantedAt:   scope.GrantedAt,
				LastUsedAt:  scope.LastUsedAt,
				GrantSource: scope.GrantSource,
				Reason:      scope.Reason,
			}
			if definition, ok := scopeDefinitionForResponse(scope.Scope); ok {
				scopeResponse.DisplayName = definition.DisplayName
				scopeResponse.Sensitivity = definition.Sensitivity
				scopeResponse.Fields = definition.Fields
			}
			scopes = append(scopes, scopeResponse)
		}
		list = append(list, adminUserAuthorizedAppResponse{
			UserID: consent.UserID,
			App:    appToJSON(consent.App),
			Scopes: scopes,
		})
	}
	return adminUserConsentListResponse{List: list, Total: result.Total}
}

func scopeDefinitionForResponse(scope string) (ScopeDefinition, bool) {
	definitions := ScopeDefinitions([]string{scope})
	if len(definitions) != 1 {
		return ScopeDefinition{}, false
	}
	definition := definitions[0]
	definition.Fields = append([]string(nil), definition.Fields...)
	return definition, true
}

func approvedAppToJSON(approved *ApprovedApp) approvedAppResponse {
	return approvedAppResponse{
		App:          appToJSON(approved.App),
		ClientSecret: approved.ClientSecret,
	}
}

func rotatedSecretToJSON(rotated *RotatedAppSecret) rotatedSecretResponse {
	return rotatedSecretResponse{
		App:          appToJSON(rotated.App),
		ClientSecret: rotated.ClientSecret,
	}
}

func appLifecycleToJSON(result *AppLifecycleResult) appLifecycleResponse {
	return appLifecycleResponse{App: appToJSON(result.App)}
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
