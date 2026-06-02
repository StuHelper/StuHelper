package admission

import (
	"context"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/rbac"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/platform/serviceaccount"
)

type BotCredentialVerifier interface {
	Verify(ctx context.Context, rawToken, audience, scope string) error
}

type Handler struct {
	service                *Service
	internalUserIDResolver middleware.InternalUserIDResolver
	botCredentialVerifier  BotCredentialVerifier
}

func NewHandler(
	service *Service,
	internalUserIDResolver middleware.InternalUserIDResolver,
	botCredentialVerifier BotCredentialVerifier,
) *Handler {
	if service == nil {
		panic("admission.NewHandler: service must not be nil")
	}
	if internalUserIDResolver == nil {
		panic("admission.NewHandler: internal user id resolver must not be nil")
	}
	return &Handler{
		service:                service,
		internalUserIDResolver: internalUserIDResolver,
		botCredentialVerifier:  botCredentialVerifier,
	}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, authMW gin.HandlerFunc) {
	admission := api.Group("/admission")
	admission.GET("/sessions/:token", h.handlePreviewAdmissionSession)
	admission.POST("/sessions/:token/link", authMW, h.handleLinkAdmissionSession)
	admission.GET("/me", authMW, h.handleAdmissionMe)
	admission.POST("/freshman/applications", authMW, h.handleCreateFreshmanApplication)
	admission.POST("/freshman/applications/:id/camera-captures", authMW, h.handleUploadFreshmanCameraCapture)
	admission.POST("/freshman/applications/:id/camera-handoffs", authMW, h.handleCreateFreshmanCameraHandoff)
	admission.GET("/freshman/camera-handoffs/:id", authMW, h.handleGetFreshmanCameraHandoff)
	admission.GET("/freshman/camera-handoffs/:id/events", authMW, h.handleWatchFreshmanCameraHandoff)
	admission.GET("/freshman/mobile-camera-handoffs/:token", h.handlePreviewFreshmanCameraHandoff)
	admission.POST("/freshman/mobile-camera-handoffs/:token/camera-capture", h.handleUploadFreshmanCameraHandoffCapture)
	admission.POST("/freshman/mobile-camera-handoffs/:token/continue", h.handleChooseFreshmanCameraHandoffContinuation)
	admission.POST("/school-email/academic-match", authMW, h.handleMatchSchoolEmailAcademicStudent)
	admission.POST("/school-email/request-otp", authMW, h.handleRequestSchoolEmailOTP)
	admission.POST("/school-email/verify-otp", authMW, h.handleVerifySchoolEmailOTP)
	admission.GET("/school-sso/:schoolCode/login", authMW, h.handleStartSchoolSSO)
	admission.GET("/school-sso/:schoolCode/callback", authMW, h.handleCompleteSchoolSSO)
}

func (h *Handler) RegisterBotRoutes(api *gin.RouterGroup) {
	bot := api.Group("/bot/admission")
	h.registerBotAdmissionRoutes(bot)

	memberBlacklist := api.Group("/bot/member-blacklist")
	h.registerBotMemberBlacklistRoutes(memberBlacklist)
}

func (h *Handler) registerBotAdmissionRoutes(bot *gin.RouterGroup) {
	bot.POST("/sessions", h.requireBotCredential(serviceaccount.ScopeBotAdmissionSession), h.handleCreateBotSession)
	bot.GET(
		"/sessions/member",
		h.requireBotCredential(serviceaccount.ScopeBotAdmissionSession),
		h.handleGetBotAdmissionSession,
	)
	bot.POST(
		"/sessions/member/resend",
		h.requireBotCredential(serviceaccount.ScopeBotAdmissionSession),
		h.handleResendBotAdmissionSession,
	)
	bot.POST(
		"/sessions/member/regenerate",
		h.requireBotCredential(serviceaccount.ScopeBotAdmissionSession),
		h.handleRegenerateBotAdmissionSession,
	)
	bot.POST(
		"/join-requests/events",
		h.requireBotCredential(serviceaccount.ScopeBotAdmissionEvent),
		h.handleRecordBotJoinRequestEvent,
	)
	bot.GET(
		"/sessions/pending",
		h.requireBotCredential(serviceaccount.ScopeBotAdmissionSession),
		h.handleListBotPendingActions,
	)
	bot.POST("/sessions/:id/events", h.requireBotCredential(serviceaccount.ScopeBotAdmissionEvent), h.handleRecordBotEvent)
	bot.GET(
		"/freshman/applications/pending-forward",
		h.requireBotCredential(serviceaccount.ScopeBotAdmissionForward),
		h.handleListBotPendingFreshmanForwards,
	)
	bot.POST(
		"/freshman/applications/:id/forwarded",
		h.requireBotCredential(serviceaccount.ScopeBotAdmissionForward),
		h.handleMarkBotFreshmanApplicationForwarded,
	)
	bot.POST(
		"/freshman/applications/:id/view",
		h.requireBotCredential(serviceaccount.ScopeBotAdmissionReview),
		h.handleBotViewFreshmanApplication,
	)
	bot.POST(
		"/freshman/applications/:id/review",
		h.requireBotCredential(serviceaccount.ScopeBotAdmissionReview),
		h.handleBotReviewFreshmanApplication,
	)
}

