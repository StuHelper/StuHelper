package app

import (
	"context"
	"errors"
	"strings"

	"github.com/StuHelper/StuHelper/server/internal/modules/storage"
	"github.com/StuHelper/StuHelper/server/internal/modules/studentverification"
	"github.com/StuHelper/StuHelper/server/internal/pkg/objectstorage"
)

type studentVerificationMaterialStorageAdapter struct {
	service  *storage.Service
	mountKey string
}

func newStudentVerificationMaterialStorageAdapter(
	service *storage.Service,
	mountKey string,
) studentVerificationMaterialStorageAdapter {
	key := strings.TrimSpace(mountKey)
	if key == "" {
		key = storage.DefaultMountKey
	}
	return studentVerificationMaterialStorageAdapter{service: service, mountKey: key}
}

func (a studentVerificationMaterialStorageAdapter) PutManualReviewMaterial(
	ctx context.Context,
	objectKey string,
	content []byte,
	contentType string,
) error {
	if a.service == nil {
		return studentverification.ErrManualMaterialStoreUnavailable
	}
	_, _, err := a.service.Put(ctx, a.mountKey, objectKey, content, contentType)
	return normalizeStudentVerificationMaterialStorageError(err)
}

func (a studentVerificationMaterialStorageAdapter) DeleteManualReviewMaterial(
	ctx context.Context,
	objectKey string,
) error {
	if a.service == nil {
		return studentverification.ErrManualMaterialStoreUnavailable
	}
	err := a.service.DeleteByMountKey(ctx, a.mountKey, objectKey)
	if errors.Is(err, storage.ErrStoredObjectMissing) ||
		objectstorage.IsKind(err, objectstorage.ErrorKindNotFound) {
		return nil
	}
	return normalizeStudentVerificationMaterialStorageError(err)
}

func (a studentVerificationMaterialStorageAdapter) GetManualReviewMaterialURL(
	ctx context.Context,
	objectKey string,
) (string, error) {
	if a.service == nil {
		return "", studentverification.ErrManualMaterialStoreUnavailable
	}
	url, err := a.service.GetDownloadURLByMountKey(ctx, a.mountKey, objectKey)
	return url, normalizeStudentVerificationMaterialStorageError(err)
}

func normalizeStudentVerificationMaterialStorageError(err error) error {
	switch {
	case errors.Is(err, storage.ErrMountNotFound),
		errors.Is(err, storage.ErrMountDisabled),
		errors.Is(err, storage.ErrDriverNotRegistered),
		errors.Is(err, storage.ErrInvalidObjectKey),
		errors.Is(err, storage.ErrStoredObjectMissing),
		errors.Is(err, storage.ErrInvalidStoredObject),
		objectstorage.IsKind(err, objectstorage.ErrorKindConfig),
		objectstorage.IsKind(err, objectstorage.ErrorKindAuthentication),
		objectstorage.IsKind(err, objectstorage.ErrorKindPermission),
		objectstorage.IsKind(err, objectstorage.ErrorKindNotFound),
		objectstorage.IsKind(err, objectstorage.ErrorKindNetwork):
		return studentverification.ErrManualMaterialStoreUnavailable
	default:
		return err
	}
}
