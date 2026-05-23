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
)

const (
	codeBytes           = 32
	authCodeRedisPrefix = "identity:auth_code:"
	revokedRedisPrefix  = "identity:revoked_jti:"
)

type Service struct {
	openPlatform openPlatformGateway
	rdb          *redis.Client
	signer       *Signer
	issuer       string
	codeTTL      time.Duration
	accessTTL    time.Duration
}

type openPlatformGateway interface {
	BeginAuthorization(ctx context.Context, req openplatform.AuthorizeRequest, userID int64) (*openplatform.AuthorizationDecision, error)
	LoadConsentChallenge(ctx context.Context, token string) (*openplatform.ConsentChallenge, error)
	AppByID(ctx context.Context, appID int64) (*openplatform.App, error)
	UserProjection(ctx context.Context, userID int64) (*openplatform.UserProjection, error)
	UserInfoForIdentityToken(ctx context.Context, clientID string, userID int64, subject string, scopes []string) (map[string]any, error)
	DeleteConsentChallenge(ctx context.Context, token string) error
	VerifyClientSecret(ctx context.Context, clientID, clientSecret string) (*openplatform.App, error)
}

func NewService(openPlatform openPlatformGateway, rdb *redis.Client, signer *Signer, issuer string, codeTTL, accessTTL time.Duration) (*Service, error) {
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
	return &Service{
		openPlatform: openPlatform,
		rdb:          rdb,
		signer:       signer,
		issuer:       strings.TrimRight(strings.TrimSpace(issuer), "/"),
		codeTTL:      codeTTL,
		accessTTL:    accessTTL,
	}, nil
}

func (s *Service) Begin(ctx context.Context, req AuthorizeRequest, userID int64) (string, error) {
	if strings.TrimSpace(req.ResponseType) != "code" {
		return "", ErrInvalidAuthorizeRequest
	}
	if err := validatePKCEChallenge(req.CodeChallenge, req.CodeChallengeMethod); err != nil {
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
	}, userID)
	if err != nil {
		return "", err
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
	return s.issueCodeRedirect(ctx, decision.App.ClientID, req.RedirectURI, decision.Scopes, decision.UserID, subject, req)
}

