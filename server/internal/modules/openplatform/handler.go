package openplatform

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

type Handler struct {
	service                *Service
	internalUserIDResolver middleware.InternalUserIDResolver
	resourceTokenVerifier  ResourceAccessTokenVerifier
	adminAuthorizers       AdminAuthorizers
}

type ResourceAccessToken struct {
	ClientID string
	Scopes   []string
}

type ResourceAccessTokenVerifier interface {
	VerifyOpenPlatformResourceAccessToken(ctx context.Context, rawToken string) (ResourceAccessToken, error)
}

type AdminAuthorizers struct {
	Manage gin.HandlerFunc
}

type HandlerOption func(*Handler)

func WithInternalUserIDResolver(resolver middleware.InternalUserIDResolver) HandlerOption {
	return func(h *Handler) {
		h.internalUserIDResolver = resolver
	}
}

func WithAdminAuthorizers(authorizers AdminAuthorizers) HandlerOption {
	return func(h *Handler) {
		h.adminAuthorizers = authorizers
	}
}

func NewHandler(service *Service, opts ...HandlerOption) *Handler {
	if service == nil {
		panic("openplatform.NewHandler: service must not be nil")
	}
	h := &Handler{service: service}
	for _, opt := range opts {
		if opt != nil {
			opt(h)
		}
	}
	return h
}

func (h *Handler) SetResourceAccessTokenVerifier(verifier ResourceAccessTokenVerifier) {
	h.resourceTokenVerifier = verifier
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, authMW gin.HandlerFunc) {
	group := api.Group("/open-platform")
	group.GET("/authorize", authMW, h.authorize)
	group.GET("/consent", authMW, h.getConsent)
	group.POST("/consent/accept", authMW, h.acceptConsent)
	group.POST("/consent/deny", authMW, h.denyConsent)
	group.GET("/profile-completion", authMW, h.getProfileCompletion)
	group.POST("/profile-completion/continue", authMW, h.continueProfileCompletion)
	group.GET("/userinfo", authMW, h.userInfo)
	group.GET("/verification", authMW, h.verification)
	group.GET("/student", authMW, h.student)
	group.GET("/phone", authMW, h.phone)
	group.GET("/apps", authMW, h.listApps)
	group.POST("/apps", authMW, h.registerApp)
	group.PATCH("/apps/:appID", authMW, h.updateOwnedAppProfile)
	group.POST("/apps/:appID/redirect-uris", authMW, h.requestRedirectURIChange)
	group.POST("/apps/:appID/redirect-uri-requests/:requestID/withdraw", authMW, h.withdrawRedirectURIRequest)
	group.POST("/apps/:appID/scopes", authMW, h.requestScopeChange)
	group.POST("/apps/:appID/scopes/:scope/withdraw", authMW, h.withdrawScopeRequest)
	group.POST("/apps/:appID/withdraw", authMW, h.withdrawOwnedApp)
	group.POST("/apps/:appID/secret/rotate", authMW, h.rotateOwnedAppSecret)
	group.GET("/apps/:appID/audit-events", authMW, h.listOwnedAppAuditEvents)
	group.GET("/consents", authMW, h.listConsents)
	group.GET("/consents/audit-events", authMW, h.listConsentAuditEvents)
	group.DELETE("/consents/:appID", authMW, h.revokeConsent)
	group.POST("/resources/access/check", h.checkResourceAccess)
}

func (h *Handler) RegisterAdminRoutes(admin *gin.RouterGroup) {
	group := admin.Group("/open-platform")
	if h.adminAuthorizers.Manage != nil {
		group.Use(h.adminAuthorizers.Manage)
	}
	group.GET("/audit-events", h.listAdminAuditEvents)
	group.GET("/consents", h.listAdminConsents)
	group.GET("/token-probe-evidence", h.listAdminTokenProbeEvidence)
	group.GET("/disclosure-report", h.getAdminDisclosureReport)
	group.GET("/apps/:appID/resource-grants", h.listAdminResourceGrants)
	group.POST("/apps/:appID/resource-grants", h.grantAdminResourceAccess)
	group.POST("/apps/:appID/resource-grants/revoke", h.revokeAdminResourceAccess)
	group.POST("/apps/:appID/consents/revoke", h.revokeAdminConsent)
	group.GET("/apps", h.listAdminApps)
	group.POST("/apps/:appID/scopes/:scope/approve", h.approveScope)
	group.POST("/apps/:appID/scopes/:scope/reject", h.rejectScope)
	group.POST("/apps/import-casdoor", h.importCasdoorApp)
	group.POST("/apps/:appID/approve", h.approveApp)
	group.POST("/apps/:appID/redirect-uri-requests/:requestID/approve", h.approveRedirectURIRequest)
	group.POST("/apps/:appID/redirect-uri-requests/:requestID/reject", h.rejectRedirectURIRequest)
	group.POST("/apps/:appID/secret/rotate", h.rotateAdminAppSecret)
	group.POST("/apps/:appID/suspend", h.suspendApp)
	group.POST("/apps/:appID/resume", h.resumeApp)
	group.POST("/apps/:appID/revoke", h.revokeApp)
}

