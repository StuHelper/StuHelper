package openplatform

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
)

var mainlandPhoneDigitsPattern = regexp.MustCompile(`1[3-9]\d{9}`)

const (
	disclosureEndpointIdentityToken = "identity_token"
	disclosureEndpointPhone         = "phone"
	disclosureEndpointStudent       = "student"
	disclosureEndpointUserInfo      = "userinfo"
	disclosureEndpointVerification  = "verification"
)

type DisclosureRequest struct {
	ClientID              string
	AuthenticatedClientID string
	AuthenticatedByBearer bool
	AccessTokenScopes     []string
	UserID                int64
	Scopes                []string
	RedirectURI           string
	ConsentBaseURL        string
	Endpoint              string
	RequestID             string
}

func (s *Service) UserInfo(ctx context.Context, req DisclosureRequest) (map[string]any, error) {
	req.Endpoint = disclosureEndpointUserInfo
	return s.disclose(ctx, req)
}

func (s *Service) Verification(ctx context.Context, req DisclosureRequest) (map[string]any, error) {
	req.Endpoint = disclosureEndpointVerification
	return s.disclose(ctx, req)
}

func (s *Service) Student(ctx context.Context, req DisclosureRequest) (map[string]any, error) {
	req.Endpoint = disclosureEndpointStudent
	return s.disclose(ctx, req)
}

func (s *Service) Phone(ctx context.Context, req DisclosureRequest) (map[string]any, error) {
	req.Endpoint = disclosureEndpointPhone
	return s.disclose(ctx, req)
}

func (s *Service) UserInfoForIdentityToken(ctx context.Context, clientID string, userID int64, subject string, scopes []string) (map[string]any, error) {
	endpoint := disclosureEndpointIdentityToken
	normalized, err := NormalizeAuthorizationScopes(scopes)
	if err != nil {
		metrics.ObserveOpenPlatformDisclosure(endpoint, "invalid_scope")
		return nil, err
	}
	app, err := s.repo.GetAppByClientID(ctx, strings.TrimSpace(clientID))
	if err != nil {
		metrics.ObserveOpenPlatformDisclosure(endpoint, "app_not_found")
		return nil, err
	}
	if app.Status != AppStatusApproved {
		return nil, s.recordDisclosureDenied(ctx, app, DisclosureRequest{
			ClientID: clientID,
			UserID:   userID,
			Endpoint: endpoint,
		}, normalized, "app_not_active", ErrAppNotActive)
	}
	if userID <= 0 {
		return nil, s.recordDisclosureDenied(ctx, app, DisclosureRequest{
			ClientID: clientID,
			UserID:   userID,
			Endpoint: endpoint,
		}, normalized, "user_unavailable", ErrDisclosureUnavailable)
	}
	if err := s.rateLimiter.checkDisclosure(ctx, app.ID, userID, endpoint); err != nil {
		return nil, s.recordDisclosureDenied(ctx, app, DisclosureRequest{
			ClientID: clientID,
			UserID:   userID,
			Endpoint: endpoint,
		}, normalized, disclosureErrorResult(err), err)
	}
	if len(normalized) > 0 {
		if err := s.ensureScopesApproved(ctx, app.ID, normalized); err != nil {
			return nil, s.recordDisclosureDenied(ctx, app, DisclosureRequest{
				ClientID: clientID,
				UserID:   userID,
				Endpoint: endpoint,
			}, normalized, disclosureErrorResult(err), err)
		}
	}
	disclosureScopes := UserConsentScopes(normalized)
	payload := map[string]any{}
	if len(disclosureScopes) > 0 {
		hasConsent, err := s.repo.HasActiveConsents(ctx, app.ID, userID, disclosureScopes)
		if err != nil {
			return nil, s.recordDisclosureDenied(ctx, app, DisclosureRequest{
				ClientID: clientID,
				UserID:   userID,
				Endpoint: endpoint,
			}, normalized, "consent_lookup_error", err)
		}
		if !hasConsent {
			return nil, s.recordDisclosureDenied(ctx, app, DisclosureRequest{
				ClientID: clientID,
				UserID:   userID,
				Endpoint: endpoint,
			}, normalized, "consent_required", ErrConsentRequired)
		}
		projection, err := s.repo.GetUserProjection(ctx, userID)
		if err != nil {
			return nil, s.recordDisclosureDenied(ctx, app, DisclosureRequest{
				ClientID: clientID,
				UserID:   userID,
				Endpoint: endpoint,
			}, normalized, "projection_unavailable", err)
		}
		payload, err = s.buildDisclosurePayload(ctx, projection, disclosureScopes)
		if err != nil {
			return nil, s.recordDisclosureDenied(ctx, app, DisclosureRequest{
				ClientID: clientID,
				UserID:   userID,
				Endpoint: endpoint,
			}, normalized, "payload_unavailable", err)
		}
	}
	if err := s.recordDisclosureGranted(ctx, app, DisclosureRequest{
		ClientID: clientID,
		UserID:   userID,
		Endpoint: endpoint,
	}, disclosureScopes); err != nil {
		return nil, err
	}
	payload["sub"] = strings.TrimSpace(subject)
	return payload, nil
}

