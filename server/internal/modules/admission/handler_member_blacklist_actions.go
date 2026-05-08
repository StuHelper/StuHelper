package admission

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

func (h *Handler) handleReleaseBotMemberBlacklist(c *gin.Context) {
	var req memberBlacklistReleaseHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	if err := h.service.ReleaseMemberBlacklist(c.Request.Context(), memberBlacklistReleaseInput(c.Param("id"), req)); err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "member blacklist released"})
}

func (h *Handler) handleReleaseBotMemberBlacklistBySubject(c *gin.Context) {
	var req memberBlacklistReleaseBySubjectHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	if err := h.service.ReleaseMemberBlacklistBySubject(c.Request.Context(), memberBlacklistReleaseBySubjectInput(req)); err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "member blacklist released"})
}

func memberBlacklistAccessQueryFromRequest(c *gin.Context) (MemberBlacklistAccessQuery, bool) {
	query := MemberBlacklistAccessQuery{
		Platform:    strings.TrimSpace(c.Query("platform")),
		SubjectType: MemberBlacklistSubjectType(strings.TrimSpace(c.Query("subjectType"))),
		SubjectID:   strings.TrimSpace(c.Query("subjectID")),
		GuildID:     strings.TrimSpace(c.Query("guildID")),
	}
	if query.SubjectType == "" {
		query.SubjectType = MemberBlacklistSubjectQQUser
	}
	if query.Platform == "" || query.SubjectID == "" || query.GuildID == "" {
		response.BadRequest(c, "platform, subjectID and guildID are required")
		return MemberBlacklistAccessQuery{}, false
	}
	return query, true
}

func memberBlacklistListFilterFromRequest(c *gin.Context) MemberBlacklistListFilter {
	page, pageSize := memberBlacklistPage(c)
	return MemberBlacklistListFilter{
		Platform:    strings.TrimSpace(c.Query("platform")),
		SubjectType: MemberBlacklistSubjectType(strings.TrimSpace(c.Query("subjectType"))),
		SubjectID:   strings.TrimSpace(c.Query("subjectID")),
		ScopeType:   MemberBlacklistScopeType(strings.TrimSpace(c.Query("scopeType"))),
		GuildID:     strings.TrimSpace(c.Query("guildID")),
		PageSize:    pageSize,
		Offset:      (page - 1) * pageSize,
		ActiveOnly:  strings.TrimSpace(c.Query("state")) != "all",
	}
}

func memberBlacklistPage(c *gin.Context) (int, int) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("pageSize"), defaultMemberBlacklistPageSize)
	return page, pageSize
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
