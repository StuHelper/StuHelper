package casdoor

import (
	"context"
	"testing"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserLookupClientValidateSubjectOwnerAllowsExpectedOwner(t *testing.T) {
	credential := validCredential()
	credential.Purpose = PurposeUserLookup
	users := &fakeUserAPI{user: &casdoorsdk.User{Owner: "stuhelper", Name: "alice"}}
	client, err := newUserLookupClient(credential, users)
	require.NoError(t, err)

	err = client.ValidateSubjectOwner(context.Background(), "casdoor-subject-1", "stuhelper")

	require.NoError(t, err)
	assert.Equal(t, "casdoor-subject-1", users.gotSubject)
}

func TestUserLookupClientValidateSubjectOwnerRejectsDifferentOwner(t *testing.T) {
	credential := validCredential()
	credential.Purpose = PurposeUserLookup
	client, err := newUserLookupClient(credential, &fakeUserAPI{user: &casdoorsdk.User{Owner: "built-in", Name: "admin"}})
	require.NoError(t, err)

	err = client.ValidateSubjectOwner(context.Background(), "admin-subject", "stuhelper")

	require.ErrorIs(t, err, ErrUserOwnerMismatch)
}

func TestUserLookupClientResolveSubjectReturnsOrganizationAdmin(t *testing.T) {
	credential := validCredential()
	credential.Purpose = PurposeUserLookup
	client, err := newUserLookupClient(credential, &fakeUserAPI{user: &casdoorsdk.User{
		Owner:   "stuhelper",
		Name:    "owner",
		IsAdmin: true,
	}})
	require.NoError(t, err)

	identity, err := client.ResolveSubject(context.Background(), "casdoor-subject-1", "stuhelper")

	require.NoError(t, err)
	assert.Equal(t, "stuhelper", identity.Organization)
	assert.True(t, identity.OrganizationAdmin)
}

func TestUserLookupClientResolveSubjectDoesNotTrustInactiveAdmin(t *testing.T) {
	for _, tc := range []struct {
		name string
		user *casdoorsdk.User
	}{
		{
			name: "forbidden",
			user: &casdoorsdk.User{
				Owner:       "stuhelper",
				Name:        "forbidden-owner",
				IsAdmin:     true,
				IsForbidden: true,
			},
		},
		{
			name: "deleted",
			user: &casdoorsdk.User{
				Owner:     "stuhelper",
				Name:      "deleted-owner",
				IsAdmin:   true,
				IsDeleted: true,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			credential := validCredential()
			credential.Purpose = PurposeUserLookup
			client, err := newUserLookupClient(credential, &fakeUserAPI{user: tc.user})
			require.NoError(t, err)

			identity, err := client.ResolveSubject(context.Background(), "casdoor-subject-1", "stuhelper")

			require.NoError(t, err)
			assert.False(t, identity.OrganizationAdmin)
		})
	}
}

func TestUserLookupClientResolveSubjectClassifiesSDKFailureAsUnavailable(t *testing.T) {
	credential := validCredential()
	credential.Purpose = PurposeUserLookup
	client, err := newUserLookupClient(credential, &fakeUserAPI{err: errFakeUserAPIUnavailable})
	require.NoError(t, err)

	_, err = client.ResolveSubject(context.Background(), "casdoor-subject-1", "stuhelper")

	require.ErrorIs(t, err, ErrUserLookupUnavailable)
	assert.Contains(t, err.Error(), errFakeUserAPIUnavailable.Error())
}