func (s *Service) IdentityAccessTokenActive(ctx context.Context, clientID string, userID int64, scopes []string) (bool, error) {
	_, active, err := s.IdentityAuthorizationFingerprint(ctx, clientID, userID, scopes)
	return active, err
}

func (s *Service) IdentityAuthorizationFingerprint(ctx context.Context, clientID string, userID int64, scopes []string) (string, bool, error) {
	normalized, err := NormalizeAuthorizationScopes(scopes)
	if err != nil {
		return "", false, err
	}
	app, err := s.repo.GetAppByClientID(ctx, strings.TrimSpace(clientID))
	if err != nil {
		return "", false, err
	}
	if app.Status != AppStatusApproved || userID <= 0 {
		return "", false, nil
	}
	if err := s.ensureScopesApproved(ctx, app.ID, normalized); err != nil {
		if errors.Is(err, ErrScopeNotApproved) {
			return "", false, nil
		}
		return "", false, err
	}
	consentScopes := UserConsentScopes(normalized)
	if len(consentScopes) == 0 {
		return "", true, nil
	}
	fingerprint, active, err := s.repo.ActiveConsentFingerprint(ctx, app.ID, userID, consentScopes)
	if err != nil {
		return "", false, err
	}
	return fingerprint, active, nil
}

func (s *Service) IdentityClientCredentialsTokenActive(ctx context.Context, clientID string, scopes []string) (bool, error) {
	normalized, err := NormalizeScopes(scopes)
	if err != nil {
		return false, err
	}
	for _, scope := range normalized {
		if scope != ScopeResourceRead && scope != ScopeResourceWrite {
			return false, nil
		}
	}
	app, err := s.repo.GetAppByClientID(ctx, strings.TrimSpace(clientID))
	if err != nil {
		return false, err
	}
	if app.Status != AppStatusApproved {
		return false, nil
	}
	if err := s.ensureScopesApproved(ctx, app.ID, normalized); err != nil {
		if errors.Is(err, ErrScopeNotApproved) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Service) disclose(ctx context.Context, req DisclosureRequest) (map[string]any, error) {
	req.Endpoint = normalizeDisclosureEndpoint(req.Endpoint)
	scopes, err := NormalizeScopes(req.Scopes)
	if err != nil {
		metrics.ObserveOpenPlatformDisclosure(req.Endpoint, "invalid_scope")
		return nil, err
	}
	disclosureScopes := UserConsentScopes(scopes)
	if len(disclosureScopes) == 0 {
		metrics.ObserveOpenPlatformDisclosure(req.Endpoint, "invalid_scope")
		return nil, ErrInvalidScope
	}
	app, err := s.loadDisclosureApp(ctx, req)
	if err != nil {
		if auditErr := s.recordDisclosureDenied(ctx, nil, req, scopes, disclosureErrorResult(err), err); auditErr != nil {
			return nil, auditErr
		}
		return nil, err
	}
	if !disclosureClientCredentialMatches(req) {
		return nil, s.recordDisclosureDenied(ctx, app, req, scopes, "client_mismatch", ErrDisclosureClientMismatch)
	}
	if req.AuthenticatedByBearer && !requestedScopesGranted(scopes, req.AccessTokenScopes) {
		return nil, s.recordDisclosureDenied(ctx, app, req, scopes, "token_scope_missing", ErrScopeNotApproved)
	}
	if !req.AuthenticatedByBearer && !redirectAllowed(app, req.RedirectURI) {
		return nil, s.recordDisclosureDenied(ctx, app, req, scopes, disclosureErrorResult(ErrRedirectURINotAllowed), ErrRedirectURINotAllowed)
	}
	if req.UserID <= 0 {
		return nil, s.recordDisclosureDenied(ctx, app, req, scopes, "user_unavailable", ErrDisclosureUnavailable)
	}
	if err := s.rateLimiter.checkDisclosure(ctx, app.ID, req.UserID, req.Endpoint); err != nil {
		return nil, s.recordDisclosureDenied(ctx, app, req, scopes, disclosureErrorResult(err), err)
	}
	if err := s.ensureScopesApproved(ctx, app.ID, scopes); err != nil {
		return nil, s.recordDisclosureDenied(ctx, app, req, scopes, disclosureErrorResult(err), err)
	}
	if !req.AuthenticatedByBearer {
		hasConsent, err := s.repo.HasActiveConsents(ctx, app.ID, req.UserID, disclosureScopes)
		if err != nil {
			return nil, s.recordDisclosureDenied(ctx, app, req, scopes, "consent_lookup_error", err)
		}
		if !hasConsent {
			if err := s.rateLimiter.checkConsentChallenge(ctx, app.ID, req.UserID); err != nil {
				return nil, s.recordDisclosureDenied(ctx, app, req, scopes, disclosureErrorResult(err), err)
			}
			err := s.consentRequired(ctx, app, req.UserID, disclosureScopes, req)
			if err != nil {
				return nil, s.recordDisclosureDenied(ctx, app, req, scopes, "consent_required", err)
			}
			return nil, err
		}
	}
	projection, err := s.repo.GetUserProjection(ctx, req.UserID)
	if err != nil {
		return nil, s.recordDisclosureDenied(ctx, app, req, scopes, "projection_unavailable", err)
	}
	payload, err := s.buildDisclosurePayload(ctx, projection, disclosureScopes)
	if err != nil {
		return nil, s.recordDisclosureDenied(ctx, app, req, scopes, "payload_unavailable", err)
	}
	if err := s.recordDisclosureGranted(ctx, app, req, disclosureScopes); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *Service) loadDisclosureApp(ctx context.Context, req DisclosureRequest) (*App, error) {
	app, err := s.repo.GetAppByClientID(ctx, strings.TrimSpace(req.ClientID))
	if err != nil {
		return nil, err
	}
	if app.Status != AppStatusApproved {
		return nil, ErrAppNotActive
	}
	return app, nil
}

func normalizeDisclosureEndpoint(raw string) string {
	switch strings.TrimSpace(raw) {
	case disclosureEndpointIdentityToken:
		return disclosureEndpointIdentityToken
	case disclosureEndpointPhone:
		return disclosureEndpointPhone
	case disclosureEndpointStudent:
		return disclosureEndpointStudent
	case disclosureEndpointVerification:
		return disclosureEndpointVerification
	default:
		return disclosureEndpointUserInfo
	}
}

func disclosureClientCredentialMatches(req DisclosureRequest) bool {
	if !req.AuthenticatedByBearer {
		return true
	}
	clientID := strings.TrimSpace(req.ClientID)
	authenticatedClientID := strings.TrimSpace(req.AuthenticatedClientID)
	if clientID == "" || authenticatedClientID == "" {
		return true
	}
	return clientID == authenticatedClientID
}

func requestedScopesGranted(requested, granted []string) bool {
	if len(requested) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(granted))
	for _, scope := range granted {
		if normalized := strings.TrimSpace(scope); normalized != "" {
			set[normalized] = struct{}{}
		}
	}
	for _, scope := range requested {
		if _, ok := set[strings.TrimSpace(scope)]; !ok {
			return false
		}
	}
	return true
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
	challenge, err := s.BuildConsentChallenge(ctx, app, userID, scopes, AuthorizeRequest{
		ClientID:    app.ClientID,
		RedirectURI: req.RedirectURI,
		Scopes:      scopes,
		Flow:        AuthorizeFlowCasdoor,
	})
	if err != nil {
		return err
	}
	definitions, err := s.scopeDefinitionsForApp(ctx, app.ID, scopes)
	if err != nil {
		return err
	}
	return ConsentRequiredError{
		ConsentURL: buildConsentURL(req.ConsentBaseURL, challenge.Token),
		Scopes:     definitions,
	}
}

