package storage

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
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

func (h *Handler) RegisterAdminRoutes(
	api *gin.RouterGroup,
	authMW gin.HandlerFunc,
	adminMiddlewares ...gin.HandlerFunc,
) {
	admin := api.Group("/admin/storage")
	middlewares := append([]gin.HandlerFunc{authMW}, adminMiddlewares...)
	admin.Use(middlewares...)
	admin.GET("/mounts", appendRouteMiddleware(h.adminAuthorizers.Read, h.listMounts)...)
	admin.POST("/mounts", appendRouteMiddleware(h.adminAuthorizers.Update, h.createMount)...)
	admin.POST("/mounts/:mountID/health-check", appendRouteMiddleware(h.adminAuthorizers.Update, h.checkMountHealth)...)
}

func appendRouteMiddleware(authorizer gin.HandlerFunc, handler gin.HandlerFunc) []gin.HandlerFunc {
	if authorizer == nil {
		return []gin.HandlerFunc{handler}
	}
	return []gin.HandlerFunc{authorizer, handler}
}

func (h *Handler) listMounts(c *gin.Context) {
	items, err := h.service.ListMounts(c.Request.Context())
	if err != nil {
		response.InternalError(c, "failed to list storage mounts")
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *Handler) createMount(c *gin.Context) {
	var req CreateMountRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Key == "" || req.Name == "" || req.Driver == "" {
		audit.LogFromGin(c, audit.Event{
			Type:         audit.EventDataCreate,
			Category:     "admin_operation",
			Resource:     "storage.mount",
			ResourceType: "storage.mount",
			Action:       "create",
			Result:       "failure",
			Reason:       "key, name and driver are required",
			Details:      map[string]any{"reason": "key, name and driver are required"},
		})
		response.BadRequest(c, "key, name and driver are required")
		return
	}
	item, err := h.service.CreateMount(c.Request.Context(), req)
	if err != nil {
		audit.LogFromGin(c, audit.Event{
			Type:         audit.EventDataCreate,
			Category:     "admin_operation",
			Resource:     "storage.mount",
			ResourceType: "storage.mount",
			ResourceID:   req.Key,
			Action:       "create",
			Result:       "failure",
			Reason:       err.Error(),
			Details:      map[string]any{"reason": err.Error(), "key": req.Key, "driver": req.Driver},
		})
		if errors.Is(err, ErrDriverNotRegistered) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, "failed to create storage mount")
		return
	}
	audit.LogFromGin(c, audit.Event{
		Type:         audit.EventDataCreate,
		Category:     "admin_operation",
		Resource:     "storage.mount",
		ResourceType: "storage.mount",
		ResourceID:   item.Key,
		Action:       "create",
		Result:       "success",
		After:        item,
		Details:      map[string]any{"mountID": item.ID, "key": item.Key, "driver": item.Driver},
	})
	response.Created(c, item)
}

func (h *Handler) checkMountHealth(c *gin.Context) {
	mountID, err := httputil.ParseIDParam(c, "mountID")
	if err != nil {
		audit.LogFromGin(c, audit.Event{
			Type:         audit.EventDataUpdate,
			Category:     "admin_operation",
			Resource:     "storage.mount",
			ResourceType: "storage.mount",
			Action:       "health_check",
			Result:       "failure",
			Reason:       "invalid mountID",
			Details:      map[string]any{"reason": "invalid mountID"},
		})
		response.BadRequest(c, "invalid mountID", errs.ErrInvalidParam)
		return
	}
	item, err := h.service.CheckMountHealth(c.Request.Context(), mountID)
	if err != nil {
		if errors.Is(err, ErrMountNotFound) {
			audit.LogFromGin(c, audit.Event{
				Type:         audit.EventDataUpdate,
				Category:     "admin_operation",
				Resource:     "storage.mount",
				ResourceType: "storage.mount",
				ResourceID:   formatInt64(mountID),
				Action:       "health_check",
				Result:       "failure",
				Reason:       "storage mount not found",
				Details:      map[string]any{"mountID": mountID, "reason": "storage mount not found"},
			})
			response.NotFound(c, "storage mount not found")
			return
		}
		if errors.Is(err, ErrDriverNotRegistered) {
			audit.LogFromGin(c, audit.Event{
				Type:         audit.EventDataUpdate,
				Category:     "admin_operation",
				Resource:     "storage.mount",
				ResourceType: "storage.mount",
				ResourceID:   formatInt64(mountID),
				Action:       "health_check",
				Result:       "failure",
				Reason:       err.Error(),
				Details:      map[string]any{"mountID": mountID, "reason": err.Error()},
			})
			response.BadRequest(c, err.Error())
			return
		}
		audit.LogFromGin(c, audit.Event{
			Type:         audit.EventDataUpdate,
			Category:     "admin_operation",
			Resource:     "storage.mount",
			ResourceType: "storage.mount",
			ResourceID:   formatInt64(mountID),
			Action:       "health_check",
			Result:       "failure",
			Reason:       err.Error(),
			Details:      map[string]any{"mountID": mountID, "reason": err.Error()},
		})
		response.InternalError(c, "failed to check storage mount health")
		return
	}
	audit.LogFromGin(c, audit.Event{
		Type:         audit.EventDataUpdate,
		Category:     "admin_operation",
		Resource:     "storage.mount",
		ResourceType: "storage.mount",
		ResourceID:   formatInt64(item.ID),
		Action:       "health_check",
		Result:       "success",
		After:        item,
		Details:      map[string]any{"mountID": item.ID, "status": item.LastHealthStatus},
	})
	response.Success(c, item)
}

func formatInt64(value int64) string {
	return strconv.FormatInt(value, 10)
}
