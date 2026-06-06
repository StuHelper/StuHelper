package resource

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/objectstorage"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	if service == nil {
		panic("resource.NewHandler: service must not be nil")
	}
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, authMW, optionalAuthMW gin.HandlerFunc) {
	resources := api.Group("/resources")
	resources.GET("", optionalAuthMW, middleware.RequireHealthyOptionalAuth(), h.listResources)
	resources.GET("/:resourceID", optionalAuthMW, middleware.RequireHealthyOptionalAuth(), h.getResource)
	resources.GET("/:resourceID/download-url", optionalAuthMW, middleware.RequireHealthyOptionalAuth(), h.getDownloadURL)
	resources.POST("", authMW, h.createResource)
	resources.PATCH("/:resourceID", authMW, h.updateResource)
	resources.DELETE("/:resourceID", authMW, h.deleteResource)
}

func (h *Handler) listResources(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	items, total, err := h.service.ListResources(c.Request.Context(), ListFilters{
		Query:        c.Query("query"),
		Tag:          c.Query("tag"),
		BindingType:  c.Query("bindingType"),
		BindingValue: c.Query("bindingValue"),
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		response.InternalError(c, "failed to list resources")
		return
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize})
}

func (h *Handler) getResource(c *gin.Context) {
	resourceID, err := httputil.ParseIDParam(c, "resourceID")
	if err != nil {
		response.BadRequest(c, "invalid resourceID", errs.ErrInvalidParam)
		return
	}
	item, err := h.service.GetResource(c.Request.Context(), resourceID, middleware.GetUserID(c))
	if err != nil {
		if errors.Is(err, ErrResourceNotFound) {
			response.NotFound(c, "resource not found")
			return
		}
		response.InternalError(c, "failed to get resource")
		return
	}
	response.Success(c, item)
}

func (h *Handler) createResource(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logResourceAudit(c, audit.EventDataCreate, "create", "failure", "", map[string]any{"reason": "invalid request body"})
		response.BadRequest(c, "invalid request body")
		return
	}
	item, err := h.service.CreateResource(c.Request.Context(), middleware.GetUserID(c), req)
	if err != nil {
		logResourceAudit(c, audit.EventDataCreate, "create", "failure", "", map[string]any{"reason": err.Error(), "title": req.Title})
		if respondResourceStorageError(c, err) {
			return
		}
		if isResourceBadRequestError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, "failed to create resource")
		return
	}
	logResourceAudit(c, audit.EventDataCreate, "create", "success", item.ID, map[string]any{"visibility": item.Visibility})
	response.Created(c, item)
}

func (h *Handler) updateResource(c *gin.Context) {
	resourceID, err := httputil.ParseIDParam(c, "resourceID")
	if err != nil {
		logResourceAudit(c, audit.EventDataUpdate, "update", "failure", "", map[string]any{"reason": "invalid resourceID"})
		response.BadRequest(c, "invalid resourceID", errs.ErrInvalidParam)
		return
	}
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logResourceAudit(c, audit.EventDataUpdate, "update", "failure", resourceID, map[string]any{"reason": "invalid request body"})
		response.BadRequest(c, "invalid request body")
		return
	}
	item, err := h.service.UpdateResource(c.Request.Context(), resourceID, middleware.GetUserID(c), req)
	if err != nil {
		if errors.Is(err, ErrResourceNotFound) {
			logResourceAudit(c, audit.EventDataUpdate, "update", "failure", resourceID, map[string]any{"reason": "resource not found"})
			response.NotFound(c, "resource not found")
			return
		}
		if errors.Is(err, ErrResourceForbidden) {
			logResourceAudit(c, audit.EventDataUpdate, "update", "failure", resourceID, map[string]any{"reason": "resource owner mismatch"})
			response.Forbidden(c, "resource owner mismatch")
			return
		}
		logResourceAudit(c, audit.EventDataUpdate, "update", "failure", resourceID, map[string]any{"reason": err.Error()})
		if isResourceBadRequestError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, "failed to update resource")
		return
	}
	logResourceAudit(c, audit.EventDataUpdate, "update", "success", resourceID, map[string]any{"visibility": item.Visibility})
	response.Success(c, item)
}

