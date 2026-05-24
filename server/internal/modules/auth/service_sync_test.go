package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingUserSyncRepo struct {
	upsertInput    UserSyncInput
	existsExternal string
}

func (r *recordingUserSyncRepo) UpsertUser(_ context.Context, input UserSyncInput) error {
	r.upsertInput = input
	return nil
}

func (r *recordingUserSyncRepo) ExistsByCasdoorSubject(_ context.Context, casdoorSubject string) (bool, error) {
	r.existsExternal = casdoorSubject
	return casdoorSubject == "exists", nil
}

func TestAuthService_UserSyncDelegation(t *testing.T) {
	repo := &recordingUserSyncRepo{}
	svc, _ := newAuthServiceForTest(t)
	svc.userSyncRepo = repo

	ctx := context.Background()
	avatar := "https://cdn.example.com/a.png"
	input := UserSyncInput{CasdoorSubject: "oidc-1", Username: "tester", Email: "tester@example.com", AvatarURL: &avatar}
	require.NoError(t, svc.SyncOIDCUser(ctx, input))
	assert.Equal(t, input, repo.upsertInput)

	exists, err := svc.UserExistsByCasdoorSubject(ctx, "exists")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, "exists", repo.existsExternal)
}
