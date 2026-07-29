package openplatform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/StuHelper/StuHelper/server/internal/pkg/ctxutil"
)

const (
	ProfileFieldUsername = "profile.username"
	ProfileFieldEmail    = "profile.email"
	ProfileFieldAvatar   = "profile.avatar"
	ProfileFieldPhone    = "profile.phone"
	ProfileFieldIdentity = "profile.identity"
	ProfileFieldStudent  = "profile.student"
	ProfileFieldSchool   = "profile.school"
)

var completionFieldCatalog = map[string]ProfileCompletionField{
	ProfileFieldUsername: {Key: ProfileFieldUsername, DisplayName: "用户名", ActionURL: "/account/profile"},
	ProfileFieldEmail:    {Key: ProfileFieldEmail, DisplayName: "邮箱", ActionURL: "/account/profile"},
	ProfileFieldAvatar:   {Key: ProfileFieldAvatar, DisplayName: "头像", ActionURL: "/account/profile"},
	ProfileFieldPhone:    {Key: ProfileFieldPhone, DisplayName: "手机号", ActionURL: "/user/phone-binding"},
	ProfileFieldIdentity: {Key: ProfileFieldIdentity, DisplayName: "实名认证", ActionURL: "/user/identity-verification"},
	ProfileFieldStudent:  {Key: ProfileFieldStudent, DisplayName: "学生认证", ActionURL: "/user/student-verification"},
	ProfileFieldSchool:   {Key: ProfileFieldSchool, DisplayName: "学校信息", ActionURL: "/user/student-verification"},
}

func NormalizeAuthorizationScopes(scopes []string) ([]string, error) {
	mapped := make([]string, 0, len(scopes))
	seenScope := false
	for _, raw := range scopes {
		switch strings.TrimSpace(raw) {
		case "":
			continue
		case "openid":
			seenScope = true
		case "profile":
			seenScope = true
			mapped = append(mapped, ScopeProfileBasicRead)
		case "email":
			seenScope = true
			mapped = append(mapped, ScopeEmailRead)
		case "phone":
			seenScope = true
			mapped = append(mapped, ScopePhoneRead)
		default:
			seenScope = true
			mapped = append(mapped, raw)
		}
	}
	if len(mapped) == 0 && seenScope {
		return []string{}, nil
	}
	return NormalizeScopes(mapped)
}

func RequiredProfileFields(projection *UserProjection, scopes []string) []ProfileCompletionField {
	if projection == nil {
		return nil
	}
	missing := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		switch scope {
		case ScopeProfileBasicRead:
			if strings.TrimSpace(projection.Username) == "" {
				missing = appendMissingField(missing, ProfileFieldUsername)
			}
			if projection.AvatarURL == nil || strings.TrimSpace(*projection.AvatarURL) == "" {
				missing = appendMissingField(missing, ProfileFieldAvatar)
			}
		case ScopeEmailRead:
			if strings.TrimSpace(projection.Email) == "" {
				missing = appendMissingField(missing, ProfileFieldEmail)
			}
		case ScopePhoneRead:
			if !projection.PhoneVerified {
				missing = appendMissingField(missing, ProfileFieldPhone)
			}
		case ScopeIdentityStatusRead, ScopeIdentityTypeRead:
			if !projection.IdentityVerified {
				missing = appendMissingField(missing, ProfileFieldIdentity)
			}
		case ScopeStudentStatusRead:
			if !isStudentVerified(projection) {
				missing = appendMissingField(missing, ProfileFieldStudent)
			}
		case ScopeStudentSchoolRead:
			if !isStudentVerified(projection) {
				missing = appendMissingField(missing, ProfileFieldStudent)
			}
			if projection.SchoolID == nil || projection.SchoolName == nil || strings.TrimSpace(*projection.SchoolName) == "" {
				missing = appendMissingField(missing, ProfileFieldSchool)
			}
		}
	}
	result := make([]ProfileCompletionField, 0, len(missing))
	for _, key := range missing {
		if field, ok := completionFieldCatalog[key]; ok {
			result = append(result, field)
		}
	}
	return result
}

func appendMissingField(keys []string, key string) []string {
	for _, existing := range keys {
		if existing == key {
			return keys
		}
	}
	return append(keys, key)
}

