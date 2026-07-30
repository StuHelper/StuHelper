package user

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
)

type fakeRoleFGAClient struct {
	tupleExists bool
	existsErr   error
	writeErr    error
	deleteErr   error
	existsCalls []fga.Tuple
	writes      [][]fga.Tuple
	deletes     [][]fga.Tuple
}

func (f *fakeRoleFGAClient) WriteMissingTuples(_ context.Context, tuples []fga.Tuple) error {
	f.writes = append(f.writes, append([]fga.Tuple(nil), tuples...))
	return f.writeErr
}

func (f *fakeRoleFGAClient) TupleExists(_ context.Context, tuple fga.Tuple) (bool, error) {
	f.existsCalls = append(f.existsCalls, tuple)
	return f.tupleExists, f.existsErr
}

func (f *fakeRoleFGAClient) DeleteTuplesIgnoringMissing(_ context.Context, tuples []fga.Tuple) error {
	f.deletes = append(f.deletes, append([]fga.Tuple(nil), tuples...))
	return f.deleteErr
}

func TestSyncGlobalRoleRelationsReconcilesAuthoritativeRoles(t *testing.T) {
	expected := fga.Tuple{
		User:     "user:42",
		Relation: "super_admin",
		Object:   "ecosystem:stuhelper",
	}

	t.Run("authoritative grant writes the exact tuple", func(t *testing.T) {
		client := &fakeRoleFGAClient{}
		repo := &UserSyncRepository{roleFGA: client}

		err := repo.syncGlobalRoleRelations(t.Context(), 42, []string{"user", "super_admin"}, true)

		require.NoError(t, err)
		assert.Equal(t, [][]fga.Tuple{{expected}}, client.writes)
		assert.Empty(t, client.existsCalls)
		assert.Empty(t, client.deletes)
	})

	t.Run("authoritative demotion deletes an existing exact tuple", func(t *testing.T) {
		client := &fakeRoleFGAClient{tupleExists: true}
		repo := &UserSyncRepository{roleFGA: client}

		err := repo.syncGlobalRoleRelations(t.Context(), 42, []string{"school_admin"}, true)

		require.NoError(t, err)
		assert.Equal(t, []fga.Tuple{expected}, client.existsCalls)
		assert.Equal(t, [][]fga.Tuple{{expected}}, client.deletes)
		assert.Empty(t, client.writes)
	})

	t.Run("already absent tuple is an idempotent no-op", func(t *testing.T) {
		client := &fakeRoleFGAClient{}
		repo := &UserSyncRepository{roleFGA: client}

		err := repo.syncGlobalRoleRelations(t.Context(), 42, []string{"user"}, true)

		require.NoError(t, err)
		assert.Equal(t, []fga.Tuple{expected}, client.existsCalls)
		assert.Empty(t, client.deletes)
		assert.Empty(t, client.writes)
	})
}

func TestSyncGlobalRoleRelationsIgnoresNonAuthoritativeRoles(t *testing.T) {
	for _, roles := range [][]string{nil, {"super_admin"}, {"school_admin"}} {
		client := &fakeRoleFGAClient{tupleExists: true}
		repo := &UserSyncRepository{roleFGA: client}

		err := repo.syncGlobalRoleRelations(t.Context(), 42, roles, false)

		require.NoError(t, err)
		assert.Empty(t, client.existsCalls)
		assert.Empty(t, client.writes)
		assert.Empty(t, client.deletes)
	}
}

func TestSyncGlobalRoleRelationsFailsClosedOnOpenFGAErrors(t *testing.T) {
	expectedReadErr := errors.New("read failed")
	readClient := &fakeRoleFGAClient{existsErr: expectedReadErr}
	readRepo := &UserSyncRepository{roleFGA: readClient}
	err := readRepo.syncGlobalRoleRelations(t.Context(), 42, []string{"user"}, true)
	require.ErrorIs(t, err, expectedReadErr)
	assert.Empty(t, readClient.deletes)

	expectedDeleteErr := errors.New("delete failed")
	deleteClient := &fakeRoleFGAClient{tupleExists: true, deleteErr: expectedDeleteErr}
	deleteRepo := &UserSyncRepository{roleFGA: deleteClient}
	err = deleteRepo.syncGlobalRoleRelations(t.Context(), 42, []string{"user"}, true)
	require.ErrorIs(t, err, expectedDeleteErr)

	expectedWriteErr := errors.New("write failed")
	writeClient := &fakeRoleFGAClient{writeErr: expectedWriteErr}
	writeRepo := &UserSyncRepository{roleFGA: writeClient}
	err = writeRepo.syncGlobalRoleRelations(t.Context(), 42, []string{"super_admin"}, true)
	require.ErrorIs(t, err, expectedWriteErr)
}

func TestGlobalRoleRevocationAuditEvent(t *testing.T) {
	event := globalRoleRevocationAuditEvent(42)

	assert.Equal(t, "iam.role.revoke", string(event.Type))
	assert.Equal(t, "authorization", event.Category)
	assert.Equal(t, "system", event.ActorType)
	assert.Equal(t, "iam.role", event.ResourceType)
	assert.Equal(t, "user:42", event.ResourceID)
	assert.Equal(t, "revoke", event.Action)
	assert.Equal(t, "success", event.Result)
	assert.Equal(t, "super_admin", event.Details["role"])
	assert.Equal(t, "ecosystem:stuhelper", event.Details["object"])
	assert.Equal(t, "fresh_oidc_roles_claim", event.Details["authority_source"])
}
