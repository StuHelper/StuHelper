package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/ctxutil"
)

type Service struct {
	repo     *Repository
	registry *Registry
	cfg      config.ObjectStorageConfig
}

const storageCleanupTimeout = 5 * time.Second

func NewService(repo *Repository, cfg config.ObjectStorageConfig) *Service {
	if repo == nil {
		panic("storage.NewService: repo must not be nil")
	}
	return &Service{
		repo:     repo,
		registry: NewRegistry(cfg),
		cfg:      cfg,
	}
}

func (s *Service) EnsureDefaultMount(ctx context.Context) error {
	return s.repo.UpsertRuntimeDefaultMount(ctx, s.cfg)
}

func (s *Service) ListMounts(ctx context.Context) ([]Mount, error) {
	items, err := s.repo.ListMounts(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		driver, err := s.registry.Get(items[i].Driver)
		if err == nil {
			items[i].Capabilities = driver.Capabilities()
		}
	}
	return items, nil
}

func (s *Service) CreateMount(ctx context.Context, req CreateMountRequest) (*Mount, error) {
	req = normalizeCreateMountRequest(req)
	if err := validateCreateMountRequest(req); err != nil {
		return nil, err
	}
	driver, err := s.registry.Get(req.Driver)
	if err != nil {
		return nil, err
	}
	mount, err := s.repo.CreateMount(ctx, req)
	if err != nil {
		return nil, err
	}
	mount.Capabilities = driver.Capabilities()
	return mount, nil
}

func (s *Service) CheckMountHealth(ctx context.Context, mountID int64) (*Mount, error) {
	mount, err := s.repo.GetMountByID(ctx, mountID)
	if err != nil {
		return nil, err
	}
	item, healthErr, err := s.probeMountHealth(ctx, mount)
	if err != nil {
		return nil, err
	}
	if healthErr != nil {
		return item, nil
	}
	return item, nil
}

func (s *Service) ValidateMountByKey(ctx context.Context, mountKey string) (*Mount, error) {
	mount, err := s.repo.GetMountByKey(ctx, normalizeMountKey(mountKey))
	if err != nil {
		return nil, err
	}
	if !mount.Enabled {
		return nil, ErrMountDisabled
	}
	item, healthErr, err := s.probeMountHealth(ctx, mount)
	if err != nil {
		return nil, err
	}
	if healthErr != nil {
		return item, healthErr
	}
	return item, nil
}

func (s *Service) Put(ctx context.Context, mountKey, objectKey string, content []byte, contentType string) (*Mount, *StoredObject, error) {
	mount, driver, err := s.getMountDriver(ctx, mountKey)
	if err != nil {
		return nil, nil, err
	}
	stored, err := driver.Put(ctx, *mount, objectKey, content, contentType)
	if err != nil {
		return nil, nil, err
	}
	if err := validateStoredObject(stored); err != nil {
		cleanupCtx, cancel := storageCleanupContext(ctx)
		cleanupErr := driver.Delete(cleanupCtx, *mount, cleanupStoredObjectKey(objectKey, stored))
		cancel()
		if cleanupErr != nil {
			return nil, nil, errors.Join(err, fmt.Errorf("cleanup invalid stored object: %w", cleanupErr))
		}
		return nil, nil, err
	}
	return mount, stored, nil
}

func storageCleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctxutil.DetachedTimeout(ctx, storageCleanupTimeout)
}

func validateStoredObject(stored *StoredObject) error {
	if stored == nil {
		return ErrStoredObjectMissing
	}
	if strings.TrimSpace(stored.ObjectKey) == "" {
		return fmt.Errorf("%w: object key is required", ErrInvalidStoredObject)
	}
	if strings.TrimSpace(stored.ContentType) == "" {
		return fmt.Errorf("%w: content type is required", ErrInvalidStoredObject)
	}
	if stored.SizeBytes < 0 {
		return fmt.Errorf("%w: sizeBytes must not be negative", ErrInvalidStoredObject)
	}
	return nil
}

func cleanupStoredObjectKey(requestedKey string, stored *StoredObject) string {
	if stored != nil && strings.TrimSpace(stored.ObjectKey) != "" {
		return stored.ObjectKey
	}
	return requestedKey
}

func (s *Service) Delete(ctx context.Context, mountID int64, objectKey string) error {
	mount, err := s.repo.GetMountByID(ctx, mountID)
	if err != nil {
		return err
	}
	driver, err := s.registry.Get(mount.Driver)
	if err != nil {
		return err
	}
	return driver.Delete(ctx, *mount, objectKey)
}

func (s *Service) GetDownloadURL(ctx context.Context, mountID int64, objectKey string) (string, error) {
	mount, err := s.repo.GetMountByID(ctx, mountID)
	if err != nil {
		return "", err
	}
	driver, err := s.registry.Get(mount.Driver)
	if err != nil {
		return "", err
	}
	return driver.GetDownloadURL(ctx, *mount, objectKey)
}

func (s *Service) GetDownloadURLByMountKey(ctx context.Context, mountKey, objectKey string) (string, error) {
	mount, driver, err := s.getMountDriver(ctx, mountKey)
	if err != nil {
		return "", err
	}
	return driver.GetDownloadURL(ctx, *mount, objectKey)
}

func (s *Service) getMountDriver(ctx context.Context, mountKey string) (*Mount, Driver, error) {
	mount, err := s.repo.GetMountByKey(ctx, normalizeMountKey(mountKey))
	if err != nil {
		return nil, nil, err
	}
	if !mount.Enabled {
		return nil, nil, ErrMountDisabled
	}
	driver, err := s.registry.Get(mount.Driver)
	if err != nil {
		return nil, nil, err
	}
	return mount, driver, nil
}

func (s *Service) probeMountHealth(ctx context.Context, mount *Mount) (*Mount, error, error) {
	driver, err := s.registry.Get(mount.Driver)
	if err != nil {
		return nil, nil, err
	}
	status := "healthy"
	var (
		message   *string
		healthErr error
	)
	if err := driver.HealthCheck(ctx, *mount); err != nil {
		status = "unhealthy"
		value := err.Error()
		message = &value
		healthErr = err
	}
	if err := s.repo.UpdateMountHealth(ctx, mount.ID, status, message); err != nil {
		return nil, nil, err
	}
	item, err := s.repo.GetMountByID(ctx, mount.ID)
	if err != nil {
		return nil, nil, err
	}
	return item, healthErr, nil
}

func normalizeCreateMountRequest(req CreateMountRequest) CreateMountRequest {
	req.Key = strings.TrimSpace(req.Key)
	req.Name = strings.TrimSpace(req.Name)
	req.Driver = strings.TrimSpace(req.Driver)
	req.BasePath = strings.Trim(strings.TrimSpace(req.BasePath), "/")
	if req.Bucket != nil {
		bucket := strings.TrimSpace(*req.Bucket)
		if bucket == "" {
			req.Bucket = nil
		} else {
			req.Bucket = &bucket
		}
	}
	return req
}

func validateCreateMountRequest(req CreateMountRequest) error {
	if req.Key == "" {
		return fmt.Errorf("%w: key is required", ErrInvalidMountConfig)
	}
	if req.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidMountConfig)
	}
	if req.Driver == "" {
		return fmt.Errorf("%w: driver is required", ErrInvalidMountConfig)
	}
	return nil
}

func normalizeMountKey(mountKey string) string {
	key := strings.TrimSpace(mountKey)
	if key == "" {
		return DefaultMountKey
	}
	return key
}

func timeDurationSeconds(value int) time.Duration {
	return time.Duration(value) * time.Second
}
