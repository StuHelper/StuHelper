package admission

import (
	"context"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/storage"
)

type AdmissionMaterialStore interface {
	PutAdmissionMaterial(ctx context.Context, objectKey string, content []byte, contentType string) error
	DeleteAdmissionMaterial(ctx context.Context, objectKey string) error
	GetAdmissionMaterialURL(ctx context.Context, objectKey string) (string, error)
}

type StorageAdmissionMaterialStore struct {
	service  *storage.Service
	mountKey string
}

func NewStorageAdmissionMaterialStore(service *storage.Service, mountKey string) *StorageAdmissionMaterialStore {
	return &StorageAdmissionMaterialStore{service: service, mountKey: mountKey}
}

func (s *StorageAdmissionMaterialStore) PutAdmissionMaterial(
	ctx context.Context,
	objectKey string,
	content []byte,
	contentType string,
) error {
	_, _, err := s.service.Put(ctx, s.mountKey, objectKey, content, contentType)
	return err
}

func (s *StorageAdmissionMaterialStore) DeleteAdmissionMaterial(ctx context.Context, objectKey string) error {
	mount, err := s.service.ValidateMountByKey(ctx, s.mountKey)
	if err != nil {
		return err
	}
	return s.service.Delete(ctx, mount.ID, objectKey)
}

func (s *StorageAdmissionMaterialStore) GetAdmissionMaterialURL(ctx context.Context, objectKey string) (string, error) {
	return s.service.GetDownloadURLByMountKey(ctx, s.mountKey, objectKey)
}
