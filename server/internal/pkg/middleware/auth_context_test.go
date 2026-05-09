package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetClaimsToContextPropagatesMFAProofOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/me", nil)
	proofAt := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)

	setClaimsToContext(c, &authResult{
		userID:      "casdoor-user-1",
		username:    "alice",
		displayName: "Alice",
		roles:       []string{"school_admin"},
		mfaProofAt:  proofAt,
	})

	assert.Equal(t, "casdoor-user-1", GetUserID(c))
	assert.Equal(t, "alice", GetUsername(c))
	assert.True(t, GetMFAProofVerifiedAt(c).Equal(proofAt))
	assert.False(t, GetMFAEnrollmentActive(c))
}

type fakeRoleScopeResolver struct {
	scopes map[string][]string
	err    error
}

func (f fakeRoleScopeResolver) ResolveRoleScopes(context.Context, string, []string) (map[string][]string, error) {
	return f.scopes, f.err
}

type countingRoleScopeResolver struct {
	calls int
}

func (f *countingRoleScopeResolver) ResolveRoleScopes(context.Context, string, []string) (map[string][]string, error) {
	f.calls++
	return map[string][]string{"school_admin": {"1001"}}, nil
}

func TestWithResolvedRoleScopesMergesIntoAuthResult(t *testing.T) {
	result, err := withResolvedRoleScopes(context.Background(), &authResult{
		userID: "casdoor-user-1",
		roles:  []string{"school_admin"},
		orgScopedRoles: map[string][]string{
			"school_admin": {"1001"},
		},
	}, fakeRoleScopeResolver{
		scopes: map[string][]string{"school_admin": {"1002"}},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"1001", "1002"}, result.orgScopedRoles["school_admin"])
}

func TestWithResolvedRoleScopesSkipsSelfSignedTokens(t *testing.T) {
	resolver := &countingRoleScopeResolver{}

	result, err := withResolvedRoleScopes(context.Background(), &authResult{
		userID:     "phone-user-1",
		roles:      []string{"school_admin"},
		selfSigned: true,
	}, resolver)

	require.NoError(t, err)
	assert.True(t, result.selfSigned)
	assert.Zero(t, resolver.calls)
	assert.Nil(t, result.orgScopedRoles)
}

func TestWithResolvedRoleScopesMarksBackendUnavailable(t *testing.T) {
	expectedErr := errors.New("openfga unavailable")

	_, err := withResolvedRoleScopes(context.Background(), &authResult{
		userID: "casdoor-user-1",
		roles:  []string{"school_admin"},
	}, fakeRoleScopeResolver{err: expectedErr})

	require.ErrorIs(t, err, errRoleScopeUnavailable)
	require.ErrorIs(t, err, expectedErr)
	assert.True(t, authBackendUnavailable(err))
}