func (h *Handler) registerBotMemberBlacklistRoutes(memberBlacklist *gin.RouterGroup) {
	memberBlacklist.GET(
		"/access",
		h.requireBotCredential(serviceaccount.ScopeBotMemberBlacklistRead),
		h.handleGetBotMemberBlacklistAccess,
	)
	memberBlacklist.GET(
		"",
		h.requireBotCredential(serviceaccount.ScopeBotMemberBlacklistRead),
		h.handleListBotMemberBlacklist,
	)
	memberBlacklist.POST(
		"",
		h.requireBotCredential(serviceaccount.ScopeBotMemberBlacklistManage),
		h.handleCreateBotMemberBlacklist,
	)
	memberBlacklist.POST(
		"/release-by-subject",
		h.requireBotCredential(serviceaccount.ScopeBotMemberBlacklistManage),
		h.handleReleaseBotMemberBlacklistBySubject,
	)
	memberBlacklist.POST(
		"/:id/release",
		h.requireBotCredential(serviceaccount.ScopeBotMemberBlacklistManage),
		h.handleReleaseBotMemberBlacklist,
	)
}

func (h *Handler) RegisterAdminRoutes(admin *gin.RouterGroup) {
	h.registerAdminAdmissionRoutes(admin)
	h.registerAdminMemberBlacklistRoutes(admin)
}

func (h *Handler) registerAdminAdmissionRoutes(admin *gin.RouterGroup) {
	admin.GET(
		"/admission/policies",
		rbac.RequireCapability(capability.AdmissionPolicyRead),
		h.handleAdminListAdmissionPolicies,
	)
	admin.PUT(
		"/admission/policies/:id",
		rbac.RequireCapability(capability.AdmissionPolicyUpdate),
		h.handleAdminUpdateAdmissionPolicy,
	)
	admin.GET(
		"/admission/sessions",
		rbac.RequireCapability(capability.AdmissionSessionRead),
		h.handleAdminListAdmissionSessions,
	)
	admin.POST(
		"/admission/sessions/:id/resend",
		rbac.RequireCapability(capability.AdmissionSessionManage),
		h.handleAdminResendAdmissionSession,
	)
	admin.POST(
		"/admission/sessions/:id/regenerate",
		rbac.RequireCapability(capability.AdmissionSessionManage),
		h.handleAdminRegenerateAdmissionSession,
	)
	admin.POST(
		"/admission/sessions/:id/cancel",
		rbac.RequireCapability(capability.AdmissionSessionManage),
		h.handleAdminCancelAdmissionSession,
	)
	admin.GET(
		"/freshman-verifications",
		rbac.RequireCapability(capability.AdmissionFreshmanRead),
		h.handleAdminListFreshmanVerifications,
	)
	admin.GET(
		"/freshman-verifications/:id",
		rbac.RequireCapability(capability.AdmissionFreshmanRead),
		h.handleAdminGetFreshmanVerification,
	)
	admin.PUT(
		"/freshman-verifications/:id",
		rbac.RequireCapability(capability.AdmissionFreshmanReview),
		h.handleAdminReviewFreshmanVerification,
	)
}

func (h *Handler) registerAdminMemberBlacklistRoutes(admin *gin.RouterGroup) {
	admin.GET(
		"/member-blacklist",
		rbac.RequireCapability(capability.MemberBlacklistRead),
		h.handleListAdminMemberBlacklist,
	)
	admin.POST(
		"/member-blacklist",
		rbac.RequireCapability(capability.MemberBlacklistManage),
		h.handleCreateAdminMemberBlacklist,
	)
	admin.POST(
		"/member-blacklist/release-by-subject",
		rbac.RequireCapability(capability.MemberBlacklistManage),
		h.handleReleaseAdminMemberBlacklistBySubject,
	)
	admin.POST(
		"/member-blacklist/:id/release",
		rbac.RequireCapability(capability.MemberBlacklistManage),
		h.handleReleaseAdminMemberBlacklist,
	)
}
