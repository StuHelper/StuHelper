package identityserver

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/openplatform"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
)

const (
	codeBytes                = 32
	refreshTokenBytes        = 32
	authCodeRedisPrefix      = "identity:auth_code:"
	refreshTokenRedisPrefix  = "identity:refresh_token:"
	refreshTokenUsedPrefix   = "identity:refresh_token_used:"
	refreshTokenFamilyPrefix = "identity:refresh_token_family:"
	revokedRedisPrefix       = "identity:revoked_jti:"
	pkceS256Length           = 43
	pkceVerifierMin          = 43
	pkceVerifierMax          = 128
	defaultRefreshTokenTTL   = 30 * 24 * time.Hour

	oidcPromptNone                = "none"
	oidcPromptLogin               = "login"
	oidcPromptConsent             = "consent"
	oidcErrorLoginRequired        = "login_required"
	oidcErrorInteractionRequired  = "interaction_required"
	oidcErrorInvalidAuthorizeFlow = "invalid_request"
)

type auditRecorder func(context.Context, audit.Event)

type Service struct {
	openPlatform openPlatformGateway
	rdb          *redis.Client
	signer       *Signer
	issuer       string
	codeTTL      time.Duration
	accessTTL    time.Duration
	refreshTTL   time.Duration
	audit        auditRecorder
}

type openPlatformGateway interface {
	BeginAuthorization(ctx context.Context, req openplatform.AuthorizeRequest, userID int64) (*openplatform.AuthorizationDecision, error)
	AuthorizeAppByClientID(ctx context.Context, clientID string) (*openplatform.App, error)
	LoadConsentChallenge(ctx context.Context, token string) (*openplatform.ConsentChallenge, error)
	AppByID(ctx context.Context, appID int64) (*openplatform.App, error)
	UserProjection(ctx context.Context, userID int64) (*openplatform.UserProjection, error)
	UserInfoForIdentityToken(ctx context.Context, clientID string, userID int64, subject string, scopes []string) (map[string]any, error)
	IdentityAccessTokenActive(ctx context.Context, clientID string, userID int64, scopes []string) (bool, error)
	IdentityAuthorizationFingerprint(ctx context.Context, clientID string, userID int64, scopes []string) (string, bool, error)
	IdentityClientCredentialsTokenActive(ctx context.Context, clientID string, scopes []string) (bool, error)
	DeleteConsentChallenge(ctx context.Context, token string) error
	VerifyClientSecret(ctx context.Context, clientID, clientSecret string) (*openplatform.App, error)
}

func NewService(
	openPlatform openPlatformGateway,
	rdb *redis.Client,
	signer *Signer,
	issuer string,
	codeTTL time.Duration,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) (*Service, error) {
	if openPlatform == nil {
		return nil, fmt.Errorf("identityserver.NewService: open platform service is required")
	}
	if rdb == nil {
		return nil, fmt.Errorf("identityserver.NewService: redis client is required")
	}
	if signer == nil {
		return nil, fmt.Errorf("identityserver.NewService: signer is required")
	}
	if strings.TrimSpace(issuer) == "" {
		return nil, fmt.Errorf("identityserver.NewService: issuer is required")
	}
	if codeTTL <= 0 {
		codeTTL = 5 * time.Minute
	}
	if accessTTL <= 0 {
		accessTTL = 15 * time.Minute
	}
	if refreshTTL <= 0 {
		refreshTTL = defaultRefreshTokenTTL
	}
	return &Service{
		openPlatform: openPlatform,
		rdb:          rdb,
		signer:       signer,
		issuer:       strings.TrimRight(strings.TrimSpace(issuer), "/"),
		codeTTL:      codeTTL,
		accessTTL:    accessTTL,
		refreshTTL:   refreshTTL,
		audit:        audit.LogContext,
	}, nil
}

func (s *Service) Begin(ctx context.Context, req AuthorizeRequest, userID int64) (string, error) {
	if err := validateAuthorizeRequest(req); err != nil {
		return "", ErrInvalidAuthorizeRequest
	}
	decision, err := s.openPlatform.BeginAuthorization(ctx, openplatform.AuthorizeRequest{
		ClientID:            req.ClientID,
		RedirectURI:         req.RedirectURI,
		Scopes:              strings.Fields(req.Scope),
		State:               req.State,
		Flow:                openplatform.AuthorizeFlowIdentity,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		Nonce:               req.Nonce,
		PromptNone:          promptNoneRequested(req),
		ForceConsent:        promptConsentRequested(req),
	}, userID)
	if err != nil {
		return "", err
	}
	if decision.InteractionRequired {
		if decision.App == nil {
			return "", ErrInvalidAuthorizeRequest
		}
		if !openplatform.RedirectURIAllowed(decision.App, req.RedirectURI) {
			return "", openplatform.ErrRedirectURINotAllowed
		}
		code := strings.TrimSpace(decision.InteractionError)
		if code == "" {
			code = oidcErrorInteractionRequired
		}
		return appendOAuthError(req.RedirectURI, code, req.State, s.issuer), nil
	}
	if decision.ProfileCompletionURL != "" {
		return decision.ProfileCompletionURL, nil
	}
	if decision.ConsentURL != "" {
		return decision.ConsentURL, nil
	}
	if decision.App == nil {
		return "", ErrInvalidAuthorizeRequest
	}
	if !openplatform.RedirectURIAllowed(decision.App, req.RedirectURI) {
		return "", openplatform.ErrRedirectURINotAllowed
	}
	subject := identitySubject(decision.UserID)
	return s.issueCodeRedirect(ctx, decision.App.ClientID, req.RedirectURI, grantedOAuthScopes(decision.OAuthScopes, decision.Scopes), decision.UserID, subject, req)
}

func (s *Service) AuthorizationErrorRedirect(ctx context.Context, req AuthorizeRequest, code string) (string, error) {
	if err := validateAuthorizeRequest(req); err != nil {
		return "", ErrInvalidAuthorizeRequest
	}
	app, err := s.openPlatform.AuthorizeAppByClientID(ctx, req.ClientID)
	if err != nil {
		return "", err
	}
	if !openplatform.RedirectURIAllowed(app, req.RedirectURI) {
		return "", openplatform.ErrRedirectURINotAllowed
	}
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		trimmed = oidcErrorInvalidAuthorizeFlow
	}
	return appendOAuthError(req.RedirectURI, trimmed, req.State, s.issuer), nil
}

