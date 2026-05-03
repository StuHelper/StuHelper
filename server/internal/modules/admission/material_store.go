package admission

import (
	"context"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/storage"
)

type AdmissionMaterialStore interface {
	PutAdmissionMaterial(ctx context.Context, objectKey string, content []byte, contentType string) error
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
