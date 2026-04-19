package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/objectstorage"
)

type fakeStoreClient struct {
	deleteKey  string
	presignKey string
	statKey    string
	uploadKey  string
}

func (s *fakeStoreClient) CheckBucket(context.Context) error { return nil }

func (s *fakeStoreClient) Delete(_ context.Context, key string) error {
	s.deleteKey = key
	return nil
}

func (s *fakeStoreClient) PresignGetURL(_ context.Context, key string) (string, error) {
	s.presignKey = key
	return "https://storage.example.test/download", nil
}

func (s *fakeStoreClient) Stat(_ context.Context, key string) (*objectstorage.ObjectInfo, error) {
	s.statKey = key
	return &objectstorage.ObjectInfo{
		Key:         key,
		SizeBytes:   16,
		ContentType: "text/plain",
	}, nil
}

func (s *fakeStoreClient) Upload(_ context.Context, key string, _ []byte, _ string) error {
	s.uploadKey = key
	return nil
}

func TestS3Driver_PersistsRelativeObjectKeyAndAppliesBasePathOnce(t *testing.T) {
	t.Parallel()

	store := &fakeStoreClient{}
	driver := newS3Driver(config.ObjectStorageConfig{})
	driver.storeFactory = func(context.Context, Mount) (storeClient, error) {
		return store, nil
	}

	mount := Mount{BasePath: "resources"}
	stored, err := driver.Put(context.Background(), mount, "users/u-1/file.txt", []byte("payload"), "text/plain")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, "users/u-1/file.txt", stored.ObjectKey)
	assert.Equal(t, "resources/users/u-1/file.txt", store.uploadKey)

	url, err := driver.GetDownloadURL(context.Background(), mount, stored.ObjectKey)
	require.NoError(t, err)
	assert.Equal(t, "https://storage.example.test/download", url)
	assert.Equal(t, "resources/users/u-1/file.txt", store.presignKey)

	err = driver.Delete(context.Background(), mount, stored.ObjectKey)
	require.NoError(t, err)
	assert.Equal(t, "resources/users/u-1/file.txt", store.deleteKey)

	info, err := driver.Stat(context.Background(), mount, stored.ObjectKey)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "users/u-1/file.txt", info.ObjectKey)
	assert.Equal(t, "resources/users/u-1/file.txt", store.statKey)
}
