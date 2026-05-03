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
	admission.POST("/freshman/applications", authMW, notImplemented)
	admission.POST("/freshman/applications/:id/camera-captures", authMW, notImplemented)
	admission.POST("/school-email/request-otp", authMW, notImplemented)
	admission.POST("/school-email/verify-otp", authMW, notImplemented)
	admission.GET("/school-sso/:schoolID/login", notImplemented)
	admission.GET("/school-sso/:schoolID/callback", notImplemented)
}

func (h *Handler) RegisterBotRoutes(api *gin.RouterGroup) {
	bot := api.Group("/bot/admission")
	bot.POST("/sessions", h.requireBotCredential(serviceaccount.ScopeBotAdmissionSession), h.handleCreateBotSession)
	bot.POST("/join-requests/events", notImplemented)
	bot.GET("/qq-users/:qqID/access", notImplemented)
	bot.GET("/sessions/pending", notImplemented)
	bot.POST("/sessions/:id/events", h.requireBotCredential(serviceaccount.ScopeBotAdmissionEvent), h.handleRecordBotEvent)
	bot.GET("/freshman/applications/pending-forward", notImplemented)
	bot.POST("/freshman/applications/:id/forwarded", notImplemented)
	bot.POST(
		"/freshman/applications/:id/review",
		h.requireBotCredential(serviceaccount.ScopeBotAdmissionReview),
		h.handleBotReviewFreshmanApplication,
	)
}

func (h *Handler) RegisterAdminRoutes(admin *gin.RouterGroup) {
	admin.GET("/admission/policies", notImplemented)
	admin.PUT("/admission/policies/:id", notImplemented)
	admin.GET("/admission/sessions", notImplemented)
	admin.GET("/freshman-verifications", notImplemented)
	admin.GET("/freshman-verifications/:id", notImplemented)
	admin.PUT(
		"/freshman-verifications/:id",
		rbac.RequireCapability(capability.AdmissionFreshmanReview),
		h.handleAdminReviewFreshmanVerification,
	)
	admin.POST("/admission/blacklist/:qqID/release", notImplemented)
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
