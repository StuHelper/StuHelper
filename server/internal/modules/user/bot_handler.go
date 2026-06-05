package user

import (
	"context"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/botcredential"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

const bearerPrefix = "Bearer "

type BotCredentialVerifier interface {
	Verify(ctx context.Context, rawToken, audience, scope string) error
}

// BotHandler 提供机器人内部调用的用户绑定接口。
type BotHandler struct {
	service            *Service
	credentialVerifier BotCredentialVerifier
}

// NewBotHandler 创建机器人内部接口处理器。
func NewBotHandler(service *Service, credentialVerifier BotCredentialVerifier) *BotHandler {
	return &BotHandler{
		service:            service,
		credentialVerifier: credentialVerifier,
	}
}

// RegisterRoutes 注册机器人调用接口。
func (h *BotHandler) RegisterRoutes(rg *gin.RouterGroup) {
	bot := rg.Group("/bot")
	bot.POST(
		"/qq-binding/consume",
		h.requireServiceCredential(botcredential.ScopeBotQQBindingConsume),
		h.handleConsumeQQBindingCode,
	)
	bot.GET(
		"/qq-users/:qqID/verification",
		h.requireServiceCredential(botcredential.ScopeBotQQVerificationRead),
		h.handleGetQQVerificationState,
	)
}

type consumeQQBindingHTTPRequest struct {
	Code string `json:"code" binding:"required,max=32"`
	QQID string `json:"qqID" binding:"required,max=64"`
}

func (h *BotHandler) handleConsumeQQBindingCode(c *gin.Context) {
	var req consumeQQBindingHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	binding, err := h.service.ConsumeQQBindingCode(c.Request.Context(), req.Code, req.QQID)
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
		"binding":           qqBindingToJSON(binding),
		"verificationState": qqVerificationStatusToJSON(status),
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

func (h *BotHandler) requireServiceCredential(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.credentialVerifier == nil {
			response.ServiceUnavailable(c, "bot service token is not configured")
			c.Abort()
			return
		}

		if c.Request == nil || len(c.Request.Header.Values("Authorization")) != 1 {
			response.Unauthorized(c, "unauthorized")
			c.Abort()
			return
		}
		rawToken, ok := parseBearerToken(c.GetHeader("Authorization"))
		if !ok {
			response.Unauthorized(c, "unauthorized")
			c.Abort()
			return
		}

		err := h.credentialVerifier.Verify(c.Request.Context(), rawToken, c.FullPath(), scope)
		if err != nil {
			respondBotCredentialError(c, err)
			return
		}
		c.Next()
	}
}

func parseBearerToken(authHeader string) (string, bool) {
	authHeader = strings.TrimSpace(authHeader)
	if len(authHeader) <= len(bearerPrefix) || !strings.EqualFold(authHeader[:len(bearerPrefix)], bearerPrefix) {
		return "", false
	}
	token := strings.TrimSpace(authHeader[len(bearerPrefix):])
	return token, token != ""
}

func respondBotCredentialError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, botcredential.ErrCredentialNotConfigured):
		response.ServiceUnavailable(c, "bot service token is not configured")
	case errors.Is(err, botcredential.ErrCredentialInvalid):
		response.Unauthorized(c, "unauthorized")
	case errors.Is(err, botcredential.ErrCredentialForbidden):
		response.Forbidden(c, "forbidden")
	case errors.Is(err, botcredential.ErrCredentialStoreUnavailable):
		logger.FromGin(c).Error("bot service credential store unavailable", zap.Error(err))
		response.ServiceUnavailable(c, "bot service credential store unavailable")
	default:
		logger.FromGin(c).Error("failed to verify bot service credential", zap.Error(err))
		response.InternalError(c, "failed to verify bot service credential")
	}
	c.Abort()
}
