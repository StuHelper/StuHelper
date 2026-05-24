package identityserver

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/openplatform"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

type Handler struct {
	service                *Service
	openPlatform           clientVerifier
	internalUserIDResolver middleware.InternalUserIDResolver
	issuer                 string
	frontendBase           string
	sessionRevoker         sessionRevoker
	cookieDomain           string
	cookieSecure           bool
}

type clientVerifier interface {
	VerifyClientSecret(ctx context.Context, clientID, clientSecret string) (*openplatform.App, error)
}

type sessionRevoker interface {
	RevokeSession(ctx context.Context, sessionID, userID, accessToken, refreshToken string) error
}

var (
	authorizeSingleValueParameters = []string{
		"response_type",
		"response_mode",
		"client_id",
		"redirect_uri",
		"scope",
		"state",
		"code_challenge",
		"code_challenge_method",
		"nonce",
		"prompt",
		"max_age",
	}
	logoutSingleValueParameters = []string{
		"id_token_hint",
		"client_id",
		"post_logout_redirect_uri",
		"state",
	}
	tokenSingleValueParameters = []string{
		"grant_type",
		"code",
		"redirect_uri",
		"code_verifier",
		"refresh_token",
		"scope",
		"client_id",
		"client_secret",
	}
	tokenInspectionSingleValueParameters = []string{
		"token",
		"token_type_hint",
		"client_id",
		"client_secret",
	}
)

func NewHandler(
	service *Service,
	openPlatform clientVerifier,
	issuer, frontendBase string,
	resolvers ...middleware.InternalUserIDResolver,
) *Handler {
	var resolver middleware.InternalUserIDResolver
	if len(resolvers) > 0 {
		resolver = resolvers[0]
	}
	return &Handler{
		service:                service,
		openPlatform:           openPlatform,
		internalUserIDResolver: resolver,
		issuer:                 strings.TrimRight(strings.TrimSpace(issuer), "/"),
		frontendBase:           strings.TrimRight(strings.TrimSpace(frontendBase), "/"),
	}
}

func (h *Handler) SetSessionRevoker(revoker sessionRevoker, cookieDomain string, cookieSecure bool) {
	h.sessionRevoker = revoker
	h.cookieDomain = strings.TrimSpace(cookieDomain)
	h.cookieSecure = cookieSecure
}

func (h *Handler) RegisterRoutes(r *gin.Engine, optionalAuthMW gin.HandlerFunc) {
	r.GET("/.well-known/openid-configuration", h.discovery)
	r.GET("/.well-known/oauth-authorization-server", h.discovery)
	r.GET("/.well-known/jwks.json", h.jwks)
	r.GET("/oauth2/authorize", optionalAuthMW, h.authorize)
	r.GET("/oauth2/continue", optionalAuthMW, h.continueAuthorize)
	r.GET("/oauth2/logout", optionalAuthMW, h.logout)
	r.POST("/oauth2/logout", optionalAuthMW, h.logout)
	r.POST("/oauth2/token", h.token)
	r.POST("/oauth2/introspect", h.introspect)
	r.POST("/oauth2/revoke", h.revoke)
	r.GET("/oidc/userinfo", h.userinfo)
	r.POST("/oidc/userinfo", h.userinfo)
}

