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
	if decision.ProfileCompletionURL != "" {
		return &AuthorizeResult{
			ProfileCompletionURL: decision.ProfileCompletionURL,
			MissingFields:        decision.MissingFields,
			Scopes:               ScopeDefinitions(decision.Scopes),
		}, nil
	}
	if decision.ConsentURL != "" {
		return &AuthorizeResult{
			ConsentURL: decision.ConsentURL,
			Scopes:     ScopeDefinitions(decision.Scopes),
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
	app, err := s.loadAuthorizeApp(ctx, req)
	if err != nil {
		return nil, err
	}
	if userID <= 0 {
		return nil, ErrDisclosureUnavailable
	}
	if err := s.ensureScopesApproved(ctx, app.ID, scopes); err != nil {
		return nil, err
	}
	projection, err := s.repo.GetUserProjection(ctx, userID)
	if err != nil {
		return nil, err
	}
	if missing := RequiredProfileFields(projection, scopes); len(missing) > 0 {
		challenge, err := s.BuildProfileCompletionChallenge(ctx, app, userID, scopes, req)
		if err != nil {
			return nil, err
		}
		return &AuthorizationDecision{
			App:                  app,
			UserID:               userID,
			Scopes:               scopes,
			ProfileCompletionURL: buildProfileCompletionURL(s.consentBaseURLForFlow(req.Flow), challenge.Token),
			MissingFields:        missing,
		}, nil
	}
	hasConsent, err := s.repo.HasActiveConsents(ctx, app.ID, userID, scopes)
	if err != nil {
		return nil, err
	}
	if !hasConsent {
		challenge, err := s.BuildConsentChallenge(ctx, app, userID, scopes, req)
		if err != nil {
			return nil, err
		}
		return &AuthorizationDecision{
			App:        app,
			UserID:     userID,
			Scopes:     scopes,
			ConsentURL: buildConsentURL(s.consentBaseURLForFlow(req.Flow), challenge.Token),
		}, nil
	}
	return &AuthorizationDecision{App: app, UserID: userID, Scopes: scopes}, nil
}

func (s *Service) GetConsentPage(ctx context.Context, token string, userID int64) (*ConsentPage, error) {
	challenge, err := s.LoadConsentChallenge(ctx, token)
	if err != nil {
		return nil, err
	}
	if err := ensureConsentActor(challenge, userID); err != nil {
		return nil, err
	}
	app, err := s.repo.GetAppByID(ctx, challenge.AppID)
	if err != nil {
		return nil, err
	}
	return &ConsentPage{
		Token:       challenge.Token,
		App:         consentApp(app),
		Scopes:      ScopeDefinitions(challenge.Scopes),
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
	keepChallenge := normalizeAuthorizeFlow(loaded.Flow) == AuthorizeFlowIdentity
	challenge, err := s.grantConsent(ctx, token, requestID, keepChallenge)
	if err != nil {
		return "", err
	}
	if normalizeAuthorizeFlow(challenge.Flow) == AuthorizeFlowIdentity {
		return buildIdentityContinueURL(s.identityBaseURL, challenge.Token), nil
	}
	app, err := s.repo.GetAppByID(ctx, challenge.AppID)
	if err != nil {
		return "", err
	}
	return s.buildOIDCRedirectURL(app, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: challenge.RedirectURI,
		Scopes:      challenge.Scopes,
		State:       challenge.State,
	}, challenge.Scopes)
}

func (s *Service) DenyConsent(ctx context.Context, token string, userID int64) (string, error) {
	challenge, err := s.LoadConsentChallenge(ctx, token)
	if err != nil {
		return "", err
	}
	if err := ensureConsentActor(challenge, userID); err != nil {
		return "", err
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
	return s.oidc.GetAuthURL(app.ClientID, req.RedirectURI, casdoorOAuthScopes(scopes), req.State), nil
}

func (s *Service) BuildConsentChallenge(
	ctx context.Context,
	app *App,
	userID int64,
	scopes []string,
	req AuthorizeRequest,
) (*ConsentChallenge, error) {
	if !redirectAllowed(app, req.RedirectURI) {
		return nil, ErrRedirectURINotAllowed
	}
	token, err := randomHex("", consentTokenBytes)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	challenge := &ConsentChallenge{
		Token:               token,
		AppID:               app.ID,
		UserID:              userID,
		Scopes:              scopes,
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
	if err := s.repo.GrantConsents(ctx, Consent{
		AppID:       challenge.AppID,
		UserID:      challenge.UserID,
		GrantSource: "web",
		RequestID:   requestID,
	}, challenge.Scopes); err != nil {
		return nil, err
	}
	if keepChallenge {
		return challenge, nil
	}
	if err := s.rdb.Del(ctx, consentRedisPrefix+token).Err(); err != nil {
		return nil, fmt.Errorf("delete consent challenge: %w", err)
	}
	return challenge, nil
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
	return &ConsentChallenge{
		Token:               token,
		AppID:               payload.AppID,
		UserID:              payload.UserID,
		Scopes:              payload.Scopes,
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

func casdoorOAuthScopes(_ []string) []string {
	return []string{"openid"}
}

func appendOAuthError(redirectURI, code, state string) string {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	query := parsed.Query()
	query.Set("error", code)
	if trimmed := strings.TrimSpace(state); trimmed != "" {
		query.Set("state", trimmed)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