func (s *Service) EndSessionRedirect(ctx context.Context, req LogoutRequest) (string, bool, error) {
	clientID, err := s.logoutClientID(req)
	if err != nil {
		return "", false, err
	}
	redirectURI := strings.TrimSpace(req.PostLogoutRedirectURI)
	if redirectURI == "" {
		return "", false, nil
	}
	if strings.TrimSpace(clientID) == "" {
		return "", false, ErrInvalidLogoutRequest
	}
	app, err := s.openPlatform.AuthorizeAppByClientID(ctx, clientID)
	if err != nil {
		return "", false, err
	}
	if !openplatform.RedirectURIAllowed(app, redirectURI) {
		return "", false, openplatform.ErrRedirectURINotAllowed
	}
	return appendOAuthState(redirectURI, req.State), true, nil
}

func (s *Service) logoutClientID(req LogoutRequest) (string, error) {
	clientID := strings.TrimSpace(req.ClientID)
	hint := strings.TrimSpace(req.IDTokenHint)
	if hint == "" {
		return clientID, nil
	}
	claims, err := s.signer.VerifyIDToken(hint)
	if err != nil {
		return "", ErrInvalidLogoutRequest
	}
	if clientID != "" && clientID != claims.ClientID {
		return "", ErrInvalidLogoutRequest
	}
	return claims.ClientID, nil
}

func (s *Service) IssueCodeFromConsentChallenge(ctx context.Context, token string, userID int64, authTime time.Time) (string, error) {
	challenge, err := s.openPlatform.LoadConsentChallenge(ctx, token)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(challenge.Flow) != openplatform.AuthorizeFlowIdentity {
		return "", openplatform.ErrConsentTokenInvalid
	}
	if err := validatePKCEChallenge(challenge.CodeChallenge, challenge.CodeChallengeMethod); err != nil {
		return "", ErrInvalidAuthorizeRequest
	}
	if challenge.UserID != userID {
		return "", openplatform.ErrConsentTokenInvalid
	}
	app, err := s.appForChallenge(ctx, challenge.AppID)
	if err != nil {
		return "", err
	}
	if !openplatform.RedirectURIAllowed(app, challenge.RedirectURI) {
		return "", openplatform.ErrRedirectURINotAllowed
	}
	subject := identitySubject(challenge.UserID)
	projection, err := s.openPlatform.UserProjection(ctx, challenge.UserID)
	if err != nil {
		return "", err
	}
	if missing := openplatform.RequiredProfileFields(projection, openplatform.UserConsentScopes(challenge.Scopes)); len(missing) > 0 {
		return "", openplatform.ErrProfileIncomplete
	}
	grantedScopes := grantedOAuthScopes(challenge.OAuthScopes, challenge.Scopes)
	if _, err := s.currentAuthorizationFingerprint(ctx, app.ClientID, challenge.UserID, grantedScopes); err != nil {
		return "", err
	}
	if err := s.openPlatform.DeleteConsentChallenge(ctx, token); err != nil {
		return "", err
	}
	return s.issueCodeRedirect(ctx, app.ClientID, challenge.RedirectURI, grantedScopes, challenge.UserID, subject, AuthorizeRequest{
		ClientID:            app.ClientID,
		RedirectURI:         challenge.RedirectURI,
		State:               challenge.State,
		CodeChallenge:       challenge.CodeChallenge,
		CodeChallengeMethod: challenge.CodeChallengeMethod,
		Nonce:               challenge.Nonce,
		AuthTime:            authTime,
	})
}

func grantedOAuthScopes(oauthScopes, fallback []string) []string {
	if len(oauthScopes) > 0 {
		return append([]string(nil), oauthScopes...)
	}
	return append([]string(nil), fallback...)
}

func (s *Service) Token(ctx context.Context, input TokenRequest) (map[string]any, error) {
	switch strings.TrimSpace(input.GrantType) {
	case "authorization_code":
		return s.ExchangeCode(ctx, input)
	case "refresh_token":
		return s.Refresh(ctx, input)
	case "client_credentials":
		return s.ClientCredentials(ctx, input)
	default:
		return nil, ErrUnsupportedGrantType
	}
}

func (s *Service) ExchangeCode(ctx context.Context, input TokenRequest) (map[string]any, error) {
	if strings.TrimSpace(input.GrantType) != "authorization_code" {
		return nil, ErrUnsupportedGrantType
	}
	app, err := s.openPlatform.VerifyClientSecret(ctx, input.ClientID, input.ClientSecret)
	if err != nil {
		return nil, ErrInvalidClient
	}
	code, err := s.consumeAuthorizationCode(ctx, input.Code)
	if err != nil {
		return nil, ErrInvalidGrant
	}
	if code.ClientID != app.ClientID {
		return nil, ErrInvalidGrant
	}
	if offlineAccessWithoutOpenID(code.Scopes) {
		return nil, ErrInvalidGrant
	}
	if strings.TrimSpace(input.RedirectURI) == "" || input.RedirectURI != code.RedirectURI {
		return nil, ErrInvalidGrant
	}
	if err := verifyPKCE(code.CodeChallenge, code.CodeChallengeMethod, input.CodeVerifier); err != nil {
		return nil, ErrInvalidGrant
	}
	authorizationFingerprint, err := s.currentAuthorizationFingerprint(ctx, app.ClientID, code.UserID, code.Scopes)
	if err != nil {
		return nil, err
	}
	accessToken, _, err := s.signer.SignAccessToken(AccessTokenInput{
		Subject:                  code.Subject,
		ClientID:                 app.ClientID,
		UserID:                   code.UserID,
		Scopes:                   code.Scopes,
		AuthorizationFingerprint: authorizationFingerprint,
		TTL:                      s.accessTTL,
	})
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   int(s.accessTTL.Seconds()),
		"scope":        strings.Join(code.Scopes, " "),
	}
	if scopeListContains(code.Scopes, "openid") {
		idToken, err := s.issueIDToken(ctx, app.ClientID, code.UserID, code.Subject, code.Scopes, code.Nonce, code.AuthTime)
		if err != nil {
			return nil, err
		}
		payload["id_token"] = idToken
	}
	if hasOfflineAccess(code.Scopes) {
		refreshToken, err := s.issueRefreshToken(ctx, RefreshToken{
			ClientID:                 code.ClientID,
			AuthorizationFingerprint: authorizationFingerprint,
			Scopes:                   code.Scopes,
			UserID:                   code.UserID,
			Subject:                  code.Subject,
			Nonce:                    code.Nonce,
			AuthTime:                 code.AuthTime,
		})
		if err != nil {
			return nil, err
		}
		payload["refresh_token"] = refreshToken
	}
	return payload, nil
}

