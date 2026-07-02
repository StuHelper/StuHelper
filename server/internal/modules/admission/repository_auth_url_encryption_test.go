package admission

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

// F062 回归：auth_url 必须以密文持久化，DB 行泄露不得暴露可用 join token。
func TestAuthURLPersistedEncryptedAtRest(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)

	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform:  "qq",
		BotSelfID: "514",
		GuildID:   "guild-1",
		ChannelID: "channel-1",
		QQID:      "10001",
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.AuthURL)
	require.NotEmpty(t, created.Token)

	// 直接读裸列：必须是密文，既不等于明文 URL，也不得包含明文 token。
	var raw []byte
	err = fixture.Pool.QueryRow(context.Background(), `
		SELECT auth_url FROM group_admission_sessions WHERE id = $1
	`, created.Session.ID).Scan(&raw)
	require.NoError(t, err)
	require.NotEmpty(t, raw, "auth_url 不应为空")
	assert.NotEqual(t, created.AuthURL, string(raw), "auth_url 必须加密存储而非明文")
	assert.False(t, strings.Contains(string(raw), created.Token), "密文不得包含明文 token")
	assert.False(t, strings.Contains(string(raw), "/verify/"), "密文不得包含可识别的 join URL 路径")

	// 经仓库读回必须解密还原为原始 URL（bot 派发链路依赖此明文）。
	reread, err := svc.repo.GetSessionByID(context.Background(), created.Session.ID)
	require.NoError(t, err)
	assert.Equal(t, created.AuthURL, reread.AuthURL, "读回应解密还原原始 auth_url")
}
