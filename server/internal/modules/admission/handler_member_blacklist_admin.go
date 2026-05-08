package admission

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

func (h *Handler) handleListAdminMemberBlacklist(c *gin.Context) {
	h.handleListBotMemberBlacklist(c)
}

func (h *Handler) handleCreateAdminMemberBlacklist(c *gin.Context) {
	userID, ok := h.resolveAdminMemberBlacklistUser(c)
	if !ok {
		return
	}
	var req botMemberBlacklistCreateHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	created, err := h.service.CreateMemberBlacklist(c.Request.Context(), adminMemberBlacklistCreateInput(req, userID))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Created(c, created)
}

func (h *Handler) handleReleaseAdminMemberBlacklist(c *gin.Context) {
	userID, ok := h.resolveAdminMemberBlacklistUser(c)
	if !ok {
		return
	}
	var req memberBlacklistReleaseHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	input := adminMemberBlacklistReleaseInput(c.Param("id"), req, userID)
	if err := h.service.ReleaseMemberBlacklist(c.Request.Context(), input); err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "member blacklist released"})
}

func (h *Handler) handleReleaseAdminMemberBlacklistBySubject(c *gin.Context) {
	userID, ok := h.resolveAdminMemberBlacklistUser(c)
	if !ok {
		return
	}
	var req memberBlacklistReleaseBySubjectHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	input := adminMemberBlacklistReleaseBySubjectInput(req, userID)
	if err := h.service.ReleaseMemberBlacklistBySubject(c.Request.Context(), input); err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "member blacklist released"})
}

func (h *Handler) resolveAdminMemberBlacklistUser(c *gin.Context) (int64, bool) {
	return middleware.ResolveRequiredInternalUserID(
		c,
		h.internalUserIDResolver,
		"failed to resolve member blacklist operator",
	)
}

func adminMemberBlacklistActorID(userID int64) string {
	return strconv.FormatInt(userID, 10)
}
