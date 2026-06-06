package resource

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"
	"time"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/ctxutil"
)

const maxResourceUploadSize = 10 * 1024 * 1024
const resourceCleanupTimeout = 5 * time.Second

var (
	ErrResourceTitleRequired        = errors.New("title is required")
	ErrResourceFilenameRequired     = errors.New("filename is required")
	ErrResourcePayloadRequired      = errors.New("dataBase64 is required")
	ErrResourcePayloadInvalid       = errors.New("invalid base64 payload")
	ErrResourcePayloadSizeInvalid   = errors.New("resource payload size is invalid")
	ErrResourceContentTypeMismatch  = errors.New("contentType does not match payload")
	ErrResourceStorageMountNotFound = errors.New("resource storage mount not found")
	ErrResourceStorageMountDisabled = errors.New("resource storage mount disabled")
	ErrResourceStorageDriverMissing = errors.New("resource storage driver unavailable")
	ErrResourceStoredObjectMissing  = errors.New("resource storage returned no object metadata")
	ErrResourceStoredObjectInvalid  = errors.New("resource storage returned invalid object metadata")
)

type StoredObject struct {
	ObjectKey   string
	SizeBytes   int64
	ContentType string
}

type objectStore interface {
	Put(ctx context.Context, mountKey, objectKey string, content []byte, contentType string) (int64, *StoredObject, error)
	Delete(ctx context.Context, mountID int64, objectKey string) error
	GetDownloadURL(ctx context.Context, mountID int64, objectKey string) (string, error)
}

type Service struct {
	repo    *Repository
	storage objectStore
}

func NewService(repo *Repository, store objectStore) *Service {
	if repo == nil {
		panic("resource.NewService: repo must not be nil")
	}
	if store == nil {
		panic("resource.NewService: storage must not be nil")
	}
	return &Service{repo: repo, storage: store}
}

func (s *Service) CreateResource(ctx context.Context, ownerUserID string, req CreateRequest) (*Item, error) {
	content, detectedType, err := decodePayload(req.ContentType, req.DataBase64)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Title) == "" {
		return nil, ErrResourceTitleRequired
	}
	if strings.TrimSpace(req.Filename) == "" {
		return nil, ErrResourceFilenameRequired
	}
	req.Visibility = normalizeVisibility(req.Visibility)
	objectKey := fmt.Sprintf("resources/%s/%d-%s", ownerUserID, time.Now().UnixNano(), sanitizeFilename(req.Filename))
	mountID, stored, err := s.storage.Put(ctx, req.MountKey, objectKey, content, detectedType)
	if err != nil {
		return nil, err
	}
	if err := validateResourceStoredObject(stored); err != nil {
		cleanupCtx, cancel := resourceCleanupContext(ctx)
		cleanupErr := s.storage.Delete(cleanupCtx, mountID, cleanupResourceObjectKey(objectKey, stored))
		cancel()
		if cleanupErr != nil {
			return nil, errors.Join(err, fmt.Errorf("cleanup invalid resource object: %w", cleanupErr))
		}
		return nil, err
	}
	item, err := s.repo.CreateResource(ctx, ownerUserID, req, mountID, stored)
	if err != nil {
		cleanupCtx, cancel := resourceCleanupContext(ctx)
		cleanupErr := s.storage.Delete(cleanupCtx, mountID, stored.ObjectKey)
		cancel()
		if cleanupErr != nil {
			return nil, errors.Join(err, fmt.Errorf("cleanup uploaded resource object: %w", cleanupErr))
		}
		return nil, err
	}
	return item, nil
}

func resourceCleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctxutil.DetachedTimeout(ctx, resourceCleanupTimeout)
}