func (h *Handler) discovery(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"issuer":                 h.issuer,
		"authorization_endpoint": h.issuer + "/oauth2/authorize",
		"authorization_response_iss_parameter_supported": true,
		"token_endpoint":                        h.issuer + "/oauth2/token",
		"userinfo_endpoint":                     h.issuer + "/oidc/userinfo",
		"jwks_uri":                              h.issuer + "/.well-known/jwks.json",
		"revocation_endpoint":                   h.issuer + "/oauth2/revoke",
		"introspection_endpoint":                h.issuer + "/oauth2/introspect",
		"end_session_endpoint":                  h.issuer + "/oauth2/logout",
		"response_types_supported":              []string{"code"},
		"response_modes_supported":              []string{"query"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token", "client_credentials"},
		"code_challenge_methods_supported":      []string{"S256"},
		"prompt_values_supported":               []string{oidcPromptNone, oidcPromptLogin, oidcPromptConsent},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"revocation_endpoint_auth_methods_supported": []string{
			"client_secret_basic",
			"client_secret_post",
		},
		"introspection_endpoint_auth_methods_supported": []string{
			"client_secret_basic",
			"client_secret_post",
		},
		"scopes_supported": append([]string{"openid", "profile", "email", "phone"},
			openplatform.ScopeProfileBasicRead,
			openplatform.ScopeEmailRead,
			openplatform.ScopePhoneRead,
			openplatform.ScopeIdentityStatusRead,
			openplatform.ScopeIdentityTypeRead,
			openplatform.ScopeStudentStatusRead,
			openplatform.ScopeStudentSchoolRead,
			openplatform.ScopeResourceRead,
			openplatform.ScopeResourceWrite,
			openplatform.ScopeOfflineAccess,
		),
		"claims_supported": []string{
			"sub",
			"username",
			"displayName",
			"name",
			"preferred_username",
			"avatar",
			"email",
			"email_verified",
			"picture",
			"phone",
			"phone_number",
			"phoneMasked",
			"phoneVerified",
			"phone_number_verified",
			"identityVerified",
			"identityType",
			"studentVerified",
			"school",
		},
	})
}

func (h *Handler) jwks(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.signer.JWKS())
}