func (s *Service) recordDisclosureGranted(
	ctx context.Context,
	app *App,
	req DisclosureRequest,
	scopes []string,
) error {
	if err := s.recordDisclosureAudit(ctx, app, req, scopes, "open_platform.disclosure.granted", "ok", nil); err != nil {
		metrics.ObserveOpenPlatformDisclosure(req.Endpoint, "audit_unavailable")
		return fmt.Errorf("%w: disclosure audit unavailable", ErrDisclosureUnavailable)
	}
	metrics.ObserveOpenPlatformDisclosure(req.Endpoint, "ok")
	return nil
}

func (s *Service) recordDisclosureDenied(
	ctx context.Context,
	app *App,
	req DisclosureRequest,
	scopes []string,
	reason string,
	cause error,
) error {
	if err := s.recordDisclosureAudit(ctx, app, req, scopes, "open_platform.disclosure.denied", reason, cause); err != nil {
		metrics.ObserveOpenPlatformDisclosure(req.Endpoint, "audit_unavailable")
		return fmt.Errorf("%w: disclosure audit unavailable", ErrDisclosureUnavailable)
	}
	metrics.ObserveOpenPlatformDisclosure(req.Endpoint, reason)
	if err := s.recordDisclosureReplayIfNeeded(ctx, app, req, scopes, reason, cause); err != nil {
		metrics.ObserveOpenPlatformDisclosure(req.Endpoint, "audit_unavailable")
		return err
	}
	return cause
}