func (s *Service) Refresh(ctx context.Context, input TokenRequest) (map[string]any, error) {
	if strings.TrimSpace(input.GrantType) != "refresh_token" {
		return nil, ErrUnsupportedGrantType
	}
	app, err := s.openPlatform.VerifyClientSecret(ctx, input.ClientID, input.ClientSecret)
	if err != nil {
		return nil, ErrInvalidClient
	}
	refreshToken, err := s.loadCurrentRefreshToken(ctx, input.RefreshToken)
	if err != nil {
		if replayErr := s.revokeRefreshTokenFamilyOnReplay(ctx, input.RefreshToken, app.ClientID); replayErr != nil {
			return nil, ErrInvalidGrant
		}
		return nil, ErrInvalidGrant
	}
	if refreshToken.ClientID != app.ClientID || !usableRefreshTokenScopes(refreshToken.Scopes) {
		return nil, ErrInvalidGrant
	}
	grantedScopes, err := refreshGrantScopes(input.Scope, refreshToken.Scopes)
	if err != nil {
		return nil, err
	}
	active, err := s.identityAuthorizationFingerprintActive(
		ctx,
		app.ClientID,
		refreshToken.UserID,
		refreshToken.Scopes,
		refreshToken.AuthorizationFingerprint,
	)
	if err != nil || !active {
		return nil, ErrInvalidGrant
	}
	refreshToken, err = s.consumeCurrentRefreshToken(ctx, input.RefreshToken)
	if err != nil {
		if replayErr := s.revokeRefreshTokenFamilyOnReplay(ctx, input.RefreshToken, app.ClientID); replayErr != nil {
			return nil, ErrInvalidGrant
		}
		return nil, ErrInvalidGrant
	}
	if refreshToken.ClientID != app.ClientID || !usableRefreshTokenScopes(refreshToken.Scopes) {
		return nil, ErrInvalidGrant
	}
	active, err = s.identityAuthorizationFingerprintActive(
		ctx,
		app.ClientID,
		refreshToken.UserID,
		refreshToken.Scopes,
		refreshToken.AuthorizationFingerprint,
	)
	if err != nil || !active {
		return nil, ErrInvalidGrant
	}
	authorizationFingerprint, err := s.currentAuthorizationFingerprint(ctx, app.ClientID, refreshToken.UserID, grantedScopes)
	if err != nil {
		return nil, ErrInvalidGrant
	}
	idToken := ""
	if scopeListContains(grantedScopes, "openid") {
		idToken, err = s.issueIDToken(ctx, app.ClientID, refreshToken.UserID, refreshToken.Subject, grantedScopes, refreshToken.Nonce, refreshToken.AuthTime)
		if err != nil {
			return nil, ErrInvalidGrant
		}
	}
	accessToken, _, err := s.signer.SignAccessToken(AccessTokenInput{
		Subject:                  refreshToken.Subject,
		ClientID:                 app.ClientID,
		UserID:                   refreshToken.UserID,
		Scopes:                   grantedScopes,
		AuthorizationFingerprint: authorizationFingerprint,
		TTL:                      s.accessTTL,
	})
	if err != nil {
		return nil, err
	}
	rotatedRefreshToken, err := s.issueRefreshToken(ctx, RefreshToken{
		ClientID:                 app.ClientID,
		FamilyID:                 refreshToken.FamilyID,
		Generation:               refreshToken.Generation + 1,
		AuthorizationFingerprint: authorizationFingerprint,
		Scopes:                   grantedScopes,
		UserID:                   refreshToken.UserID,
		Subject:                  refreshToken.Subject,
		Nonce:                    refreshToken.Nonce,
		AuthTime:                 refreshToken.AuthTime,
	})
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"access_token":  accessToken,
		"refresh_token": rotatedRefreshToken,
		"token_type":    "Bearer",
		"expires_in":    int(s.accessTTL.Seconds()),
		"scope":         strings.Join(grantedScopes, " "),
	}
	if idToken != "" {
		payload["id_token"] = idToken
	}
	return payload, nil
}

func (s *Service) currentAuthorizationFingerprint(ctx context.Context, clientID string, userID int64, scopes []string) (string, error) {
	fingerprint, active, err := s.openPlatform.IdentityAuthorizationFingerprint(ctx, clientID, userID, scopes)
	if err != nil || !active {
		return "", ErrInvalidGrant
	}
	return strings.TrimSpace(fingerprint), nil
}

func (s *Service) identityAuthorizationFingerprintActive(
	ctx context.Context,
	clientID string,
	userID int64,
	scopes []string,
	expectedFingerprint string,
) (bool, error) {
	fingerprint, active, err := s.openPlatform.IdentityAuthorizationFingerprint(ctx, clientID, userID, scopes)
	if err != nil || !active {
		return false, err
	}
	return strings.TrimSpace(fingerprint) == strings.TrimSpace(expectedFingerprint), nil
}

func (s *Service) issueIDToken(ctx context.Context, clientID string, userID int64, subject string, scopes []string, nonce string, authTime time.Time) (string, error) {
	profile, err := s.openPlatform.UserInfoForIdentityToken(ctx, clientID, userID, subject, scopes)
	if err != nil {
		return "", err
	}
	return s.signer.SignIDToken(IDTokenInput{
		Subject:  subject,
		ClientID: clientID,
		Scopes:   scopes,
		Nonce:    nonce,
		AuthTime: authTime,
		Profile:  idTokenProfileClaims(profile),
		TTL:      s.accessTTL,
	})
}

func refreshGrantScopes(raw string, original []string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return append([]string(nil), original...), nil
	}
	requested, err := openplatform.NormalizeGrantedOAuthScopes(strings.Fields(raw))
	if err != nil {
		return nil, ErrInvalidScope
	}
	if !scopeListContains(requested, "openid") || !scopeListContains(requested, openplatform.ScopeOfflineAccess) {
		return nil, ErrInvalidScope
	}
	if !scopeListSubset(requested, original) {
		return nil, ErrInvalidScope
	}
	return requested, nil
}

