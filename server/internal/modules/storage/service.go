package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/ctxutil"
)

type Service struct {
	repo     *Repository
	registry *Registry
	cfg      config.ObjectStorageConfig
}

const storageCleanupTimeout = 5 * time.Second

type mountHealthProbeResult struct {
	mount     *Mount
	healthErr error
}

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
	if err := validateMountID(mountID); err != nil {
		return nil, err
	}
	mount, err := s.repo.GetMountByID(ctx, mountID)
	if err != nil {
		return nil, err
	}
	result, err := s.probeMountHealth(ctx, mount)
	if err != nil {
		return nil, err
	}
	return result.mount, nil
}

func (s *Service) ValidateMountByKey(ctx context.Context, mountKey string) (*Mount, error) {
	mount, err := s.repo.GetMountByKey(ctx, normalizeMountKey(mountKey))
	if err != nil {
		return nil, err
	}
	if !mount.Enabled {
		return nil, ErrMountDisabled
	}
	result, err := s.probeMountHealth(ctx, mount)
	if err != nil {
		return nil, err
	}
	if result.healthErr != nil {
		return result.mount, result.healthErr
	}
	return result.mount, nil
}

func (s *Service) Put(ctx context.Context, mountKey, objectKey string, content []byte, contentType string) (*Mount, *StoredObject, error) {
	objectKey, err := validateObjectKey(objectKey)
	if err != nil {
		return nil, nil, err
	}
	mount, driver, err := s.getMountDriver(ctx, mountKey)
	if err != nil {
		return nil, nil, err
	}
	stored, err := driver.Put(ctx, *mount, objectKey, content, contentType)
	if err != nil {
		return nil, nil, err
	}
	normalizedStored, err := normalizeStoredObject(stored)
	if err != nil {
		cleanupCtx, cancel := storageCleanupContext(ctx)
		cleanupErr := driver.Delete(cleanupCtx, *mount, cleanupStoredObjectKey(objectKey, stored))
		cancel()
		if cleanupErr != nil {
			return nil, nil, errors.Join(err, fmt.Errorf("cleanup invalid stored object: %w", cleanupErr))
		}
		return nil, nil, err
	}
	return mount, normalizedStored, nil
}

func storageCleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctxutil.DetachedTimeout(ctx, storageCleanupTimeout)
}

func normalizeStoredObject(stored *StoredObject) (*StoredObject, error) {
	if stored == nil {
		return nil, ErrStoredObjectMissing
	}
	objectKey := strings.TrimSpace(stored.ObjectKey)
	if objectKey == "" {
		return nil, fmt.Errorf("%w: object key is required", ErrInvalidStoredObject)
	}
	contentType := strings.TrimSpace(stored.ContentType)
	if contentType == "" {
		return nil, fmt.Errorf("%w: content type is required", ErrInvalidStoredObject)
	}
	if stored.SizeBytes < 0 {
		return nil, fmt.Errorf("%w: sizeBytes must not be negative", ErrInvalidStoredObject)
	}
	return &StoredObject{
		ObjectKey:   objectKey,
		SizeBytes:   stored.SizeBytes,
		ContentType: contentType,
	}, nil
}

func cleanupStoredObjectKey(requestedKey string, stored *StoredObject) string {
	if stored != nil {
		if objectKey := strings.TrimSpace(stored.ObjectKey); objectKey != "" {
			return objectKey
		}
	}
	return requestedKey
}

func (s *Service) Delete(ctx context.Context, mountID int64, objectKey string) error {
	if err := validateMountID(mountID); err != nil {
		return err
	}
	objectKey, err := validateObjectKey(objectKey)
	if err != nil {
		return err
	}
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

func (s *Service) DeleteByMountKey(ctx context.Context, mountKey, objectKey string) error {
	objectKey, err := validateObjectKey(objectKey)
	if err != nil {
		return err
	}
	mount, driver, err := s.getMountDriver(ctx, mountKey)
	if err != nil {
		return err
	}
	return driver.Delete(ctx, *mount, objectKey)
}

func (s *Service) GetDownloadURL(ctx context.Context, mountID int64, objectKey string) (string, error) {
	if err := validateMountID(mountID); err != nil {
		return "", err
	}
	objectKey, err := validateObjectKey(objectKey)
	if err != nil {
		return "", err
	}
	mount, err := s.repo.GetMountByID(ctx, mountID)
	if err != nil {
		return "", err
	}
	if !mount.Enabled {
		return "", ErrMountDisabled
	}
	driver, err := s.registry.Get(mount.Driver)
	if err != nil {
		return "", err
	}
	if err := ensureObjectExists(ctx, driver, mount, objectKey); err != nil {
		return "", err
	}
	return driver.GetDownloadURL(ctx, *mount, objectKey)
}

func (s *Service) GetDownloadURLByMountKey(ctx context.Context, mountKey, objectKey string) (string, error) {
	objectKey, err := validateObjectKey(objectKey)
	if err != nil {
		return "", err
	}
	mount, driver, err := s.getMountDriver(ctx, mountKey)
	if err != nil {
		return "", err
	}
	if err := ensureObjectExists(ctx, driver, mount, objectKey); err != nil {
		return "", err
	}
	return driver.GetDownloadURL(ctx, *mount, objectKey)
}

func ensureObjectExists(ctx context.Context, driver Driver, mount *Mount, objectKey string) error {
	stored, err := driver.Stat(ctx, *mount, objectKey)
	if err != nil {
		return err
	}
	if stored == nil {
		return ErrStoredObjectMissing
	}
	return nil
}

func validateMountID(mountID int64) error {
	if mountID <= 0 {
		return ErrInvalidMountID
	}
	return nil
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

func (s *Service) probeMountHealth(ctx context.Context, mount *Mount) (mountHealthProbeResult, error) {
	driver, err := s.registry.Get(mount.Driver)
	if err != nil {
		return mountHealthProbeResult{}, err
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
		return mountHealthProbeResult{}, err
	}
	item, err := s.repo.GetMountByID(ctx, mount.ID)
	if err != nil {
		return mountHealthProbeResult{}, err
	}
	return mountHealthProbeResult{mount: item, healthErr: healthErr}, nil
}

func normalizeCreateMountRequest(req CreateMountRequest) CreateMountRequest {
	req.Key = strings.TrimSpace(req.Key)
	req.Name = strings.TrimSpace(req.Name)
	req.Driver = strings.TrimSpace(req.Driver)
	req.BasePath = normalizeObjectKey(req.BasePath)
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
