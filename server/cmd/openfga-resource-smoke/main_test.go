package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
)

func TestRunSmokeGrantListAndRevoke(t *testing.T) {
	client := newFakeSmokeFGA()

	evidence, err := runSmoke(context.Background(), client, smokeConfig{
		APIURL:     "http://openfga.example.test",
		AppID:      "app1",
		ModelID:    "model1",
		ResourceID: "resource1",
		StoreID:    "store1",
	})

	require.NoError(t, err)
	assert.Equal(t, "open_platform_app:app1", evidence.AppObject)
	assert.Equal(t, "resource_item:resource1", evidence.ResourceObject)
	assert.True(t, evidence.ReadAfterGrant)
	assert.True(t, evidence.WriteAfterGrant)
	assert.True(t, evidence.ListedReadGrant)
	assert.False(t, evidence.ReadAfterRevoke)
	assert.False(t, evidence.WriteAfterRevoke)
	assert.False(t, evidence.ListedReadAfterRevoke)
	assert.Empty(t, client.tuples)
}

func TestRunSmokeFailsWhenOpenFGACheckFails(t *testing.T) {
	client := newFakeSmokeFGA()
	client.checkErr = errors.New("openfga unavailable")

	_, err := runSmoke(context.Background(), client, smokeConfig{
		APIURL:     "http://openfga.example.test",
		AppID:      "app1",
		ModelID:    "model1",
		ResourceID: "resource1",
		StoreID:    "store1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "check read grant")
}

type fakeSmokeFGA struct {
	checkErr error
	tuples   map[fga.Tuple]struct{}
}

func newFakeSmokeFGA() *fakeSmokeFGA {
	return &fakeSmokeFGA{tuples: map[fga.Tuple]struct{}{}}
}

func (f *fakeSmokeFGA) Check(_ context.Context, user, relation, object string) (bool, error) {
	if f.checkErr != nil {
		return false, f.checkErr
	}
	_, ok := f.tuples[fga.Tuple{User: user, Relation: relation, Object: object}]
	return ok, nil
}

func (f *fakeSmokeFGA) WriteMissingTuples(_ context.Context, tuples []fga.Tuple) error {
	for _, tuple := range tuples {
		f.tuples[tuple] = struct{}{}
	}
	return nil
}

func (f *fakeSmokeFGA) DeleteTuples(_ context.Context, tuples []fga.Tuple) error {
	for _, tuple := range tuples {
		delete(f.tuples, tuple)
	}
	return nil
}

func (f *fakeSmokeFGA) ListObjects(_ context.Context, user, relation, objectType string) ([]string, error) {
	prefix := objectType + ":"
	objects := make([]string, 0)
	for tuple := range f.tuples {
		if tuple.User == user && tuple.Relation == relation && len(tuple.Object) > len(prefix) && tuple.Object[:len(prefix)] == prefix {
			objects = append(objects, tuple.Object)
		}
	}
	return objects, nil
}
