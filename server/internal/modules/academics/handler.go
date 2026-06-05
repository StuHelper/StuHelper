package academics

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

type Handler struct {
	service          *Service
	adminAuthorizers AdminAuthorizers
}

type AdminAuthorizers struct {
	Read   gin.HandlerFunc
	Update gin.HandlerFunc
}

type HandlerOption func(*Handler)

func WithAdminAuthorizers(authorizers AdminAuthorizers) HandlerOption {
	return func(h *Handler) {
		h.adminAuthorizers = authorizers
	}
}

func NewHandler(service *Service, opts ...HandlerOption) *Handler {
	h := &Handler{service: service}
	for _, opt := range opts {
		if opt != nil {
			opt(h)
		}
	}
	return h
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, authMW gin.HandlerFunc) {
	academicsGroup := api.Group("/academics")
	academicsGroup.GET("/terms", h.listTerms)
	academicsGroup.GET("/offerings", h.listOfferings)
	academicsGroup.GET("/offerings/:offeringID", h.getOffering)

	myGroup := academicsGroup.Group("/me")
	myGroup.Use(authMW)
	myGroup.GET("/courses", h.listMyCourses)
	myGroup.GET("/schedule", h.listMySchedule)

	admin := api.Group("/admin/academics")
	admin.Use(authMW)
	admin.GET("/sources", httputil.RouteHandlers(h.listSources, h.adminAuthorizers.Read)...)
	admin.GET("/import-jobs", httputil.RouteHandlers(h.listImportJobs, h.adminAuthorizers.Read)...)
	admin.POST("/import-jobs", httputil.RouteHandlers(h.triggerImport, h.adminAuthorizers.Update)...)
}

func (h *Handler) listSources(c *gin.Context) {
	items, err := h.service.ListSources(c.Request.Context())
	if err != nil {
		response.InternalError(c, "failed to list academic sources")
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *Handler) listImportJobs(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	items, total, err := h.service.ListImportJobs(c.Request.Context(), page, pageSize)
	if err != nil {
		response.InternalError(c, "failed to list academic import jobs")
		return
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize})
}

func (h *Handler) triggerImport(c *gin.Context) {
	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.SourceKey == "" {
		response.BadRequest(c, "sourceKey is required")
		return
	}
	job, err := h.service.TriggerImport(c.Request.Context(), req.SourceKey, middleware.GetUserID(c))
	if err != nil {
		if errors.Is(err, ErrSourceNotFound) {
			response.NotFound(c, "academic source not found", errs.ErrNotFound)
			return
		}
		if errors.Is(err, ErrSourceDisabled) {
			response.NotFound(c, "academic source is unavailable", errs.ErrNotFound)
			return
		}
		audit.LogFromGin(c, audit.Event{
			Type:         audit.EventDataCreate,
			Category:     "admin_operation",
			Resource:     "academics.import_job",
			ResourceType: "academics.import_job",
			ResourceID:   req.SourceKey,
			Action:       "trigger",
			Result:       "failure",
			Reason:       err.Error(),
			Details:      map[string]any{"sourceKey": req.SourceKey, "reason": err.Error()},
		})
		response.InternalError(c, "failed to trigger academic import")
		return
	}
	audit.LogFromGin(c, audit.Event{
		Type:         audit.EventDataCreate,
		Category:     "admin_operation",
		Resource:     "academics.import_job",
		ResourceType: "academics.import_job",
		ResourceID:   strconv.FormatInt(job.ID, 10),
		Action:       "trigger",
		Result:       "success",
		Details:      map[string]any{"sourceKey": req.SourceKey, "jobID": job.ID},
	})
	response.Created(c, job)
}

func (h *Handler) listTerms(c *gin.Context) {
	items, err := h.service.ListTerms(c.Request.Context())
	if err != nil {
		response.InternalError(c, "failed to list academic terms")
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *Handler) listOfferings(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	filters := OfferingFilters{
		TermCode:       c.Query("termCode"),
		SchoolName:     c.Query("schoolName"),
		DepartmentName: c.Query("departmentName"),
		CourseQuery:    c.Query("query"),
		TeacherQuery:   c.Query("teacherName"),
		Page:           page,
		PageSize:       pageSize,
	}
	items, total, err := h.service.ListOfferings(c.Request.Context(), filters)
	if err != nil {
		response.InternalError(c, "failed to list academic offerings")
		return
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize})
}

func (h *Handler) getOffering(c *gin.Context) {
	offeringID, err := httputil.ParseIDParam(c, "offeringID")
	if err != nil {
		response.BadRequest(c, "invalid offeringID", errs.ErrInvalidParam)
		return
	}
	item, err := h.service.GetOffering(c.Request.Context(), offeringID)
	if err != nil {
		if errors.Is(err, ErrOfferingNotFound) {
			response.NotFound(c, "academic offering not found")
			return
		}
		response.InternalError(c, "failed to get academic offering")
		return
	}
	response.Success(c, item)
}

func (h *Handler) listMyCourses(c *gin.Context) {
	items, err := h.service.ListMyCourses(c.Request.Context(), middleware.GetUserID(c), c.Query("termCode"))
	if err != nil {
		response.InternalError(c, "failed to list my courses")
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *Handler) listMySchedule(c *gin.Context) {
	items, err := h.service.ListMySchedule(c.Request.Context(), middleware.GetUserID(c), c.Query("termCode"))
	if err != nil {
		response.InternalError(c, "failed to list my schedule")
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *Handler) Health(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
