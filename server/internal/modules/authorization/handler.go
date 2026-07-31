package authorization

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

type AdminAuthorizers struct {
	Manage    gin.HandlerFunc
	StepUpMFA gin.HandlerFunc
}

type Handler struct {
	service     *Service
	authorizers AdminAuthorizers
}

func NewHandler(service *Service, authorizers AdminAuthorizers) *Handler {
	if service == nil {
		panic("authorization.NewHandler: service is required")
	}
	return &Handler{service: service, authorizers: authorizers}
}

func (h *Handler) RegisterAdminRoutes(admin *gin.RouterGroup) {
	grants := admin.Group("/authorization/grants")
	grants.GET("", httputil.RouteHandlers(h.listGrants, h.authorizers.Manage)...)
	grants.GET("/:grantID", httputil.RouteHandlers(h.getGrant, h.authorizers.Manage)...)
	grants.POST(
		"",
		httputil.RouteHandlers(h.createGrant, h.authorizers.Manage, h.authorizers.StepUpMFA)...,
	)
	grants.POST(
		"/:grantID/revoke",
		httputil.RouteHandlers(h.revokeGrant, h.authorizers.Manage, h.authorizers.StepUpMFA)...,
	)
	grants.POST(
		"/:grantID/reconcile",
		httputil.RouteHandlers(h.reconcileGrant, h.authorizers.Manage, h.authorizers.StepUpMFA)...,
	)
}

type createGrantHTTPRequest struct {
	SubjectUserID int64   `json:"subjectUserID" binding:"required,min=1"`
	Role          Role    `json:"role" binding:"required"`
	SchoolID      *int64  `json:"schoolID"`
	SectionID     *string `json:"sectionID"`
	Reason        string  `json:"reason" binding:"required"`
}

type mutateGrantHTTPRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type grantResponse struct {
	ID                 int64            `json:"id"`
	SubjectUserID      int64            `json:"subjectUserID"`
	SubjectUsername    string           `json:"subjectUsername"`
	SubjectDisplayName string           `json:"subjectDisplayName"`
	Role               Role             `json:"role"`
	SchoolID           *int64           `json:"schoolID,omitempty"`
	SectionID          *string          `json:"sectionID,omitempty"`
	DesiredState       DesiredState     `json:"desiredState"`
	ProjectionStatus   ProjectionStatus `json:"projectionStatus"`
	Revision           int64            `json:"revision"`
	Reason             string           `json:"reason"`
	ActivatedAt        *time.Time       `json:"activatedAt,omitempty"`
	RevokedAt          *time.Time       `json:"revokedAt,omitempty"`
	ProjectedAt        *time.Time       `json:"projectedAt,omitempty"`
	LastError          *string          `json:"lastError,omitempty"`
	CreatedAt          time.Time        `json:"createdAt"`
	UpdatedAt          time.Time        `json:"updatedAt"`
}

type grantListResponse struct {
	List  []grantResponse `json:"list"`
	Total int             `json:"total"`
}

type grantMutationResponse struct {
	Grant   grantResponse `json:"grant"`
	Changed bool          `json:"changed"`
}

func (h *Handler) listGrants(c *gin.Context) {
	filter, ok := parseGrantListFilter(c)
	if !ok {
		return
	}
	result, err := h.service.ListGrants(c.Request.Context(), filter)
	if err != nil {
		h.respondError(c, "list", 0, err)
		return
	}
	items := make([]grantResponse, 0, len(result.Items))
	for _, grant := range result.Items {
		items = append(items, grantToResponse(grant))
	}
	response.Success(c, grantListResponse{List: items, Total: result.Total})
}

func (h *Handler) getGrant(c *gin.Context) {
	grantID, ok := parseGrantID(c)
	if !ok {
		return
	}
	grant, err := h.service.GetGrant(c.Request.Context(), grantID)
	if err != nil {
		h.respondError(c, "get", grantID, err)
		return
	}
	response.Success(c, grantToResponse(grant))
}

func (h *Handler) createGrant(c *gin.Context) {
	var request createGrantHTTPRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid authorization grant request")
		return
	}
	actorID, ok := h.resolveActor(c)
	if !ok {
		return
	}
	result, err := h.service.CreateGrant(c.Request.Context(), CreateGrantInput{
		SubjectUserID: request.SubjectUserID,
		Role:          request.Role,
		SchoolID:      request.SchoolID,
		SectionID:     request.SectionID,
		Reason:        request.Reason,
		ActorUserID:   actorID,
	})
	if err != nil {
		h.respondError(c, "create", 0, err)
		return
	}
	response.Created(c, grantMutationResponse{
		Grant:   grantToResponse(result.Grant),
		Changed: result.Changed,
	})
}

