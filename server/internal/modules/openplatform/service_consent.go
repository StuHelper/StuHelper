package openplatform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type ConsentRequiredError struct {
	ConsentURL string
	Scopes     []ScopeDefinition
	State      string
}

func (e ConsentRequiredError) Error() string {
	return ErrConsentRequired.Error()
}

type ProfileCompletionRequiredError struct {
	CompletionURL string
	MissingFields []ProfileCompletionField
	Scopes        []ScopeDefinition
	State         string
}

func (e ProfileCompletionRequiredError) Error() string {
	return ErrProfileIncomplete.Error()
}

func (s *Service) Authorize(ctx context.Context, req AuthorizeRequest, userID int64) (*AuthorizeResult, error) {
	req.Flow = AuthorizeFlowCasdoor
	decision, err := s.BeginAuthorization(ctx, req, userID)
	if err != nil {
		return nil, err
	}
	if decision.InteractionRequired {
		if decision.InteractionError == "consent_required" {
			return nil, ErrConsentRequired
		}
		return nil, ErrProfileIncomplete
	}
	if decision.ProfileCompletionURL != "" {
		definitions, err := s.scopeDefinitionsForApp(ctx, decision.App.ID, decision.Scopes)
		if err != nil {
			return nil, err
		}
		return &AuthorizeResult{
			ProfileCompletionURL: decision.ProfileCompletionURL,
			MissingFields:        decision.MissingFields,
			Scopes:               definitions,
		}, nil
	}
	if decision.ConsentURL != "" {
		definitions, err := s.scopeDefinitionsForApp(ctx, decision.App.ID, decision.Scopes)
		if err != nil {
			return nil, err
		}
		return &AuthorizeResult{
			ConsentURL: decision.ConsentURL,
			Scopes:     definitions,
		}, nil
	}
	redirectURL, err := s.buildOIDCRedirectURL(decision.App, req, decision.Scopes)
	if err != nil {
		return nil, err
	}
	return &AuthorizeResult{RedirectURL: redirectURL}, nil
}

func (s *Service) BeginAuthorization(ctx context.Context, req AuthorizeRequest, userID int64) (*AuthorizationDecision, error) {
	req.Flow = normalizeAuthorizeFlow(req.Flow)
	scopes, err := NormalizeAuthorizationScopes(req.Scopes)
	if err != nil {
		return nil, err
	}
	oauthScopes, err := NormalizeGrantedOAuthScopes(req.Scopes)
	if err != nil {
		return nil, err
	}
	app, err := s.loadAuthorizeApp(ctx, req)
	if err != nil {
		return nil, err
	}
	if userID <= 0 {
		return nil, ErrDisclosureUnavailable
	}
	if !redirectAllowed(app, req.RedirectURI) {
		return nil, ErrRedirectURINotAllowed
	}
	if err := s.ensureScopesApproved(ctx, app.ID, scopes); err != nil {
		return nil, err
	}
	consentScopes := UserConsentScopes(scopes)
	projection, err := s.repo.GetUserProjection(ctx, userID)
	if err != nil {
		return nil, err
	}
	if missing := RequiredProfileFields(projection, consentScopes); len(missing) > 0 {
		if req.PromptNone {
			return &AuthorizationDecision{
				App:                 app,
				UserID:              userID,
				Scopes:              consentScopes,
				OAuthScopes:         oauthScopes,
				InteractionRequired: true,
				InteractionError:    "interaction_required",
				MissingFields:       missing,
			}, nil
		}
		challenge, err := s.BuildProfileCompletionChallenge(ctx, app, userID, scopes, req)
		if err != nil {
			return nil, err
		}
		return &AuthorizationDecision{
			App:                  app,
			UserID:               userID,
			Scopes:               consentScopes,
			OAuthScopes:          oauthScopes,
			ProfileCompletionURL: buildProfileCompletionURL(s.consentBaseURLForFlow(req.Flow), challenge.Token),
			MissingFields:        missing,
		}, nil
	}
	if len(consentScopes) == 0 {
		return &AuthorizationDecision{App: app, UserID: userID, Scopes: scopes, OAuthScopes: oauthScopes}, nil
	}
	hasConsent, err := s.repo.HasActiveConsents(ctx, app.ID, userID, consentScopes)
	if err != nil {
		return nil, err
	}
	if !hasConsent || req.ForceConsent {
		if req.PromptNone {
			return &AuthorizationDecision{
				App:                 app,
				UserID:              userID,
				Scopes:              consentScopes,
				OAuthScopes:         oauthScopes,
				InteractionRequired: true,
				InteractionError:    "consent_required",
			}, nil
		}
		if err := s.rateLimiter.checkConsentChallenge(ctx, app.ID, userID); err != nil {
			return nil, err
		}
		challenge, err := s.BuildConsentChallenge(ctx, app, userID, scopes, req)
		if err != nil {
			return nil, err
		}
		return &AuthorizationDecision{
			App:         app,
			UserID:      userID,
			Scopes:      consentScopes,
			OAuthScopes: oauthScopes,
			ConsentURL:  buildConsentURL(s.consentBaseURLForFlow(req.Flow), challenge.Token),
		}, nil
	}
	return &AuthorizationDecision{App: app, UserID: userID, Scopes: scopes, OAuthScopes: oauthScopes}, nil
}

