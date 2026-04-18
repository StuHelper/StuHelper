package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingUserSyncRepo struct {
	upsertInput    UserSyncInput
	upsertPhone    string
	existsExternal string
	user           *PhoneUser
}

func (r *recordingUserSyncRepo) UpsertUser(_ context.Context, input UserSyncInput) error {
	r.upsertInput = input
	return nil
}

func (r *recordingUserSyncRepo) UpsertByPhone(_ context.Context, phone string) (*PhoneUser, error) {
	r.upsertPhone = phone
	if r.user != nil {
		return r.user, nil
	}
	return &PhoneUser{ExternalID: "phone-user-1", Username: "Phone User"}, nil
}

func (r *recordingUserSyncRepo) ExistsByExternalID(_ context.Context, externalID string) (bool, error) {
	r.existsExternal = externalID
	return externalID == "exists", nil
}

func TestAuthService_UserSyncDelegation(t *testing.T) {
	repo := &recordingUserSyncRepo{user: &PhoneUser{ExternalID: "phone-1", Username: "Phone Tester"}}
	svc, _ := newAuthServiceForTest(t)
	svc.userSyncRepo = repo

	ctx := context.Background()
	avatar := "https://cdn.example.com/a.png"
	input := UserSyncInput{ExternalID: "oidc-1", Username: "tester", Email: "tester@example.com", AvatarURL: &avatar}
	require.NoError(t, svc.SyncOIDCUser(ctx, input))
	assert.Equal(t, input, repo.upsertInput)

	user, err := svc.SyncPhoneUser(ctx, "13800138000")
	require.NoError(t, err)
	assert.Equal(t, "13800138000", repo.upsertPhone)
	require.NotNil(t, user)
	assert.Equal(t, "phone-1", user.ExternalID)

	exists, err := svc.UserExistsByExternalID(ctx, "exists")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, "exists", repo.existsExternal)
}
