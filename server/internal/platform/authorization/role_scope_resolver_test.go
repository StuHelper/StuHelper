package authorization

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
	"github.com/StuHelper/StuHelper/server/internal/pkg/metrics"
)

type fakeScopeReader struct {
	listResponses map[string][]string
	readResponses map[string][]fga.Tuple
	err           error
	mu            sync.Mutex
	listCalls     []scopeListCall
	readCalls     []scopeReadCall
}

type scopeListCall struct {
	user       string
	relation   string
	objectType string
}

type scopeReadCall struct {
	object   string
	relation string
}

func (f *fakeScopeReader) ListObjects(_ context.Context, user, relation, objectType string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listCalls = append(f.listCalls, scopeListCall{user: user, relation: relation, objectType: objectType})
	if f.err != nil {
		return nil, f.err
	}
	return f.listResponses[listScopeKey(user, relation, objectType)], nil
}

func (f *fakeScopeReader) ReadTuples(_ context.Context, object, relation string) ([]fga.Tuple, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.readCalls = append(f.readCalls, scopeReadCall{object: object, relation: relation})
	if f.err != nil {
		return nil, f.err
	}
	return f.readResponses[readScopeKey(object, relation)], nil
}

func TestRoleScopeResolverResolvesSchoolAdminScopes(t *testing.T) {
	reader := newFakeScopeReader()
	reader.listResponses[listScopeKey("user:42", "effective_admin", "school")] = []string{
		"school:4111010002", "school:4111010001", "bad", "school:4111010001",
	}
	resolver, err := NewRoleScopeResolver(reader, func(_ context.Context, subject string) (int64, error) {
		assert.Equal(t, "casdoor-subject-1", subject)
		return 42, nil
	})
	require.NoError(t, err)

	scopes, err := resolver.ResolveRoleScopes(context.Background(), "casdoor-subject-1", []string{"school_admin"})

	require.NoError(t, err)
	assert.Equal(t, map[string][]string{"school_admin": {"4111010001", "4111010002"}}, scopes)
	assert.Equal(t, []scopeListCall{{user: "user:42", relation: "effective_admin", objectType: "school"}}, reader.listCalls)
	assert.Empty(t, reader.readCalls)
}

func TestRoleScopeResolverResolvesSectionRoleScopesToSections(t *testing.T) {
	reader := newFakeScopeReader()
	reader.listResponses[listScopeKey("user:42", "section_moderator", "section")] = []string{
		"section:school_10002_review_moderation",
		"section:school_10001_review_moderation",
		"section:school_10002_review_moderation",
		"school:bad",
	}
	resolver, err := NewRoleScopeResolver(reader, func(context.Context, string) (int64, error) {
		return 42, nil
	})
	require.NoError(t, err)

	scopes, err := resolver.ResolveRoleScopes(context.Background(), "casdoor-subject-1", []string{"section_moderator"})

	require.NoError(t, err)
	assert.Equal(t, map[string][]string{"section_moderator": {"school_10001_review_moderation", "school_10002_review_moderation"}}, scopes)
	assert.Equal(t, []scopeListCall{{user: "user:42", relation: "section_moderator", objectType: "section"}}, reader.listCalls)
	assert.Empty(t, reader.readCalls)
}

func TestRoleScopeResolverIgnoresUnsupportedSectionScope(t *testing.T) {
	reader := newFakeScopeReader()
	reader.listResponses[listScopeKey("user:42", "section_admin", "section")] = []string{"section:orphan"}
	resolver, err := NewRoleScopeResolver(reader, func(context.Context, string) (int64, error) {
		return 42, nil
	})
	require.NoError(t, err)
	before := testutil.ToFloat64(metrics.IAMInvalidRoleScopeTotal)

	scopes, err := resolver.ResolveRoleScopes(context.Background(), "casdoor-subject-1", []string{"section_admin"})

	require.NoError(t, err)
	assert.Nil(t, scopes)
	assert.Equal(t, before+1, testutil.ToFloat64(metrics.IAMInvalidRoleScopeTotal))
}

func TestRoleScopeResolverPreservesValidSectionsAlongsideInvalidScope(t *testing.T) {
	reader := newFakeScopeReader()
	reader.listResponses[listScopeKey("user:42", "section_moderator", "section")] = []string{
		"section:orphan",
		"section:school_10001_review_moderation",
		"section:school_10002_review_moderation",
	}
	resolver, err := NewRoleScopeResolver(reader, func(context.Context, string) (int64, error) {
		return 42, nil
	})
	require.NoError(t, err)
	before := testutil.ToFloat64(metrics.IAMInvalidRoleScopeTotal)

	scopes, err := resolver.ResolveRoleScopes(context.Background(), "casdoor-subject-1", []string{"section_moderator"})

	require.NoError(t, err)
	assert.Equal(t, map[string][]string{
		"section_moderator": {"school_10001_review_moderation", "school_10002_review_moderation"},
	}, scopes)
	assert.Equal(t, before+1, testutil.ToFloat64(metrics.IAMInvalidRoleScopeTotal))
}

func TestRoleScopeResolverSkipsRolesWithoutSchoolScope(t *testing.T) {
	reader := newFakeScopeReader()
	resolver, err := NewRoleScopeResolver(reader, func(context.Context, string) (int64, error) {
		t.Fatal("internal user resolver should not be called")
		return 0, nil
	})
	require.NoError(t, err)

	scopes, err := resolver.ResolveRoleScopes(context.Background(), "casdoor-subject-1", []string{"user"})

	require.NoError(t, err)
	assert.Nil(t, scopes)
	assert.Empty(t, reader.listCalls)
	assert.Empty(t, reader.readCalls)
}

func TestRoleScopeResolverPropagatesDependencies(t *testing.T) {
	for _, role := range []string{"school_admin", "section_admin"} {
		t.Run(role, func(t *testing.T) {
			expectedErr := errors.New("openfga unavailable")
			reader := newFakeScopeReader()
			reader.err = expectedErr
			resolver, err := NewRoleScopeResolver(reader, func(context.Context, string) (int64, error) {
				return 42, nil
			})
			require.NoError(t, err)

			_, err = resolver.ResolveRoleScopes(context.Background(), "casdoor-subject-1", []string{role})

			require.ErrorIs(t, err, expectedErr)
		})
	}
}

func TestNewRoleScopeResolverRequiresDependencies(t *testing.T) {
	resolver, err := NewRoleScopeResolver(nil, func(context.Context, string) (int64, error) { return 1, nil })
	require.Error(t, err)
	assert.Nil(t, resolver)

	resolver, err = NewRoleScopeResolver(newFakeScopeReader(), nil)
	require.Error(t, err)
	assert.Nil(t, resolver)
}

func newFakeScopeReader() *fakeScopeReader {
	return &fakeScopeReader{
		listResponses: map[string][]string{},
		readResponses: map[string][]fga.Tuple{},
	}
}

func listScopeKey(user, relation, objectType string) string {
	return user + "|" + relation + "|" + objectType
}

func readScopeKey(object, relation string) string {
	return object + "|" + relation
}