func (s *Service) GetConsentPage(ctx context.Context, token string, userID int64) (*ConsentPage, error) {
	challenge, err := s.LoadConsentChallenge(ctx, token)
	if err != nil {
		return nil, err
	}
	if err := ensureConsentActor(challenge, userID); err != nil {
		return nil, err
	}
	app, err := s.loadCurrentChallengeApp(ctx, challenge.AppID, challenge.RedirectURI, challenge.Scopes)
	if err != nil {
		return nil, err
	}
	definitions, err := s.scopeDefinitionsForApp(ctx, app.ID, challenge.ConsentScopes)
	if err != nil {
		return nil, err
	}
	return &ConsentPage{
		Token:       challenge.Token,
		App:         consentApp(app),
		Scopes:      definitions,
		RedirectURI: challenge.RedirectURI,
		ExpiresAt:   challenge.ExpiresAt,
	}, nil
}

func (s *Service) AcceptConsent(ctx context.Context, token, requestID string, userID int64) (string, error) {
	loaded, err := s.LoadConsentChallenge(ctx, token)
	if err != nil {
		return "", err
	}
	if err := ensureConsentActor(loaded, userID); err != nil {
		return "", err
	}
	if normalizeAuthorizeFlow(loaded.Flow) == AuthorizeFlowAccount {
		challenge, err := s.grantConsent(ctx, token, requestID, true)
		if err != nil {
			return "", err
		}
		return buildAccountContinueURL(s.accountBaseURL, challenge.Token), nil
	}
	app, err := s.loadCurrentChallengeApp(ctx, loaded.AppID, loaded.RedirectURI, loaded.Scopes)
	if err != nil {
		return "", err
	}
	redirectURL, err := s.buildOIDCRedirectURL(app, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: loaded.RedirectURI,
		Scopes:      grantedOAuthScopes(loaded.OAuthScopes, loaded.Scopes),
		State:       loaded.State,
	}, loaded.Scopes)
	if err != nil {
		return "", err
	}
	if err := s.grantLoadedConsent(ctx, loaded, requestID); err != nil {
		return "", err
	}
	if err := s.DeleteConsentChallenge(ctx, token); err != nil {
		return "", err
	}
	return redirectURL, nil
}

func (s *Service) DenyConsent(ctx context.Context, token, requestID string, userID int64) (string, error) {
	challenge, err := s.LoadConsentChallenge(ctx, token)
	if err != nil {
		return "", err
	}
	if err := ensureConsentActor(challenge, userID); err != nil {
		return "", err
	}
	if _, err := s.loadCurrentChallengeApp(ctx, challenge.AppID, challenge.RedirectURI, challenge.Scopes); err != nil {
		return "", err
	}
	if err := s.repo.RecordAuditEvent(ctx, auditEvent{
		AppID:     challenge.AppID,
		UserID:    challenge.UserID,
		EventType: "open_platform.consent.denied",
		RequestID: requestID,
		Metadata: map[string]any{
			"actor":  "user",
			"flow":   normalizeAuthorizeFlow(challenge.Flow),
			"result": "access_denied",
			"scopes": append([]string(nil), challenge.ConsentScopes...),
		},
	}); err != nil {
		return "", fmt.Errorf("%w: consent denial audit unavailable", ErrDisclosureUnavailable)
	}
	if err := s.rdb.Del(ctx, consentRedisPrefix+token).Err(); err != nil {
		return "", fmt.Errorf("delete denied consent challenge: %w", err)
	}
	return appendOAuthError(challenge.RedirectURI, "access_denied", challenge.State), nil
}