func (s *Service) IssueCodeFromConsentChallenge(ctx context.Context, token string, userID int64) (string, error) {
	challenge, err := s.openPlatform.LoadConsentChallenge(ctx, token)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(challenge.Flow) != openplatform.AuthorizeFlowIdentity {
		return "", openplatform.ErrConsentTokenInvalid
	}
	if challenge.UserID != userID {
		return "", openplatform.ErrConsentTokenInvalid
	}
	app, err := s.appForChallenge(ctx, challenge.AppID)
	if err != nil {
		return "", err
	}
	subject := identitySubject(challenge.UserID)
	projection, err := s.openPlatform.UserProjection(ctx, challenge.UserID)
	if err != nil {
		return "", err
	}
	if missing := openplatform.RequiredProfileFields(projection, challenge.Scopes); len(missing) > 0 {
		return "", openplatform.ErrProfileIncomplete
	}
	if _, err := s.openPlatform.UserInfoForIdentityToken(ctx, app.ClientID, challenge.UserID, subject, challenge.Scopes); err != nil {
		return "", err
	}
	if err := s.openPlatform.DeleteConsentChallenge(ctx, token); err != nil {
		return "", err
	}
	return s.issueCodeRedirect(ctx, app.ClientID, challenge.RedirectURI, challenge.Scopes, challenge.UserID, subject, AuthorizeRequest{
		ClientID:            app.ClientID,
		RedirectURI:         challenge.RedirectURI,
		State:               challenge.State,
		CodeChallenge:       challenge.CodeChallenge,
		CodeChallengeMethod: challenge.CodeChallengeMethod,
		Nonce:               challenge.Nonce,
	})
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
	if input.RedirectURI != "" && input.RedirectURI != code.RedirectURI {
		return nil, ErrInvalidGrant
	}
	if err := verifyPKCE(code.CodeChallenge, code.CodeChallengeMethod, input.CodeVerifier); err != nil {
		return nil, ErrInvalidGrant
	}
	profile, err := s.openPlatform.UserInfoForIdentityToken(ctx, app.ClientID, code.UserID, code.Subject, code.Scopes)
	if err != nil {
		return nil, err
	}
	accessToken, _, err := s.signer.SignAccessToken(AccessTokenInput{
		Subject:  code.Subject,
		ClientID: app.ClientID,
		UserID:   code.UserID,
		Scopes:   code.Scopes,
		TTL:      s.accessTTL,
	})
	if err != nil {
		return nil, err
	}
	idToken, err := s.signer.SignIDToken(IDTokenInput{
		Subject:  code.Subject,
		ClientID: app.ClientID,
		Scopes:   code.Scopes,
		Nonce:    code.Nonce,
		Profile:  idTokenProfileClaims(profile),
		TTL:      s.accessTTL,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"access_token": accessToken,
		"id_token":     idToken,
		"token_type":   "Bearer",
		"expires_in":   int(s.accessTTL.Seconds()),
		"scope":        strings.Join(code.Scopes, " "),
	}, nil
}

type TokenRequest struct {
	GrantType    string
	Code         string
	RedirectURI  string
	CodeVerifier string
	ClientID     string
	ClientSecret string
}

var (
	ErrInvalidClient           = errors.New("invalid_client")
	ErrInvalidGrant            = errors.New("invalid_grant")
	ErrInvalidAuthorizeRequest = errors.New("invalid_authorize_request")
	ErrUnsupportedGrantType    = errors.New("unsupported_grant_type")
)

func (s *Service) UserInfo(ctx context.Context, rawToken string) (map[string]any, error) {
	claims, err := s.verifyUsableAccessToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	payload, err := s.openPlatform.UserInfoForIdentityToken(ctx, claims.ClientID, claims.UserID, claims.Subject, claims.Scopes)
	if err != nil {
		return nil, err
	}
	addOIDCProfileAliases(payload)
	return payload, nil
}

func (s *Service) Introspect(ctx context.Context, rawToken string) map[string]any {
	claims, err := s.verifyUsableAccessToken(ctx, rawToken)
	if err != nil {
		return map[string]any{"active": false}
	}
	return map[string]any{
		"active":    true,
		"iss":       s.issuer,
		"sub":       claims.Subject,
		"aud":       claims.ClientID,
		"client_id": claims.ClientID,
		"scope":     strings.Join(claims.Scopes, " "),
		"exp":       claims.Expires.Unix(),
		"iat":       claims.IssuedAt.Unix(),
	}
}

func (s *Service) Revoke(ctx context.Context, rawToken string) error {
	claims, err := s.signer.VerifyAccessToken(rawToken)
	if err != nil {
		return nil
	}
	ttl := time.Until(claims.Expires)
	if ttl <= 0 {
		return nil
	}
	return s.rdb.Set(ctx, revokedRedisPrefix+claims.JTI, "1", ttl).Err()
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
	return appendOAuthCode(redirectURI, code, req.State), nil
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

func validatePKCEChallenge(challenge, method string) error {
	if strings.TrimSpace(challenge) == "" {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(method), "S256") {
		return errors.New("code_challenge_method must be S256")
	}
	return nil
}

func verifyPKCE(challenge, method, verifier string) error {
	challenge = strings.TrimSpace(challenge)
	if challenge == "" {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(method), "S256") || strings.TrimSpace(verifier) == "" {
		return ErrInvalidGrant
	}
	sum := sha256.Sum256([]byte(verifier))
	actual := base64.RawURLEncoding.EncodeToString(sum[:])
	if actual != challenge {
		return ErrInvalidGrant
	}
	return nil
}

func appendOAuthCode(redirectURI, code, state string) string {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	query := parsed.Query()
	query.Set("code", code)
	if trimmed := strings.TrimSpace(state); trimmed != "" {
		query.Set("state", trimmed)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func idTokenProfileClaims(profile map[string]any) map[string]any {
	out := map[string]any{}
	if username, ok := profile["username"].(string); ok && username != "" {
		out["preferred_username"] = username
		out["name"] = username
	}
	if email, ok := profile["email"].(string); ok && email != "" {
		out["email"] = email
	}
	if avatar, ok := profile["avatar"].(string); ok && avatar != "" {
		out["picture"] = avatar
	}
	if identityType, ok := profile["identityType"].(string); ok && identityType != "" {
		out["stuhelper_identity_type"] = identityType
	}
	if studentVerified, ok := profile["studentVerified"].(bool); ok {
		out["stuhelper_student_verified"] = studentVerified
	}
	return out
}

func addOIDCProfileAliases(payload map[string]any) {
	if username, ok := payload["username"].(string); ok && username != "" {
		payload["preferred_username"] = username
		payload["name"] = username
	}
	if email, ok := payload["email"].(string); ok && email != "" {
		payload["email_verified"] = true
	}
	if avatar, ok := payload["avatar"].(string); ok && avatar != "" {
		payload["picture"] = avatar
	}
}