func (s *Service) BuildProfileCompletionChallenge(
	ctx context.Context,
	app *App,
	userID int64,
	scopes []string,
	req AuthorizeRequest,
) (*ProfileCompletionChallenge, error) {
	if app == nil {
		return nil, ErrAppNotFound
	}
	if userID <= 0 {
		return nil, ErrDisclosureUnavailable
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
	challenge := &ProfileCompletionChallenge{
		Token:               token,
		AppID:               currentApp.ID,
		UserID:              userID,
		Scopes:              scopes,
		OAuthScopes:         oauthScopes,
		RedirectURI:         req.RedirectURI,
		State:               strings.TrimSpace(req.State),
		Flow:                normalizeAuthorizeFlow(req.Flow),
		CodeChallenge:       strings.TrimSpace(req.CodeChallenge),
		CodeChallengeMethod: strings.TrimSpace(req.CodeChallengeMethod),
		Nonce:               strings.TrimSpace(req.Nonce),
		CreatedAt:           now,
		ExpiresAt:           now.Add(completionTokenTTL),
	}
	payload, err := challenge.MarshalPayload()
	if err != nil {
		return nil, fmt.Errorf("marshal profile completion challenge: %w", err)
	}
	if err := s.rdb.Set(ctx, completionRedisPrefix+token, payload, completionTokenTTL).Err(); err != nil {
		return nil, fmt.Errorf("store profile completion challenge: %w", err)
	}
	return challenge, nil
}

func (s *Service) LoadProfileCompletionChallenge(ctx context.Context, token string) (*ProfileCompletionChallenge, error) {
	token = normalizeChallengeToken(token)
	if token == "" {
		return nil, ErrCompletionTokenInvalid
	}
	raw, err := s.rdb.Get(ctx, completionRedisPrefix+token).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrCompletionTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("load profile completion challenge: %w", err)
	}
	return decodeProfileCompletionChallenge(token, []byte(raw))
}

func (s *Service) GetProfileCompletionPage(ctx context.Context, token string, userID int64) (*ProfileCompletionPage, error) {
	challenge, err := s.LoadProfileCompletionChallenge(ctx, token)
	if err != nil {
		return nil, err
	}
	if err := ensureProfileCompletionActor(challenge, userID); err != nil {
		return nil, err
	}
	app, err := s.loadCurrentChallengeApp(ctx, challenge.AppID, challenge.RedirectURI, challenge.Scopes)
	if err != nil {
		return nil, err
	}
	projection, err := s.repo.GetUserProjection(ctx, challenge.UserID)
	if err != nil {
		return nil, err
	}
	consentScopes := UserConsentScopes(challenge.Scopes)
	definitions, err := s.scopeDefinitionsForApp(ctx, app.ID, consentScopes)
	if err != nil {
		return nil, err
	}
	return &ProfileCompletionPage{
		Token:         challenge.Token,
		App:           consentApp(app),
		Scopes:        definitions,
		MissingFields: RequiredProfileFields(projection, consentScopes),
		RedirectURI:   challenge.RedirectURI,
		ExpiresAt:     challenge.ExpiresAt,
	}, nil
}

func (s *Service) ContinueProfileCompletion(ctx context.Context, token string, userID int64) (*AuthorizeResult, error) {
	challenge, err := s.LoadProfileCompletionChallenge(ctx, token)
	if err != nil {
		return nil, err
	}
	if err := ensureProfileCompletionActor(challenge, userID); err != nil {
		return nil, err
	}
	app, err := s.loadCurrentChallengeApp(ctx, challenge.AppID, challenge.RedirectURI, challenge.Scopes)
	if err != nil {
		return nil, err
	}
	projection, err := s.repo.GetUserProjection(ctx, challenge.UserID)
	if err != nil {
		return nil, err
	}
	consentScopes := UserConsentScopes(challenge.Scopes)
	if missing := RequiredProfileFields(projection, consentScopes); len(missing) > 0 {
		definitions, err := s.scopeDefinitionsForApp(ctx, app.ID, consentScopes)
		if err != nil {
			return nil, err
		}
		return &AuthorizeResult{
			ProfileCompletionURL: buildProfileCompletionURL(s.consentBaseURLForFlow(challenge.Flow), challenge.Token),
			MissingFields:        missing,
			Scopes:               definitions,
		}, nil
	}
	req := AuthorizeRequest{
		ClientID:            app.ClientID,
		RedirectURI:         challenge.RedirectURI,
		Scopes:              grantedOAuthScopes(challenge.OAuthScopes, challenge.Scopes),
		State:               challenge.State,
		Flow:                normalizeAuthorizeFlow(challenge.Flow),
		CodeChallenge:       challenge.CodeChallenge,
		CodeChallengeMethod: challenge.CodeChallengeMethod,
		Nonce:               challenge.Nonce,
	}
	hasConsent := len(consentScopes) == 0
	if !hasConsent {
		hasConsent, err = s.repo.HasActiveConsents(ctx, app.ID, challenge.UserID, consentScopes)
		if err != nil {
			return nil, err
		}
	}
	if !hasConsent {
		definitions, err := s.scopeDefinitionsForApp(ctx, app.ID, consentScopes)
		if err != nil {
			return nil, err
		}
		consentChallenge, err := s.BuildConsentChallenge(ctx, app, challenge.UserID, challenge.Scopes, req)
		if err != nil {
			return nil, err
		}
		if err := s.deleteProfileCompletionChallenge(ctx, token); err != nil {
			if cleanupErr := s.deleteConsentChallengeDetached(ctx, consentChallenge.Token); cleanupErr != nil {
				return nil, fmt.Errorf("%w; cleanup consent challenge: %v", err, cleanupErr)
			}
			return nil, err
		}
		return &AuthorizeResult{
			ConsentURL: buildConsentURL(s.consentBaseURLForFlow(challenge.Flow), consentChallenge.Token),
			Scopes:     definitions,
		}, nil
	}
	if normalizeAuthorizeFlow(challenge.Flow) == AuthorizeFlowAccount {
		consentChallenge, err := s.BuildConsentChallenge(ctx, app, challenge.UserID, challenge.Scopes, req)
		if err != nil {
			return nil, err
		}
		if err := s.deleteProfileCompletionChallenge(ctx, token); err != nil {
			if cleanupErr := s.deleteConsentChallengeDetached(ctx, consentChallenge.Token); cleanupErr != nil {
				return nil, fmt.Errorf("%w; cleanup consent challenge: %v", err, cleanupErr)
			}
			return nil, err
		}
		return &AuthorizeResult{RedirectURL: buildAccountContinueURL(s.accountBaseURL, consentChallenge.Token)}, nil
	}
	redirectURL, err := s.buildOIDCRedirectURL(app, req, challenge.Scopes)
	if err != nil {
		return nil, err
	}
	if err := s.deleteProfileCompletionChallengeDetached(ctx, token); err != nil {
		return nil, err
	}
	return &AuthorizeResult{RedirectURL: redirectURL}, nil
}

