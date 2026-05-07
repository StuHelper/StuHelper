package admission

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/platform/serviceaccount"
)

func memberBlacklistAccessQueryFromGin(c *gin.Context) MemberBlacklistAccessQuery {
	return MemberBlacklistAccessQuery{
		Platform: c.Query("platform"), SubjectType: MemberBlacklistSubjectType(c.Query("subjectType")),
		SubjectID: c.Query("subjectID"), GuildID: queryStringPtr(c, "guildID"),
	}
}

func memberBlacklistListFilterFromGin(c *gin.Context) MemberBlacklistListFilter {
	page, pageSize := httputil.ParsePage(c)
	return MemberBlacklistListFilter{
		Platform: c.Query("platform"), SubjectType: MemberBlacklistSubjectType(c.Query("subjectType")),
		SubjectID: c.Query("subjectID"), ScopeType: MemberBlacklistScopeType(c.Query("scopeType")),
		Source: MemberBlacklistSource(c.Query("source")), GuildID: c.Query("guildID"),
		CreatedByID: c.Query("createdByID"),
		Status:      MemberBlacklistStatus(c.Query("status")), PageSize: pageSize, Offset: httputil.SafeOffset(page, pageSize),
	}
}

func adminMemberBlacklistCreateInput(req memberBlacklistCreateHTTPRequest, userID int64) MemberBlacklistCreateInput {
	input := memberBlacklistCreateInput(req)
	input.CreatedByType = BlacklistActorAdminUser
	input.CreatedByID = strconv.FormatInt(userID, 10)
	input.CreatedFrom = BlacklistCreatedFromAdminConsole
	return input
}

func botMemberBlacklistCreateInput(req memberBlacklistCreateHTTPRequest) MemberBlacklistCreateInput {
	input := memberBlacklistCreateInput(req)
	input.CreatedFrom = botMemberBlacklistCreatedFrom(req)
	input.CreatedByType = botMemberBlacklistActorType(req)
	input.CreatedByID = botMemberBlacklistActorID(req)
	return input
}

func memberBlacklistCreateInput(req memberBlacklistCreateHTTPRequest) MemberBlacklistCreateInput {
	return MemberBlacklistCreateInput{
		Platform: req.Platform, SubjectType: req.SubjectType, SubjectID: req.SubjectID,
		ScopeType: req.ScopeType, GuildID: req.GuildID, Source: req.Source,
		ReasonCode: req.ReasonCode, ReasonText: req.ReasonText, ExpiresAt: req.ExpiresAt, Metadata: req.Metadata,
	}
}

func adminMemberBlacklistReleaseInput(
	id string,
	req memberBlacklistReleaseHTTPRequest,
	userID int64,
) MemberBlacklistReleaseInput {
	input := botMemberBlacklistReleaseInput(id, req)
	input.ReleasedByType = BlacklistActorAdminUser
	input.ReleasedByID = strconv.FormatInt(userID, 10)
	return input
}

func botMemberBlacklistReleaseInput(id string, req memberBlacklistReleaseHTTPRequest) MemberBlacklistReleaseInput {
	return MemberBlacklistReleaseInput{
		ID: id, ReleasedByType: BlacklistActorQQOperator, ReleasedByID: req.OperatorQQID,
		ReleaseReasonCode: req.ReleaseReasonCode, ReleaseReason: req.ReleaseReason,
	}
}

func adminMemberBlacklistReleaseBySubjectInput(
	req memberBlacklistReleaseBySubjectHTTPRequest,
	userID int64,
) MemberBlacklistReleaseBySubjectInput {
	input := botMemberBlacklistReleaseBySubjectInput(req)
	input.ReleasedByType = BlacklistActorAdminUser
	input.ReleasedByID = strconv.FormatInt(userID, 10)
	return input
}

func botMemberBlacklistReleaseBySubjectInput(
	req memberBlacklistReleaseBySubjectHTTPRequest,
) MemberBlacklistReleaseBySubjectInput {
	return MemberBlacklistReleaseBySubjectInput{
		Platform: req.Platform, SubjectType: req.SubjectType, SubjectID: req.SubjectID,
		ScopeType: req.ScopeType, GuildID: req.GuildID, ReleasedByType: BlacklistActorQQOperator,
		ReleasedByID: req.OperatorQQID, ReleaseReasonCode: req.ReleaseReasonCode, ReleaseReason: req.ReleaseReason,
	}
}

func botMemberBlacklistCreatedFrom(req memberBlacklistCreateHTTPRequest) MemberBlacklistCreatedFrom {
	switch req.Source {
	case BlacklistSourceManualAdmin:
		return req.CreatedFrom
	case BlacklistSourceKickBlacklist:
		return BlacklistCreatedFromQQCommand
	case BlacklistSourceModerationAction:
		return BlacklistCreatedFromModerationReview
	default:
		return ""
	}
}

func botMemberBlacklistActorType(req memberBlacklistCreateHTTPRequest) MemberBlacklistActorType {
	if req.Source == BlacklistSourceModerationAction {
		return BlacklistActorServiceAccount
	}
	if req.Source == BlacklistSourceManualAdmin && botMemberBlacklistCreatedFrom(req) == BlacklistCreatedFromKoishiConsole {
		return BlacklistActorServiceAccount
	}
	return BlacklistActorQQOperator
}

func botMemberBlacklistActorID(req memberBlacklistCreateHTTPRequest) string {
	if botMemberBlacklistActorType(req) == BlacklistActorServiceAccount {
		return serviceaccount.KoishiRuntimeCredentialName
	}
	return metadataString(req.Metadata, "operatorQQID")
}

func metadataString(metadata map[string]any, key string) string {
	if value, ok := metadata[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func queryStringPtr(c *gin.Context, key string) *string {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return nil
	}
	return &value
}
