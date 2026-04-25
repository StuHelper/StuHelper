package user

import (
	"crypto/subtle"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// BotHandler 提供机器人内部调用的用户绑定接口。
type BotHandler struct {
	service      *Service
	serviceToken string
}

// NewBotHandler 创建机器人内部接口处理器。
func NewBotHandler(service *Service, serviceToken string) *BotHandler {
	return &BotHandler{
		service:      service,
		serviceToken: strings.TrimSpace(serviceToken),
	}
}

// RegisterRoutes 注册机器人调用接口。
func (h *BotHandler) RegisterRoutes(rg *gin.RouterGroup) {
	bot := rg.Group("/bot")
	bot.Use(h.requireServiceToken())
	bot.POST("/qq-binding/consume", h.handleConsumeQQBindingCode)
	bot.GET("/qq-users/:qqID/verification", h.handleGetQQVerificationState)
}

type consumeQQBindingHTTPRequest struct {
	Code       string  `json:"code" binding:"required,max=32"`
	QQID       string  `json:"qqID" binding:"required,max=64"`
	QQNickname *string `json:"qqNickname"`
}

func (h *BotHandler) handleConsumeQQBindingCode(c *gin.Context) {
	var req consumeQQBindingHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	binding, err := h.service.ConsumeQQBindingCode(c.Request.Context(), req.Code, req.QQID, req.QQNickname)
	if err != nil {
		if respondQQBindingConsumeError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to consume qq binding code", zap.Error(err))
		response.InternalError(c, "failed to consume qq binding code")
		return
	}

	status, err := h.service.GetQQVerificationStateByQQID(c.Request.Context(), req.QQID)
	if err != nil {
		if respondQQVerificationError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to load qq verification state after binding", zap.Error(err))
		response.InternalError(c, "failed to load qq verification state")
		return
	}

	response.Success(c, gin.H{
		"binding":            qqBindingToJSON(binding),
		"verificationState":  qqVerificationStatusToJSON(status),
	})
}

func (h *BotHandler) handleGetQQVerificationState(c *gin.Context) {
	qqID := c.Param("qqID")
	status, err := h.service.GetQQVerificationStateByQQID(c.Request.Context(), qqID)
	if err != nil {
		if respondQQVerificationError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to get qq verification state", zap.Error(err))
		response.InternalError(c, "failed to get qq verification state")
		return
	}

	response.Success(c, qqVerificationStatusToJSON(status))
}

func (h *BotHandler) requireServiceToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.serviceToken == "" {
			response.ServiceUnavailable(c, "bot service token is not configured")
			c.Abort()
			return
		}

		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		expected := "Bearer " + h.serviceToken
		if subtle.ConstantTimeCompare([]byte(authHeader), []byte(expected)) != 1 {
			response.Unauthorized(c, "unauthorized")
			c.Abort()
			return
		}

		c.Next()
	}
}