func scopeListContains(scopes []string, target string) bool {
	for _, scope := range scopes {
		if strings.TrimSpace(scope) == target {
			return true
		}
	}
	return false
}

func scopeListSubset(requested, original []string) bool {
	granted := make(map[string]struct{}, len(original))
	for _, scope := range original {
		trimmed := strings.TrimSpace(scope)
		if trimmed != "" {
			granted[trimmed] = struct{}{}
		}
	}
	for _, scope := range requested {
		if _, ok := granted[strings.TrimSpace(scope)]; !ok {
			return false
		}
	}
	return true
}

func (s *Service) ClientCredentials(ctx context.Context, input TokenRequest) (map[string]any, error) {
	if strings.TrimSpace(input.GrantType) != "client_credentials" {
		return nil, ErrUnsupportedGrantType
	}
	app, err := s.openPlatform.VerifyClientSecret(ctx, input.ClientID, input.ClientSecret)
	if err != nil {
		return nil, ErrInvalidClient
	}
	scopes, err := clientCredentialsScopes(input.Scope)
	if err != nil {
		return nil, err
	}
	active, err := s.openPlatform.IdentityClientCredentialsTokenActive(ctx, app.ClientID, scopes)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, ErrInvalidScope
	}
	accessToken, _, err := s.signer.SignAccessToken(AccessTokenInput{
		Subject:   clientCredentialsSubject(app.ClientID),
		ClientID:  app.ClientID,
		UserID:    0,
		Scopes:    scopes,
		GrantType: "client_credentials",
		TTL:       s.accessTTL,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   int(s.accessTTL.Seconds()),
		"scope":        strings.Join(scopes, " "),
	}, nil
}

func clientCredentialsScopes(raw string) ([]string, error) {
	scopes := strings.Fields(raw)
	if len(scopes) == 0 {
		return nil, ErrInvalidScope
	}
	normalized, err := openplatform.NormalizeScopes(scopes)
	if err != nil {
		return nil, ErrInvalidScope
	}
	for _, scope := range normalized {
		if !clientCredentialsScopeAllowed(scope) {
			return nil, ErrInvalidScope
		}
	}
	return normalized, nil
}

func clientCredentialsScopeAllowed(scope string) bool {
	switch scope {
	case openplatform.ScopeResourceRead, openplatform.ScopeResourceWrite:
		return true
	default:
		return false
	}
}

func clientCredentialsSubject(clientID string) string {
	return "client:" + strings.TrimSpace(clientID)
}

type TokenRequest struct {
	GrantType    string
	Code         string
	RedirectURI  string
	CodeVerifier string
	RefreshToken string
	Scope        string
	ClientID     string
	ClientSecret string
}

type refreshTokenFamily struct {
	ClientID   string `json:"clientID"`
	CurrentKey string `json:"currentKey"`
}

var (
	ErrInvalidClient           = errors.New("invalid_client")
	ErrInvalidGrant            = errors.New("invalid_grant")
	ErrInvalidScope            = errors.New("invalid_scope")
	ErrInvalidAuthorizeRequest = errors.New("invalid_authorize_request")
	ErrInvalidLogoutRequest    = errors.New("invalid_logout_request")
	ErrUnsupportedGrantType    = errors.New("unsupported_grant_type")
)

func (s *Service) UserInfo(ctx context.Context, rawToken string) (map[string]any, error) {
	claims, err := s.verifyUsableAccessToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	if claims.GrantType == "client_credentials" || claims.UserID <= 0 {
		return nil, errors.New("identity client credentials token cannot access userinfo")
	}
	if !scopeListContains(claims.Scopes, "openid") {
		return nil, errors.New("identity access token missing openid scope")
	}
	active, err := s.identityAuthorizationFingerprintActive(
		ctx,
		claims.ClientID,
		claims.UserID,
		claims.Scopes,
		claims.AuthorizationFingerprint,
	)
	if err != nil || !active {
		return nil, errors.New("identity access token authorization is inactive")
	}
	payload, err := s.openPlatform.UserInfoForIdentityToken(ctx, claims.ClientID, claims.UserID, claims.Subject, claims.Scopes)
	if err != nil {
		return nil, err
	}
	addOIDCProfileAliases(payload)
	return payload, nil
}

func (s *Service) VerifyOpenPlatformResourceAccessToken(ctx context.Context, rawToken string) (openplatform.ResourceAccessToken, error) {
	claims, err := s.verifyUsableAccessToken(ctx, rawToken)
	if err != nil {
		return openplatform.ResourceAccessToken{}, ErrInvalidGrant
	}
	if claims.GrantType != "client_credentials" || claims.UserID != 0 {
		return openplatform.ResourceAccessToken{}, ErrInvalidGrant
	}
	active, err := s.openPlatform.IdentityClientCredentialsTokenActive(ctx, claims.ClientID, claims.Scopes)
	if err != nil {
		return openplatform.ResourceAccessToken{}, err
	}
	if !active {
		return openplatform.ResourceAccessToken{}, ErrInvalidGrant
	}
	return openplatform.ResourceAccessToken{
		ClientID: claims.ClientID,
		Scopes:   append([]string(nil), claims.Scopes...),
	}, nil
}

func (s *Service) Introspect(ctx context.Context, rawToken string, requesterClientID string) map[string]any {
	claims, err := s.verifyUsableAccessToken(ctx, rawToken)
	if err != nil {
		return s.introspectRefreshToken(ctx, rawToken, requesterClientID)
	}
	if strings.TrimSpace(requesterClientID) == "" || claims.ClientID != strings.TrimSpace(requesterClientID) {
		return map[string]any{"active": false}
	}
	if claims.GrantType == "client_credentials" {
		return s.introspectClientCredentialsAccessToken(ctx, claims)
	}
	active, err := s.identityAuthorizationFingerprintActive(
		ctx,
		claims.ClientID,
		claims.UserID,
		claims.Scopes,
		claims.AuthorizationFingerprint,
	)
	if err != nil || !active {
		return map[string]any{"active": false}
	}
	return map[string]any{
		"active":     true,
		"iss":        s.issuer,
		"sub":        claims.Subject,
		"aud":        claims.ClientID,
		"client_id":  claims.ClientID,
		"scope":      strings.Join(claims.Scopes, " "),
		"token_type": "Bearer",
		"token_kind": "access_token",
		"exp":        claims.Expires.Unix(),
		"iat":        claims.IssuedAt.Unix(),
	}
}

