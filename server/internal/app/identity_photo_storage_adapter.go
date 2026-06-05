package app

import (
	"context"
	"errors"
	"strings"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/storage"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/objectstorage"
)

type identityPhotoStorageAdapter struct {
	service  *storage.Service
	mountKey string
}

func newIdentityPhotoStorageAdapter(service *storage.Service, mountKey string) identityPhotoStorageAdapter {
	key := strings.TrimSpace(mountKey)
	if key == "" {
		key = storage.DefaultMountKey
	}
	return identityPhotoStorageAdapter{
		service:  service,
		mountKey: key,
	}
}

func (a identityPhotoStorageAdapter) Upload(ctx context.Context, key string, content []byte, contentType string) error {
	_, _, err := a.service.Put(ctx, a.mountKey, key, content, contentType)
	return normalizeIdentityPhotoStorageError(err)
}

func (a identityPhotoStorageAdapter) PresignGetURL(ctx context.Context, key string) (string, error) {
	url, err := a.service.GetDownloadURLByMountKey(ctx, a.mountKey, key)
	return url, normalizeIdentityPhotoStorageError(err)
}

func normalizeIdentityPhotoStorageError(err error) error {
	switch {
	case errors.Is(err, storage.ErrMountNotFound),
		errors.Is(err, storage.ErrMountDisabled),
		errors.Is(err, storage.ErrDriverNotRegistered),
		objectstorage.IsKind(err, objectstorage.ErrorKindConfig),
		objectstorage.IsKind(err, objectstorage.ErrorKindAuthentication),
		objectstorage.IsKind(err, objectstorage.ErrorKindPermission),
		objectstorage.IsKind(err, objectstorage.ErrorKindNotFound):
		return user.ErrIdentityPhotoStorageUnavailable
	case objectstorage.IsKind(err, objectstorage.ErrorKindNetwork):
		return user.ErrIdentityPhotoStorageTemporaryUnavailable
	default:
		return err
	}
}