func (h *Handler) authorize(c *gin.Context) {
	if repeated := repeatedParameterName(c.Request.URL.Query(), authorizeSingleValueParameters...); repeated != "" {
		logger.FromGin(c).Warn("identity authorize repeated parameter", zap.String("parameter", repeated))
		c.String(http.StatusBadRequest, "invalid authorization request")
		return
	}
	req := authorizeRequestFromQuery(c)
	if err := validateAuthorizeRequest(req); err != nil {
		logger.FromGin(c).Warn("identity authorize prompt invalid", zap.Error(err))
		c.String(http.StatusBadRequest, "invalid authorization request")
		return
	}
	if !middleware.IsAuthenticated(c) {
		if promptNoneRequested(req) {
			redirectURL, err := h.service.AuthorizationErrorRedirect(c.Request.Context(), req, oidcErrorLoginRequired)
			if err != nil {
				logger.FromGin(c).Warn("identity authorize prompt none failed", zap.Error(err))
				c.String(http.StatusBadRequest, "invalid authorization request")
				return
			}
			c.Redirect(http.StatusFound, redirectURL)
			return
		}
		if reauthenticationRequested(c, req) {
			c.Redirect(http.StatusFound, h.reauthLoginURL(c, req))
			return
		}
		c.Redirect(http.StatusFound, h.loginURL(c))
		return
	}
	if reauthenticationRequested(c, req) {
		c.Redirect(http.StatusFound, h.reauthLoginURL(c, req))
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	req.AuthTime = middleware.GetAuthenticationTime(c)
	redirectURL, err := h.service.Begin(c.Request.Context(), req, userID)
	if err != nil {
		logger.FromGin(c).Warn("identity authorize failed", zap.Error(err))
		c.String(http.StatusBadRequest, "invalid authorization request")
		return
	}
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *Handler) logout(c *gin.Context) {
	req, ok := logoutRequestFromRequest(c)
	if !ok {
		return
	}
	redirectURL, hasRedirect, err := h.service.EndSessionRedirect(c.Request.Context(), req)
	if err != nil {
		logger.FromGin(c).Warn("identity logout request invalid", zap.Error(err))
		c.String(http.StatusBadRequest, "invalid logout request")
		return
	}
	if err := h.revokeCurrentSession(c); err != nil {
		logger.FromGin(c).Error("identity logout failed to revoke session", zap.Error(err))
		h.clearAuthCookies(c)
		c.String(http.StatusInternalServerError, "logout partially failed")
		return
	}
	h.clearAuthCookies(c)
	if hasRedirect {
		c.Redirect(http.StatusFound, redirectURL)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) continueAuthorize(c *gin.Context) {
	token, ok := singleRequiredQueryValue(c.Request.URL.Query(), "token")
	if !ok {
		logger.FromGin(c).Warn("identity continue rejected invalid token query")
		c.String(http.StatusBadRequest, "invalid authorization request")
		return
	}
	if !middleware.IsAuthenticated(c) {
		c.Redirect(http.StatusFound, h.loginURL(c))
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	redirectURL, err := h.service.IssueCodeFromConsentChallenge(c.Request.Context(), token, userID, middleware.GetAuthenticationTime(c))
	if err != nil {
		logger.FromGin(c).Warn("identity continue failed", zap.Error(err))
		c.String(http.StatusBadRequest, "invalid authorization request")
		return
	}
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *Handler) token(c *gin.Context) {
	if !formURLEncodedContentType(c) {
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := c.Request.ParseForm(); err != nil {
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	if hasURLQuery(c) {
		logger.FromGin(c).Warn("identity token rejected query parameters")
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	if clientAuthenticationAmbiguous(c) {
		oauthInvalidClient(c)
		return
	}
	if repeated := repeatedParameterName(c.Request.PostForm, tokenSingleValueParameters...); repeated != "" {
		logger.FromGin(c).Warn("identity token repeated parameter", zap.String("parameter", repeated))
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	clientID, clientSecret, ok := h.clientCredentials(c)
	if !ok {
		oauthInvalidClient(c)
		return
	}
	if strings.TrimSpace(c.PostForm("grant_type")) == "" {
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	payload, err := h.service.Token(c.Request.Context(), TokenRequest{
		GrantType:    c.PostForm("grant_type"),
		Code:         c.PostForm("code"),
		RedirectURI:  c.PostForm("redirect_uri"),
		CodeVerifier: c.PostForm("code_verifier"),
		RefreshToken: c.PostForm("refresh_token"),
		Scope:        c.PostForm("scope"),
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})
	if err != nil {
		h.tokenError(c, err)
		return
	}
	oauthJSON(c, http.StatusOK, payload)
}

func (h *Handler) userinfo(c *gin.Context) {
	if hasURLQuery(c) || identityRequestHasBody(c) {
		logger.FromGin(c).Warn("identity userinfo rejected non-header token source")
		oauthInvalidToken(c)
		return
	}
	rawToken := bearerToken(c)
	if rawToken == "" {
		oauthInvalidToken(c)
		return
	}
	payload, err := h.service.UserInfo(c.Request.Context(), rawToken)
	if err != nil {
		logger.FromGin(c).Warn("identity userinfo failed", zap.Error(err))
		oauthInvalidToken(c)
		return
	}
	oauthJSON(c, http.StatusOK, payload)
}

func (h *Handler) introspect(c *gin.Context) {
	if !formURLEncodedContentType(c) {
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := c.Request.ParseForm(); err != nil {
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	if hasURLQuery(c) {
		logger.FromGin(c).Warn("identity introspect rejected query parameters")
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	if clientAuthenticationAmbiguous(c) {
		oauthInvalidClient(c)
		return
	}
	if repeated := repeatedParameterName(c.Request.PostForm, tokenInspectionSingleValueParameters...); repeated != "" {
		logger.FromGin(c).Warn("identity introspect repeated parameter", zap.String("parameter", repeated))
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	app, ok := h.authenticateClient(c)
	if !ok {
		oauthInvalidClient(c)
		return
	}
	rawToken := c.PostForm("token")
	if strings.TrimSpace(rawToken) == "" {
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	oauthJSON(c, http.StatusOK, h.service.Introspect(c.Request.Context(), rawToken, app.ClientID))
}

func (h *Handler) revoke(c *gin.Context) {
	if !formURLEncodedContentType(c) {
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := c.Request.ParseForm(); err != nil {
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	if hasURLQuery(c) {
		logger.FromGin(c).Warn("identity revoke rejected query parameters")
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	if clientAuthenticationAmbiguous(c) {
		oauthInvalidClient(c)
		return
	}
	if repeated := repeatedParameterName(c.Request.PostForm, tokenInspectionSingleValueParameters...); repeated != "" {
		logger.FromGin(c).Warn("identity revoke repeated parameter", zap.String("parameter", repeated))
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	app, ok := h.authenticateClient(c)
	if !ok {
		oauthInvalidClient(c)
		return
	}
	rawToken := c.PostForm("token")
	if strings.TrimSpace(rawToken) == "" {
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := h.service.Revoke(c.Request.Context(), rawToken, app.ClientID, c.PostForm("token_type_hint")); err != nil {
		logger.FromGin(c).Error("identity revoke failed", zap.Error(err))
		oauthJSONError(c, http.StatusServiceUnavailable, "server_error")
		return
	}
	oauthNoStore(c)
	c.Status(http.StatusOK)
}

func (h *Handler) authenticateClient(c *gin.Context) (*openplatform.App, bool) {
	clientID, clientSecret, ok := h.clientCredentials(c)
	if !ok {
		return nil, false
	}
	app, err := h.openPlatform.VerifyClientSecret(c.Request.Context(), clientID, clientSecret)
	if err != nil || app == nil || strings.TrimSpace(app.ClientID) == "" {
		return nil, false
	}
	return app, true
}

func (h *Handler) clientCredentials(c *gin.Context) (string, string, bool) {
	if repeatedAuthorizationHeader(c) {
		return "", "", false
	}
	if hasAuthorizationHeader(c) {
		clientID, clientSecret, ok := c.Request.BasicAuth()
		if !ok {
			return "", "", false
		}
		if formClientCredentialPresent(c.Request.PostForm) {
			return "", "", false
		}
		return strings.TrimSpace(clientID), strings.TrimSpace(clientSecret), true
	}
	return strings.TrimSpace(c.PostForm("client_id")), strings.TrimSpace(c.PostForm("client_secret")), true
}

func clientAuthenticationAmbiguous(c *gin.Context) bool {
	return repeatedAuthorizationHeader(c) || (hasAuthorizationHeader(c) && formClientCredentialPresent(c.Request.PostForm))
}

func repeatedAuthorizationHeader(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	return len(c.Request.Header.Values("Authorization")) > 1
}

func hasAuthorizationHeader(c *gin.Context) bool {
	return c != nil && strings.TrimSpace(c.GetHeader("Authorization")) != ""
}

func hasURLQuery(c *gin.Context) bool {
	return c != nil &&
		c.Request != nil &&
		c.Request.URL != nil &&
		c.Request.URL.RawQuery != ""
}

func formURLEncodedContentType(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	raw := strings.TrimSpace(c.GetHeader("Content-Type"))
	if raw == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(raw)
	if err != nil {
		return false
	}
	return strings.EqualFold(mediaType, "application/x-www-form-urlencoded")
}

func formClientCredentialPresent(values url.Values) bool {
	if values == nil {
		return false
	}
	if _, ok := values["client_id"]; ok {
		return true
	}
	if _, ok := values["client_secret"]; ok {
		return true
	}
	return false
}

func (h *Handler) resolveCurrentUserID(c *gin.Context) (int64, bool) {
	if h.internalUserIDResolver == nil {
		logger.FromGin(c).Error("identity internal user resolver is not configured")
		c.String(http.StatusInternalServerError, "failed to resolve user")
		return 0, false
	}
	return middleware.ResolveRequiredInternalUserID(c, h.internalUserIDResolver, "failed to resolve user")
}

func (h *Handler) tokenError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidClient):
		oauthInvalidClient(c)
	case errors.Is(err, ErrInvalidGrant):
		oauthJSONError(c, http.StatusBadRequest, "invalid_grant")
	case errors.Is(err, ErrInvalidScope):
		oauthJSONError(c, http.StatusBadRequest, "invalid_scope")
	case errors.Is(err, ErrUnsupportedGrantType):
		oauthJSONError(c, http.StatusBadRequest, "unsupported_grant_type")
	default:
		logger.FromGin(c).Error("identity token exchange failed", zap.Error(err))
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
	}
}

func (h *Handler) loginURL(c *gin.Context) string {
	return h.loginURLForTarget(c.Request.URL.RequestURI(), false)
}

func (h *Handler) reauthLoginURL(c *gin.Context, req AuthorizeRequest) string {
	return h.loginURLForTarget(authorizationTargetAfterReauth(c.Request.URL, req), true)
}

func (h *Handler) loginURLForTarget(target string, reauth bool) string {
	base := h.frontendBase
	if base == "" {
		base = h.issuer
	}
	values := url.Values{}
	values.Set("redirect", target)
	if reauth {
		values.Set("reauth", "1")
	}
	return base + "/login?" + values.Encode()
}

func reauthenticationRequested(c *gin.Context, req AuthorizeRequest) bool {
	if promptLoginRequested(req) {
		return true
	}
	maxAge, ok, err := maxAgeDuration(req.MaxAge)
	if err != nil || !ok {
		return false
	}
	if maxAge == 0 {
		return true
	}
	authTime := middleware.GetAuthenticationTime(c)
	if authTime.IsZero() {
		return true
	}
	return time.Since(authTime) > maxAge
}

func authorizationTargetAfterReauth(original *url.URL, req AuthorizeRequest) string {
	if original == nil {
		return ""
	}
	next := *original
	query := next.Query()
	removePromptQueryValue(query, oidcPromptLogin)
	if maxAge, ok, err := maxAgeDuration(req.MaxAge); err == nil && ok && maxAge == 0 {
		query.Del("max_age")
	}
	next.RawQuery = query.Encode()
	return next.RequestURI()
}

func removePromptQueryValue(query url.Values, value string) {
	values := strings.Fields(query.Get("prompt"))
	if len(values) == 0 {
		return
	}
	filtered := values[:0]
	for _, current := range values {
		if current != value {
			filtered = append(filtered, current)
		}
	}
	if len(filtered) == 0 {
		query.Del("prompt")
		return
	}
	query.Set("prompt", strings.Join(filtered, " "))
}

func logoutRequestFromRequest(c *gin.Context) (LogoutRequest, bool) {
	if c.Request.Method == http.MethodPost {
		if hasURLQuery(c) {
			logger.FromGin(c).Warn("identity logout rejected query parameters on POST")
			c.String(http.StatusBadRequest, "invalid logout request")
			return LogoutRequest{}, false
		}
		if identityRequestHasBody(c) && !formURLEncodedContentType(c) {
			logger.FromGin(c).Warn("identity logout rejected unsupported content type")
			c.String(http.StatusBadRequest, "invalid logout request")
			return LogoutRequest{}, false
		}
		if err := c.Request.ParseForm(); err != nil {
			c.String(http.StatusBadRequest, "invalid logout request")
			return LogoutRequest{}, false
		}
		if repeated := repeatedParameterName(c.Request.PostForm, logoutSingleValueParameters...); repeated != "" {
			logger.FromGin(c).Warn("identity logout repeated form parameter", zap.String("parameter", repeated))
			c.String(http.StatusBadRequest, "invalid logout request")
			return LogoutRequest{}, false
		}
		return LogoutRequest{
			IDTokenHint:           c.PostForm("id_token_hint"),
			ClientID:              c.PostForm("client_id"),
			PostLogoutRedirectURI: c.PostForm("post_logout_redirect_uri"),
			State:                 c.PostForm("state"),
		}, true
	}
	if repeated := repeatedParameterName(c.Request.URL.Query(), logoutSingleValueParameters...); repeated != "" {
		logger.FromGin(c).Warn("identity logout repeated query parameter", zap.String("parameter", repeated))
		c.String(http.StatusBadRequest, "invalid logout request")
		return LogoutRequest{}, false
	}
	return LogoutRequest{
		IDTokenHint:           c.Query("id_token_hint"),
		ClientID:              c.Query("client_id"),
		PostLogoutRedirectURI: c.Query("post_logout_redirect_uri"),
		State:                 c.Query("state"),
	}, true
}

func identityRequestHasBody(c *gin.Context) bool {
	return c != nil &&
		c.Request != nil &&
		c.Request.Body != nil &&
		c.Request.Body != http.NoBody &&
		c.Request.ContentLength != 0
}

func authorizeRequestFromQuery(c *gin.Context) AuthorizeRequest {
	return AuthorizeRequest{
		ResponseType:        c.Query("response_type"),
		ResponseMode:        c.Query("response_mode"),
		ClientID:            c.Query("client_id"),
		RedirectURI:         c.Query("redirect_uri"),
		Scope:               c.Query("scope"),
		State:               c.Query("state"),
		CodeChallenge:       c.Query("code_challenge"),
		CodeChallengeMethod: c.Query("code_challenge_method"),
		Nonce:               c.Query("nonce"),
		Prompt:              c.Query("prompt"),
		MaxAge:              c.Query("max_age"),
	}
}

func (h *Handler) revokeCurrentSession(c *gin.Context) error {
	if h.sessionRevoker == nil || !middleware.IsAuthenticated(c) {
		return nil
	}
	refreshToken, err := c.Cookie(middleware.CookieRefreshToken)
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		return err
	}
	sessionID, err := c.Cookie(middleware.CookieSessionID)
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		return err
	}
	return h.sessionRevoker.RevokeSession(
		c.Request.Context(),
		strings.TrimSpace(sessionID),
		middleware.GetUserID(c),
		middleware.GetAccessToken(c),
		strings.TrimSpace(refreshToken),
	)
}

func (h *Handler) clearAuthCookies(c *gin.Context) {
	h.writeExpiredCookie(c, middleware.CookieAccessToken, middleware.CookieAccessTokenPath, true)
	h.writeExpiredCookie(c, middleware.CookieRefreshToken, middleware.CookieRefreshTokenPath, true)
	h.writeExpiredCookie(c, middleware.CookieSessionID, "/", true)
	h.writeExpiredCookie(c, middleware.CSRFCookieName, "/", false)
	c.Header(middleware.CSRFHeaderName, "")
}

func (h *Handler) writeExpiredCookie(c *gin.Context, name, path string, httpOnly bool) {
	http.SetCookie(c.Writer, &http.Cookie{ //nolint:gosec // Secure is supplied by deployment token cookie config.
		Name:     name,
		Value:    "",
		MaxAge:   -1,
		Path:     path,
		Domain:   h.cookieDomain,
		Secure:   h.cookieSecure,
		HttpOnly: httpOnly,
		SameSite: http.SameSiteLaxMode,
	})
}

func bearerToken(c *gin.Context) string {
	if repeatedAuthorizationHeader(c) {
		return ""
	}
	header := c.GetHeader("Authorization")
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func repeatedParameterName(values url.Values, names ...string) string {
	for _, name := range names {
		if len(values[name]) > 1 {
			return name
		}
	}
	return ""
}

func singleRequiredQueryValue(values url.Values, name string) (string, bool) {
	raw := values[name]
	if len(raw) != 1 {
		return "", false
	}
	value := strings.TrimSpace(raw[0])
	if value == "" {
		return "", false
	}
	return value, true
}

func oauthJSONError(c *gin.Context, status int, code string) {
	oauthJSON(c, status, gin.H{"error": code})
}

func oauthInvalidClient(c *gin.Context) {
	c.Header("WWW-Authenticate", `Basic realm="StuHelper Identity"`)
	oauthJSONError(c, http.StatusUnauthorized, "invalid_client")
}

func oauthInvalidToken(c *gin.Context) {
	c.Header("WWW-Authenticate", `Bearer realm="StuHelper Identity", error="invalid_token"`)
	oauthJSONError(c, http.StatusUnauthorized, "invalid_token")
}

func oauthJSON(c *gin.Context, status int, payload any) {
	oauthNoStore(c)
	c.JSON(status, payload)
}

func oauthNoStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
}
