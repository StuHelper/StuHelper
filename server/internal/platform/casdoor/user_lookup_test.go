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
