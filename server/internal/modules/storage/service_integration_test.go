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
	putObject   *StoredObject
	putErr      error
	deletedKeys []string
	onPut       func()
}

func (d *fakeDriver) Capabilities() CapabilitySet {
	return CapabilitySet{Put: true, Delete: true, Stat: true, PresignedDownload: true}
}

func (d *fakeDriver) HealthCheck(context.Context, Mount) error {
	return d.healthErr
}

func (d *fakeDriver) Put(context.Context, Mount, string, []byte, string) (*StoredObject, error) {
	if d.onPut != nil {
		d.onPut()
	}
	if d.putErr != nil {
		return nil, d.putErr
	}
	return d.putObject, nil
}

func (d *fakeDriver) Stat(context.Context, Mount, string) (*StoredObject, error) {
	return nil, errors.New("unexpected Stat call")
}

func (d *fakeDriver) Delete(ctx context.Context, _ Mount, objectKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d.deletedKeys = append(d.deletedKeys, objectKey)
	return nil
}

func (d *fakeDriver) GetDownloadURL(context.Context, Mount, string) (string, error) {
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
	svc.registry.drivers["s3"] = &fakeDriver{}
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

	svc.registry.drivers["s3"] = &fakeDriver{healthErr: errors.New("network timeout")}
	unhealthy, err := svc.CheckMountHealth(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, unhealthy.LastHealthStatus)
	require.NotNil(t, unhealthy.LastHealthError)
	assert.Equal(t, "unhealthy", *unhealthy.LastHealthStatus)
	assert.Equal(t, "network timeout", *unhealthy.LastHealthError)
}

func TestMountIDOperationsRejectInvalidIDBeforeDependencies(t *testing.T) {
	ctx := context.Background()
	svc := &Service{}

	for _, mountID := range []int64{0, -1} {
		mount, err := svc.CheckMountHealth(ctx, mountID)
		require.ErrorIs(t, err, ErrInvalidMountID)
		assert.Nil(t, mount)

		err = svc.Delete(ctx, mountID, "resources/object.txt")
		require.ErrorIs(t, err, ErrInvalidMountID)

		url, err := svc.GetDownloadURL(ctx, mountID, "resources/object.txt")
		require.ErrorIs(t, err, ErrInvalidMountID)
		assert.Empty(t, url)
	}
}

func TestCreateMount_NormalizesRequest(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, config.ObjectStorageConfig{})
	svc.registry.drivers["s3"] = &fakeDriver{}
	ctx := context.Background()

	bucket := "  resource-bucket  "
	created, err := svc.CreateMount(ctx, CreateMountRequest{
		Key:      " campus-share-trimmed ",
		Name:     " Campus Share ",
		Driver:   " s3 ",
		Bucket:   &bucket,
		BasePath: " /tenant/../resources/materials/ ",
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

func TestCreateMount_RejectsDuplicateKey(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, config.ObjectStorageConfig{})
	svc.registry.drivers["s3"] = &fakeDriver{}
	ctx := context.Background()

	req := CreateMountRequest{
		Key:     "campus-share-duplicate",
		Name:    "Campus Share",
		Driver:  "s3",
		Enabled: true,
	}
	_, err := svc.CreateMount(ctx, req)
	require.NoError(t, err)

	_, err = svc.CreateMount(ctx, req)
	require.ErrorIs(t, err, ErrMountAlreadyExists)
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
	svc.registry.drivers["s3"] = &fakeDriver{downloadURL: "https://storage.example.test/identity/front.png"}
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

func TestObjectOperationsRejectInvalidObjectKey(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, config.ObjectStorageConfig{})
	svc.registry.drivers["s3"] = &fakeDriver{
		onPut: func() {
			t.Fatal("driver Put must not be called for invalid object key")
		},
	}

	mount, err := repo.GetMountByKey(context.Background(), DefaultMountKey)
	require.NoError(t, err)

	for _, tc := range []struct {
		name string
		key  string
	}{
		{name: "blank", key: " "},
		{name: "root", key: "/"},
		{name: "parent only", key: "../"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := svc.Put(context.Background(), DefaultMountKey, tc.key, []byte("hello"), "text/plain")
			require.ErrorIs(t, err, ErrInvalidObjectKey)

			err = svc.Delete(context.Background(), mount.ID, tc.key)
			require.ErrorIs(t, err, ErrInvalidObjectKey)

			_, err = svc.GetDownloadURL(context.Background(), mount.ID, tc.key)
			require.ErrorIs(t, err, ErrInvalidObjectKey)

			_, err = svc.GetDownloadURLByMountKey(context.Background(), DefaultMountKey, tc.key)
			require.ErrorIs(t, err, ErrInvalidObjectKey)
		})
	}
}