func (h *Handler) revokeGrant(c *gin.Context) {
	grantID, ok := parseGrantID(c)
	if !ok {
		return
	}
	request, ok := parseMutationRequest(c)
	if !ok {
		return
	}
	actorID, ok := h.resolveActor(c)
	if !ok {
		return
	}
	result, err := h.service.RevokeGrant(c.Request.Context(), RevokeGrantInput{
		GrantID:     grantID,
		Reason:      request.Reason,
		ActorUserID: actorID,
	})
	if err != nil {
		h.respondError(c, "revoke", grantID, err)
		return
	}
	response.Success(c, grantMutationResponse{
		Grant:   grantToResponse(result.Grant),
		Changed: result.Changed,
	})
}

func (h *Handler) reconcileGrant(c *gin.Context) {
	grantID, ok := parseGrantID(c)
	if !ok {
		return
	}
	request, ok := parseMutationRequest(c)
	if !ok {
		return
	}
	actorID, ok := h.resolveActor(c)
	if !ok {
		return
	}
	result, err := h.service.ReconcileGrant(c.Request.Context(), ReconcileGrantInput{
		GrantID:     grantID,
		Reason:      request.Reason,
		ActorUserID: actorID,
	})
	if err != nil {
		h.respondError(c, "reconcile", grantID, err)
		return
	}
	response.Success(c, grantMutationResponse{
		Grant:   grantToResponse(result.Grant),
		Changed: result.Changed,
	})
}

func (h *Handler) resolveActor(c *gin.Context) (int64, bool) {
	actorID, err := h.service.ResolveInternalUserID(c.Request.Context(), middleware.GetUserID(c))
	if err == nil {
		return actorID, true
	}
	h.respondError(c, "resolve_actor", 0, err)
	return 0, false
}

func (h *Handler) respondError(c *gin.Context, operation string, grantID int64, err error) {
	switch {
	case errors.Is(err, ErrInvalidGrant):
		response.BadRequest(c, "invalid authorization grant")
	case errors.Is(err, ErrTargetUserNotFound):
		response.NotFound(c, "authorization target user not found")
	case errors.Is(err, ErrSchoolNotFound):
		response.NotFound(c, "authorization scope school not found")
	case errors.Is(err, ErrGrantNotFound):
		response.NotFound(c, "authorization grant not found")
	case errors.Is(err, ErrActorUserNotFound):
		response.Forbidden(c, "authenticated user is not provisioned")
	case errors.Is(err, ErrLastSuperAdmin):
		response.Conflict(c, "at least one applied super administrator must remain")
	default:
		logger.FromGin(c).Error(
			"authorization grant operation failed",
			zap.String("operation", operation),
			zap.Int64("grant_id", grantID),
			zap.Error(err),
		)
		response.InternalError(c, "authorization grant operation failed")
	}
}

func parseGrantID(c *gin.Context) (int64, bool) {
	grantID, err := strconv.ParseInt(c.Param("grantID"), 10, 64)
	if err != nil || grantID <= 0 {
		response.BadRequest(c, "invalid authorization grant ID")
		return 0, false
	}
	return grantID, true
}

func parseMutationRequest(c *gin.Context) (mutateGrantHTTPRequest, bool) {
	var request mutateGrantHTTPRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid authorization grant request")
		return mutateGrantHTTPRequest{}, false
	}
	return request, true
}

func parseGrantListFilter(c *gin.Context) (ListGrantsFilter, bool) {
	page, pageSize := httputil.ParsePage(c)
	filter := ListGrantsFilter{
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}
	if raw := strings.TrimSpace(c.Query("subjectUserID")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			response.BadRequest(c, "invalid authorization subject user ID")
			return ListGrantsFilter{}, false
		}
		filter.SubjectUserID = &value
	}
	if raw := strings.TrimSpace(c.Query("role")); raw != "" {
		value := Role(raw)
		filter.Role = &value
	}
	if raw := strings.TrimSpace(c.Query("desiredState")); raw != "" {
		value := DesiredState(raw)
		filter.DesiredState = &value
	}
	if raw := strings.TrimSpace(c.Query("projectionStatus")); raw != "" {
		value := ProjectionStatus(raw)
		filter.Projection = &value
	}
	normalized, err := normalizeListGrantsFilter(filter)
	if err != nil {
		response.BadRequest(c, "invalid authorization grant filter")
		return ListGrantsFilter{}, false
	}
	return normalized, true
}

func grantToResponse(grant Grant) grantResponse {
	return grantResponse{
		ID:                 grant.ID,
		SubjectUserID:      grant.SubjectUserID,
		SubjectUsername:    grant.SubjectUsername,
		SubjectDisplayName: grant.SubjectDisplayName,
		Role:               grant.Role,
		SchoolID:           grant.SchoolID,
		SectionID:          grant.SectionID,
		DesiredState:       grant.DesiredState,
		ProjectionStatus:   grant.ProjectionStatus,
		Revision:           grant.Revision,
		Reason:             grant.Reason,
		ActivatedAt:        grant.ActivatedAt,
		RevokedAt:          grant.RevokedAt,
		ProjectedAt:        grant.ProjectedAt,
		LastError:          grant.LastError,
		CreatedAt:          grant.CreatedAt,
		UpdatedAt:          grant.UpdatedAt,
	}
}
