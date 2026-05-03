package casdoor

import (
	"context"
	"errors"
	"testing"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRoleSyncClientRequiresDedicatedPurposes(t *testing.T) {
	roleCredential := validCredential()
	userLookupCredential := validCredential()
	roleCredential.Purpose = PurposeAppProvisioning
	userLookupCredential.Purpose = PurposeAppProvisioning

	client, err := NewRoleSyncClient(roleCredential, userLookupCredential)

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "credential purpose")
}

func TestSyncRoleGrantsUserByCasdoorSubject(t *testing.T) {
	roles := &fakeRoleAPI{role: &casdoorsdk.Role{Name: "verified_student", Users: []string{"existing"}}}
	users := &fakeUserAPI{user: &casdoorsdk.User{Name: "alice"}}
	client := newRoleSyncTestClient(t, roles, users)

	err := client.SyncRole(context.Background(), "casdoor-subject-1", "verified_student", true)

	require.NoError(t, err)
	assert.Equal(t, "casdoor-subject-1", users.gotSubject)
	require.NotNil(t, roles.updated)
	assert.Equal(t, []string{"existing", "alice"}, roles.updated.Users)
	assert.Equal(t, []string{"users"}, roles.columns)
}

func TestSyncRoleRevokesUser(t *testing.T) {
	roles := &fakeRoleAPI{role: &casdoorsdk.Role{Name: "verified_student", Users: []string{"alice", "bob"}}}
	users := &fakeUserAPI{user: &casdoorsdk.User{Name: "alice"}}
	client := newRoleSyncTestClient(t, roles, users)

	err := client.SyncRole(context.Background(), "casdoor-subject-1", "verified_student", false)

	require.NoError(t, err)
	require.NotNil(t, roles.updated)
	assert.Equal(t, []string{"bob"}, roles.updated.Users)
}

func TestSyncRoleNoopsWhenRoleAlreadyMatches(t *testing.T) {
	roles := &fakeRoleAPI{role: &casdoorsdk.Role{Name: "verified_student", Users: []string{"alice"}}}
	users := &fakeUserAPI{user: &casdoorsdk.User{Name: "alice"}}
	client := newRoleSyncTestClient(t, roles, users)

	err := client.SyncRole(context.Background(), "casdoor-subject-1", "verified_student", true)

	require.NoError(t, err)
	assert.Nil(t, roles.updated)
}

func TestUserHasRoleChecksRoleMembership(t *testing.T) {
	roles := &fakeRoleAPI{role: &casdoorsdk.Role{Name: "super_admin", Users: []string{"alice"}}}
	users := &fakeUserAPI{user: &casdoorsdk.User{Name: "alice"}}
	client := newRoleSyncTestClient(t, roles, users)

	allowed, err := client.UserHasRole(context.Background(), "casdoor-subject-1", "super_admin")

	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestSyncRoleRejectsMissingCasdoorUser(t *testing.T) {
	roles := &fakeRoleAPI{role: &casdoorsdk.Role{Name: "verified_student"}}
	users := &fakeUserAPI{}
	client := newRoleSyncTestClient(t, roles, users)

	err := client.SyncRole(context.Background(), "casdoor-subject-1", "verified_student", true)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSyncRolePropagatesCasdoorUpdateFailure(t *testing.T) {
	roles := &fakeRoleAPI{role: &casdoorsdk.Role{Name: "verified_student"}, updateErr: errors.New("casdoor down")}
	users := &fakeUserAPI{user: &casdoorsdk.User{Name: "alice"}}
	client := newRoleSyncTestClient(t, roles, users)

	err := client.SyncRole(context.Background(), "casdoor-subject-1", "verified_student", true)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "casdoor down")
}

func TestBuildRoleSyncFuncRequiresConfiguredClient(t *testing.T) {
	called := false
	syncFn := BuildRoleSyncFunc(nil, func(context.Context, int64) (string, error) {
		called = true
		return "casdoor-subject-1", nil
	})

	err := syncFn(context.Background(), 42, "verified_student", true)

	require.ErrorIs(t, err, ErrRoleSyncCredentialNotConfigured)
	assert.False(t, called)
}

func TestBuildRoleSyncFuncDelegatesToCasdoor(t *testing.T) {
	roles := &fakeRoleAPI{role: &casdoorsdk.Role{Name: "verified_student"}}
	users := &fakeUserAPI{user: &casdoorsdk.User{Name: "alice"}}
	client := newRoleSyncTestClient(t, roles, users)
	syncFn := BuildRoleSyncFunc(client, func(_ context.Context, userID int64) (string, error) {
		assert.Equal(t, int64(7), userID)
		return "casdoor-subject-7", nil
	})

	err := syncFn(context.Background(), 7, "verified_student", true)

	require.NoError(t, err)
	assert.Equal(t, "casdoor-subject-7", users.gotSubject)
	require.NotNil(t, roles.updated)
	assert.Equal(t, []string{"alice"}, roles.updated.Users)
}

func TestBuildRoleMembershipFuncDelegatesToCasdoor(t *testing.T) {
	roles := &fakeRoleAPI{role: &casdoorsdk.Role{Name: "super_admin", Users: []string{"alice"}}}
	users := &fakeUserAPI{user: &casdoorsdk.User{Name: "alice"}}
	client := newRoleSyncTestClient(t, roles, users)
	check := BuildRoleMembershipFunc(client, func(_ context.Context, userID int64) (string, error) {
		assert.Equal(t, int64(7), userID)
		return "casdoor-subject-7", nil
	})

	allowed, err := check(context.Background(), 7, "super_admin")

	require.NoError(t, err)
	assert.True(t, allowed)
}

func newRoleSyncTestClient(t *testing.T, roles *fakeRoleAPI, users *fakeUserAPI) *RoleSyncClient {
	t.Helper()
	roleCredential := validCredential()
	userLookupCredential := validCredential()
	roleCredential.Purpose = PurposeRoleSync
	userLookupCredential.Purpose = PurposeUserLookup
	client, err := newRoleSyncClient(roleCredential, userLookupCredential, roles, users)
	require.NoError(t, err)
	return client
}

type fakeRoleAPI struct {
	role      *casdoorsdk.Role
	added     *casdoorsdk.Role
	updated   *casdoorsdk.Role
	columns   []string
	updateErr error
}

func (f *fakeRoleAPI) GetRole(name string) (*casdoorsdk.Role, error) {
	if f.role == nil {
		return nil, nil
	}
	role := *f.role
	role.Users = append([]string(nil), f.role.Users...)
	return &role, nil
}

func (f *fakeRoleAPI) AddRole(role *casdoorsdk.Role) (bool, error) {
	f.added = role
	return true, nil
}

func (f *fakeRoleAPI) UpdateRoleForColumns(role *casdoorsdk.Role, columns []string) (bool, error) {
	f.updated = role
	f.columns = append([]string(nil), columns...)
	return f.updateErr == nil, f.updateErr
}

type fakeUserAPI struct {
	user       *casdoorsdk.User
	gotSubject string
}

func (f *fakeUserAPI) GetUserByUserId(subject string) (*casdoorsdk.User, error) {
	f.gotSubject = subject
	return f.user, nil
}