func (s *Service) loadAuthorizeApp(ctx context.Context, req AuthorizeRequest) (*App, error) {
	app, err := s.repo.GetAppByClientID(ctx, strings.TrimSpace(req.ClientID))
	if err != nil {
		return nil, err
	}
	if app.Status != AppStatusApproved {
		return nil, ErrAppNotActive
	}
	return app, nil
}

func (s *Service) loadCurrentChallengeApp(ctx context.Context, appID int64, redirectURI string, scopes []string) (*App, error) {
	if appID <= 0 {
		return nil, ErrAppNotFound
	}
	app, err := s.repo.GetAppByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if app.Status != AppStatusApproved {
		return nil, ErrAppNotActive
	}
	if !redirectAllowed(app, redirectURI) {
		return nil, ErrRedirectURINotAllowed
	}
	if err := s.ensureScopesApproved(ctx, app.ID, scopes); err != nil {
		return nil, err
	}
	return app, nil
}

func (s *Service) AuthorizeAppByClientID(ctx context.Context, clientID string) (*App, error) {
	return s.loadAuthorizeApp(ctx, AuthorizeRequest{ClientID: clientID})
}

func ensureConsentActor(challenge *ConsentChallenge, userID int64) error {
	if userID != challenge.UserID {
		return ErrConsentTokenInvalid
	}
	return nil
}

func (s *Service) buildOIDCRedirectURL(app *App, req AuthorizeRequest, scopes []string) (string, error) {
	if !redirectAllowed(app, req.RedirectURI) {
		return "", ErrRedirectURINotAllowed
	}
	if s.oidc == nil {
		return "", fmt.Errorf("open platform OIDC URL builder is not configured")
	}
	return s.oidc.GetAuthURL(app.ClientID, req.RedirectURI, oidcRedirectScopes(req.Scopes, scopes), req.State), nil
}

func (s *Service) BuildConsentChallenge(
	ctx context.Context,
	app *App,
	userID int64,
	scopes []string,
	req AuthorizeRequest,
) (*ConsentChallenge, error) {
	if app == nil {
		return nil, ErrAppNotFound
	}
	currentApp, err := s.loadCurrentChallengeApp(ctx, app.ID, req.RedirectURI, scopes)
	if err != nil {
		return nil, err
	}
	oauthScopes, err := NormalizeGrantedOAuthScopes(req.Scopes)
	if err != nil {
		oauthScopes, err = NormalizeGrantedOAuthScopes(scopes)
		if err != nil {
			return nil, err
		}
	}
	token, err := randomHex("", consentTokenBytes)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	challenge := &ConsentChallenge{
		Token:               token,
		AppID:               currentApp.ID,
		UserID:              userID,
		Scopes:              scopes,
		OAuthScopes:         oauthScopes,
		ConsentScopes:       UserConsentScopes(scopes),
		RedirectURI:         req.RedirectURI,
		State:               strings.TrimSpace(req.State),
		Flow:                normalizeAuthorizeFlow(req.Flow),
		CodeChallenge:       strings.TrimSpace(req.CodeChallenge),
		CodeChallengeMethod: strings.TrimSpace(req.CodeChallengeMethod),
		Nonce:               strings.TrimSpace(req.Nonce),
		CreatedAt:           now,
		ExpiresAt:           now.Add(consentTokenTTL),
	}
	payload, err := challenge.MarshalPayload()
	if err != nil {
		return nil, fmt.Errorf("marshal consent challenge: %w", err)
	}
	key := consentRedisPrefix + token
	if err := s.rdb.Set(ctx, key, payload, consentTokenTTL).Err(); err != nil {
		return nil, fmt.Errorf("store consent challenge: %w", err)
	}
	return challenge, nil
}