func TestPutRejectsMissingObjectMetadata(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, config.ObjectStorageConfig{})
	driver := &fakeDriver{}
	svc.registry.drivers["s3"] = driver

	_, _, err := svc.Put(context.Background(), DefaultMountKey, "resources/1/file.txt", []byte("hello"), "text/plain")
	require.ErrorIs(t, err, ErrStoredObjectMissing)
	assert.Equal(t, []string{"resources/1/file.txt"}, driver.deletedKeys)
}

func TestPutNormalizesStoredObjectMetadata(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, config.ObjectStorageConfig{})
	driver := &fakeDriver{
		putObject: &StoredObject{
			ObjectKey:   " resources/1/actual-upload-key.txt ",
			SizeBytes:   5,
			ContentType: " text/plain ",
		},
	}
	svc.registry.drivers["s3"] = driver

	_, stored, err := svc.Put(context.Background(), DefaultMountKey, "resources/1/file.txt", []byte("hello"), "text/plain")

	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, "resources/1/actual-upload-key.txt", stored.ObjectKey)
	assert.Equal(t, "text/plain", stored.ContentType)
	assert.Empty(t, driver.deletedKeys)
}

func TestPutCleanupSurvivesRequestCancellation(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, config.ObjectStorageConfig{})

	ctx, cancel := context.WithCancel(context.Background())
	driver := &fakeDriver{
		putObject: &StoredObject{ObjectKey: " ", SizeBytes: 5, ContentType: "text/plain"},
		onPut:     cancel,
	}
	svc.registry.drivers["s3"] = driver

	_, _, err := svc.Put(ctx, DefaultMountKey, "resources/1/file.txt", []byte("hello"), "text/plain")
	require.ErrorIs(t, err, ErrInvalidStoredObject)
	assert.Equal(t, []string{"resources/1/file.txt"}, driver.deletedKeys)
}

func TestPutRejectsInvalidObjectMetadata(t *testing.T) {
	for _, tc := range []struct {
		name       string
		stored     *StoredObject
		cleanupKey string
	}{
		{
			name:       "blank object key",
			stored:     &StoredObject{ObjectKey: " ", SizeBytes: 5, ContentType: "text/plain"},
			cleanupKey: "resources/1/file.txt",
		},
		{
			name:       "blank content type",
			stored:     &StoredObject{ObjectKey: "resources/1/actual-upload-key.txt", SizeBytes: 5, ContentType: " "},
			cleanupKey: "resources/1/actual-upload-key.txt",
		},
		{
			name:       "negative size",
			stored:     &StoredObject{ObjectKey: "resources/1/actual-upload-key.txt", SizeBytes: -1, ContentType: "text/plain"},
			cleanupKey: "resources/1/actual-upload-key.txt",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := postgresfixture.Start(t)
			repo := NewRepository(fixture.DB)
			svc := NewService(repo, config.ObjectStorageConfig{})
			driver := &fakeDriver{putObject: tc.stored}
			svc.registry.drivers["s3"] = driver

			_, _, err := svc.Put(context.Background(), DefaultMountKey, "resources/1/file.txt", []byte("hello"), "text/plain")
			require.ErrorIs(t, err, ErrInvalidStoredObject)
			assert.Equal(t, []string{tc.cleanupKey}, driver.deletedKeys)
		})
	}
}
