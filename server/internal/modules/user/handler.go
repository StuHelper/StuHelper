package user

import (
	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
)

// Handler 用户模块 HTTP 处理器
type Handler struct {
	service          *Service
	adminAuthorizers AdminAuthorizers
}

type AdminAuthorizers struct {
	SystemRead   gin.HandlerFunc
	SystemUpdate gin.HandlerFunc
	StepUpMFA    gin.HandlerFunc
}

type HandlerOption func(*Handler)

func WithAdminAuthorizers(authorizers AdminAuthorizers) HandlerOption {
	return func(h *Handler) {
		h.adminAuthorizers = authorizers
	}
}

// NewHandler 创建用户处理器
func NewHandler(
	service *Service,
	opts ...HandlerOption,
) *Handler {
	if service == nil {
		panic("user.NewHandler: service must not be nil")
	}
	h := &Handler{service: service}
	for _, opt := range opts {
		if opt != nil {
			opt(h)
		}
	}
	return h
}

// RegisterRoutes 注册用户中心路由（挂载到 /api/v1 级别）
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMW gin.HandlerFunc) {
	user := rg.Group("/user")
	user.Use(authMW)
	{
		user.GET("/me", h.handleGetUserSurface)
		user.GET("/qq-binding", h.handleGetQQBinding)
		user.POST("/qq-binding/code", h.handleCreateQQBindingCode)
	}
}

// RegisterAdminRoutes 注册管理后台路由
func (h *Handler) RegisterAdminRoutes(admin *gin.RouterGroup) {
	admin.GET(
		"/system-configs",
		httputil.RouteHandlers(h.handleAdminListSystemConfigs, h.adminAuthorizers.SystemRead)...,
	)
	admin.PUT(
		"/system-configs/:key",
		httputil.RouteHandlers(
			h.handleAdminUpdateSystemConfig,
			h.adminAuthorizers.SystemUpdate,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
}