func (s *Service) introspectClientCredentialsAccessToken(ctx context.Context, claims AccessTokenClaims) map[string]any {
	active, err := s.openPlatform.IdentityClientCredentialsTokenActive(ctx, claims.ClientID, claims.Scopes)
	if err != nil || !active {
		return map[string]any{"active": false}
	}
	return map[string]any{
		"active":     true,
		"iss":        s.issuer,
		"sub":        claims.Subject,
		"aud":        claims.ClientID,
		"client_id":  claims.ClientID,
		"scope":      strings.Join(claims.Scopes, " "),
		"token_type": "Bearer",
		"token_kind": "access_token",
		"grant_type": "client_credentials",
		"exp":        claims.Expires.Unix(),
		"iat":        claims.IssuedAt.Unix(),
	}
}

func (s *Service) introspectRefreshToken(ctx context.Context, rawToken string, requesterClientID string) map[string]any {
	refreshToken, err := s.loadCurrentRefreshToken(ctx, rawToken)
	if err != nil {
		return map[string]any{"active": false}
	}
	if strings.TrimSpace(requesterClientID) == "" || refreshToken.ClientID != strings.TrimSpace(requesterClientID) {
		return map[string]any{"active": false}
	}
	if !usableRefreshTokenScopes(refreshToken.Scopes) {
		return map[string]any{"active": false}
	}
	active, err := s.identityAuthorizationFingerprintActive(
		ctx,
		refreshToken.ClientID,
		refreshToken.UserID,
		refreshToken.Scopes,
		refreshToken.AuthorizationFingerprint,
	)
	if err != nil || !active {
		return map[string]any{"active": false}
	}
	return map[string]any{
		"active":     true,
		"iss":        s.issuer,
		"sub":        refreshToken.Subject,
		"aud":        refreshToken.ClientID,
		"client_id":  refreshToken.ClientID,
		"scope":      strings.Join(refreshToken.Scopes, " "),
		"token_type": "refresh_token",
		"token_kind": "refresh_token",
		"exp":        refreshToken.ExpiresAt.Unix(),
		"iat":        refreshToken.CreatedAt.Unix(),
	}
}

func (s *Service) Revoke(ctx context.Context, rawToken string, requesterClientID string, tokenTypeHint ...string) error {
	hint := ""
	if len(tokenTypeHint) > 0 {
		hint = strings.TrimSpace(tokenTypeHint[0])
	}
	if hint == "refresh_token" {
		handled, err := s.revokeRefreshToken(ctx, rawToken, requesterClientID)
		if err != nil || handled {
			return err
		}
		_, err = s.revokeAccessToken(ctx, rawToken, requesterClientID)
		return err
	}
	handled, err := s.revokeAccessToken(ctx, rawToken, requesterClientID)
	if err != nil || handled {
		return err
	}
	_, err = s.revokeRefreshToken(ctx, rawToken, requesterClientID)
	return err
}

func (s *Service) revokeAccessToken(ctx context.Context, rawToken string, requesterClientID string) (bool, error) {
	claims, err := s.signer.VerifyAccessToken(rawToken)
	if err != nil {
		return false, nil
	}
	if strings.TrimSpace(requesterClientID) == "" || claims.ClientID != strings.TrimSpace(requesterClientID) {
		return true, nil
	}
	ttl := time.Until(claims.Expires)
	if ttl <= 0 {
		return true, nil
	}
	if err := s.rdb.Set(ctx, revokedRedisPrefix+claims.JTI, "1", ttl).Err(); err != nil {
		return true, err
	}
	s.recordAccessTokenRevocationAudit(ctx, claims)
	return true, nil
}

func (s *Service) verifyUsableAccessToken(ctx context.Context, rawToken string) (AccessTokenClaims, error) {
	claims, err := s.signer.VerifyAccessToken(rawToken)
	if err != nil {
		return AccessTokenClaims{}, err
	}
	revoked, err := s.isRevoked(ctx, claims.JTI)
	if err != nil {
		return AccessTokenClaims{}, err
	}
	if revoked {
		return AccessTokenClaims{}, errors.New("identity token revoked")
	}
	return claims, nil
}

func (s *Service) isRevoked(ctx context.Context, jti string) (bool, error) {
	raw, err := s.rdb.Get(ctx, revokedRedisPrefix+jti).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return raw != "", nil
}

func (s *Service) issueCodeRedirect(ctx context.Context, clientID, redirectURI string, scopes []string, userID int64, subject string, req AuthorizeRequest) (string, error) {
	if clientID == "" {
		clientID = strings.TrimSpace(req.ClientID)
	}
	if subject == "" {
		subject = identitySubject(userID)
	}
	code, err := randomCode()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	payload := AuthorizationCode{
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		Scopes:              scopes,
		UserID:              userID,
		Subject:             subject,
		CodeChallenge:       strings.TrimSpace(req.CodeChallenge),
		CodeChallengeMethod: strings.TrimSpace(req.CodeChallengeMethod),
		Nonce:               strings.TrimSpace(req.Nonce),
		AuthTime:            authTimeOrNow(req.AuthTime, now),
		CreatedAt:           now,
		ExpiresAt:           now.Add(s.codeTTL),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal identity authorization code: %w", err)
	}
	if err := s.rdb.Set(ctx, authCodeRedisPrefix+code, raw, s.codeTTL).Err(); err != nil {
		return "", fmt.Errorf("store identity authorization code: %w", err)
	}
	return appendOAuthCode(redirectURI, code, req.State, s.issuer), nil
}

func (s *Service) consumeAuthorizationCode(ctx context.Context, code string) (*AuthorizationCode, error) {
	raw, err := s.rdb.GetDel(ctx, authCodeRedisPrefix+strings.TrimSpace(code)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrInvalidGrant
	}
	if err != nil {
		return nil, err
	}
	var payload AuthorizationCode
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	if time.Now().After(payload.ExpiresAt) {
		return nil, ErrInvalidGrant
	}
	return &payload, nil
}

