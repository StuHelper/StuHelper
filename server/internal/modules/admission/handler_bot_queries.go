package admission

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
	"github.com/StuHelper/StuHelper/server/internal/pkg/sse"
)

const maxBotPendingActionFilterLength = 64
const maxBotSessionSubjectGuildLength = 128

type botJoinRequestEventHTTPRequest struct {
	Platform  string         `json:"platform" binding:"required"`
	GuildID   string         `json:"guildID" binding:"required"`
	QQID      string         `json:"qqID" binding:"required"`
	RequestID string         `json:"requestID" binding:"required"`
	Decision  string         `json:"decision"`
	Success   bool           `json:"success"`
	Error     string         `json:"error"`
	RawEvent  map[string]any `json:"rawEvent"`
}

type botJoinRequestDecisionHTTPRequest struct {
	Platform  string         `json:"platform" binding:"required"`
	GuildID   string         `json:"guildID" binding:"required"`
	QQID      string         `json:"qqID" binding:"required"`
	RequestID string         `json:"requestID" binding:"required"`
	RawEvent  map[string]any `json:"rawEvent"`
}

func botSessionSubjectFromQuery(c *gin.Context) (BotSessionSubjectInput, bool) {
	platform := strings.TrimSpace(c.Query("platform"))
	guildID := strings.TrimSpace(c.Query("guildID"))
	qqID := strings.TrimSpace(c.Query("qqID"))
	if platform == "" || guildID == "" || qqID == "" {
		response.BadRequest(c, "admission session query requires platform, guildID and qqID")
		return BotSessionSubjectInput{}, false
	}
	if len(platform) > maxBotPendingActionFilterLength ||
		len(guildID) > maxBotSessionSubjectGuildLength ||
		len(qqID) > maxBotPendingActionFilterLength {
		response.BadRequest(c, "admission session query too long")
		return BotSessionSubjectInput{}, false
	}
	return BotSessionSubjectInput{Platform: platform, GuildID: guildID, QQID: qqID}, true
}

func (h *Handler) handleRecordBotJoinRequestEvent(c *gin.Context) {
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

func (h *Handler) handleResolveBotJoinRequestDecision(c *gin.Context) {
	var req botJoinRequestDecisionHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	decision, err := h.service.ResolveJoinRequestDecision(c.Request.Context(), botJoinRequestDecisionInput(req))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, decision)
}

func (h *Handler) handleListBotAdmissionPolicyTargets(c *gin.Context) {
	items, err := h.service.ListAdmissionPolicyTargets(c.Request.Context())
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) handleListBotPendingActions(c *gin.Context) {
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

func (h *Handler) handleStreamBotAdmissionActions(c *gin.Context) {
	filter, ok := botPendingActionFilter(c)
	if !ok {
		return
	}
	if err := sse.DisableWriteTimeout(c.Writer); err != nil {
		response.InternalError(c, "failed to initialize admission action stream")
		return
	}
	headers := c.Writer.Header()
	headers.Set("Content-Type", "text/event-stream")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("X-Accel-Buffering", "no")
	if err := sse.WriteComment(c.Writer, "connected"); err != nil {
		return
	}
	c.Writer.Flush()

	if !h.writeQueuedAdmissionActions(c, filter) {
		return
	}
	if h.writeStreamShutdownEvent(c) {
		return
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.streamStop:
			h.writeStreamShutdownEvent(c)
			return
		case <-keepalive.C:
			if h.writeStreamShutdownEvent(c) {
				return
			}
			c.SSEvent("keepalive", time.Now().UTC().Format(time.RFC3339))
			c.Writer.Flush()
		case <-ticker.C:
			if h.writeStreamShutdownEvent(c) {
				return
			}
			if !h.writeQueuedAdmissionActions(c, filter) {
				return
			}
		}
	}
}

func (h *Handler) writeStreamShutdownEvent(c *gin.Context) bool {
	select {
	case <-h.streamStop:
		c.SSEvent("end", "shutdown")
		c.Writer.Flush()
		return true
	default:
		return false
	}
}

func (h *Handler) handleClaimBotAdmissionActions(c *gin.Context) {
	filter, ok := botPendingActionFilter(c)
	if !ok {
		return
	}
	actions, err := h.service.ClaimQueuedAdmissionActions(c.Request.Context(), filter)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, actions)
}

func (h *Handler) writeQueuedAdmissionActions(c *gin.Context, filter AdmissionPendingActionFilter) bool {
	actions, err := h.service.ClaimQueuedAdmissionActions(c.Request.Context(), filter)
	if err != nil {
		c.SSEvent("error", "admission queued action unavailable")
		c.Writer.Flush()
		return false
	}
	for i := range actions {
		c.SSEvent("action", actions[i])
		c.Writer.Flush()
	}
	return true
}

func botPendingActionFilter(c *gin.Context) (AdmissionPendingActionFilter, bool) {
	platform := strings.TrimSpace(c.Query("platform"))
	botSelfID := strings.TrimSpace(c.Query("botSelfID"))
	if platform == "" || botSelfID == "" {
		response.BadRequest(c, "admission pending action filter requires platform and botSelfID")
		return AdmissionPendingActionFilter{}, false
	}
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

func botJoinRequestEventInput(req botJoinRequestEventHTTPRequest) AdmissionJoinRequestEventInput {
	return AdmissionJoinRequestEventInput{
		Platform:  req.Platform,
		GuildID:   req.GuildID,
		QQID:      req.QQID,
		RequestID: req.RequestID,
		Decision:  AdmissionJoinRequestDecisionAction(req.Decision),
		Success:   req.Success,
		Error:     req.Error,
		RawEvent:  req.RawEvent,
	}
}

func botJoinRequestDecisionInput(req botJoinRequestDecisionHTTPRequest) AdmissionJoinRequestDecisionInput {
	return AdmissionJoinRequestDecisionInput(req)
}
