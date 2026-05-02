package authorization

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeObjectLister struct {
	user       string
	relation   string
	objectType string
	objects    []string
	err        error
	calls      int
}

func (f *fakeObjectLister) ListObjects(_ context.Context, user, relation, objectType string) ([]string, error) {
	f.calls++
	f.user = user
	f.relation = relation
	f.objectType = objectType
	return f.objects, f.err
}

func TestRoleScopeResolverResolvesSchoolAdminScopes(t *testing.T) {
	lister := &fakeObjectLister{objects: []string{"school:1002", "school:1001", "bad", "school:1001"}}
	resolver, err := NewRoleScopeResolver(lister, func(_ context.Context, subject string) (int64, error) {
		assert.Equal(t, "casdoor-subject-1", subject)
		return 42, nil
	})
	require.NoError(t, err)

	scopes, err := resolver.ResolveRoleScopes(context.Background(), "casdoor-subject-1", []string{"school_admin"})

	require.NoError(t, err)
	assert.Equal(t, map[string][]string{"school_admin": {"1001", "1002"}}, scopes)
	assert.Equal(t, "user:42", lister.user)
	assert.Equal(t, "effective_admin", lister.relation)
	assert.Equal(t, "school", lister.objectType)
	assert.Equal(t, 1, lister.calls)
}

func TestRoleScopeResolverSkipsRolesWithoutSchoolScope(t *testing.T) {
	lister := &fakeObjectLister{}
	resolver, err := NewRoleScopeResolver(lister, func(context.Context, string) (int64, error) {
		t.Fatal("internal user resolver should not be called")
		return 0, nil
	})
	require.NoError(t, err)

	scopes, err := resolver.ResolveRoleScopes(context.Background(), "casdoor-subject-1", []string{"user"})

	require.NoError(t, err)
	assert.Nil(t, scopes)
	assert.Equal(t, 0, lister.calls)
}

func TestRoleScopeResolverPropagatesDependencies(t *testing.T) {
	expectedErr := errors.New("openfga unavailable")
	lister := &fakeObjectLister{err: expectedErr}
	resolver, err := NewRoleScopeResolver(lister, func(context.Context, string) (int64, error) {
		return 42, nil
	})
	require.NoError(t, err)

	_, err = resolver.ResolveRoleScopes(context.Background(), "casdoor-subject-1", []string{"school_admin"})

	require.ErrorIs(t, err, expectedErr)
}

func TestNewRoleScopeResolverRequiresDependencies(t *testing.T) {
	resolver, err := NewRoleScopeResolver(nil, func(context.Context, string) (int64, error) { return 1, nil })
	require.Error(t, err)
	assert.Nil(t, resolver)

	resolver, err = NewRoleScopeResolver(&fakeObjectLister{}, nil)
	require.Error(t, err)
	assert.Nil(t, resolver)
}
