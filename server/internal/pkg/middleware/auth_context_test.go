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

func TestSetClaimsToContextPropagatesAuthTimeAndMFAProof(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/me", nil)
	authTime := time.Date(2026, 5, 2, 11, 30, 0, 0, time.UTC)
	proofAt := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	accessExpiry := time.Date(2026, 5, 2, 13, 0, 0, 0, time.UTC)

	setClaimsToContext(c, &authResult{
		userID:               "casdoor-user-1",
		username:             "alice",
		displayName:          "Alice",
		roles:                []string{"school_admin"},
		authTime:             authTime,
		mfaProofAt:           proofAt,
		accessTokenExpiresAt: accessExpiry,
	})

	assert.Equal(t, "casdoor-user-1", GetUserID(c))
	assert.Equal(t, "alice", GetUsername(c))
	assert.True(t, GetAuthenticationTime(c).Equal(authTime))
	assert.True(t, GetMFAProofVerifiedAt(c).Equal(proofAt))
	assert.True(t, GetAccessTokenExpiresAt(c).Equal(accessExpiry))
	assert.False(t, GetMFAEnrollmentActive(c))
}

func TestSetClaimsToContextNormalizesScopedRoleGrants(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	setClaimsToContext(c, &authResult{
		userID: "casdoor-user-1",
		roles:  []string{"school_admin"},
		scopedRoleGrants: map[string][]string{
			" school_admin ": {" 4111010002 ", "4111010001", "4111010002", " "},
			" ":              {"ignored"},
		},
	})

	value, exists := c.Get(CtxKeyScopedRoleGrants)
	require.True(t, exists)
	assert.Equal(t, map[string][]string{
		"school_admin": {"4111010001", "4111010002"},
	}, value)
}

type fakeAccessSnapshotResolver struct {
	roles  []string
	scopes map[string][]string
	err    error
}

func (f fakeAccessSnapshotResolver) ResolveAccessSnapshot(
	context.Context,
	string,
) ([]string, map[string][]string, error) {
	return f.roles, f.scopes, f.err
}

type countingAccessSnapshotResolver struct {
	calls int
}

func (f *countingAccessSnapshotResolver) ResolveAccessSnapshot(
	context.Context,
	string,
) ([]string, map[string][]string, error) {
	f.calls++
	return []string{"user", "school_admin"}, map[string][]string{"school_admin": {"4111010001"}}, nil
}

func TestWithResolvedAccessSnapshotDiscardsProviderAuthorizationClaims(t *testing.T) {
	result, err := withResolvedAccessSnapshot(context.Background(), &authResult{
		userID: "casdoor-user-1",
		roles:  []string{"super_admin"},
		scopedRoleGrants: map[string][]string{
			" school_admin ": {"4111010002", " 4111010001 ", "4111010002", ""},
		},
	}, fakeAccessSnapshotResolver{
		roles: []string{"user", "school_admin"},
		scopes: map[string][]string{
			"school_admin": {"4111010003", " 4111010002 "},
			" ":            {"ignored"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"user", "school_admin"}, result.roles)
	assert.Equal(t, map[string][]string{
		"school_admin": {"4111010002", "4111010003"},
	}, result.scopedRoleGrants)
}

func TestWithResolvedAccessSnapshotResolvesAuthenticatedTokens(t *testing.T) {
	resolver := &countingAccessSnapshotResolver{}

	result, err := withResolvedAccessSnapshot(context.Background(), &authResult{
		userID: "casdoor-user-1",
		roles:  []string{"super_admin"},
	}, resolver)

	require.NoError(t, err)
	assert.Equal(t, 1, resolver.calls)
	assert.Equal(t, []string{"user", "school_admin"}, result.roles)
	assert.Equal(t, []string{"4111010001"}, result.scopedRoleGrants["school_admin"])
}

func TestWithResolvedAccessSnapshotMarksBackendUnavailable(t *testing.T) {
	expectedErr := errors.New("postgres unavailable")

	_, err := withResolvedAccessSnapshot(context.Background(), &authResult{
		userID: "casdoor-user-1",
		roles:  []string{"school_admin"},
	}, fakeAccessSnapshotResolver{err: expectedErr})

	require.ErrorIs(t, err, errAccessSnapshotUnavailable)
	require.ErrorIs(t, err, expectedErr)
	assert.True(t, authBackendUnavailable(err))
}

func TestWithResolvedAccessSnapshotWithoutResolverUsesIdentityOnlyRole(t *testing.T) {
	result, err := withResolvedAccessSnapshot(context.Background(), &authResult{
		userID: "casdoor-user-1",
		roles:  []string{"super_admin"},
		scopedRoleGrants: map[string][]string{
			"school_admin": {"4111010001"},
		},
	}, nil)

	require.NoError(t, err)
	assert.Equal(t, []string{"user"}, result.roles)
	assert.Nil(t, result.scopedRoleGrants)
}
