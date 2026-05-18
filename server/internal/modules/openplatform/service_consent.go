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

func (s *Service) Authorize(ctx context.Context, req AuthorizeRequest, casdoorSubject string) (*AuthorizeResult, error) {
	scopes, err := NormalizeScopes(req.Scopes)
	if err != nil {
		return nil, err
	}
	app, userID, err := s.loadAuthorizeActors(ctx, req, casdoorSubject)
	if err != nil {
		return nil, err
	}
	if err := s.ensureScopesApproved(ctx, app.ID, scopes); err != nil {
		return nil, err
	}
	hasConsent, err := s.repo.HasActiveConsents(ctx, app.ID, userID, scopes)
	if err != nil {
		return nil, err
	}
	if !hasConsent {
		challenge, err := s.BuildConsentChallenge(ctx, app, userID, scopes, req.RedirectURI, req.State)
		if err != nil {
			return nil, err
		}
		return &AuthorizeResult{
			ConsentURL: buildConsentURL(s.consentBaseURL, challenge.Token),
			Scopes:     ScopeDefinitions(scopes),
		}, nil
	}
	redirectURL, err := s.buildOIDCRedirectURL(app, req, scopes)
	if err != nil {
		return nil, err
	}
	return &AuthorizeResult{RedirectURL: redirectURL}, nil
}

func (s *Service) GetConsentPage(ctx context.Context, token, casdoorSubject string) (*ConsentPage, error) {
	challenge, err := s.LoadConsentChallenge(ctx, token)
	if err != nil {
		return nil, err
	}
	if err := s.ensureConsentActor(ctx, challenge, casdoorSubject); err != nil {
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

func (s *Service) AcceptConsent(ctx context.Context, token, requestID, casdoorSubject string) (string, error) {
	loaded, err := s.LoadConsentChallenge(ctx, token)
	if err != nil {
		return "", err
	}
	if err := s.ensureConsentActor(ctx, loaded, casdoorSubject); err != nil {
		return "", err
	}
	challenge, err := s.GrantConsent(ctx, token, requestID)
	if err != nil {
		return "", err
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

func (s *Service) DenyConsent(ctx context.Context, token, casdoorSubject string) (string, error) {
	challenge, err := s.LoadConsentChallenge(ctx, token)
	if err != nil {
		return "", err
	}
	if err := s.ensureConsentActor(ctx, challenge, casdoorSubject); err != nil {
		return "", err
	}
	if err := s.rdb.Del(ctx, consentRedisPrefix+token).Err(); err != nil {
		return "", fmt.Errorf("delete denied consent challenge: %w", err)
	}
	return appendOAuthError(challenge.RedirectURI, "access_denied", challenge.State), nil
}

func (s *Service) loadAuthorizeActors(ctx context.Context, req AuthorizeRequest, casdoorSubject string) (*App, int64, error) {
	app, err := s.repo.GetAppByClientID(ctx, strings.TrimSpace(req.ClientID))
	if err != nil {
		return nil, 0, err
	}
	if app.Status != AppStatusApproved {
		return nil, 0, ErrAppNotActive
	}
	userID, err := s.repo.GetInternalUserID(ctx, strings.TrimSpace(casdoorSubject))
	if err != nil {
		return nil, 0, err
	}
	return app, userID, nil
}

func (s *Service) ensureConsentActor(ctx context.Context, challenge *ConsentChallenge, casdoorSubject string) error {
	userID, err := s.repo.GetInternalUserID(ctx, strings.TrimSpace(casdoorSubject))
	if err != nil {
		return err
	}
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
	redirectURI string,
	state string,
) (*ConsentChallenge, error) {
	if !redirectAllowed(app, redirectURI) {
		return nil, ErrRedirectURINotAllowed
	}
	token, err := randomHex("", consentTokenBytes)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	challenge := &ConsentChallenge{
		Token:       token,
		AppID:       app.ID,
		UserID:      userID,
		Scopes:      scopes,
		RedirectURI: redirectURI,
		State:       strings.TrimSpace(state),
		CreatedAt:   now,
		ExpiresAt:   now.Add(consentTokenTTL),
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
		Token:       token,
		AppID:       payload.AppID,
		UserID:      payload.UserID,
		Scopes:      payload.Scopes,
		RedirectURI: payload.RedirectURI,
		State:       payload.State,
		CreatedAt:   createdAt,
		ExpiresAt:   expiresAt,
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
