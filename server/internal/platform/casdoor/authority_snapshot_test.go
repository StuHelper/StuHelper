package casdoor

import (
	"testing"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAuthoritySnapshotAPI struct {
	users []*casdoorsdk.User
	roles []*casdoorsdk.Role
}

func (f fakeAuthoritySnapshotAPI) GetUsers() ([]*casdoorsdk.User, error) {
	return f.users, nil
}

func (f fakeAuthoritySnapshotAPI) GetRoles() ([]*casdoorsdk.Role, error) {
	return f.roles, nil
}

func TestAuthoritySnapshotCapturesActiveAdminsAndLegacyRoleMembers(t *testing.T) {
	client, err := newAuthoritySnapshotClient(
		authorityCutoverTestCredential(),
		fakeAuthoritySnapshotAPI{
			users: []*casdoorsdk.User{
				{Id: "subject-1", Owner: "stuhelper", Name: "admin", IsAdmin: true},
				{Id: "subject-2", Owner: "stuhelper", Name: "disabled", IsAdmin: true, IsForbidden: true},
				{Id: "foreign", Owner: "other", Name: "foreign", IsAdmin: true},
			},
			roles: []*casdoorsdk.Role{
				{Owner: "stuhelper", Name: "school_admin", IsEnabled: true, Users: []string{"stuhelper/admin"}},
			},
		},
	)
	require.NoError(t, err)

	snapshot, err := client.Snapshot(t.Context(), []string{"school_admin", "section_admin"})

	require.NoError(t, err)
	require.Len(t, snapshot.Users, 2)
	assert.True(t, snapshot.Users[0].OrganizationAdmin)
	assert.True(t, snapshot.Users[1].ForbiddenOrDeleted)
	assert.Equal(t, []string{"stuhelper/admin"}, snapshot.RoleMembers["school_admin"])
	assert.Empty(t, snapshot.RoleMembers["section_admin"])
}

func TestAuthoritySnapshotRejectsNestedLegacyRoleMembership(t *testing.T) {
	client, err := newAuthoritySnapshotClient(
		authorityCutoverTestCredential(),
		fakeAuthoritySnapshotAPI{
			roles: []*casdoorsdk.Role{
				{Owner: "stuhelper", Name: "school_admin", IsEnabled: true, Groups: []string{"stuhelper/operators"}},
			},
		},
	)
	require.NoError(t, err)

	_, err = client.Snapshot(t.Context(), []string{"school_admin"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "flatten membership before cutover")
}

func authorityCutoverTestCredential() Credential {
	credential := validCredential()
	credential.Purpose = PurposeAuthorityCutover
	credential.Application = "casdoor-admin-authority-cutover"
	return credential
}