func (s *Service) recordDisclosureReplayIfNeeded(
	ctx context.Context,
	app *App,
	req DisclosureRequest,
	scopes []string,
	reason string,
	cause error,
) error {
	appID := int64(0)
	if app != nil {
		appID = app.ID
	}
	detection, err := s.replayDetector.check(ctx, appID, req.UserID, req.Endpoint, reason, scopes)
	if err != nil {
		return fmt.Errorf("%w: disclosure replay detector unavailable", ErrDisclosureUnavailable)
	}
	if !detection.Detected {
		return nil
	}
	metadata := map[string]any{
		"clientID":      strings.TrimSpace(req.ClientID),
		"endpoint":      req.Endpoint,
		"result":        reason,
		"scopes":        append([]string(nil), scopes...),
		"signatureHash": detection.SignatureHash,
		"count":         detection.Count,
	}
	if cause != nil {
		metadata["error"] = cause.Error()
		if dimension := disclosureRateLimitDimension(cause); dimension != "" {
			metadata["rateLimitDimension"] = dimension
		}
	}
	if err := s.repo.RecordAuditEvent(ctx, auditEvent{
		AppID:     appID,
		UserID:    req.UserID,
		EventType: "open_platform.disclosure.replay_detected",
		RequestID: req.RequestID,
		Metadata:  metadata,
	}); err != nil {
		return fmt.Errorf("%w: disclosure replay audit unavailable", ErrDisclosureUnavailable)
	}
	return nil
}

