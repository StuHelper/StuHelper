package user

import (
	"context"
	"strings"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/storage"
)

type storageIdentityPhotoStore struct {
	service  *storage.Service
	mountKey string
}

func WithIdentityPhotoStorageService(storageService *storage.Service, mountKey string) ServiceOption {
	return func(s *Service) {
		if storageService == nil {
			return
		}
		key := strings.TrimSpace(mountKey)
		if key == "" {
			key = storage.DefaultMountKey
		}
		s.photoStore = &storageIdentityPhotoStore{
			service:  storageService,
			mountKey: key,
		}
	}
}

func (s *storageIdentityPhotoStore) Upload(ctx context.Context, key string, content []byte, contentType string) error {
	_, _, err := s.service.Put(ctx, s.mountKey, key, content, contentType)
	return err
}

func (s *storageIdentityPhotoStore) PresignGetURL(ctx context.Context, key string) (string, error) {
	return s.service.GetDownloadURLByMountKey(ctx, s.mountKey, key)
}
