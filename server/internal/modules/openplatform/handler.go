package openplatform

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/rbac"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

type Handler struct {
	service                *Service
	internalUserIDResolver middleware.InternalUserIDResolver
}

func NewHandler(service *Service, resolvers ...middleware.InternalUserIDResolver) *Handler {
	var resolver middleware.InternalUserIDResolver
	if len(resolvers) > 0 {
		resolver = resolvers[0]
	}
	return &Handler{service: service, internalUserIDResolver: resolver}
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
	group.POST("/apps", authMW, h.registerApp)
}

func (h *Handler) RegisterAdminRoutes(admin *gin.RouterGroup) {
	group := admin.Group("/open-platform")
	group.POST("/apps/:appID/scopes/:scope/approve",
		rbac.RequireGlobalCapability(capability.OpenPlatformManage),
		h.approveScope,
	)
	group.POST("/apps/import-casdoor",
		rbac.RequireGlobalCapability(capability.OpenPlatformManage),
		h.importCasdoorApp,
	)
	group.POST("/apps/:appID/approve",
		rbac.RequireGlobalCapability(capability.OpenPlatformManage),
		h.approveApp,
	)
}

func (h *Handler) authorize(c *gin.Context) {
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
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	page, err := h.service.GetConsentPage(c.Request.Context(), c.Query("token"), userID)
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
	redirectURL, err := h.service.DenyConsent(c.Request.Context(), req.Token, userID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, redirectResponse{RedirectURL: redirectURL})
}

func (h *Handler) getProfileCompletion(c *gin.Context) {
	userID, ok := h.resolveCurrentUserID(c)
	if !ok {
		return
	}
	page, err := h.service.GetProfileCompletionPage(c.Request.Context(), c.Query("token"), userID)
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

func (h *Handler) disclose(c *gin.Context, fn func(context.Context, DisclosureRequest) (map[string]any, error)) {
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

func (h *Handler) approveScope(c *gin.Context) {
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
	err := h.service.ApproveScope(c.Request.Context(), ApproveScopeInput{
		AppID:          appID,
		Scope:          c.Param("scope"),
		ReviewerUserID: reviewerID,
		DecisionNote:   req.DecisionNote,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, messageResponse{Message: "scope approved"})
}

func (h *Handler) approveApp(c *gin.Context) {
	appID, ok := parseInt64Path(c, "appID")
	if !ok {
		return
	}
	approved, err := h.service.ApproveApp(c.Request.Context(), appID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, approvedAppToJSON(approved))
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
	imported, err := h.service.ImportCasdoorApp(c.Request.Context(), importCasdoorAppInput(importerID, req))
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
	case errors.Is(err, ErrInvalidScope):
		response.BadRequest(c, "open platform scope is invalid", errs.ErrOpenPlatformScopeInvalid)
	case errors.Is(err, ErrScopeNotApproved):
		response.Forbidden(c, "open platform scope is not approved", errs.ErrOpenPlatformScopeDenied)
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
	return DisclosureRequest{
		ClientID:       middleware.GetAppID(c),
		UserID:         userID,
		Scopes:         strings.Fields(c.Query("scope")),
		RedirectURI:    c.Query("redirect_uri"),
		ConsentBaseURL: c.Query("consent_base_url"),
	}
}
