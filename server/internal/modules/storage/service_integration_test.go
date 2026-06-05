package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

type fakeDriver struct {
	healthErr   error
	downloadURL string
}

func (d fakeDriver) Capabilities() CapabilitySet {
	return CapabilitySet{Put: true, Delete: true, Stat: true, PresignedDownload: true}
}

func (d fakeDriver) HealthCheck(context.Context, Mount) error {
	return d.healthErr
}

func (d fakeDriver) Put(context.Context, Mount, string, []byte, string) (*StoredObject, error) {
	return nil, errors.New("unexpected Put call")
}

func (d fakeDriver) Stat(context.Context, Mount, string) (*StoredObject, error) {
	return nil, errors.New("unexpected Stat call")
}

func (d fakeDriver) Delete(context.Context, Mount, string) error {
	return errors.New("unexpected Delete call")
}

func (d fakeDriver) GetDownloadURL(context.Context, Mount, string) (string, error) {
	if d.downloadURL == "" {
		return "", errors.New("unexpected GetDownloadURL call")
	}
	return d.downloadURL, nil
}

func TestEnsureDefaultMountAndListMounts(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, config.ObjectStorageConfig{
		Endpoint: "minio:9000",
		Bucket:   "stuhelper-assets",
	})
	ctx := context.Background()

	err := svc.EnsureDefaultMount(ctx)
	require.NoError(t, err)

	defaultMount, err := repo.GetMountByKey(ctx, DefaultMountKey)
	require.NoError(t, err)
	require.NotNil(t, defaultMount.Bucket)
	assert.Equal(t, "stuhelper-assets", *defaultMount.Bucket)

	items, err := svc.ListMounts(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, items)
	assert.Equal(t, DefaultMountKey, items[0].Key)
	assert.True(t, items[0].Capabilities.PresignedDownload)
}

func TestCreateMountAndCheckMountHealth(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, config.ObjectStorageConfig{})
	svc.registry.drivers["s3"] = fakeDriver{}
	ctx := context.Background()

	bucket := "resource-bucket"
	created, err := svc.CreateMount(ctx, CreateMountRequest{
		Key:      "campus-share",
		Name:     "Campus Share",
		Driver:   "s3",
		Bucket:   &bucket,
		BasePath: "resources",
		Enabled:  true,
	})
	require.NoError(t, err)
	assert.True(t, created.Capabilities.Put)

	healthy, err := svc.CheckMountHealth(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, healthy.LastHealthStatus)
	assert.Equal(t, "healthy", *healthy.LastHealthStatus)
	assert.Nil(t, healthy.LastHealthError)

	svc.registry.drivers["s3"] = fakeDriver{healthErr: errors.New("network timeout")}
	unhealthy, err := svc.CheckMountHealth(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, unhealthy.LastHealthStatus)
	require.NotNil(t, unhealthy.LastHealthError)
	assert.Equal(t, "unhealthy", *unhealthy.LastHealthStatus)
	assert.Equal(t, "network timeout", *unhealthy.LastHealthError)
}

func TestCreateMount_NormalizesRequest(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, config.ObjectStorageConfig{})
	svc.registry.drivers["s3"] = fakeDriver{}
	ctx := context.Background()

	bucket := "  resource-bucket  "
	created, err := svc.CreateMount(ctx, CreateMountRequest{
		Key:      " campus-share-trimmed ",
		Name:     " Campus Share ",
		Driver:   " s3 ",
		Bucket:   &bucket,
		BasePath: " /resources/materials/ ",
		Enabled:  true,
	})

	require.NoError(t, err)
	assert.Equal(t, "campus-share-trimmed", created.Key)
	assert.Equal(t, "Campus Share", created.Name)
	assert.Equal(t, "s3", created.Driver)
	require.NotNil(t, created.Bucket)
	assert.Equal(t, "resource-bucket", *created.Bucket)
	assert.Equal(t, "resources/materials", created.BasePath)
}

func TestCreateMount_RejectsUnknownDriver(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, config.ObjectStorageConfig{})

	_, err := svc.CreateMount(context.Background(), CreateMountRequest{
		Key:      "invalid-driver",
		Name:     "Invalid Driver",
		Driver:   "webdav",
		BasePath: "",
		Enabled:  true,
	})
	require.ErrorIs(t, err, ErrDriverNotRegistered)
}

func TestCreateMount_RejectsBlankRequiredFields(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, config.ObjectStorageConfig{})

	_, err := svc.CreateMount(context.Background(), CreateMountRequest{
		Key:     " ",
		Name:    "Blank Key",
		Driver:  "s3",
		Enabled: true,
	})

	require.ErrorIs(t, err, ErrInvalidMountConfig)
}

func TestValidateMountByKeyAndGetDownloadURLByMountKey(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, config.ObjectStorageConfig{})
	svc.registry.drivers["s3"] = fakeDriver{downloadURL: "https://storage.example.test/identity/front.png"}
	ctx := context.Background()

	created, err := repo.GetMountByKey(ctx, DefaultMountKey)
	require.NoError(t, err)

	validated, err := svc.ValidateMountByKey(ctx, DefaultMountKey)
	require.NoError(t, err)
	require.NotNil(t, validated.LastHealthStatus)
	assert.Equal(t, created.ID, validated.ID)
	assert.Equal(t, "healthy", *validated.LastHealthStatus)

	url, err := svc.GetDownloadURLByMountKey(ctx, DefaultMountKey, "identities/42/front.png")
	require.NoError(t, err)
	assert.Equal(t, "https://storage.example.test/identity/front.png", url)
}