func (s *Service) issueRefreshToken(ctx context.Context, input RefreshToken) (string, error) {
	token, err := randomRefreshToken()
	if err != nil {
		return "", err
	}
	familyID := strings.TrimSpace(input.FamilyID)
	if familyID == "" {
		familyID, err = randomRefreshToken()
		if err != nil {
			return "", err
		}
	}
	generation := input.Generation
	if generation < 0 {
		generation = 0
	}
	now := time.Now().UTC()
	payload := RefreshToken{
		ClientID:                 strings.TrimSpace(input.ClientID),
		FamilyID:                 familyID,
		Generation:               generation,
		AuthorizationFingerprint: strings.TrimSpace(input.AuthorizationFingerprint),
		Scopes:                   append([]string(nil), input.Scopes...),
		UserID:                   input.UserID,
		Subject:                  strings.TrimSpace(input.Subject),
		Nonce:                    strings.TrimSpace(input.Nonce),
		AuthTime:                 authTimeOrNow(input.AuthTime, now),
		CreatedAt:                now,
		ExpiresAt:                now.Add(s.refreshTTL),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal identity refresh token: %w", err)
	}
	tokenKey := refreshTokenKey(token)
	if err := s.rdb.Set(ctx, tokenKey, raw, s.refreshTTL).Err(); err != nil {
		return "", fmt.Errorf("store identity refresh token: %w", err)
	}
	if err := s.storeRefreshTokenFamily(ctx, payload, tokenKey); err != nil {
		if cleanupErr := s.rdb.Del(ctx, tokenKey).Err(); cleanupErr != nil {
			return "", fmt.Errorf("store identity refresh token family: %w; cleanup refresh token: %v", err, cleanupErr)
		}
		return "", err
	}
	return token, nil
}

func (s *Service) consumeRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	tokenKey := refreshTokenKey(token)
	raw, err := s.rdb.GetDel(ctx, tokenKey).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrInvalidGrant
	}
	if err != nil {
		return nil, err
	}
	payload, err := decodeRefreshToken([]byte(raw))
	if err != nil {
		return nil, ErrInvalidGrant
	}
	if err := s.storeUsedRefreshToken(ctx, token, *payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *Service) consumeCurrentRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	payload, err := s.consumeRefreshToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if err := s.ensureRefreshTokenFamilyCurrent(ctx, *payload, refreshTokenKey(token)); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *Service) loadRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	raw, err := s.rdb.Get(ctx, refreshTokenKey(token)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrInvalidGrant
	}
	if err != nil {
		return nil, err
	}
	return decodeRefreshToken([]byte(raw))
}

func (s *Service) loadCurrentRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	payload, err := s.loadRefreshToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if err := s.ensureRefreshTokenFamilyCurrent(ctx, *payload, refreshTokenKey(token)); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *Service) ensureRefreshTokenFamilyCurrent(ctx context.Context, token RefreshToken, tokenKey string) error {
	familyID := strings.TrimSpace(token.FamilyID)
	if familyID == "" {
		return ErrInvalidGrant
	}
	raw, err := s.rdb.Get(ctx, refreshTokenFamilyKey(familyID)).Result()
	if errors.Is(err, redis.Nil) {
		return ErrInvalidGrant
	}
	if err != nil {
		return err
	}
	var family refreshTokenFamily
	if err := json.Unmarshal([]byte(raw), &family); err != nil {
		return ErrInvalidGrant
	}
	if strings.TrimSpace(family.ClientID) != strings.TrimSpace(token.ClientID) ||
		strings.TrimSpace(family.CurrentKey) != strings.TrimSpace(tokenKey) {
		return ErrInvalidGrant
	}
	return nil
}

func decodeRefreshToken(raw []byte) (*RefreshToken, error) {
	var payload RefreshToken
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	if time.Now().After(payload.ExpiresAt) {
		return nil, ErrInvalidGrant
	}
	return &payload, nil
}

func (s *Service) storeUsedRefreshToken(ctx context.Context, token string, payload RefreshToken) error {
	ttl := time.Until(payload.ExpiresAt)
	if ttl <= 0 {
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal used identity refresh token: %w", err)
	}
	return s.rdb.Set(ctx, refreshTokenUsedKey(token), raw, ttl).Err()
}

func (s *Service) loadUsedRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	raw, err := s.rdb.Get(ctx, refreshTokenUsedKey(token)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrInvalidGrant
	}
	if err != nil {
		return nil, err
	}
	return decodeRefreshToken([]byte(raw))
}

func (s *Service) storeRefreshTokenFamily(ctx context.Context, token RefreshToken, currentKey string) error {
	familyID := strings.TrimSpace(token.FamilyID)
	if familyID == "" {
		return nil
	}
	raw, err := json.Marshal(refreshTokenFamily{
		ClientID:   strings.TrimSpace(token.ClientID),
		CurrentKey: strings.TrimSpace(currentKey),
	})
	if err != nil {
		return fmt.Errorf("marshal identity refresh token family: %w", err)
	}
	return s.rdb.Set(ctx, refreshTokenFamilyKey(familyID), raw, s.refreshTTL).Err()
}

func (s *Service) revokeRefreshTokenFamilyOnReplay(ctx context.Context, token string, requesterClientID string) error {
	used, err := s.loadUsedRefreshToken(ctx, token)
	if err != nil {
		return nil
	}
	if strings.TrimSpace(requesterClientID) == "" || used.ClientID != strings.TrimSpace(requesterClientID) {
		return nil
	}
	if err := s.revokeRefreshTokenFamily(ctx, used.FamilyID, used.ClientID); err != nil {
		return err
	}
	s.recordRefreshTokenReplayAudit(ctx, *used)
	return nil
}

func (s *Service) recordRefreshTokenReplayAudit(ctx context.Context, token RefreshToken) {
	if s.audit == nil {
		return
	}
	familyHash := refreshTokenFamilyAuditID(token.FamilyID)
	s.audit(ctx, audit.Event{
		Type:         audit.EventTokenRevoked,
		ActorType:    "system",
		UserID:       strconv.FormatInt(token.UserID, 10),
		ResourceType: "identity.refresh_token_family",
		ResourceID:   familyHash,
		Action:       "identity_refresh_reuse_detected",
		Result:       "failure",
		Reason:       "refresh token reuse detected",
		Details: map[string]any{
			"clientID":   token.ClientID,
			"familyHash": familyHash,
			"generation": token.Generation,
			"scopeCount": len(token.Scopes),
		},
	})
}

