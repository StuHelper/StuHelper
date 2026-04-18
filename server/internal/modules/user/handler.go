package user

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/rbac"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

// Handler 用户模块 HTTP 处理器
type Handler struct {
	service       *Service
	verifyLimiter *middleware.RedisRateLimiter
	otpService    OTPGenerator
	smsService    SMSSender
}

// NewHandler 创建用户处理器
func NewHandler(service *Service, rdb *redis.Client, otpService OTPGenerator, smsService SMSSender) *Handler {
	var verifyLimiter *middleware.RedisRateLimiter
	if rdb != nil {
		verifyLimiter = middleware.NewRedisRateLimiter(rdb, 5, time.Minute)
	}
	return &Handler{
		service:       service,
		verifyLimiter: verifyLimiter,
		otpService:    otpService,
		smsService:    smsService,
	}
}

// RegisterRoutes 注册用户中心路由（挂载到 /api/v1 级别）
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMW gin.HandlerFunc) {
	user := rg.Group("/user")
	user.Use(authMW)
	{
		user.GET("/me", h.handleGetUserSurface)
		user.GET("/identity", h.handleGetIdentity)
		user.POST("/identity", h.handleSubmitIdentity)
		user.POST("/identity/uploads", h.handleUploadIdentityPhoto)
		user.GET("/profile", h.handleGetProfile)
		if h.verifyLimiter != nil {
			user.POST("/profile/verify", middleware.EndpointRateLimitMiddleware(h.verifyLimiter, "user-profile-verify"), h.handleVerifyStudent)
		} else {
			user.POST("/profile/verify", h.handleVerifyStudent)
		}
		user.POST("/profile/bind-phone/otp", h.handleRequestBindPhoneOTP)
		user.POST("/profile/bind-phone", h.handleBindPhone)
		user.GET("/profile/academic-info", h.handleGetAcademicInfo)
	}

	rg.GET("/user/schools", h.handleListSchools)
}

// RegisterAdminRoutes 注册管理后台路由
func (h *Handler) RegisterAdminRoutes(admin *gin.RouterGroup) {
	admin.GET("/identities", rbac.RequireGlobalCapability(capability.UserIdentityRead), h.handleAdminListIdentities)
	admin.PUT("/identities/:userID", rbac.RequireGlobalCapability(capability.UserIdentityReview), h.handleAdminReviewIdentity)
	admin.GET("/student-verifications", rbac.RequireCapability(capability.UserStudentRead), h.handleAdminListStudentVerifications)
	admin.PUT("/student-verifications/:userID", rbac.RequireCapability(capability.UserStudentReview), h.handleAdminReviewStudentVerification)
	admin.GET("/school-configs", rbac.RequireCapability(capability.UserSchoolRead), h.handleAdminListSchoolConfigs)
	admin.PUT("/school-configs/:schoolID", rbac.RequireCapability(capability.UserSchoolUpdate), h.handleAdminUpdateSchoolConfig)
	admin.GET("/system-configs", rbac.RequireGlobalCapability(capability.UserSystemRead), h.handleAdminListSystemConfigs)
	admin.PUT("/system-configs/:key", rbac.RequireGlobalCapability(capability.UserSystemUpdate), h.handleAdminUpdateSystemConfig)
}