func (h *Handler) authorize(c *gin.Context) {
	if rejectRepeatedQueryParameters(c, "client_id", "redirect_uri", "scope", "state") {
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	result, err := h.service.Authorize(c.Request.Context(), authorizeRequestFromQuery(c), userID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, authorizeResponse{
		RedirectURL:          result.RedirectURL,
		ConsentURL:           result.ConsentURL,
		ProfileCompletionURL: result.ProfileCompletionURL,
		Scopes:               result.Scopes,
		MissingFields:        result.MissingFields,
	})
}

func (h *Handler) getConsent(c *gin.Context) {
	token, ok := singleRequiredQueryValue(c, "token")
	if !ok {
		h.respondError(c, ErrConsentTokenInvalid)
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	page, err := h.service.GetConsentPage(c.Request.Context(), token, userID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, consentPageToJSON(page))
}

func (h *Handler) acceptConsent(c *gin.Context) {
	var req consentDecisionRequest
	if !bindJSON(c, &req) {
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	redirectURL, err := h.service.AcceptConsent(c.Request.Context(), req.Token, middleware.GetRequestID(c), userID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, redirectResponse{RedirectURL: redirectURL})
}

func (h *Handler) denyConsent(c *gin.Context) {
	var req consentDecisionRequest
	if !bindJSON(c, &req) {
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	redirectURL, err := h.service.DenyConsent(c.Request.Context(), req.Token, middleware.GetRequestID(c), userID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, redirectResponse{RedirectURL: redirectURL})
}

func (h *Handler) getProfileCompletion(c *gin.Context) {
	token, ok := singleRequiredQueryValue(c, "token")
	if !ok {
		h.respondError(c, ErrCompletionTokenInvalid)
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	page, err := h.service.GetProfileCompletionPage(c.Request.Context(), token, userID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, profileCompletionPageToJSON(page))
}

func (h *Handler) continueProfileCompletion(c *gin.Context) {
	var req consentDecisionRequest
	if !bindJSON(c, &req) {
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	result, err := h.service.ContinueProfileCompletion(c.Request.Context(), req.Token, userID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, authorizeResponse{
		RedirectURL:          result.RedirectURL,
		ConsentURL:           result.ConsentURL,
		ProfileCompletionURL: result.ProfileCompletionURL,
		Scopes:               result.Scopes,
		MissingFields:        result.MissingFields,
	})
}

func (h *Handler) userInfo(c *gin.Context) {
	h.disclose(c, h.service.UserInfo)
}

func (h *Handler) verification(c *gin.Context) {
	h.disclose(c, h.service.Verification)
}

func (h *Handler) student(c *gin.Context) {
	h.disclose(c, h.service.Student)
}

func (h *Handler) phone(c *gin.Context) {
	h.disclose(c, h.service.Phone)
}

func (h *Handler) checkResourceAccess(c *gin.Context) {
	var req resourceAccessCheckRequest
	if !bindJSON(c, &req) {
		return
	}
	input := ResourceAccessCheckInput{
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Action:       req.Action,
		RequestID:    middleware.GetRequestID(c),
	}
	rawToken, hasBearerToken, unsupportedAuthorization := resourceAccessAuthorization(c)
	if unsupportedAuthorization {
		h.respondError(c, ErrInvalidResourceAccessToken)
		return
	}
	if hasBearerToken {
		if rawToken == "" || h.resourceTokenVerifier == nil {
			h.respondError(c, ErrInvalidResourceAccessToken)
			return
		}
		if strings.TrimSpace(req.ClientID) != "" || strings.TrimSpace(req.ClientSecret) != "" {
			h.respondError(c, ErrInvalidResourceAccess)
			return
		}
		token, err := h.resourceTokenVerifier.VerifyOpenPlatformResourceAccessToken(c.Request.Context(), rawToken)
		if err != nil {
			h.respondError(c, ErrInvalidResourceAccessToken)
			return
		}
		input.AccessTokenClientID = token.ClientID
		input.AccessTokenScopes = token.Scopes
	}
	decision, err := h.service.CheckResourceAccess(c.Request.Context(), input)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, resourceAccessDecisionToJSON(decision))
}

func (h *Handler) disclose(c *gin.Context, fn func(context.Context, DisclosureRequest) (map[string]any, error)) {
	if rejectRepeatedQueryParameters(c, "client_id", "redirect_uri", "scope", "consent_base_url") {
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	payload, err := fn(c.Request.Context(), disclosureRequestFromQuery(c, userID))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, payload)
}

func (h *Handler) registerApp(c *gin.Context) {
	var req registerAppRequest
	if !bindJSON(c, &req) {
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	registered, err := h.service.RegisterApp(c.Request.Context(), registerAppInput(userID, req))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, registeredAppToJSON(registered))
}

func (h *Handler) listApps(c *gin.Context) {
	if rejectRepeatedListQueryParameters(c, "status") {
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	page, pageSize := httputil.ParsePage(c)
	result, err := h.service.ListApps(c.Request.Context(), ListAppsInput{
		OwnerUserID: userID,
		Status:      c.DefaultQuery("status", "all"),
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, appListToJSON(result))
}

func (h *Handler) listAdminApps(c *gin.Context) {
	if rejectRepeatedListQueryParameters(c, "status") {
		return
	}
	page, pageSize := httputil.ParsePage(c)
	result, err := h.service.ListApps(c.Request.Context(), ListAppsInput{
		Status:   c.DefaultQuery("status", AppStatusPending),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, appListToJSON(result))
}

func (h *Handler) updateOwnedAppProfile(c *gin.Context) {
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	var req updateAppProfileRequest
	if !bindJSON(c, &req) {
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	result, err := h.service.UpdateAppProfile(
		c.Request.Context(),
		updateAppProfileInput(appID, userID, middleware.GetRequestID(c), req),
	)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, appLifecycleToJSON(result))
}

func (h *Handler) listAdminAuditEvents(c *gin.Context) {
	if rejectRepeatedListQueryParameters(c, "appID", "userID", "eventType", "scope") {
		return
	}
	appID, ok := httputil.ParseOptionalInt64Query(c, "appID")
	if !ok {
		response.BadRequest(c, "invalid appID", errs.ErrInvalidParam)
		return
	}
	userID, ok := httputil.ParseOptionalInt64Query(c, "userID")
	if !ok {
		response.BadRequest(c, "invalid userID", errs.ErrInvalidParam)
		return
	}
	page, pageSize := httputil.ParsePage(c)
	result, err := h.service.ListAuditEvents(c.Request.Context(), ListAuditEventsInput{
		AppID:     appID,
		UserID:    userID,
		EventType: c.Query("eventType"),
		Scope:     c.Query("scope"),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, auditEventListToJSON(result))
}

func (h *Handler) listAdminConsents(c *gin.Context) {
	if rejectRepeatedListQueryParameters(c, "appID", "userID") {
		return
	}
	appID, ok := httputil.ParseOptionalInt64Query(c, "appID")
	if !ok {
		response.BadRequest(c, "invalid appID", errs.ErrInvalidParam)
		return
	}
	userID, ok := httputil.ParseOptionalInt64Query(c, "userID")
	if !ok {
		response.BadRequest(c, "invalid userID", errs.ErrInvalidParam)
		return
	}
	page, pageSize := httputil.ParsePage(c)
	result, err := h.service.ListAdminUserConsents(c.Request.Context(), ListAdminUserConsentsInput{
		AppID:    appID,
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, adminUserConsentListToJSON(result))
}

func (h *Handler) listOwnedAppAuditEvents(c *gin.Context) {
	if rejectRepeatedListQueryParameters(c, "eventType", "scope") {
		return
	}
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	page, pageSize := httputil.ParsePage(c)
	result, err := h.service.ListDeveloperAppAuditEvents(c.Request.Context(), ListDeveloperAppAuditEventsInput{
		OwnerUserID: userID,
		AppID:       appID,
		EventType:   c.Query("eventType"),
		Scope:       c.Query("scope"),
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, developerAppAuditEventListToJSON(result))
}

func (h *Handler) listAdminTokenProbeEvidence(c *gin.Context) {
	if rejectRepeatedListQueryParameters(c, "appID", "reviewerUserID", "result", "clientID") {
		return
	}
	appID, ok := httputil.ParseOptionalInt64Query(c, "appID")
	if !ok {
		response.BadRequest(c, "invalid appID", errs.ErrInvalidParam)
		return
	}
	reviewerUserID, ok := httputil.ParseOptionalInt64Query(c, "reviewerUserID")
	if !ok {
		response.BadRequest(c, "invalid reviewerUserID", errs.ErrInvalidParam)
		return
	}
	page, pageSize := httputil.ParsePage(c)
	result, err := h.service.ListTokenProbeEvidence(c.Request.Context(), ListTokenProbeEvidenceInput{
		AppID:          appID,
		ReviewerUserID: reviewerUserID,
		Result:         c.Query("result"),
		ClientID:       c.Query("clientID"),
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, tokenProbeEvidenceListToJSON(result))
}

func (h *Handler) listAdminResourceGrants(c *gin.Context) {
	if rejectRepeatedQueryParameters(c, "resourceType") {
		return
	}
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	result, err := h.service.ListResourceGrants(c.Request.Context(), ResourceGrantListInput{
		AppID:        appID,
		ResourceType: c.Query("resourceType"),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, resourceGrantResultToJSON(result))
}

func (h *Handler) grantAdminResourceAccess(c *gin.Context) {
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	var req resourceGrantRequest
	if !bindJSON(c, &req) {
		return
	}
	actorID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	result, err := h.service.GrantResourceAccess(c.Request.Context(), ResourceGrantInput{
		AppID:          appID,
		ReviewerUserID: actorID,
		ResourceType:   req.ResourceType,
		ResourceID:     req.ResourceID,
		Actions:        req.Actions,
		Reason:         req.Reason,
		RequestID:      middleware.GetRequestID(c),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, resourceGrantResultToJSON(result))
}

func (h *Handler) revokeAdminResourceAccess(c *gin.Context) {
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	var req resourceGrantRequest
	if !bindJSON(c, &req) {
		return
	}
	actorID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	result, err := h.service.RevokeResourceAccess(c.Request.Context(), ResourceGrantRevokeInput{
		AppID:          appID,
		ReviewerUserID: actorID,
		ResourceType:   req.ResourceType,
		ResourceID:     req.ResourceID,
		Actions:        req.Actions,
		Reason:         req.Reason,
		RequestID:      middleware.GetRequestID(c),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, resourceGrantResultToJSON(result))
}

func (h *Handler) revokeAdminConsent(c *gin.Context) {
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	var req adminConsentRevokeRequest
	if !bindJSON(c, &req) {
		return
	}
	actorID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	if err := h.service.RevokeAdminUserConsent(c.Request.Context(), AdminRevokeConsentInput{
		AppID:       appID,
		UserID:      req.UserID,
		ActorUserID: actorID,
		Reason:      req.Reason,
		Scopes:      req.Scopes,
		RequestID:   middleware.GetRequestID(c),
	}); err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, messageResponse{Message: "consent revoked"})
}

func (h *Handler) getAdminDisclosureReport(c *gin.Context) {
	if rejectRepeatedQueryParameters(c, "windowHours") {
		return
	}
	windowHours, ok := httputil.ParseOptionalIntQuery(c, "windowHours")
	if !ok {
		response.BadRequest(c, "invalid windowHours", errs.ErrInvalidParam)
		return
	}
	result, err := h.service.DisclosureReport(c.Request.Context(), DisclosureReportInput{
		WindowHours: windowHours,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, disclosureReportToJSON(result))
}

func (h *Handler) listConsents(c *gin.Context) {
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	consents, err := h.service.ListUserConsents(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, userConsentsToJSON(consents))
}

func (h *Handler) listConsentAuditEvents(c *gin.Context) {
	if rejectRepeatedListQueryParameters(c, "appID", "eventType", "scope") {
		return
	}
	appID, ok := httputil.ParseOptionalInt64Query(c, "appID")
	if !ok {
		response.BadRequest(c, "invalid appID", errs.ErrInvalidParam)
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	page, pageSize := httputil.ParsePage(c)
	result, err := h.service.ListUserConsentAuditEvents(c.Request.Context(), ListUserConsentAuditEventsInput{
		UserID:    userID,
		AppID:     appID,
		EventType: c.Query("eventType"),
		Scope:     c.Query("scope"),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, userConsentAuditEventListToJSON(result))
}

func (h *Handler) revokeConsent(c *gin.Context) {
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	if err := h.service.RevokeUserConsent(c.Request.Context(), RevokeConsentInput{
		UserID:    userID,
		AppID:     appID,
		Scopes:    revokeScopesFromQuery(c),
		RequestID: middleware.GetRequestID(c),
	}); err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, messageResponse{Message: "consent revoked"})
}

func (h *Handler) rotateOwnedAppSecret(c *gin.Context) {
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	var req secretRotationRequest
	if !bindOptionalJSON(c, &req) {
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	rotated, err := h.service.RotateAppSecret(c.Request.Context(), RotateAppSecretInput{
		AppID:       appID,
		ActorUserID: userID,
		OwnerUserID: userID,
		ActorType:   "developer",
		Reason:      req.Reason,
		RequestID:   middleware.GetRequestID(c),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, rotatedSecretToJSON(rotated))
}

func (h *Handler) requestRedirectURIChange(c *gin.Context) {
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	var req redirectURIChangeRequest
	if !bindJSON(c, &req) {
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	redirectRequest, err := h.service.RequestRedirectURIChange(c.Request.Context(), RedirectURIChangeInput{
		AppID:        appID,
		OwnerUserID:  userID,
		RedirectURIs: req.RedirectURIs,
		Reason:       req.Reason,
		RequestID:    middleware.GetRequestID(c),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, redirectURIRequestToJSON(redirectRequest))
}

func (h *Handler) withdrawRedirectURIRequest(c *gin.Context) {
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	redirectURIRequestID, ok := parseInt64Path(c, "requestID")
	if !ok {
		return
	}
	var req lifecycleActionRequest
	if !bindJSON(c, &req) {
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	withdrawn, err := h.service.WithdrawRedirectURIRequest(c.Request.Context(), RedirectURIWithdrawalInput{
		AppID:                appID,
		RedirectURIRequestID: redirectURIRequestID,
		OwnerUserID:          userID,
		Reason:               req.Reason,
		RequestID:            middleware.GetRequestID(c),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, redirectURIRequestToJSON(withdrawn))
}

func (h *Handler) requestScopeChange(c *gin.Context) {
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	var req scopeChangeRequest
	if !bindJSON(c, &req) {
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	result, err := h.service.RequestScopeChange(c.Request.Context(), ScopeChangeInput{
		AppID:       appID,
		OwnerUserID: userID,
		Scopes:      scopeChangeInput(req),
		RequestID:   middleware.GetRequestID(c),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, scopeChangeToJSON(result))
}

func (h *Handler) withdrawScopeRequest(c *gin.Context) {
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	var req lifecycleActionRequest
	if !bindJSON(c, &req) {
		return
	}
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	withdrawn, err := h.service.WithdrawScopeRequest(c.Request.Context(), ScopeWithdrawalInput{
		AppID:       appID,
		OwnerUserID: userID,
		Scope:       c.Param("scope"),
		Reason:      req.Reason,
		RequestID:   middleware.GetRequestID(c),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, scopeRequestToJSON(withdrawn))
}

func (h *Handler) approveScope(c *gin.Context) {
	h.reviewScope(c, func(ctx context.Context, appID int64, scope string, reviewerID int64, note string, requestID string) error {
		return h.service.ApproveScope(ctx, ApproveScopeInput{
			AppID:          appID,
			Scope:          scope,
			ReviewerUserID: reviewerID,
			DecisionNote:   note,
			RequestID:      requestID,
		})
	})
}

func (h *Handler) rejectScope(c *gin.Context) {
	h.reviewScope(c, func(ctx context.Context, appID int64, scope string, reviewerID int64, note string, requestID string) error {
		return h.service.RejectScope(ctx, RejectScopeInput{
			AppID:          appID,
			Scope:          scope,
			ReviewerUserID: reviewerID,
			DecisionNote:   note,
			RequestID:      requestID,
		})
	})
}

func (h *Handler) reviewScope(
	c *gin.Context,
	fn func(context.Context, int64, string, int64, string, string) error,
) {
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	var req approveScopeRequest
	if !bindOptionalJSON(c, &req) {
		return
	}
	reviewerID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	err := fn(c.Request.Context(), appID, c.Param("scope"), reviewerID, req.DecisionNote, middleware.GetRequestID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, messageResponse{Message: "scope reviewed"})
}

func (h *Handler) approveApp(c *gin.Context) {
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	reviewerID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	approved, err := h.service.ApproveAppWithAudit(c.Request.Context(), ApproveAppInput{
		AppID:          appID,
		ReviewerUserID: reviewerID,
		RequestID:      middleware.GetRequestID(c),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, approvedAppToJSON(approved))
}

func (h *Handler) approveRedirectURIRequest(c *gin.Context) {
	h.reviewRedirectURIRequest(c, h.service.ApproveRedirectURIRequest)
}

func (h *Handler) rejectRedirectURIRequest(c *gin.Context) {
	h.reviewRedirectURIRequest(c, h.service.RejectRedirectURIRequest)
}

func (h *Handler) reviewRedirectURIRequest(
	c *gin.Context,
	fn func(context.Context, RedirectURIReviewInput) (RedirectURIRequest, error),
) {
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	redirectURIRequestID, ok := parseInt64Path(c, "requestID")
	if !ok {
		return
	}
	var req redirectURIReviewRequest
	if !bindOptionalJSON(c, &req) {
		return
	}
	reviewerID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	reviewed, err := fn(c.Request.Context(), RedirectURIReviewInput{
		AppID:                appID,
		RedirectURIRequestID: redirectURIRequestID,
		ReviewerUserID:       reviewerID,
		DecisionNote:         req.DecisionNote,
		RequestID:            middleware.GetRequestID(c),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, redirectURIRequestToJSON(reviewed))
}

func (h *Handler) rotateAdminAppSecret(c *gin.Context) {
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	var req secretRotationRequest
	if !bindOptionalJSON(c, &req) {
		return
	}
	actorID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	rotated, err := h.service.RotateAppSecret(c.Request.Context(), RotateAppSecretInput{
		AppID:       appID,
		ActorUserID: actorID,
		ActorType:   "admin",
		Reason:      req.Reason,
		RequestID:   middleware.GetRequestID(c),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, rotatedSecretToJSON(rotated))
}

func (h *Handler) suspendApp(c *gin.Context) {
	h.updateAppLifecycle(c, h.service.SuspendApp)
}

func (h *Handler) resumeApp(c *gin.Context) {
	h.updateAppLifecycle(c, h.service.ResumeApp)
}

func (h *Handler) revokeApp(c *gin.Context) {
	h.updateAppLifecycle(c, h.service.RevokeApp)
}

func (h *Handler) withdrawOwnedApp(c *gin.Context) {
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	var req lifecycleActionRequest
	if !bindJSON(c, &req) {
		return
	}
	ownerID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	result, err := h.service.WithdrawApp(c.Request.Context(), AppWithdrawalInput{
		AppID:       appID,
		OwnerUserID: ownerID,
		Reason:      req.Reason,
		RequestID:   middleware.GetRequestID(c),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, appLifecycleToJSON(result))
}

func (h *Handler) updateAppLifecycle(
	c *gin.Context,
	fn func(context.Context, AppLifecycleActionInput) (*AppLifecycleResult, error),
) {
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	var req lifecycleActionRequest
	if !bindJSON(c, &req) {
		return
	}
	actorID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	result, err := fn(c.Request.Context(), AppLifecycleActionInput{
		AppID:       appID,
		ActorUserID: actorID,
		Reason:      req.Reason,
		RequestID:   middleware.GetRequestID(c),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, appLifecycleToJSON(result))
}

func (h *Handler) importCasdoorApp(c *gin.Context) {
	var req importCasdoorAppRequest
	if !bindJSON(c, &req) {
		return
	}
	importerID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	imported, err := h.service.ImportCasdoorApp(
		c.Request.Context(),
		importCasdoorAppInput(importerID, middleware.GetRequestID(c), req),
	)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, importedAppToJSON(imported))
}

func (h *Handler) resolveCurrentUserID(c *gin.Context) (int64, bool) {
	if h.internalUserIDResolver == nil {
		logger.FromGin(c).Error("open platform internal user resolver is not configured")
		response.InternalError(c, "failed to resolve user")
		return 0, false
	}
	return middleware.ResolveRequiredInternalUserID(c, h.internalUserIDResolver, "failed to resolve user")
}

func resourceAccessBearerToken(c *gin.Context) (string, bool) {
	token, ok, _ := resourceAccessAuthorization(c)
	return token, ok
}

func resourceAccessAuthorization(c *gin.Context) (string, bool, bool) {
	if c != nil && c.Request != nil && len(c.Request.Header.Values("Authorization")) > 1 {
		return "", false, true
	}
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if header == "" {
		return "", false, false
	}
	const prefix = "bearer "
	if !strings.HasPrefix(strings.ToLower(header), prefix) {
		return "", false, true
	}
	return strings.TrimSpace(header[len(prefix):]), true, false
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		response.BadRequest(c, "invalid request body")
		return false
	}
	return true
}

func bindOptionalJSON(c *gin.Context, target any) bool {
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		return true
	}
	if err := c.ShouldBindJSON(target); err != nil {
		if errors.Is(err, io.EOF) {
			return true
		}
		response.BadRequest(c, "invalid request body")
		return false
	}
	return true
}

func parseInt64Path(c *gin.Context, key string) (int64, bool) {
	value, err := parseInt64(c.Param(key))
	if err != nil {
		response.BadRequest(c, "invalid "+key, errs.ErrInvalidParam)
		return 0, false
	}
	return value, true
}

func (h *Handler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrAppNotFound):
		response.NotFound(c, "open platform app not found", errs.ErrOpenPlatformAppNotFound)
	case errors.Is(err, ErrAppAlreadyExists):
		response.Conflict(c, "open platform app already exists")
	case errors.Is(err, ErrAppNotActive), errors.Is(err, ErrAppNotApproved):
		response.Forbidden(c, "open platform app is not active", errs.ErrOpenPlatformAppInactive)
	case errors.Is(err, ErrInvalidAppProfile):
		response.BadRequest(c, "open platform app profile is invalid", errs.ErrInvalidParam)
	case errors.Is(err, ErrInvalidAuditFilter):
		response.BadRequest(c, "open platform audit filter is invalid", errs.ErrInvalidParam)
	case errors.Is(err, ErrInvalidTokenProbeFilter):
		response.BadRequest(c, "open platform token probe filter is invalid", errs.ErrInvalidParam)
	case errors.Is(err, ErrInvalidResourceAccess):
		response.BadRequest(c, "open platform resource access request is invalid", errs.ErrInvalidParam)
	case errors.Is(err, ErrInvalidResourceAccessToken):
		response.Unauthorized(c, "open platform resource access token is invalid", errs.ErrTokenInvalid)
	case errors.Is(err, ErrResourceAccessReasonRequired):
		response.BadRequest(c, "open platform resource access reason is required", errs.ErrInvalidParam)
	case errors.Is(err, ErrResourceAccessUnavailable):
		response.ServiceUnavailable(c, "open platform resource authorization unavailable", errs.ErrServiceUnavailable)
	case errors.Is(err, ErrInvalidAppStatus):
		response.BadRequest(c, "open platform app status is invalid", errs.ErrInvalidParam)
	case errors.Is(err, ErrLifecycleReasonRequired):
		response.BadRequest(c, "open platform lifecycle reason is required", errs.ErrInvalidParam)
	case errors.Is(err, ErrRedirectURIRequestNotFound):
		response.NotFound(c, "open platform redirect URI request not found", errs.ErrOpenPlatformAppNotFound)
	case errors.Is(err, ErrRedirectURIReasonRequired):
		response.BadRequest(c, "open platform redirect URI reason is required", errs.ErrInvalidParam)
	case errors.Is(err, ErrInvalidScope):
		response.BadRequest(c, "open platform scope is invalid", errs.ErrOpenPlatformScopeInvalid)
	case errors.Is(err, ErrScopeAlreadyApproved):
		response.Conflict(c, "open platform scope is already approved")
	case errors.Is(err, ErrScopeAlreadyPending):
		response.Conflict(c, "open platform scope is already pending")
	case errors.Is(err, ErrScopeReasonRequired):
		response.BadRequest(c, "open platform scope reason is required", errs.ErrInvalidParam)
	case errors.Is(err, ErrScopeNotApproved):
		response.Forbidden(c, "open platform scope is not approved", errs.ErrOpenPlatformScopeDenied)
	case errors.Is(err, ErrTokenMinimizationProbe):
		response.Forbidden(c, "open platform token minimization probe failed", errs.ErrForbidden)
	case errors.Is(err, ErrDisclosureClientMismatch):
		response.Forbidden(c, "open platform disclosure client does not match authenticated credential", errs.ErrForbidden)
	case errors.Is(err, ErrDisclosureRateLimited):
		response.RateLimitExceeded(c, "open platform disclosure rate limit exceeded")
	case errors.Is(err, ErrConsentRequired):
		h.respondConsentRequired(c, err)
	case errors.Is(err, ErrConsentTokenInvalid):
		response.BadRequest(c, "open platform consent token is invalid", errs.ErrOpenPlatformConsentInvalid)
	case errors.Is(err, ErrProfileIncomplete):
		h.respondProfileIncomplete(c, err)
	case errors.Is(err, ErrCompletionTokenInvalid):
		response.BadRequest(c, "open platform profile completion token is invalid", errs.ErrOpenPlatformConsentInvalid)
	case errors.Is(err, ErrRedirectURINotAllowed):
		response.BadRequest(c, "open platform redirect URI is not allowed", errs.ErrOpenPlatformScopeInvalid)
	case errors.Is(err, ErrDisclosureUnavailable):
		response.ServiceUnavailable(c, "open platform disclosure unavailable", errs.ErrServiceUnavailable)
	default:
		logger.FromGin(c).Error("open platform request failed", zap.Error(err))
		response.InternalError(c, "open platform request failed")
	}
}

func (h *Handler) respondProfileIncomplete(c *gin.Context, err error) {
	var completionErr ProfileCompletionRequiredError
	if errors.As(err, &completionErr) {
		response.ErrorWithDetails(c, http.StatusPreconditionRequired,
			errs.ErrOpenPlatformProfileIncomplete,
			"open platform profile completion is required",
			map[string]any{
				"profileCompletionURL": completionErr.CompletionURL,
				"missingFields":        completionErr.MissingFields,
				"scopes":               completionErr.Scopes,
			},
		)
		return
	}
	response.Error(c, http.StatusPreconditionRequired,
		errs.ErrOpenPlatformProfileIncomplete,
		"open platform profile completion is required",
	)
}

func (h *Handler) respondConsentRequired(c *gin.Context, err error) {
	var consentErr ConsentRequiredError
	if errors.As(err, &consentErr) {
		response.ErrorWithDetails(c, http.StatusPreconditionRequired,
			errs.ErrOpenPlatformConsentRequired,
			"open platform consent is required",
			map[string]any{"consentURL": consentErr.ConsentURL, "scopes": consentErr.Scopes},
		)
		return
	}
	response.Error(c, http.StatusPreconditionRequired,
		errs.ErrOpenPlatformConsentRequired,
		"open platform consent is required",
	)
}

func authorizeRequestFromQuery(c *gin.Context) AuthorizeRequest {
	return AuthorizeRequest{
		ClientID:    c.Query("client_id"),
		RedirectURI: c.Query("redirect_uri"),
		Scopes:      strings.Fields(c.Query("scope")),
		State:       c.Query("state"),
	}
}

func disclosureRequestFromQuery(c *gin.Context, userID int64) DisclosureRequest {
	tokenScopes := middleware.GetTokenScopes(c)
	return DisclosureRequest{
		ClientID:              disclosureClientIDFromQuery(c),
		AuthenticatedClientID: strings.TrimSpace(middleware.GetAppID(c)),
		AuthenticatedByBearer: requestHasBearerAuthorization(c),
		AccessTokenScopes:     normalizeDisclosureTokenScopes(tokenScopes),
		UserID:                userID,
		Scopes:                disclosureScopesFromQuery(c, tokenScopes),
		RedirectURI:           c.Query("redirect_uri"),
		ConsentBaseURL:        c.Query("consent_base_url"),
		RequestID:             middleware.GetRequestID(c),
	}
}

func disclosureScopesFromQuery(c *gin.Context, tokenScopes []string) []string {
	queryScopes := strings.Fields(c.Query("scope"))
	if len(queryScopes) > 0 || !requestHasBearerAuthorization(c) {
		return queryScopes
	}
	return normalizeDisclosureTokenScopes(tokenScopes)
}

func normalizeDisclosureTokenScopes(scopes []string) []string {
	seen := make(map[string]struct{}, len(scopes))
	normalized := make([]string, 0, len(scopes))
	for _, raw := range scopes {
		for _, scope := range disclosureScopesFromTokenScope(raw) {
			if _, ok := seen[scope]; ok {
				continue
			}
			seen[scope] = struct{}{}
			normalized = append(normalized, scope)
		}
	}
	return normalized
}

func disclosureScopesFromTokenScope(raw string) []string {
	scope := strings.TrimSpace(raw)
	switch scope {
	case "", "openid":
		return nil
	case "profile":
		return []string{ScopeProfileBasicRead}
	case "email":
		return []string{ScopeEmailRead}
	case "phone":
		return []string{ScopePhoneRead}
	case ScopeResourceRead, ScopeResourceWrite, ScopeOfflineAccess:
		return nil
	default:
		if _, ok := scopeCatalog[scope]; !ok {
			return nil
		}
		return []string{scope}
	}
}

func disclosureClientIDFromQuery(c *gin.Context) string {
	clientID, ok := c.GetQuery("client_id")
	if !ok {
		return strings.TrimSpace(middleware.GetAppID(c))
	}
	return strings.TrimSpace(clientID)
}

func requestHasBearerAuthorization(c *gin.Context) bool {
	header := c.GetHeader("Authorization")
	parts := strings.SplitN(header, " ", 2)
	return len(parts) == 2 &&
		strings.EqualFold(parts[0], "Bearer") &&
		strings.TrimSpace(parts[1]) != ""
}

func singleRequiredQueryValue(c *gin.Context, name string) (string, bool) {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return "", false
	}
	values := c.Request.URL.Query()[name]
	if len(values) != 1 {
		return "", false
	}
	value := strings.TrimSpace(values[0])
	if value == "" {
		return "", false
	}
	return value, true
}

func rejectRepeatedQueryParameters(c *gin.Context, names ...string) bool {
	if name := repeatedQueryParameterName(c, names...); name != "" {
		response.BadRequest(c, "repeated query parameter: "+name, errs.ErrInvalidParam)
		return true
	}
	return false
}

func rejectRepeatedListQueryParameters(c *gin.Context, names ...string) bool {
	listNames := append([]string{}, names...)
	listNames = append(listNames, "page", "page_size", "pageSize")
	if rejectRepeatedQueryParameters(c, listNames...) {
		return true
	}
	if queryParameterPresent(c, "page_size") && queryParameterPresent(c, "pageSize") {
		response.BadRequest(c, "ambiguous query parameter: page_size/pageSize", errs.ErrInvalidParam)
		return true
	}
	return false
}

func repeatedQueryParameterName(c *gin.Context, names ...string) string {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return ""
	}
	query := c.Request.URL.Query()
	for _, name := range names {
		if len(query[name]) > 1 {
			return name
		}
	}
	return ""
}

func queryParameterPresent(c *gin.Context, name string) bool {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return false
	}
	_, ok := c.Request.URL.Query()[name]
	return ok
}

func revokeScopesFromQuery(c *gin.Context) []string {
	scopes := c.QueryArray("scope")
	if len(scopes) == 0 {
		scopes = strings.Fields(c.Query("scopes"))
	}
	return scopes
}
