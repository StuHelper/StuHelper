package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
)

type fakeProfileFGAClient struct {
	readByRelation map[string][]fga.Tuple
	deleted        [][]fga.Tuple
	written        [][]fga.Tuple
}

func (f *fakeProfileFGAClient) ReadTuples(_ context.Context, _ string, relation string) ([]fga.Tuple, error) {
	return append([]fga.Tuple(nil), f.readByRelation[relation]...), nil
}

func (f *fakeProfileFGAClient) WriteTuples(_ context.Context, tuples []fga.Tuple) error {
	f.written = append(f.written, append([]fga.Tuple(nil), tuples...))
	return nil
}

func (f *fakeProfileFGAClient) DeleteTuples(_ context.Context, tuples []fga.Tuple) error {
	f.deleted = append(f.deleted, append([]fga.Tuple(nil), tuples...))
	return nil
}

func TestSyncUserProfileProjectionUsesTargetStudentAuthority(t *testing.T) {
	schoolID := int64(4111010006)
	fgaClient := &fakeProfileFGAClient{readByRelation: map[string][]fga.Tuple{
		"school": {{User: "school:4111010001", Relation: "school", Object: "user_profile:123"}},
	}}
	svc, err := NewService(
		&mockRepo{},
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithProfileFGAClient(fgaClient),
	)
	require.NoError(t, err)
	svc.SetVerificationStatusGateway(fakeVerificationStatusGateway{
		student: CurrentStudentStatus{Eligible: true, SchoolID: &schoolID},
	})

	require.NoError(t, svc.syncUserProfileProjection(context.Background(), 123))
	require.Len(t, fgaClient.deleted, 1)
	require.Len(t, fgaClient.written, 1)
	assert.Equal(t, []fga.Tuple{
		{User: "user:123", Relation: "owner", Object: "user_profile:123"},
		{User: "school:4111010006", Relation: "school", Object: "user_profile:123"},
	}, fgaClient.written[0])
}

func TestSyncUserProfileProjectionPurgesLegacySchoolTupleWithoutProfileShell(t *testing.T) {
	legacy := fga.Tuple{User: "school:4111010001", Relation: "school", Object: "user_profile:123"}
	fgaClient := &fakeProfileFGAClient{readByRelation: map[string][]fga.Tuple{"school": {legacy}}}
	svc, err := NewService(
		&mockRepo{},
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithProfileFGAClient(fgaClient),
	)
	require.NoError(t, err)
	svc.SetVerificationStatusGateway(fakeVerificationStatusGateway{})

	require.NoError(t, svc.syncUserProfileProjection(context.Background(), 123))
	assert.Equal(t, [][]fga.Tuple{{legacy}}, fgaClient.deleted)
}

func TestSyncUserProfileProjectionFailsClosedWithoutTargetAuthority(t *testing.T) {
	svc, err := NewService(
		&mockRepo{},
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithProfileFGAClient(&fakeProfileFGAClient{}),
	)
	require.NoError(t, err)

	err = svc.syncUserProfileProjection(context.Background(), 123)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "student verification status dependency")
}
