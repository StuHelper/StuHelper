package app

import (
	"context"
	"errors"
	"strings"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/admission"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/storage"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/objectstorage"
)

type admissionMaterialStorageAdapter struct {
	service  *storage.Service
	mountKey string
}

func newAdmissionMaterialStorageAdapter(service *storage.Service, mountKey string) admissionMaterialStorageAdapter {
	key := strings.TrimSpace(mountKey)
	if key == "" {
		key = storage.DefaultMountKey
	}
	return admissionMaterialStorageAdapter{
		service:  service,
		mountKey: key,
	}
}

func (a admissionMaterialStorageAdapter) PutAdmissionMaterial(
	ctx context.Context,
	objectKey string,
	content []byte,
	contentType string,
) error {
	_, _, err := a.service.Put(ctx, a.mountKey, objectKey, content, contentType)
	return normalizeAdmissionMaterialStorageError(err)
}

func (a admissionMaterialStorageAdapter) DeleteAdmissionMaterial(ctx context.Context, objectKey string) error {
	mount, err := a.service.ValidateMountByKey(ctx, a.mountKey)
	if err != nil {
		return normalizeAdmissionMaterialStorageError(err)
	}
	return normalizeAdmissionMaterialStorageError(a.service.Delete(ctx, mount.ID, objectKey))
}

func (a admissionMaterialStorageAdapter) GetAdmissionMaterialURL(ctx context.Context, objectKey string) (string, error) {
	url, err := a.service.GetDownloadURLByMountKey(ctx, a.mountKey, objectKey)
	return url, normalizeAdmissionMaterialStorageError(err)
}

func normalizeAdmissionMaterialStorageError(err error) error {
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
		return admission.ErrAdmissionMaterialStoreUnavailable
	default:
		return err
	}
}
