package admission

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

const maxBotPendingActionFilterLength = 64

type botJoinRequestEventHTTPRequest struct {
	Platform  string         `json:"platform" binding:"required"`
	GuildID   string         `json:"guildID" binding:"required"`
	QQID      string         `json:"qqID" binding:"required"`
	RequestID string         `json:"requestID" binding:"required"`
	Success   bool           `json:"success"`
	Error     string         `json:"error"`
	RawEvent  map[string]any `json:"rawEvent"`
}

type botFreshmanCommandHTTPRequest struct {
	OperatorQQID string  `json:"operatorQQID" binding:"required"`
	GuildID      string  `json:"guildID" binding:"required"`
	ChannelID    *string `json:"channelID"`
	RawCommand   string  `json:"rawCommand" binding:"required"`
}

func (h *Handler) handleRecordBotJoinRequestEvent(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	var req botJoinRequestEventHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	err := h.service.RecordJoinRequestEvent(c.Request.Context(), botJoinRequestEventInput(req))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "admission join request event recorded"})
}

func (h *Handler) handleGetBotAdmissionQQAccess(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	access, err := h.service.GetAdmissionQQAccess(c.Request.Context(), c.Param("qqID"))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, access)
}

func (h *Handler) handleListBotPendingActions(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	filter, ok := botPendingActionFilter(c)
	if !ok {
		return
	}
	actions, err := h.service.ListPendingAdmissionActions(c.Request.Context(), filter)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, actions)
}

func botPendingActionFilter(c *gin.Context) (AdmissionPendingActionFilter, bool) {
	platform := strings.TrimSpace(c.Query("platform"))
	botSelfID := strings.TrimSpace(c.Query("botSelfID"))
	if len(platform) > maxBotPendingActionFilterLength || len(botSelfID) > maxBotPendingActionFilterLength {
		response.BadRequest(c, "admission pending action filter too long")
		return AdmissionPendingActionFilter{}, false
	}
	limit, ok := botPendingActionLimit(c)
	if !ok {
		response.BadRequest(c, "invalid admission pending action limit")
		return AdmissionPendingActionFilter{}, false
	}
	return AdmissionPendingActionFilter{Platform: platform, BotSelfID: botSelfID, Limit: limit}, true
}

func botPendingActionLimit(c *gin.Context) (int, bool) {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return 0, true
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 0, false
	}
	return limit, true
}

func (h *Handler) handleListBotPendingFreshmanForwards(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	items, err := h.service.ListPendingFreshmanForwards(c.Request.Context())
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) handleMarkBotFreshmanApplicationForwarded(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	if err := h.service.MarkFreshmanApplicationForwarded(c.Request.Context(), c.Param("id")); err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "freshman application forwarded"})
}

func (h *Handler) handleBotViewFreshmanApplication(c *gin.Context) {
	command, ok := h.bindBotFreshmanCommand(c)
	if !ok {
		return
	}
	app, err := h.service.ViewFreshmanApplicationFromBot(c.Request.Context(), command)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, app)
}

func (h *Handler) handleBotReleaseAdmissionBlacklist(c *gin.Context) {
	command, ok := h.bindBotFreshmanCommand(c)
	if !ok {
		return
	}
	if err := h.service.ReleaseAdmissionBlacklistFromBot(c.Request.Context(), c.Param("qqID"), command); err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "admission blacklist released"})
}

func (h *Handler) bindBotFreshmanCommand(c *gin.Context) (BotFreshmanCommandInput, bool) {
	if !h.ready(c) {
		return BotFreshmanCommandInput{}, false
	}
	var req botFreshmanCommandHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return BotFreshmanCommandInput{}, false
	}
	return botFreshmanCommandInput(c.Param("id"), req), true
}

func botJoinRequestEventInput(req botJoinRequestEventHTTPRequest) AdmissionJoinRequestEventInput {
	return AdmissionJoinRequestEventInput{
		Platform:  req.Platform,
		GuildID:   req.GuildID,
		QQID:      req.QQID,
		RequestID: req.RequestID,
		Success:   req.Success,
		Error:     req.Error,
		RawEvent:  req.RawEvent,
	}
}

func botFreshmanCommandInput(applicationID string, req botFreshmanCommandHTTPRequest) BotFreshmanCommandInput {
	return BotFreshmanCommandInput{
		ApplicationID: applicationID,
		OperatorQQID:  req.OperatorQQID,
		GuildID:       req.GuildID,
		ChannelID:     req.ChannelID,
		RawCommand:    req.RawCommand,
	}
}
