package app

import (
	"context"
	"errors"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/resource"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/storage"
)

type resourceStorageAdapter struct {
	service *storage.Service
}

func newResourceStorageAdapter(service *storage.Service) resourceStorageAdapter {
	return resourceStorageAdapter{service: service}
}

func (a resourceStorageAdapter) Put(
	ctx context.Context,
	mountKey string,
	objectKey string,
	content []byte,
	contentType string,
) (int64, *resource.StoredObject, error) {
	mount, stored, err := a.service.Put(ctx, mountKey, objectKey, content, contentType)
	if err != nil {
		return 0, nil, normalizeResourceStorageError(err)
	}
	return mount.ID, &resource.StoredObject{
		ObjectKey:   stored.ObjectKey,
		SizeBytes:   stored.SizeBytes,
		ContentType: stored.ContentType,
	}, nil
}

func (a resourceStorageAdapter) Delete(ctx context.Context, mountID int64, objectKey string) error {
	return normalizeResourceStorageError(a.service.Delete(ctx, mountID, objectKey))
}

func (a resourceStorageAdapter) GetDownloadURL(ctx context.Context, mountID int64, objectKey string) (string, error) {
	url, err := a.service.GetDownloadURL(ctx, mountID, objectKey)
	return url, normalizeResourceStorageError(err)
}

func normalizeResourceStorageError(err error) error {
	switch {
	case errors.Is(err, storage.ErrMountNotFound):
		return resource.ErrResourceStorageMountNotFound
	case errors.Is(err, storage.ErrMountDisabled):
		return resource.ErrResourceStorageMountDisabled
	case errors.Is(err, storage.ErrDriverNotRegistered):
		return resource.ErrResourceStorageDriverMissing
	case errors.Is(err, storage.ErrStoredObjectMissing):
		return resource.ErrResourceStoredObjectMissing
	case errors.Is(err, storage.ErrInvalidObjectKey),
		errors.Is(err, storage.ErrInvalidStoredObject):
		return resource.ErrResourceStoredObjectInvalid
	default:
		return err
	}
}