func (s *Service) recordDisclosureAudit(
	ctx context.Context,
	app *App,
	req DisclosureRequest,
	scopes []string,
	eventType string,
	result string,
	cause error,
) error {
	appID := int64(0)
	if app != nil {
		appID = app.ID
	}
	metadata := map[string]any{
		"clientID": strings.TrimSpace(req.ClientID),
		"endpoint": req.Endpoint,
		"result":   result,
		"scopes":   append([]string(nil), scopes...),
	}
	if cause != nil {
		metadata["error"] = cause.Error()
		if dimension := disclosureRateLimitDimension(cause); dimension != "" {
			metadata["rateLimitDimension"] = dimension
		}
	}
	return s.repo.RecordAuditEvent(ctx, auditEvent{
		AppID:     appID,
		UserID:    req.UserID,
		EventType: eventType,
		RequestID: req.RequestID,
		Metadata:  metadata,
	})
}

func disclosureErrorResult(err error) string {
	switch {
	case errors.Is(err, ErrAppNotFound):
		return "app_not_found"
	case errors.Is(err, ErrAppNotActive), errors.Is(err, ErrAppNotApproved):
		return "app_not_active"
	case errors.Is(err, ErrScopeNotApproved):
		return "scope_denied"
	case errors.Is(err, ErrRedirectURINotAllowed):
		return "redirect_uri_not_allowed"
	case errors.Is(err, ErrDisclosureClientMismatch):
		return "client_mismatch"
	case errors.Is(err, ErrConsentRequired):
		return "consent_required"
	case errors.Is(err, ErrDisclosureRateLimited):
		return "rate_limited"
	case errors.Is(err, ErrDisclosureUnavailable):
		return "unavailable"
	default:
		return "error"
	}
}

func buildConsentURL(baseURL, token string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "/consent?token=" + token
	}
	return base + "/consent?token=" + token
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
	case ScopeQQStatusRead:
		out["qqBound"] = projection.QQID != nil && strings.TrimSpace(*projection.QQID) != ""
	case ScopeQQNumberRead:
		addQQNumberPayload(out, projection)
	}
	return nil
}

func (s *Service) addPhonePayload(ctx context.Context, out map[string]any, projection *UserProjection) error {
	if !projection.PhoneVerified || len(projection.PhoneEnc) == 0 {
		out["phoneVerified"] = false
		return nil
	}
	if s.phoneCipher == nil {
		return fmt.Errorf("%w: phone decryptor is not configured", ErrDisclosureUnavailable)
	}
	phone, err := s.phoneCipher.Decrypt(projection.PhoneEnc)
	if err != nil {
		return fmt.Errorf("decrypt phone projection: %w", err)
	}
	normalized, ok := normalizeCasdoorMainlandPhone(phone)
	if !ok {
		return fmt.Errorf("%w: phone projection is unavailable", ErrDisclosureUnavailable)
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
	if projection.SchoolID == nil {
		return
	}
	out["school"] = map[string]any{
		"id": *projection.SchoolID,
	}
}

func addQQNumberPayload(out map[string]any, projection *UserProjection) {
	if projection.QQID == nil || strings.TrimSpace(*projection.QQID) == "" {
		out["qqBound"] = false
		return
	}
	qq := map[string]any{
		"bound": true,
		"id":    strings.TrimSpace(*projection.QQID),
	}
	out["qqBound"] = true
	out["qq"] = qq
}