func validateResourceStoredObject(stored *StoredObject) error {
	if stored == nil {
		return ErrResourceStoredObjectMissing
	}
	if strings.TrimSpace(stored.ObjectKey) == "" {
		return fmt.Errorf("%w: object key is required", ErrResourceStoredObjectInvalid)
	}
	if strings.TrimSpace(stored.ContentType) == "" {
		return fmt.Errorf("%w: content type is required", ErrResourceStoredObjectInvalid)
	}
	if stored.SizeBytes <= 0 {
		return fmt.Errorf("%w: sizeBytes must be positive", ErrResourceStoredObjectInvalid)
	}
	return nil
}

func cleanupResourceObjectKey(requestedKey string, stored *StoredObject) string {
	if stored != nil && strings.TrimSpace(stored.ObjectKey) != "" {
		return stored.ObjectKey
	}
	return requestedKey
}

func (s *Service) ListResources(ctx context.Context, filters ListFilters) ([]Item, int, error) {
	return s.repo.ListResources(ctx, filters)
}

func (s *Service) GetResource(ctx context.Context, resourceID int64, viewerUserID string) (*Item, error) {
	item, err := s.repo.GetResourceByID(ctx, resourceID)
	if err != nil {
		return nil, err
	}
	if item.Visibility == "private" && item.OwnerUserID != viewerUserID {
		return nil, ErrResourceNotFound
	}
	return item, nil
}

func (s *Service) UpdateResource(ctx context.Context, resourceID int64, ownerUserID string, req UpdateRequest) (*Item, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, ErrResourceTitleRequired
	}
	req.Visibility = normalizeVisibility(req.Visibility)
	return s.repo.UpdateResource(ctx, resourceID, ownerUserID, req)
}

func (s *Service) DeleteResource(ctx context.Context, resourceID int64, ownerUserID string) error {
	item, err := s.repo.GetResourceByID(ctx, resourceID)
	if err != nil {
		return err
	}
	if item.OwnerUserID != ownerUserID {
		return ErrResourceForbidden
	}
	return s.repo.DeleteResource(ctx, resourceID, ownerUserID)
}

func (s *Service) GetDownloadURL(ctx context.Context, resourceID int64, viewerUserID string) (string, error) {
	item, err := s.GetResource(ctx, resourceID, viewerUserID)
	if err != nil {
		return "", err
	}
	return s.storage.GetDownloadURL(ctx, item.LatestVersion.MountID, item.LatestVersion.ObjectKey)
}

func decodePayload(contentType, dataBase64 string) ([]byte, string, error) {
	raw := strings.TrimSpace(dataBase64)
	if raw == "" {
		return nil, "", ErrResourcePayloadRequired
	}
	if idx := strings.Index(raw, ","); strings.HasPrefix(raw, "data:") && idx > 0 {
		raw = raw[idx+1:]
	}
	content, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, "", ErrResourcePayloadInvalid
	}
	if len(content) == 0 || len(content) > maxResourceUploadSize {
		return nil, "", ErrResourcePayloadSizeInvalid
	}
	detectedType := http.DetectContentType(content)
	if provided := strings.TrimSpace(contentType); provided != "" {
		if !resourceMediaTypesMatch(provided, detectedType) {
			return nil, "", ErrResourceContentTypeMismatch
		}
	}
	return content, detectedType, nil
}

func resourceMediaTypesMatch(provided, detected string) bool {
	providedType, _, providedErr := mime.ParseMediaType(provided)
	detectedType, _, detectedErr := mime.ParseMediaType(detected)
	if providedErr != nil || detectedErr != nil {
		return provided == detected
	}
	return providedType == detectedType
}

func isResourceBadRequestError(err error) bool {
	return errors.Is(err, ErrResourceTitleRequired) ||
		errors.Is(err, ErrResourceFilenameRequired) ||
		errors.Is(err, ErrResourcePayloadRequired) ||
		errors.Is(err, ErrResourcePayloadInvalid) ||
		errors.Is(err, ErrResourcePayloadSizeInvalid) ||
		errors.Is(err, ErrResourceContentTypeMismatch)
}

func normalizeVisibility(value string) string {
	if strings.TrimSpace(value) == "private" {
		return "private"
	}
	return "public"
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "_")
	return replacer.Replace(strings.TrimSpace(name))
}
