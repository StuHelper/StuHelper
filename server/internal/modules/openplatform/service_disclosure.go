package openplatform

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
)

var mainlandPhoneDigitsPattern = regexp.MustCompile(`1[3-9]\d{9}`)

type DisclosureRequest struct {
	ClientID       string
	CasdoorSubject string
	Scopes         []string
	RedirectURI    string
	ConsentBaseURL string
}

func (s *Service) UserInfo(ctx context.Context, req DisclosureRequest) (map[string]any, error) {
	return s.disclose(ctx, req)
}

func (s *Service) Verification(ctx context.Context, req DisclosureRequest) (map[string]any, error) {
	return s.disclose(ctx, req)
}

func (s *Service) Student(ctx context.Context, req DisclosureRequest) (map[string]any, error) {
	return s.disclose(ctx, req)
}

func (s *Service) Phone(ctx context.Context, req DisclosureRequest) (map[string]any, error) {
	return s.disclose(ctx, req)
}

func (s *Service) disclose(ctx context.Context, req DisclosureRequest) (map[string]any, error) {
	scopes, err := NormalizeScopes(req.Scopes)
	if err != nil {
		return nil, err
	}
	app, userID, err := s.loadDisclosureActors(ctx, req)
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
		return nil, s.consentRequired(ctx, app, userID, scopes, req)
	}
	projection, err := s.repo.GetUserProjection(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.buildDisclosurePayload(ctx, projection, scopes)
}

func (s *Service) loadDisclosureActors(ctx context.Context, req DisclosureRequest) (*App, int64, error) {
	app, err := s.repo.GetAppByClientID(ctx, strings.TrimSpace(req.ClientID))
	if err != nil {
		return nil, 0, err
	}
	if app.Status != AppStatusApproved {
		return nil, 0, ErrAppNotActive
	}
	userID, err := s.repo.GetInternalUserID(ctx, strings.TrimSpace(req.CasdoorSubject))
	if err != nil {
		return nil, 0, err
	}
	return app, userID, nil
}

func (s *Service) ensureScopesApproved(ctx context.Context, appID int64, scopes []string) error {
	approved, err := s.repo.ListApprovedScopes(ctx, appID)
	if err != nil {
		return err
	}
	set := make(map[string]struct{}, len(approved))
	for _, scope := range approved {
		set[scope] = struct{}{}
	}
	for _, scope := range scopes {
		if _, ok := set[scope]; !ok {
			return ErrScopeNotApproved
		}
	}
	return nil
}

func (s *Service) consentRequired(
	ctx context.Context,
	app *App,
	userID int64,
	scopes []string,
	req DisclosureRequest,
) error {
	challenge, err := s.BuildConsentChallenge(ctx, app, userID, scopes, req.RedirectURI, "")
	if err != nil {
		return err
	}
	return ConsentRequiredError{
		ConsentURL: buildConsentURL(req.ConsentBaseURL, challenge.Token),
		Scopes:     ScopeDefinitions(scopes),
	}
}

func buildConsentURL(baseURL, token string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "/connect/consent?token=" + token
	}
	return base + "/connect/consent?token=" + token
}

func (s *Service) buildDisclosurePayload(ctx context.Context, projection *UserProjection, scopes []string) (map[string]any, error) {
	out := map[string]any{}
	for _, scope := range scopes {
		if err := s.addScopePayload(ctx, out, projection, scope); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Service) addScopePayload(ctx context.Context, out map[string]any, projection *UserProjection, scope string) error {
	switch scope {
	case ScopeProfileBasicRead:
		out["username"] = projection.Username
		out["displayName"] = projection.Username
		if projection.AvatarURL != nil {
			out["avatar"] = *projection.AvatarURL
		}
	case ScopeEmailRead:
		out["email"] = projection.Email
	case ScopePhoneRead:
		return s.addPhonePayload(ctx, out, projection)
	case ScopeIdentityStatusRead:
		out["identityVerified"] = projection.IdentityVerified
	case ScopeIdentityTypeRead:
		out["identityType"] = identityType(projection)
	case ScopeStudentStatusRead:
		out["studentVerified"] = isStudentVerified(projection)
	case ScopeStudentSchoolRead:
		addSchoolPayload(out, projection)
	}
	return nil
}

func (s *Service) addPhonePayload(ctx context.Context, out map[string]any, projection *UserProjection) error {
	if !projection.PhoneVerified {
		out["phoneVerified"] = false
		return nil
	}
	if s.phoneReader == nil {
		return fmt.Errorf("%w: Casdoor phone reader is not configured", ErrDisclosureUnavailable)
	}
	phone, err := s.phoneReader.GetPhone(ctx, projection.CasdoorSubject)
	if err != nil {
		return fmt.Errorf("read Casdoor phone: %w", err)
	}
	normalized, ok := normalizeCasdoorMainlandPhone(phone)
	if !ok {
		return fmt.Errorf("%w: Casdoor phone is unavailable", ErrDisclosureUnavailable)
	}
	out["phone"] = normalized
	out["phoneMasked"] = phoneutil.Mask(normalized)
	out["phoneVerified"] = true
	return nil
}

func normalizeCasdoorMainlandPhone(raw string) (string, bool) {
	compact := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(strings.TrimSpace(raw))
	match := mainlandPhoneDigitsPattern.FindString(compact)
	if match == "" {
		return "", false
	}
	return match, true
}

func identityType(projection *UserProjection) string {
	if isStudentVerified(projection) {
		return "student"
	}
	return "other"
}

func isStudentVerified(projection *UserProjection) bool {
	return projection.ProfileStatus != nil && *projection.ProfileStatus == "verified"
}

func addSchoolPayload(out map[string]any, projection *UserProjection) {
	if projection.SchoolID == nil || projection.SchoolName == nil {
		return
	}
	out["school"] = map[string]any{
		"id":   *projection.SchoolID,
		"name": *projection.SchoolName,
	}
}
