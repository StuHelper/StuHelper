package identityserver

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

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
}

type clientVerifier interface {
	VerifyClientSecret(ctx context.Context, clientID, clientSecret string) (*openplatform.App, error)
}

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

func (h *Handler) RegisterRoutes(r *gin.Engine, optionalAuthMW gin.HandlerFunc) {
	r.GET("/.well-known/openid-configuration", h.discovery)
	r.GET("/.well-known/jwks.json", h.jwks)
	r.GET("/oauth2/authorize", optionalAuthMW, h.authorize)
	r.GET("/oauth2/continue", optionalAuthMW, h.continueAuthorize)
	r.POST("/oauth2/token", h.token)
	r.POST("/oauth2/introspect", h.introspect)
	r.POST("/oauth2/revoke", h.revoke)
	r.GET("/oidc/userinfo", h.userinfo)
	r.POST("/oidc/userinfo", h.userinfo)
}

func (h *Handler) discovery(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"issuer":                                h.issuer,
		"authorization_endpoint":                h.issuer + "/oauth2/authorize",
		"token_endpoint":                        h.issuer + "/oauth2/token",
		"userinfo_endpoint":                     h.issuer + "/oidc/userinfo",
		"jwks_uri":                              h.issuer + "/.well-known/jwks.json",
		"revocation_endpoint":                   h.issuer + "/oauth2/revoke",
		"introspection_endpoint":                h.issuer + "/oauth2/introspect",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"scopes_supported": append([]string{"openid", "profile", "email", "phone"},
			openplatform.ScopeProfileBasicRead,
			openplatform.ScopeEmailRead,
			openplatform.ScopePhoneRead,
			openplatform.ScopeIdentityStatusRead,
			openplatform.ScopeIdentityTypeRead,
			openplatform.ScopeStudentStatusRead,
			openplatform.ScopeStudentSchoolRead,
		),
		"claims_supported": []string{
			"sub",
			"name",
			"preferred_username",
			"email",
			"email_verified",
			"picture",
			"phone",
			"phoneVerified",
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
	req := authorizeRequestFromQuery(c)
	if !middleware.IsAuthenticated(c) {
		c.Redirect(http.StatusFound, h.loginURL(c))
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	redirectURL, err := h.service.Begin(c.Request.Context(), req, userID)
	if err != nil {
		logger.FromGin(c).Warn("identity authorize failed", zap.Error(err))
		c.String(http.StatusBadRequest, "invalid authorization request")
		return
	}
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *Handler) continueAuthorize(c *gin.Context) {
	token := c.Query("token")
	if !middleware.IsAuthenticated(c) {
		c.Redirect(http.StatusFound, h.loginURL(c))
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	redirectURL, err := h.service.IssueCodeFromConsentChallenge(c.Request.Context(), token, userID)
	if err != nil {
		logger.FromGin(c).Warn("identity continue failed", zap.Error(err))
		c.String(http.StatusBadRequest, "invalid authorization request")
		return
	}
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *Handler) token(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	clientID, clientSecret := h.clientCredentials(c)
	payload, err := h.service.ExchangeCode(c.Request.Context(), TokenRequest{
		GrantType:    c.PostForm("grant_type"),
		Code:         c.PostForm("code"),
		RedirectURI:  c.PostForm("redirect_uri"),
		CodeVerifier: c.PostForm("code_verifier"),
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})
	if err != nil {
		h.tokenError(c, err)
		return
	}
	c.JSON(http.StatusOK, payload)
}

func (h *Handler) userinfo(c *gin.Context) {
	rawToken := bearerToken(c)
	if rawToken == "" {
		oauthJSONError(c, http.StatusUnauthorized, "invalid_token")
		return
	}
	payload, err := h.service.UserInfo(c.Request.Context(), rawToken)
	if err != nil {
		logger.FromGin(c).Warn("identity userinfo failed", zap.Error(err))
		oauthJSONError(c, http.StatusUnauthorized, "invalid_token")
		return
	}
	c.JSON(http.StatusOK, payload)
}

func (h *Handler) introspect(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	if !h.validClient(c) {
		oauthJSONError(c, http.StatusUnauthorized, "invalid_client")
		return
	}
	c.JSON(http.StatusOK, h.service.Introspect(c.Request.Context(), c.PostForm("token")))
}

func (h *Handler) revoke(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
		return
	}
	if !h.validClient(c) {
		oauthJSONError(c, http.StatusUnauthorized, "invalid_client")
		return
	}
	if err := h.service.Revoke(c.Request.Context(), c.PostForm("token")); err != nil {
		logger.FromGin(c).Warn("identity revoke failed", zap.Error(err))
	}
	c.Status(http.StatusOK)
}

func (h *Handler) validClient(c *gin.Context) bool {
	clientID, clientSecret := h.clientCredentials(c)
	_, err := h.openPlatform.VerifyClientSecret(c.Request.Context(), clientID, clientSecret)
	return err == nil
}

func (h *Handler) clientCredentials(c *gin.Context) (string, string) {
	if clientID, clientSecret, ok := c.Request.BasicAuth(); ok {
		return strings.TrimSpace(clientID), strings.TrimSpace(clientSecret)
	}
	return strings.TrimSpace(c.PostForm("client_id")), strings.TrimSpace(c.PostForm("client_secret"))
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
		oauthJSONError(c, http.StatusUnauthorized, "invalid_client")
	case errors.Is(err, ErrInvalidGrant):
		oauthJSONError(c, http.StatusBadRequest, "invalid_grant")
	case errors.Is(err, ErrUnsupportedGrantType):
		oauthJSONError(c, http.StatusBadRequest, "unsupported_grant_type")
	default:
		logger.FromGin(c).Error("identity token exchange failed", zap.Error(err))
		oauthJSONError(c, http.StatusBadRequest, "invalid_request")
	}
}

func (h *Handler) loginURL(c *gin.Context) string {
	target := c.Request.URL.RequestURI()
	base := h.frontendBase
	if base == "" {
		base = h.issuer
	}
	values := url.Values{}
	values.Set("redirect", target)
	return base + "/login?" + values.Encode()
}

func authorizeRequestFromQuery(c *gin.Context) AuthorizeRequest {
	return AuthorizeRequest{
		ResponseType:        c.Query("response_type"),
		ClientID:            c.Query("client_id"),
		RedirectURI:         c.Query("redirect_uri"),
		Scope:               c.Query("scope"),
		State:               c.Query("state"),
		CodeChallenge:       c.Query("code_challenge"),
		CodeChallengeMethod: c.Query("code_challenge_method"),
		Nonce:               c.Query("nonce"),
	}
}

func bearerToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func oauthJSONError(c *gin.Context, status int, code string) {
	c.JSON(status, gin.H{"error": code})
}