func (s *Service) LoadConsentChallenge(ctx context.Context, token string) (*ConsentChallenge, error) {
	raw, err := s.rdb.Get(ctx, consentRedisPrefix+token).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrConsentTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("load consent challenge: %w", err)
	}
	return decodeConsentChallenge(token, []byte(raw))
}

func (s *Service) GrantConsent(ctx context.Context, token, requestID string) (*ConsentChallenge, error) {
	return s.grantConsent(ctx, token, requestID, false)
}

func (s *Service) grantConsent(ctx context.Context, token, requestID string, keepChallenge bool) (*ConsentChallenge, error) {
	challenge, err := s.LoadConsentChallenge(ctx, token)
	if err != nil {
		return nil, err
	}
	if _, err := s.loadCurrentChallengeApp(ctx, challenge.AppID, challenge.RedirectURI, challenge.Scopes); err != nil {
		return nil, err
	}
	if err := s.grantLoadedConsent(ctx, challenge, requestID); err != nil {
		return nil, err
	}
	if keepChallenge {
		return challenge, nil
	}
	if err := s.DeleteConsentChallenge(ctx, token); err != nil {
		return nil, err
	}
	return challenge, nil
}

func (s *Service) grantLoadedConsent(ctx context.Context, challenge *ConsentChallenge, requestID string) error {
	return s.repo.GrantConsents(ctx, Consent{
		AppID:       challenge.AppID,
		UserID:      challenge.UserID,
		GrantSource: "web",
		RequestID:   requestID,
	}, challenge.ConsentScopes)
}

func decodeConsentChallenge(token string, raw []byte) (*ConsentChallenge, error) {
	var payload ConsentChallengePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode consent challenge: %w", err)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, payload.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("decode consent created_at: %w", err)
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, payload.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("decode consent expires_at: %w", err)
	}
	consentScopes := append([]string(nil), payload.ConsentScopes...)
	if payload.ConsentScopes == nil {
		consentScopes = append([]string(nil), payload.Scopes...)
	}
	oauthScopes := append([]string(nil), payload.OAuthScopes...)
	if payload.OAuthScopes == nil {
		oauthScopes = append([]string(nil), payload.Scopes...)
	}
	return &ConsentChallenge{
		Token:               token,
		AppID:               payload.AppID,
		UserID:              payload.UserID,
		Scopes:              payload.Scopes,
		OAuthScopes:         oauthScopes,
		ConsentScopes:       consentScopes,
		RedirectURI:         payload.RedirectURI,
		State:               payload.State,
		Flow:                normalizeAuthorizeFlow(payload.Flow),
		CodeChallenge:       payload.CodeChallenge,
		CodeChallengeMethod: payload.CodeChallengeMethod,
		Nonce:               payload.Nonce,
		CreatedAt:           createdAt,
		ExpiresAt:           expiresAt,
	}, nil
}

func redirectAllowed(app *App, redirectURI string) bool {
	parsed, err := url.Parse(redirectURI)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Fragment != "" {
		return false
	}
	for _, candidate := range app.RedirectURIs {
		if redirectURI == candidate {
			return true
		}
	}
	return false
}

func RedirectURIAllowed(app *App, redirectURI string) bool {
	return redirectAllowed(app, redirectURI)
}

func consentApp(app *App) ConsentApp {
	return ConsentApp{
		ID:               app.ID,
		ClientID:         app.ClientID,
		DisplayName:      app.DisplayName,
		Description:      app.Description,
		HomepageURL:      app.HomepageURL,
		PrivacyPolicyURL: app.PrivacyPolicyURL,
	}
}

func casdoorOAuthScopes(scopes []string) []string {
	oauthScopes, err := NormalizeGrantedOAuthScopes(scopes)
	if err != nil || len(oauthScopes) == 0 {
		return []string{"openid"}
	}
	return oauthScopes
}

func oidcRedirectScopes(oauthScopes, fallbackScopes []string) []string {
	normalized, err := NormalizeGrantedOAuthScopes(oauthScopes)
	if err == nil && len(normalized) > 0 {
		return normalized
	}
	return casdoorOAuthScopes(fallbackScopes)
}

func appendOAuthError(redirectURI, code, state string) string {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	query := parsed.Query()
	query.Set("error", code)
	if state != "" {
		query.Set("state", state)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
