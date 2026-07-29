package user

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
)

// Handler 用户模块 HTTP 处理器
type Handler struct {
	service                  *Service
	verifyLimiter            *middleware.RedisRateLimiter
	identityPhotoLimiter     *middleware.RedisRateLimiter
	bindPhoneUserLimiter     *middleware.RedisRateLimiter
	bindPhoneEndpointLimiter *middleware.RedisRateLimiter
	otpService               OTPGenerator
	smsService               SMSSender
	adminAuthorizers         AdminAuthorizers
}

type AdminAuthorizers struct {
	IdentityRead   gin.HandlerFunc
	IdentityReview gin.HandlerFunc
	StudentRead    gin.HandlerFunc
	StudentReview  gin.HandlerFunc
	SchoolRead     gin.HandlerFunc
	SchoolUpdate   gin.HandlerFunc
	SystemRead     gin.HandlerFunc
	SystemUpdate   gin.HandlerFunc
	StepUpMFA      gin.HandlerFunc
}

type HandlerOption func(*Handler)

func WithAdminAuthorizers(authorizers AdminAuthorizers) HandlerOption {
	return func(h *Handler) {
		h.adminAuthorizers = authorizers
	}
}

const (
	verifyRateLimitPerMinute        = 5
	identityPhotoUploadLimitPerDay  = 12
	bindPhoneOTPUserLimitPerMinute  = 5
	bindPhoneOTPRouteLimitPerMinute = 5
)

// NewHandler 创建用户处理器
func NewHandler(
	service *Service,
	rdb *redis.Client,
	otpService OTPGenerator,
	smsService SMSSender,
	opts ...HandlerOption,
) *Handler {
	if service == nil {
		panic("user.NewHandler: service must not be nil")
	}
	var (
		verifyLimiter            *middleware.RedisRateLimiter
		identityPhotoLimiter     *middleware.RedisRateLimiter
		bindPhoneUserLimiter     *middleware.RedisRateLimiter
		bindPhoneEndpointLimiter *middleware.RedisRateLimiter
	)
	if rdb != nil {
		verifyLimiter = middleware.NewRedisRateLimiter(rdb, verifyRateLimitPerMinute, time.Minute)
		identityPhotoLimiter = middleware.NewRedisRateLimiter(rdb, identityPhotoUploadLimitPerDay, 24*time.Hour)
		bindPhoneUserLimiter = middleware.NewRedisRateLimiter(rdb, bindPhoneOTPUserLimitPerMinute, time.Minute)
		bindPhoneEndpointLimiter = middleware.NewRedisRateLimiter(rdb, bindPhoneOTPRouteLimitPerMinute, time.Minute)
	}
	h := &Handler{
		service:                  service,
		verifyLimiter:            verifyLimiter,
		identityPhotoLimiter:     identityPhotoLimiter,
		bindPhoneUserLimiter:     bindPhoneUserLimiter,
		bindPhoneEndpointLimiter: bindPhoneEndpointLimiter,
		otpService:               otpService,
		smsService:               smsService,
	}
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
		user.GET("/identity", h.handleGetIdentity)
		user.POST("/identity", h.handleSubmitIdentity)
		if h.identityPhotoLimiter != nil {
			user.POST(
				"/identity/uploads",
				middleware.EndpointRateLimitMiddleware(h.identityPhotoLimiter, "user-identity-photo-upload"),
				h.handleUploadIdentityPhoto,
			)
		} else {
			user.POST("/identity/uploads", h.handleUploadIdentityPhoto)
		}
		user.GET("/qq-binding", h.handleGetQQBinding)
		user.POST("/qq-binding/code", h.handleCreateQQBindingCode)
		user.GET("/profile", h.handleGetProfile)
		if h.verifyLimiter != nil {
			user.POST("/profile/verify", middleware.EndpointRateLimitMiddleware(h.verifyLimiter, "user-profile-verify"), h.handleVerifyStudent)
			user.POST("/profile/school-email/academic-match", middleware.EndpointRateLimitMiddleware(h.verifyLimiter, "user-profile-school-email-academic-match"), h.handleMatchStudentEmailAcademicStudent)
			user.POST("/profile/school-email/request-otp", middleware.EndpointRateLimitMiddleware(h.verifyLimiter, "user-profile-school-email-request-otp"), h.handleRequestStudentEmailOTP)
			user.POST("/profile/school-email/verify-otp", middleware.EndpointRateLimitMiddleware(h.verifyLimiter, "user-profile-school-email-verify-otp"), h.handleVerifyStudentEmailOTP)
		} else {
			user.POST("/profile/verify", h.handleVerifyStudent)
			user.POST("/profile/school-email/academic-match", h.handleMatchStudentEmailAcademicStudent)
			user.POST("/profile/school-email/request-otp", h.handleRequestStudentEmailOTP)
			user.POST("/profile/school-email/verify-otp", h.handleVerifyStudentEmailOTP)
		}
		if h.bindPhoneUserLimiter != nil && h.bindPhoneEndpointLimiter != nil {
			user.POST(
				"/profile/bind-phone/otp",
				middleware.UserRateLimitMiddleware(h.bindPhoneUserLimiter),
				middleware.EndpointRateLimitMiddleware(h.bindPhoneEndpointLimiter, "user-profile-bind-phone-otp"),
				h.handleRequestBindPhoneOTP,
			)
		} else {
			user.POST("/profile/bind-phone/otp", h.handleRequestBindPhoneOTP)
		}
		user.POST("/profile/bind-phone", h.handleBindPhone)
		user.GET("/profile/academic-info", h.handleGetAcademicInfo)
	}

	rg.GET("/user/schools", h.handleListSchools)
}

// RegisterAdminRoutes 注册管理后台路由
func (h *Handler) RegisterAdminRoutes(admin *gin.RouterGroup) {
	admin.GET(
		"/identities",
		httputil.RouteHandlers(
			h.handleAdminListIdentities,
			h.adminAuthorizers.IdentityRead,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
	admin.GET(
		"/identities/:userID",
		httputil.RouteHandlers(
			h.handleAdminGetIdentity,
			h.adminAuthorizers.IdentityRead,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
	admin.PUT(
		"/identities/:userID",
		httputil.RouteHandlers(
			h.handleAdminReviewIdentity,
			h.adminAuthorizers.IdentityReview,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
	admin.GET(
		"/student-verifications",
		httputil.RouteHandlers(
			h.handleAdminListStudentVerifications,
			h.adminAuthorizers.StudentRead,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
	admin.PUT(
		"/student-verifications/:userID",
		httputil.RouteHandlers(
			h.handleAdminReviewStudentVerification,
			h.adminAuthorizers.StudentReview,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
	admin.GET(
		"/school-configs",
		httputil.RouteHandlers(h.handleAdminListSchoolConfigs, h.adminAuthorizers.SchoolRead)...,
	)
	admin.PUT(
		"/school-configs/:schoolID",
		httputil.RouteHandlers(
			h.handleAdminUpdateSchoolConfig,
			h.adminAuthorizers.SchoolUpdate,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
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