func (h *Handler) deleteResource(c *gin.Context) {
	resourceID, err := httputil.ParseIDParam(c, "resourceID")
	if err != nil {
		logResourceAudit(c, audit.EventDataDelete, "delete", "failure", "", map[string]any{"reason": "invalid resourceID"})
		response.BadRequest(c, "invalid resourceID", errs.ErrInvalidParam)
		return
	}
	if err := h.service.DeleteResource(c.Request.Context(), resourceID, middleware.GetUserID(c)); err != nil {
		if errors.Is(err, ErrResourceForbidden) {
			logResourceAudit(c, audit.EventDataDelete, "delete", "failure", resourceID, map[string]any{"reason": "resource owner mismatch"})
			response.Forbidden(c, "resource owner mismatch")
			return
		}
		if errors.Is(err, ErrResourceNotFound) {
			logResourceAudit(c, audit.EventDataDelete, "delete", "failure", resourceID, map[string]any{"reason": "resource not found"})
			response.NotFound(c, "resource not found")
			return
		}
		logResourceAudit(c, audit.EventDataDelete, "delete", "failure", resourceID, map[string]any{"reason": err.Error()})
		response.InternalError(c, "failed to delete resource")
		return
	}
	logResourceAudit(c, audit.EventDataDelete, "delete", "success", resourceID, nil)
	response.Success(c, gin.H{"message": "resource deleted"})
}

func (h *Handler) getDownloadURL(c *gin.Context) {
	resourceID, err := httputil.ParseIDParam(c, "resourceID")
	if err != nil {
		response.BadRequest(c, "invalid resourceID", errs.ErrInvalidParam)
		return
	}
	url, err := h.service.GetDownloadURL(c.Request.Context(), resourceID, middleware.GetUserID(c))
	if err != nil {
		if errors.Is(err, ErrResourceNotFound) {
			response.NotFound(c, "resource not found")
			return
		}
		if respondResourceStorageError(c, err) {
			return
		}
		response.InternalError(c, "failed to get download url")
		return
	}
	response.Success(c, gin.H{"url": url})
}

func respondResourceStorageError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, ErrResourceStorageMountNotFound):
		response.NotFound(c, "storage mount not found")
	case errors.Is(err, ErrResourceStorageMountDisabled):
		response.Conflict(c, "storage mount is disabled")
	case errors.Is(err, ErrResourceStorageDriverMissing):
		response.ServiceUnavailable(c, "resource storage driver is unavailable")
	case objectstorage.IsKind(err, objectstorage.ErrorKindConfig),
		objectstorage.IsKind(err, objectstorage.ErrorKindAuthentication),
		objectstorage.IsKind(err, objectstorage.ErrorKindPermission):
		response.ServiceUnavailable(c, "resource storage is misconfigured")
	case objectstorage.IsKind(err, objectstorage.ErrorKindNetwork):
		response.ServiceUnavailable(c, "resource storage is temporarily unavailable")
	case objectstorage.IsKind(err, objectstorage.ErrorKindNotFound):
		response.InternalError(c, "resource file is missing from storage")
	default:
		return false
	}
	return true
}

func logResourceAudit(c *gin.Context, eventType audit.EventType, action, result string, resourceID any, details map[string]any) {
	event := audit.Event{
		Type:         eventType,
		Resource:     "resource.item",
		ResourceType: "resource.item",
		Action:       action,
		Result:       result,
		Details:      details,
	}
	if id := stringifyResourceID(resourceID); id != "" {
		event.ResourceID = id
	}
	if result == "failure" && details != nil {
		if reason, ok := details["reason"].(string); ok {
			event.Reason = reason
		}
	}
	audit.LogFromGin(c, event)
}

func stringifyResourceID(value any) string {
	switch v := value.(type) {
	case int64:
		return strconv.FormatInt(v, 10)
	case string:
		return v
	default:
		return ""
	}
}
