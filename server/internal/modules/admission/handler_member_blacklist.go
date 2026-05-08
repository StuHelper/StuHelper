package admission

import (
	"time"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

type memberBlacklistListResponse struct {
	Items []MemberBlacklistEntry `json:"items"`
	Total int                    `json:"total"`
}

type botMemberBlacklistCreateHTTPRequest struct {
	Platform    string                     `json:"platform" binding:"required"`
	SubjectType MemberBlacklistSubjectType `json:"subjectType" binding:"required"`
	SubjectID   string                     `json:"subjectID" binding:"required"`
	ScopeType   MemberBlacklistScopeType   `json:"scopeType" binding:"required"`
	GuildID     string                     `json:"guildID"`
	Source      MemberBlacklistSource      `json:"source" binding:"required"`
	ReasonCode  string                     `json:"reasonCode" binding:"required"`
	ReasonText  string                     `json:"reasonText"`
	CreatedFrom MemberBlacklistCreatedFrom `json:"createdFrom" binding:"required"`
	OperatorID  string                     `json:"operatorID"`
	ExpiresAt   *time.Time                 `json:"expiresAt"`
	Metadata    map[string]any             `json:"metadata"`
}

type memberBlacklistReleaseHTTPRequest struct {
	ReleaseReasonCode string `json:"releaseReasonCode" binding:"required"`
	ReleaseReason     string `json:"releaseReason"`
	OperatorID        string `json:"operatorID"`
}

type memberBlacklistReleaseBySubjectHTTPRequest struct {
	Platform          string                     `json:"platform" binding:"required"`
	SubjectType       MemberBlacklistSubjectType `json:"subjectType" binding:"required"`
	SubjectID         string                     `json:"subjectID" binding:"required"`
	ScopeType         MemberBlacklistScopeType   `json:"scopeType" binding:"required"`
	GuildID           string                     `json:"guildID"`
	ReleaseReasonCode string                     `json:"releaseReasonCode" binding:"required"`
	ReleaseReason     string                     `json:"releaseReason"`
	OperatorID        string                     `json:"operatorID"`
}

func (h *Handler) handleGetBotMemberBlacklistAccess(c *gin.Context) {
	query, ok := memberBlacklistAccessQueryFromRequest(c)
	if !ok {
		return
	}
	decision, err := h.service.GetMemberBlacklistAccess(c.Request.Context(), query)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, decision)
}

func (h *Handler) handleListBotMemberBlacklist(c *gin.Context) {
	items, total, err := h.service.ListMemberBlacklist(c.Request.Context(), memberBlacklistListFilterFromRequest(c))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, memberBlacklistListResponse{Items: items, Total: total})
}

func (h *Handler) handleCreateBotMemberBlacklist(c *gin.Context) {
	var req botMemberBlacklistCreateHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	if !validBotMemberBlacklistCreateRequest(req) {
		response.BadRequest(c, "member blacklist source is not allowed for bot API")
		return
	}
	created, err := h.service.CreateMemberBlacklist(c.Request.Context(), botMemberBlacklistCreateInput(req))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Created(c, created)
}
