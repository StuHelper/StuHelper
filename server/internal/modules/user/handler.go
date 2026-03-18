package user

import (
	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/rbac"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
)

// Handler 用户模块 HTTP 处理器
type Handler struct {
	service *Service
}

// NewHandler 创建用户处理器
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes 注册用户中心路由（挂载到 /api/v1 级别）
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMW gin.HandlerFunc) {
	user := rg.Group("/user")
	user.Use(authMW)
	{
		user.GET("/identity", h.handleGetIdentity)
		user.POST("/identity", h.handleSubmitIdentity)
		user.GET("/profile", h.handleGetProfile)
		user.POST("/profile/verify", h.handleVerifyStudent)
		user.POST("/profile/bind-phone", h.handleBindPhone)
		user.GET("/profile/academic-info", h.handleGetAcademicInfo)
	}

	// 学校列表无需认证
	rg.GET("/user/schools", h.handleListSchools)
}

// RegisterAdminRoutes 注册管理后台路由（调用方负责挂载到 /admin 组并添加认证中间件）
func (h *Handler) RegisterAdminRoutes(admin *gin.RouterGroup, permissionService rbac.PermissionService) {
	admin.GET("/identities", rbac.RequirePermission(permissionService, capability.UserIdentityRead), h.handleAdminListIdentities)
	admin.PUT("/identities/:userID", rbac.RequirePermission(permissionService, capability.UserIdentityReview), h.handleAdminReviewIdentity)
	admin.GET("/student-verifications", rbac.RequirePermission(permissionService, capability.UserStudentRead), h.handleAdminListStudentVerifications)
	admin.PUT("/student-verifications/:userID", rbac.RequirePermission(permissionService, capability.UserStudentReview), h.handleAdminReviewStudentVerification)
	admin.GET("/school-configs", rbac.RequirePermission(permissionService, capability.UserSchoolRead), h.handleAdminListSchoolConfigs)
	admin.PUT("/school-configs/:schoolID", rbac.RequirePermission(permissionService, capability.UserSchoolUpdate), h.handleAdminUpdateSchoolConfig)
	admin.GET("/system-configs", rbac.RequirePermission(permissionService, capability.UserSystemRead), h.handleAdminListSystemConfigs)
	admin.PUT("/system-configs/:key", rbac.RequirePermission(permissionService, capability.UserSystemUpdate), h.handleAdminUpdateSystemConfig)
}
