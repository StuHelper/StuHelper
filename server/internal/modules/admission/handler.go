package admission

import (
	"context"
	"net/http"

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
	admission.POST("/school-email/request-otp", authMW, h.handleRequestSchoolEmailOTP)
	admission.POST("/school-email/verify-otp", authMW, h.handleVerifySchoolEmailOTP)
	admission.GET("/school-sso/:schoolID/login", authMW, h.handleStartSchoolSSO)
	admission.GET("/school-sso/:schoolID/callback", authMW, h.handleCompleteSchoolSSO)
}

func (h *Handler) RegisterBotRoutes(api *gin.RouterGroup) {
	bot := api.Group("/bot/admission")
	bot.POST("/sessions", h.requireBotCredential(serviceaccount.ScopeBotAdmissionSession), h.handleCreateBotSession)
	bot.POST(
		"/join-requests/events",
		h.requireBotCredential(serviceaccount.ScopeBotAdmissionEvent),
		h.handleRecordBotJoinRequestEvent,
	)
	bot.GET(
		"/qq-users/:qqID/access",
		h.requireBotCredential(serviceaccount.ScopeBotAdmissionSession),
		h.handleGetBotAdmissionQQAccess,
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
	bot.POST(
		"/blacklist/:qqID/release",
		h.requireBotCredential(serviceaccount.ScopeBotAdmissionReview),
		h.handleBotReleaseAdmissionBlacklist,
	)
}

func (h *Handler) RegisterAdminRoutes(admin *gin.RouterGroup) {
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
	admin.POST(
		"/admission/blacklist/:qqID/release",
		rbac.RequireCapability(capability.AdmissionBlacklistManage),
		h.handleAdminReleaseAdmissionBlacklist,
	)
}

func notImplemented(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusNotImplemented, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "B0000004",
			"message": "admission endpoint not implemented",
		},
	})
}