func (s *Service) deleteProfileCompletionChallengeDetached(ctx context.Context, token string) error {
	cleanupCtx, cancel := challengeCleanupContext(ctx)
	defer cancel()
	return s.deleteProfileCompletionChallenge(cleanupCtx, token)
}

func (s *Service) deleteProfileCompletionChallenge(ctx context.Context, token string) error {
	token = normalizeChallengeToken(token)
	if token == "" {
		return ErrCompletionTokenInvalid
	}
	if err := s.rdb.Del(ctx, completionRedisPrefix+token).Err(); err != nil {
		return fmt.Errorf("delete profile completion challenge: %w", err)
	}
	return nil
}

func (s *Service) DeleteConsentChallenge(ctx context.Context, token string) error {
	return s.deleteConsentChallenge(ctx, token)
}

func (s *Service) deleteConsentChallengeDetached(ctx context.Context, token string) error {
	cleanupCtx, cancel := challengeCleanupContext(ctx)
	defer cancel()
	return s.deleteConsentChallenge(cleanupCtx, token)
}

func (s *Service) deleteConsentChallenge(ctx context.Context, token string) error {
	token = normalizeChallengeToken(token)
	if token == "" {
		return ErrConsentTokenInvalid
	}
	if err := s.rdb.Del(ctx, consentRedisPrefix+token).Err(); err != nil {
		return fmt.Errorf("delete consent challenge: %w", err)
	}
	return nil
}

func normalizeChallengeToken(token string) string {
	return strings.TrimSpace(token)
}

func challengeCleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctxutil.DetachedTimeout(ctx, challengeCleanupTimeout)
}

func ensureProfileCompletionActor(challenge *ProfileCompletionChallenge, userID int64) error {
	if userID != challenge.UserID {
		return ErrCompletionTokenInvalid
	}
	return nil
}

func decodeProfileCompletionChallenge(token string, raw []byte) (*ProfileCompletionChallenge, error) {
	var payload ProfileCompletionChallengePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode profile completion challenge: %w", err)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, payload.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("decode profile completion created_at: %w", err)
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, payload.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("decode profile completion expires_at: %w", err)
	}
	return &ProfileCompletionChallenge{
		Token:               token,
		AppID:               payload.AppID,
		UserID:              payload.UserID,
		Scopes:              payload.Scopes,
		OAuthScopes:         grantedOAuthScopes(payload.OAuthScopes, payload.Scopes),
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

func grantedOAuthScopes(oauthScopes, fallback []string) []string {
	if len(oauthScopes) > 0 {
		return append([]string(nil), oauthScopes...)
	}
	return append([]string(nil), fallback...)
}

func normalizeAuthorizeFlow(flow string) string {
	if strings.EqualFold(strings.TrimSpace(flow), AuthorizeFlowAccount) {
		return AuthorizeFlowAccount
	}
	return AuthorizeFlowCasdoor
}

func (s *Service) consentBaseURLForFlow(flow string) string {
	if normalizeAuthorizeFlow(flow) == AuthorizeFlowAccount && strings.TrimSpace(s.accountBaseURL) != "" {
		return s.accountBaseURL
	}
	return s.consentBaseURL
}

func buildProfileCompletionURL(baseURL, token string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "/complete-profile?token=" + token
	}
	return base + "/complete-profile?token=" + token
}

func buildAccountContinueURL(baseURL, token string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "/oauth2/continue?token=" + token
	}
	return base + "/oauth2/continue?token=" + token
}
