package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/usersync"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestUserSyncRepositoryUpsertsOnlyIdentityProjection(t *testing.T) {
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	repo := NewUserSyncRepository(
		postgres.DB,
		[]byte("identity-projection-test-hmac-key"),
	)
	avatar := "https://cdn.example.com/alice.png"

	require.NoError(t, repo.UpsertUser(ctx, usersync.Input{
		CasdoorSubject: "casdoor-alice",
		Username:       "alice",
		Email:          "alice@example.com",
		AvatarURL:      &avatar,
	}))

	var (
		username  string
		email     *string
		gotAvatar *string
	)
	require.NoError(t, postgres.Pool.QueryRow(ctx, `
		SELECT username, email, avatar_url
		FROM users
		WHERE casdoor_subject = $1
	`, "casdoor-alice").Scan(&username, &email, &gotAvatar))
	assert.Equal(t, "alice", username)
	require.NotNil(t, email)
	assert.Equal(t, "alice@example.com", *email)
	require.NotNil(t, gotAvatar)
	assert.Equal(t, avatar, *gotAvatar)
}