func (s *Service) recordAccessTokenRevocationAudit(ctx context.Context, claims AccessTokenClaims) {
	if s.audit == nil {
		return
	}
	s.audit(ctx, audit.Event{
		Type:         audit.EventTokenRevoked,
		ActorType:    "client",
		UserID:       strconv.FormatInt(claims.UserID, 10),
		ResourceType: "identity.access_token",
		ResourceID:   strings.TrimSpace(claims.JTI),
		Action:       "identity_access_token_revoked",
		Result:       "success",
		Reason:       "revoked by client",
		Details: map[string]any{
			"clientID":   claims.ClientID,
			"scopeCount": len(claims.Scopes),
			"tokenType":  "access_token",
		},
	})
}

func (s *Service) recordRefreshTokenRevocationAudit(ctx context.Context, token RefreshToken) {
	if s.audit == nil {
		return
	}
	familyHash := refreshTokenFamilyAuditID(token.FamilyID)
	s.audit(ctx, audit.Event{
		Type:         audit.EventTokenRevoked,
		ActorType:    "client",
		UserID:       strconv.FormatInt(token.UserID, 10),
		ResourceType: "identity.refresh_token_family",
		ResourceID:   familyHash,
		Action:       "identity_refresh_token_revoked",
		Result:       "success",
		Reason:       "revoked by client",
		Details: map[string]any{
			"clientID":   token.ClientID,
			"familyHash": familyHash,
			"generation": token.Generation,
			"scopeCount": len(token.Scopes),
			"tokenType":  "refresh_token",
		},
	})
}

func (s *Service) revokeRefreshTokenFamily(ctx context.Context, familyID string, requesterClientID string) error {
	familyID = strings.TrimSpace(familyID)
	if familyID == "" {
		return nil
	}
	familyKey := refreshTokenFamilyKey(familyID)
	raw, err := s.rdb.Get(ctx, familyKey).Result()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return err
	}
	var family refreshTokenFamily
	if err := json.Unmarshal([]byte(raw), &family); err != nil {
		return nil
	}
	if strings.TrimSpace(requesterClientID) == "" || family.ClientID != strings.TrimSpace(requesterClientID) {
		return nil
	}
	keys := []string{familyKey}
	if currentKey := strings.TrimSpace(family.CurrentKey); currentKey != "" {
		keys = append(keys, currentKey)
	}
	return s.rdb.Del(ctx, keys...).Err()
}

func (s *Service) revokeRefreshToken(ctx context.Context, token string, requesterClientID string) (bool, error) {
	if strings.TrimSpace(token) == "" {
		return false, nil
	}
	key := refreshTokenKey(token)
	raw, err := s.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return s.revokeUsedRefreshToken(ctx, token, requesterClientID)
	}
	if err != nil {
		return true, err
	}
	var payload RefreshToken
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return true, nil
	}
	if strings.TrimSpace(requesterClientID) == "" || payload.ClientID != strings.TrimSpace(requesterClientID) {
		return true, nil
	}
	keys := []string{key}
	if familyID := strings.TrimSpace(payload.FamilyID); familyID != "" {
		keys = append(keys, refreshTokenFamilyKey(familyID))
	}
	if err := s.rdb.Del(ctx, keys...).Err(); err != nil {
		return true, err
	}
	s.recordRefreshTokenRevocationAudit(ctx, payload)
	return true, nil
}

func (s *Service) revokeUsedRefreshToken(ctx context.Context, token string, requesterClientID string) (bool, error) {
	if strings.TrimSpace(token) == "" {
		return false, nil
	}
	used, err := s.loadUsedRefreshToken(ctx, token)
	if err != nil {
		return false, nil
	}
	if strings.TrimSpace(requesterClientID) == "" || used.ClientID != strings.TrimSpace(requesterClientID) {
		return true, nil
	}
	if err := s.revokeRefreshTokenFamily(ctx, used.FamilyID, used.ClientID); err != nil {
		return true, err
	}
	if err := s.rdb.Del(ctx, refreshTokenUsedKey(token)).Err(); err != nil {
		return true, err
	}
	s.recordRefreshTokenRevocationAudit(ctx, *used)
	return true, nil
}

func refreshTokenKey(token string) string {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return refreshTokenRedisPrefix
	}
	sum := sha256.Sum256([]byte(trimmed))
	return refreshTokenRedisPrefix + hex.EncodeToString(sum[:])
}

func refreshTokenUsedKey(token string) string {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return refreshTokenUsedPrefix
	}
	sum := sha256.Sum256([]byte(trimmed))
	return refreshTokenUsedPrefix + hex.EncodeToString(sum[:])
}

func refreshTokenFamilyKey(familyID string) string {
	return refreshTokenFamilyPrefix + strings.TrimSpace(familyID)
}

func refreshTokenFamilyAuditID(familyID string) string {
	trimmed := strings.TrimSpace(familyID)
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])
}

func (s *Service) appForChallenge(ctx context.Context, appID int64) (*openplatform.App, error) {
	return s.openPlatform.AppByID(ctx, appID)
}

func identitySubject(userID int64) string {
	if userID <= 0 {
		return ""
	}
	return "stuhelper:" + strconv.FormatInt(userID, 10)
}

