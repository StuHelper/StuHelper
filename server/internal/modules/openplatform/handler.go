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
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, authMW gin.HandlerFunc) {
	group := api.Group("/open-platform")
	group.GET("/authorize", authMW, h.authorize)
	group.GET("/consent", authMW, h.getConsent)
	group.POST("/consent/accept", authMW, h.acceptConsent)
	group.POST("/consent/deny", authMW, h.denyConsent)
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
	group.POST("/apps/:appID/approve",
		rbac.RequireGlobalCapability(capability.OpenPlatformManage),
		h.approveApp,
	)
}

func (h *Handler) authorize(c *gin.Context) {
	result, err := h.service.Authorize(c.Request.Context(), authorizeRequestFromQuery(c), middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, authorizeResponse{
		RedirectURL: result.RedirectURL,
		ConsentURL:  result.ConsentURL,
		Scopes:      result.Scopes,
	})
}

func (h *Handler) getConsent(c *gin.Context) {
	page, err := h.service.GetConsentPage(c.Request.Context(), c.Query("token"), middleware.GetUserID(c))
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
	redirectURL, err := h.service.AcceptConsent(c.Request.Context(), req.Token, middleware.GetRequestID(c), middleware.GetUserID(c))
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
	redirectURL, err := h.service.DenyConsent(c.Request.Context(), req.Token, middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, redirectResponse{RedirectURL: redirectURL})
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
	payload, err := fn(c.Request.Context(), disclosureRequestFromQuery(c))
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
	userID, ok := middleware.ResolveRequiredInternalUserID(c, h.service.repo.GetInternalUserID, "failed to resolve user")
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
	reviewerID, ok := middleware.ResolveRequiredInternalUserID(c, h.service.repo.GetInternalUserID, "failed to resolve user")
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
	case errors.Is(err, ErrRedirectURINotAllowed):
		response.BadRequest(c, "open platform redirect URI is not allowed", errs.ErrOpenPlatformScopeInvalid)
	case errors.Is(err, ErrDisclosureUnavailable):
		response.ServiceUnavailable(c, "open platform disclosure unavailable", errs.ErrServiceUnavailable)
	default:
		logger.FromGin(c).Error("open platform request failed", zap.Error(err))
		response.InternalError(c, "open platform request failed")
	}
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

func disclosureRequestFromQuery(c *gin.Context) DisclosureRequest {
	return DisclosureRequest{
		ClientID:       middleware.GetAppID(c),
		CasdoorSubject: middleware.GetUserID(c),
		Scopes:         strings.Fields(c.Query("scope")),
		RedirectURI:    c.Query("redirect_uri"),
		ConsentBaseURL: c.Query("consent_base_url"),
	}
}
