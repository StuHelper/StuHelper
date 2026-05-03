package admission

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/platform/serviceaccount"
)

const admissionBearerPrefix = "Bearer "

type botSessionCreateHTTPRequest struct {
	Platform   string  `json:"platform" binding:"required,max=32"`
	GuildID    string  `json:"guildID" binding:"required,max=128"`
	ChannelID  string  `json:"channelID" binding:"required,max=128"`
	QQID       string  `json:"qqID" binding:"required,max=64"`
	QQNickname *string `json:"qqNickname"`
	BotSelfID  string  `json:"botSelfID"`
}

type botAdmissionEventHTTPRequest struct {
	Action    BotAction `json:"action" binding:"required"`
	Success   bool      `json:"success"`
	MessageID string    `json:"messageID"`
	Error     string    `json:"error"`
}

type botFreshmanReviewHTTPRequest struct {
	Action        FreshmanReviewAction `json:"action" binding:"required"`
	Reason        *string              `json:"reason"`
	ExpiresInDays *int                 `json:"expiresInDays"`
	OperatorQQID  string               `json:"operatorQQID" binding:"required"`
	GuildID       string               `json:"guildID" binding:"required"`
	ChannelID     *string              `json:"channelID"`
	RawCommand    string               `json:"rawCommand" binding:"required"`
}

func (h *Handler) handleCreateBotSession(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	var req botSessionCreateHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	created, err := h.service.CreateBotSession(c.Request.Context(), botSessionCreateInput(req))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Created(c, created)
}

func (h *Handler) handleRecordBotEvent(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	var req botAdmissionEventHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	if err := h.service.RecordBotEvent(c.Request.Context(), c.Param("id"), botEventInput(req)); err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "admission event recorded"})
}

func (h *Handler) handleBotReviewFreshmanApplication(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	var req botFreshmanReviewHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	app, err := h.service.ReviewFreshmanApplicationFromBot(
		c.Request.Context(),
		botFreshmanReviewInput(c.Param("id"), req),
	)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, app)
}

func (h *Handler) requireBotCredential(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.botCredentialVerifier == nil {
			response.ServiceUnavailable(c, "bot service token is not configured")
			c.Abort()
			return
		}
		rawToken, ok := parseAdmissionBearerToken(c.GetHeader("Authorization"))
		if !ok {
			response.Unauthorized(c, "unauthorized")
			c.Abort()
			return
		}
		if err := h.botCredentialVerifier.Verify(c.Request.Context(), rawToken, c.FullPath(), scope); err != nil {
			respondAdmissionBotCredentialError(c, err)
			return
		}
		c.Next()
	}
}

func botSessionCreateInput(req botSessionCreateHTTPRequest) BotSessionCreateInput {
	return BotSessionCreateInput{
		Platform:   req.Platform,
		GuildID:    req.GuildID,
		ChannelID:  req.ChannelID,
		QQID:       req.QQID,
		QQNickname: req.QQNickname,
		BotSelfID:  req.BotSelfID,
	}
}

func botEventInput(req botAdmissionEventHTTPRequest) BotEventInput {
	return BotEventInput{
		Action:    req.Action,
		Success:   req.Success,
		MessageID: req.MessageID,
		Error:     req.Error,
	}
}

func botFreshmanReviewInput(applicationID string, req botFreshmanReviewHTTPRequest) BotFreshmanReviewInput {
	return BotFreshmanReviewInput{
		ApplicationID: applicationID,
		Action:        req.Action,
		Reason:        req.Reason,
		ExpiresInDays: req.ExpiresInDays,
		OperatorQQID:  req.OperatorQQID,
		GuildID:       req.GuildID,
		ChannelID:     req.ChannelID,
		RawCommand:    req.RawCommand,
	}
}

func parseAdmissionBearerToken(authHeader string) (string, bool) {
	authHeader = strings.TrimSpace(authHeader)
	if len(authHeader) <= len(admissionBearerPrefix) {
		return "", false
	}
	if !strings.EqualFold(authHeader[:len(admissionBearerPrefix)], admissionBearerPrefix) {
		return "", false
	}
	token := strings.TrimSpace(authHeader[len(admissionBearerPrefix):])
	return token, token != ""
}

func respondAdmissionBotCredentialError(c *gin.Context, err error) {
	switch {
	case err == nil:
		return
	case errors.Is(err, serviceaccount.ErrCredentialNotConfigured):
		response.ServiceUnavailable(c, "bot service token is not configured")
	case errors.Is(err, serviceaccount.ErrCredentialInvalid):
		response.Unauthorized(c, "unauthorized")
	case errors.Is(err, serviceaccount.ErrCredentialForbidden):
		response.Forbidden(c, "forbidden")
	default:
		response.InternalError(c, "failed to verify bot service credential")
	}
	c.Abort()
}
