package admission

import (
	"time"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

type memberBlacklistCreateHTTPRequest struct {
	Platform    string                     `json:"platform" binding:"required"`
	SubjectType MemberBlacklistSubjectType `json:"subjectType" binding:"required"`
	SubjectID   string                     `json:"subjectID" binding:"required"`
	ScopeType   MemberBlacklistScopeType   `json:"scopeType" binding:"required"`
	GuildID     *string                    `json:"guildID"`
	Source      MemberBlacklistSource      `json:"source" binding:"required"`
	ReasonCode  MemberBlacklistReasonCode  `json:"reasonCode" binding:"required"`
	ReasonText  string                     `json:"reasonText" binding:"required"`
	CreatedFrom MemberBlacklistCreatedFrom `json:"createdFrom"`
	ExpiresAt   *time.Time                 `json:"expiresAt"`
	Metadata    map[string]any             `json:"metadata"`
}

type memberBlacklistReleaseHTTPRequest struct {
	ReleaseReasonCode MemberBlacklistReleaseReasonCode `json:"releaseReasonCode" binding:"required"`
	ReleaseReason     *string                          `json:"releaseReason"`
	OperatorQQID      string                           `json:"operatorQQID"`
}

type memberBlacklistReleaseBySubjectHTTPRequest struct {
	Platform          string                           `json:"platform" binding:"required"`
	SubjectType       MemberBlacklistSubjectType       `json:"subjectType" binding:"required"`
	SubjectID         string                           `json:"subjectID" binding:"required"`
	ScopeType         MemberBlacklistScopeType         `json:"scopeType" binding:"required"`
	GuildID           *string                          `json:"guildID"`
	ReleaseReasonCode MemberBlacklistReleaseReasonCode `json:"releaseReasonCode" binding:"required"`
	ReleaseReason     *string                          `json:"releaseReason"`
	OperatorQQID      string                           `json:"operatorQQID"`
}

func (h *Handler) handleGetBotMemberBlacklistAccess(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	decision, err := h.service.GetMemberBlacklistAccess(c.Request.Context(), memberBlacklistAccessQueryFromGin(c))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, decision)
}

func (h *Handler) handleListBotMemberBlacklist(c *gin.Context) {
	h.handleListMemberBlacklist(c)
}

func (h *Handler) handleListAdminMemberBlacklist(c *gin.Context) {
	h.handleListMemberBlacklist(c)
}

func (h *Handler) handleListMemberBlacklist(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	items, total, err := h.service.ListMemberBlacklist(c.Request.Context(), memberBlacklistListFilterFromGin(c))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, gin.H{"list": items, "total": total})
}

func (h *Handler) handleCreateBotMemberBlacklist(c *gin.Context) {
	req, ok := h.bindMemberBlacklistCreateRequest(c)
	if !ok {
		return
	}
	entry, err := h.service.CreateMemberBlacklistFromBot(c.Request.Context(), botMemberBlacklistCreateInput(req))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Created(c, entry)
}

func (h *Handler) handleCreateAdminMemberBlacklist(c *gin.Context) {
	userID, ok := h.resolveAdminUserID(c, "failed to resolve member blacklist admin")
	if !ok {
		return
	}
	req, ok := h.bindMemberBlacklistCreateRequest(c)
	if !ok {
		return
	}
	input := adminMemberBlacklistCreateInput(req, userID)
	entry, err := h.service.CreateMemberBlacklistFromAdmin(c.Request.Context(), input)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Created(c, entry)
}

func (h *Handler) handleReleaseBotMemberBlacklist(c *gin.Context) {
	var req memberBlacklistReleaseHTTPRequest
	if !h.bindMemberBlacklistJSON(c, &req) {
		return
	}
	entry, err := h.service.ReleaseMemberBlacklistFromBot(c.Request.Context(), botMemberBlacklistReleaseInput(c.Param("id"), req))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, entry)
}

func (h *Handler) handleReleaseAdminMemberBlacklist(c *gin.Context) {
	userID, ok := h.resolveAdminUserID(c, "failed to resolve member blacklist admin")
	if !ok {
		return
	}
	var req memberBlacklistReleaseHTTPRequest
	if !h.bindMemberBlacklistJSON(c, &req) {
		return
	}
	input := adminMemberBlacklistReleaseInput(c.Param("id"), req, userID)
	entry, err := h.service.ReleaseMemberBlacklistFromAdmin(c.Request.Context(), input)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, entry)
}

func (h *Handler) handleReleaseBotMemberBlacklistBySubject(c *gin.Context) {
	var req memberBlacklistReleaseBySubjectHTTPRequest
	if !h.bindMemberBlacklistJSON(c, &req) {
		return
	}
	entry, err := h.service.ReleaseMemberBlacklistBySubjectFromBot(c.Request.Context(), botMemberBlacklistReleaseBySubjectInput(req))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, entry)
}

func (h *Handler) handleReleaseAdminMemberBlacklistBySubject(c *gin.Context) {
	userID, ok := h.resolveAdminUserID(c, "failed to resolve member blacklist admin")
	if !ok {
		return
	}
	var req memberBlacklistReleaseBySubjectHTTPRequest
	if !h.bindMemberBlacklistJSON(c, &req) {
		return
	}
	input := adminMemberBlacklistReleaseBySubjectInput(req, userID)
	entry, err := h.service.ReleaseMemberBlacklistBySubjectFromAdmin(c.Request.Context(), input)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, entry)
}

func (h *Handler) resolveAdminUserID(c *gin.Context, failureMessage string) (int64, bool) {
	return middleware.ResolveRequiredInternalUserID(c, h.internalUserIDResolver, failureMessage)
}

func (h *Handler) bindMemberBlacklistCreateRequest(c *gin.Context) (memberBlacklistCreateHTTPRequest, bool) {
	var req memberBlacklistCreateHTTPRequest
	ok := h.bindMemberBlacklistJSON(c, &req)
	return req, ok
}

func (h *Handler) bindMemberBlacklistJSON(c *gin.Context, req any) bool {
	if !h.ready(c) {
		return false
	}
	if err := c.ShouldBindJSON(req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return false
	}
	return true
}
