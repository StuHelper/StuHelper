package admission

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBotMemberBlacklistCreateInputUsesTopLevelCreatedFrom(t *testing.T) {
	req := memberBlacklistCreateHTTPRequest{
		Platform:    "qq",
		SubjectType: BlacklistSubjectQQUser,
		SubjectID:   "10001",
		ScopeType:   BlacklistScopeGuild,
		GuildID:     stringPtr("guild-1"),
		Source:      BlacklistSourceManualAdmin,
		ReasonCode:  BlacklistReasonManualBlacklist,
		ReasonText:  "manual test",
		CreatedFrom: BlacklistCreatedFromKoishiConsole,
		Metadata: map[string]any{
			"createdFrom":  string(BlacklistCreatedFromQQCommand),
			"operatorQQID": "90001",
		},
	}

	input := botMemberBlacklistCreateInput(req)

	assert.Equal(t, BlacklistCreatedFromKoishiConsole, input.CreatedFrom)
	assert.Equal(t, BlacklistActorServiceAccount, input.CreatedByType)
	assert.Equal(t, "koishi-runtime", input.CreatedByID)
}

func TestBotMemberBlacklistReleaseInputUsesServiceAccountWithoutOperator(t *testing.T) {
	input := botMemberBlacklistReleaseInput("entry-1", memberBlacklistReleaseHTTPRequest{
		ReleaseReasonCode: BlacklistReleaseManualPardon,
	})

	assert.Equal(t, BlacklistActorServiceAccount, input.ReleasedByType)
	assert.Equal(t, "koishi-runtime", input.ReleasedByID)
}

func TestBotMemberBlacklistReleaseInputUsesQQOperatorWhenProvided(t *testing.T) {
	input := botMemberBlacklistReleaseBySubjectInput(memberBlacklistReleaseBySubjectHTTPRequest{
		Platform: "qq", SubjectType: BlacklistSubjectQQUser, SubjectID: "10001",
		ScopeType: BlacklistScopeGuild, GuildID: stringPtr("guild-1"),
		ReleaseReasonCode: BlacklistReleaseOnly, OperatorQQID: " 90001 ",
	})

	assert.Equal(t, BlacklistActorQQOperator, input.ReleasedByType)
	assert.Equal(t, "90001", input.ReleasedByID)
}

func TestBotMemberBlacklistListFilterRequiresPlatform(t *testing.T) {
	c := newMemberBlacklistQueryContext("/api/v1/bot/member-blacklist?status=active")

	_, err := botMemberBlacklistListFilterFromGin(c)

	require.ErrorIs(t, err, ErrMemberBlacklistInvalidInput)
}

func TestBotMemberBlacklistListFilterAcceptsPlatform(t *testing.T) {
	c := newMemberBlacklistQueryContext("/api/v1/bot/member-blacklist?platform=qq&status=active")

	filter, err := botMemberBlacklistListFilterFromGin(c)

	require.NoError(t, err)
	assert.Equal(t, "qq", filter.Platform)
}

func newMemberBlacklistQueryContext(rawURL string) *gin.Context {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, rawURL, nil)
	return c
}
