package admission

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, authMW gin.HandlerFunc) {
	admission := api.Group("/admission")
	admission.GET("/sessions/:token", notImplemented)
	admission.POST("/sessions/:token/link", authMW, notImplemented)
	admission.GET("/me", authMW, notImplemented)
	admission.POST("/freshman/applications", authMW, notImplemented)
	admission.POST("/freshman/applications/:id/camera-captures", authMW, notImplemented)
	admission.POST("/school-email/request-otp", authMW, notImplemented)
	admission.POST("/school-email/verify-otp", authMW, notImplemented)
	admission.GET("/school-sso/:schoolID/login", notImplemented)
	admission.GET("/school-sso/:schoolID/callback", notImplemented)
}

func (h *Handler) RegisterBotRoutes(api *gin.RouterGroup) {
	bot := api.Group("/bot/admission")
	bot.POST("/sessions", notImplemented)
	bot.POST("/join-requests/events", notImplemented)
	bot.GET("/qq-users/:qqID/access", notImplemented)
	bot.GET("/sessions/pending", notImplemented)
	bot.POST("/sessions/:id/events", notImplemented)
	bot.GET("/freshman/applications/pending-forward", notImplemented)
	bot.POST("/freshman/applications/:id/forwarded", notImplemented)
	bot.POST("/freshman/applications/:id/review", notImplemented)
}

func (h *Handler) RegisterAdminRoutes(admin *gin.RouterGroup) {
	admin.GET("/admission/policies", notImplemented)
	admin.PUT("/admission/policies/:id", notImplemented)
	admin.GET("/admission/sessions", notImplemented)
	admin.GET("/freshman-verifications", notImplemented)
	admin.GET("/freshman-verifications/:id", notImplemented)
	admin.PUT("/freshman-verifications/:id", notImplemented)
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
