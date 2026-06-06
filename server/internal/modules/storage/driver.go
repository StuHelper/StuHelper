package storage

import (
	"context"
	"fmt"
	"path"
	"strings"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/objectstorage"
)

type Driver interface {
	Capabilities() CapabilitySet
	HealthCheck(ctx context.Context, mount Mount) error
	Put(ctx context.Context, mount Mount, objectKey string, content []byte, contentType string) (*StoredObject, error)
	Stat(ctx context.Context, mount Mount, objectKey string) (*StoredObject, error)
	Delete(ctx context.Context, mount Mount, objectKey string) error
	GetDownloadURL(ctx context.Context, mount Mount, objectKey string) (string, error)
}

type storeClient interface {
	CheckBucket(ctx context.Context) error
	Delete(ctx context.Context, key string) error
	PresignGetURL(ctx context.Context, key string) (string, error)
	Stat(ctx context.Context, key string) (*objectstorage.ObjectInfo, error)
	Upload(ctx context.Context, key string, content []byte, contentType string) error
}

type storeFactory func(ctx context.Context, mount Mount) (storeClient, error)

type Registry struct {
	drivers map[string]Driver
}

func NewRegistry(cfg config.ObjectStorageConfig) *Registry {
	return &Registry{
		drivers: map[string]Driver{
			"s3": newS3Driver(cfg),
		},
	}
}

func (r *Registry) Get(name string) (Driver, error) {
	driver, ok := r.drivers[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrDriverNotRegistered, name)
	}
	return driver, nil
}

type s3Driver struct {
	cfg          config.ObjectStorageConfig
	storeFactory storeFactory
}

func newS3Driver(cfg config.ObjectStorageConfig) *s3Driver {
	driver := &s3Driver{cfg: cfg}
	driver.storeFactory = driver.newStore
	return driver
}

func (d *s3Driver) Capabilities() CapabilitySet {
	return CapabilitySet{Put: true, Delete: true, Stat: true, PresignedDownload: true}
}

func (d *s3Driver) HealthCheck(ctx context.Context, mount Mount) error {
	store, err := d.storeFactory(ctx, mount)
	if err != nil {
		return err
	}
	return store.CheckBucket(ctx)
}

func (d *s3Driver) Put(ctx context.Context, mount Mount, objectKey string, content []byte, contentType string) (*StoredObject, error) {
	store, err := d.storeFactory(ctx, mount)
	if err != nil {
		return nil, err
	}
	objectKey = normalizeObjectKey(objectKey)
	key := d.mountKey(mount, objectKey)
	if err := store.Upload(ctx, key, content, contentType); err != nil {
		return nil, err
	}
	// 资源层只持久化 mount 内相对 key，避免 base_path 变更时重复拼接。
	return &StoredObject{ObjectKey: objectKey, SizeBytes: int64(len(content)), ContentType: contentType}, nil
}

func (d *s3Driver) Stat(ctx context.Context, mount Mount, objectKey string) (*StoredObject, error) {
	store, err := d.storeFactory(ctx, mount)
	if err != nil {
		return nil, err
	}
	objectKey = normalizeObjectKey(objectKey)
	info, err := store.Stat(ctx, d.mountKey(mount, objectKey))
	if err != nil {
		return nil, err
	}
	return &StoredObject{ObjectKey: objectKey, SizeBytes: info.SizeBytes, ContentType: info.ContentType}, nil
}

func (d *s3Driver) Delete(ctx context.Context, mount Mount, objectKey string) error {
	store, err := d.storeFactory(ctx, mount)
	if err != nil {
		return err
	}
	return store.Delete(ctx, d.mountKey(mount, objectKey))
}

func (d *s3Driver) GetDownloadURL(ctx context.Context, mount Mount, objectKey string) (string, error) {
	store, err := d.storeFactory(ctx, mount)
	if err != nil {
		return "", err
	}
	return store.PresignGetURL(ctx, d.mountKey(mount, objectKey))
}

func (d *s3Driver) newStore(ctx context.Context, mount Mount) (storeClient, error) {
	bucket := d.cfg.Bucket
	if mount.Bucket != nil && *mount.Bucket != "" {
		bucket = *mount.Bucket
	}
	return objectstorage.New(ctx, objectstorage.Config{
		Endpoint:        d.cfg.Endpoint,
		Region:          d.cfg.Region,
		Bucket:          bucket,
		AccessKeyID:     d.cfg.AccessKeyID,
		SecretAccessKey: d.cfg.SecretAccessKey,
		UseSSL:          d.cfg.UseSSL,
		ForcePathStyle:  d.cfg.ForcePathStyle,
		PresignTTL:      timeDurationSeconds(d.cfg.PresignTTL),
		TLSCAFile:       d.cfg.TLSCAFile,
	})
}

func (d *s3Driver) mountKey(mount Mount, objectKey string) string {
	objectKey = normalizeObjectKey(objectKey)
	base := normalizeObjectKey(mount.BasePath)
	if base == "" {
		return objectKey
	}
	if objectKey == "" {
		return base
	}
	return base + "/" + objectKey
}

// normalizeObjectKey converts an object or base path into a mount-relative key.
func normalizeObjectKey(key string) string {
	key = path.Clean("/" + strings.TrimSpace(key))
	return strings.Trim(key, "/")
}
