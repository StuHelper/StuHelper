package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
)

type fakeProfileFGAClient struct {
	deleteCalls [][]fga.Tuple
	writeCalls  [][]fga.Tuple
}

func (f *fakeProfileFGAClient) WriteTuples(_ context.Context, tuples []fga.Tuple) error {
	copied := append([]fga.Tuple(nil), tuples...)
	f.writeCalls = append(f.writeCalls, copied)
	return nil
}

func (f *fakeProfileFGAClient) DeleteTuples(_ context.Context, tuples []fga.Tuple) error {
	copied := append([]fga.Tuple(nil), tuples...)
	f.deleteCalls = append(f.deleteCalls, copied)
	return nil
}

func TestReconcileFGAUserProfileTuples_ApproveWritesOwnerAndSchool(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
			require.Equal(t, int64(123), userID)
			schoolID := "10006"
			return &Profile{UserID: userID, SchoolID: &schoolID}, nil
		},
	}

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	fgaClient := &fakeProfileFGAClient{}
	h := NewHandler(svc, fgaClient, nil, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/student-verifications/123", nil)
	c.Request = req
	c.Set("userID", "external-user-123")

	h.reconcileFGAUserProfileTuples(c, 123, true)

	require.Len(t, fgaClient.deleteCalls, 1)
	require.Len(t, fgaClient.writeCalls, 1)

	assert.Equal(t, []fga.Tuple{{
		User:     "school:10006",
		Relation: "school",
		Object:   "user_profile:123",
	}}, fgaClient.deleteCalls[0])

	assert.Equal(t, []fga.Tuple{
		{User: "user:test-external-id", Relation: "owner", Object: "user_profile:123"},
		{User: "school:10006", Relation: "school", Object: "user_profile:123"},
	}, fgaClient.writeCalls[0])
}

func TestReconcileFGAUserProfileTuples_RejectDeletesSchoolAndKeepsOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
			require.Equal(t, int64(123), userID)
			schoolID := "10006"
			return &Profile{UserID: userID, SchoolID: &schoolID}, nil
		},
	}

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	fgaClient := &fakeProfileFGAClient{}
	h := NewHandler(svc, fgaClient, nil, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/student-verifications/123", nil)
	c.Request = req
	c.Set("userID", "external-user-123")

	h.reconcileFGAUserProfileTuples(c, 123, false)

	require.Len(t, fgaClient.deleteCalls, 1)
	require.Len(t, fgaClient.writeCalls, 1)

	assert.Equal(t, []fga.Tuple{{
		User:     "school:10006",
		Relation: "school",
		Object:   "user_profile:123",
	}}, fgaClient.deleteCalls[0])

	assert.Equal(t, []fga.Tuple{
		{User: "user:test-external-id", Relation: "owner", Object: "user_profile:123"},
	}, fgaClient.writeCalls[0])
}