func randomCode() (string, error) {
	bytes := make([]byte, codeBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("identity random code: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func randomRefreshToken() (string, error) {
	bytes := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("identity random refresh token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func hasOfflineAccess(scopes []string) bool {
	for _, scope := range scopes {
		if strings.TrimSpace(scope) == openplatform.ScopeOfflineAccess {
			return true
		}
	}
	return false
}

func usableRefreshTokenScopes(scopes []string) bool {
	return hasOfflineAccess(scopes) && !offlineAccessWithoutOpenID(scopes)
}

func authTimeOrNow(authTime, now time.Time) time.Time {
	if authTime.IsZero() {
		return now.UTC()
	}
	return authTime.UTC()
}

func validateAuthorizeRequest(req AuthorizeRequest) error {
	if strings.TrimSpace(req.ResponseType) != "code" {
		return ErrInvalidAuthorizeRequest
	}
	if err := validateResponseMode(req.ResponseMode); err != nil {
		return ErrInvalidAuthorizeRequest
	}
	if err := validatePrompt(req.Prompt); err != nil {
		return ErrInvalidAuthorizeRequest
	}
	if _, _, err := maxAgeDuration(req.MaxAge); err != nil {
		return ErrInvalidAuthorizeRequest
	}
	if err := validateAuthorizeScopes(req.Scope); err != nil {
		return ErrInvalidAuthorizeRequest
	}
	if err := validatePKCEChallenge(req.CodeChallenge, req.CodeChallengeMethod); err != nil {
		return ErrInvalidAuthorizeRequest
	}
	return nil
}

func validateAuthorizeScopes(raw string) error {
	oauthScopes, err := openplatform.NormalizeGrantedOAuthScopes(strings.Fields(raw))
	if err != nil {
		return err
	}
	if offlineAccessWithoutOpenID(oauthScopes) {
		return errors.New("offline_access requires openid")
	}
	return nil
}

func offlineAccessWithoutOpenID(scopes []string) bool {
	return scopeListContains(scopes, openplatform.ScopeOfflineAccess) && !scopeListContains(scopes, "openid")
}

func validateResponseMode(mode string) error {
	if mode != strings.TrimSpace(mode) {
		return fmt.Errorf("unsupported response_mode %q", mode)
	}
	if mode == "" || mode == "query" {
		return nil
	}
	return fmt.Errorf("unsupported response_mode %q", mode)
}

func validatePrompt(prompt string) error {
	values := strings.Fields(prompt)
	if len(values) == 0 {
		return nil
	}
	hasNone := false
	for _, value := range values {
		if value != oidcPromptNone && value != oidcPromptLogin && value != oidcPromptConsent {
			return fmt.Errorf("unsupported prompt value %q", value)
		}
		if value == oidcPromptNone {
			hasNone = true
		}
	}
	if hasNone && len(values) != 1 {
		return errors.New("prompt none cannot be combined with other prompt values")
	}
	return nil
}

func promptNoneRequested(req AuthorizeRequest) bool {
	values := strings.Fields(req.Prompt)
	return len(values) == 1 && values[0] == oidcPromptNone
}

func promptLoginRequested(req AuthorizeRequest) bool {
	for _, value := range strings.Fields(req.Prompt) {
		if value == oidcPromptLogin {
			return true
		}
	}
	return false
}

func promptConsentRequested(req AuthorizeRequest) bool {
	for _, value := range strings.Fields(req.Prompt) {
		if value == oidcPromptConsent {
			return true
		}
	}
	return false
}

func maxAgeDuration(raw string) (time.Duration, bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false, nil
	}
	seconds, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || seconds < 0 {
		return 0, true, errors.New("max_age must be a non-negative integer")
	}
	return time.Duration(seconds) * time.Second, true, nil
}

func validatePKCEChallenge(challenge, method string) error {
	if strings.TrimSpace(challenge) == "" {
		return errors.New("code_challenge is required")
	}
	if challenge != strings.TrimSpace(challenge) {
		return errors.New("code_challenge must be a valid S256 challenge")
	}
	if !strings.EqualFold(strings.TrimSpace(method), "S256") {
		return errors.New("code_challenge_method must be S256")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(challenge)
	if err != nil || len(decoded) != sha256.Size || len(challenge) != pkceS256Length {
		return errors.New("code_challenge must be a valid S256 challenge")
	}
	return nil
}

func verifyPKCE(challenge, method, verifier string) error {
	if strings.TrimSpace(challenge) == "" || challenge != strings.TrimSpace(challenge) {
		return ErrInvalidGrant
	}
	if !strings.EqualFold(strings.TrimSpace(method), "S256") || !validPKCEVerifier(verifier) {
		return ErrInvalidGrant
	}
	sum := sha256.Sum256([]byte(verifier))
	actual := base64.RawURLEncoding.EncodeToString(sum[:])
	if actual != challenge {
		return ErrInvalidGrant
	}
	return nil
}

func validPKCEVerifier(value string) bool {
	if len(value) < pkceVerifierMin || len(value) > pkceVerifierMax {
		return false
	}
	for _, r := range value {
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		switch r {
		case '-', '.', '_', '~':
			continue
		default:
			return false
		}
	}
	return true
}

func appendOAuthCode(redirectURI, code, state, issuer string) string {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	query := parsed.Query()
	query.Set("code", code)
	if trimmed := strings.TrimSpace(issuer); trimmed != "" {
		query.Set("iss", strings.TrimRight(trimmed, "/"))
	}
	if state != "" {
		query.Set("state", state)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func appendOAuthError(redirectURI, code, state, issuer string) string {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	query := parsed.Query()
	query.Set("error", code)
	if trimmed := strings.TrimSpace(issuer); trimmed != "" {
		query.Set("iss", strings.TrimRight(trimmed, "/"))
	}
	if state != "" {
		query.Set("state", state)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func appendOAuthState(redirectURI, state string) string {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	if state != "" {
		query := parsed.Query()
		query.Set("state", state)
		parsed.RawQuery = query.Encode()
	}
	return parsed.String()
}

func idTokenProfileClaims(profile map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range []string{
		"username",
		"displayName",
		"avatar",
		"email",
		"phone",
		"phoneMasked",
		"phoneVerified",
		"identityVerified",
		"identityType",
		"studentVerified",
		"school",
	} {
		if value, ok := profile[key]; ok {
			out[key] = value
		}
	}
	addOIDCProfileAliases(out)
	return out
}

func addOIDCProfileAliases(payload map[string]any) {
	if username, ok := payload["username"].(string); ok && username != "" {
		payload["preferred_username"] = username
		if _, ok := payload["name"]; !ok {
			payload["name"] = username
		}
	}
	if displayName, ok := payload["displayName"].(string); ok && displayName != "" {
		payload["name"] = displayName
	}
	if email, ok := payload["email"].(string); ok && email != "" {
		payload["email_verified"] = true
	}
	if avatar, ok := payload["avatar"].(string); ok && avatar != "" {
		payload["picture"] = avatar
	}
	if phone, ok := payload["phone"].(string); ok && phone != "" {
		payload["phone_number"] = phone
	}
	if phoneVerified, ok := payload["phoneVerified"].(bool); ok {
		payload["phone_number_verified"] = phoneVerified
	}
}
