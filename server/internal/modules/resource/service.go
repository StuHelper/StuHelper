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
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
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
	ErrResourceOwnerRequired        = errors.New("owner user id is required")
	ErrResourceIDInvalid            = errors.New("resource id is invalid")
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
	ownerUserID, err := normalizeRequiredResourceOwner(ownerUserID)
	if err != nil {
		return nil, err
	}
	req = normalizeCreateResourceRequest(req)
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
	objectKey, err := newResourceObjectKey(ownerUserID, req.Filename)
	if err != nil {
		return nil, err
	}
	mountID, stored, err := s.storage.Put(ctx, req.MountKey, objectKey, content, detectedType)
	if err != nil {
		return nil, err
	}
	normalizedStored, err := normalizeResourceStoredObject(stored)
	if err != nil {
		cleanupCtx, cancel := resourceCleanupContext(ctx)
		cleanupErr := s.storage.Delete(cleanupCtx, mountID, cleanupResourceObjectKey(objectKey, stored))
		cancel()
		if cleanupErr != nil {
			return nil, errors.Join(err, fmt.Errorf("cleanup invalid resource object: %w", cleanupErr))
		}
		return nil, err
	}
	item, err := s.repo.CreateResource(ctx, ownerUserID, req, mountID, normalizedStored)
	if err != nil {
		cleanupCtx, cancel := resourceCleanupContext(ctx)
		cleanupErr := s.storage.Delete(cleanupCtx, mountID, normalizedStored.ObjectKey)
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

func normalizeResourceStoredObject(stored *StoredObject) (*StoredObject, error) {
	if stored == nil {
		return nil, ErrResourceStoredObjectMissing
	}
	objectKey := strings.TrimSpace(stored.ObjectKey)
	if objectKey == "" {
		return nil, fmt.Errorf("%w: object key is required", ErrResourceStoredObjectInvalid)
	}
	contentType := strings.TrimSpace(stored.ContentType)
	if contentType == "" {
		return nil, fmt.Errorf("%w: content type is required", ErrResourceStoredObjectInvalid)
	}
	if stored.SizeBytes <= 0 {
		return nil, fmt.Errorf("%w: sizeBytes must be positive", ErrResourceStoredObjectInvalid)
	}
	return &StoredObject{
		ObjectKey:   objectKey,
		SizeBytes:   stored.SizeBytes,
		ContentType: contentType,
	}, nil
}

func cleanupResourceObjectKey(requestedKey string, stored *StoredObject) string {
	if stored != nil {
		if objectKey := strings.TrimSpace(stored.ObjectKey); objectKey != "" {
			return objectKey
		}
	}
	return requestedKey
}

func (s *Service) ListResources(ctx context.Context, filters ListFilters) ([]Item, int, error) {
	filters = normalizeListFilters(filters)
	return s.repo.ListResources(ctx, filters)
}

func (s *Service) GetResource(ctx context.Context, resourceID int64, viewerUserID string) (*Item, error) {
	if err := validateResourceID(resourceID); err != nil {
		return nil, err
	}
	viewerUserID = strings.TrimSpace(viewerUserID)
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
	if err := validateResourceID(resourceID); err != nil {
		return nil, err
	}
	ownerUserID, err := normalizeRequiredResourceOwner(ownerUserID)
	if err != nil {
		return nil, err
	}
	req = normalizeUpdateResourceRequest(req)
	if strings.TrimSpace(req.Title) == "" {
		return nil, ErrResourceTitleRequired
	}
	req.Visibility = normalizeVisibility(req.Visibility)
	return s.repo.UpdateResource(ctx, resourceID, ownerUserID, req)
}

func (s *Service) DeleteResource(ctx context.Context, resourceID int64, ownerUserID string) error {
	if err := validateResourceID(resourceID); err != nil {
		return err
	}
	ownerUserID, err := normalizeRequiredResourceOwner(ownerUserID)
	if err != nil {
		return err
	}
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

func normalizeRequiredResourceOwner(ownerUserID string) (string, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return "", ErrResourceOwnerRequired
	}
	return ownerUserID, nil
}

func validateResourceID(resourceID int64) error {
	if resourceID <= 0 {
		return ErrResourceIDInvalid
	}
	return nil
}

func isResourceBadRequestError(err error) bool {
	return errors.Is(err, ErrResourceTitleRequired) ||
		errors.Is(err, ErrResourceOwnerRequired) ||
		errors.Is(err, ErrResourceIDInvalid) ||
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

func normalizeCreateResourceRequest(req CreateRequest) CreateRequest {
	req.Title = strings.TrimSpace(req.Title)
	req.Description = normalizeOptionalResourceString(req.Description)
	req.Category = normalizeOptionalResourceString(req.Category)
	req.Filename = strings.TrimSpace(req.Filename)
	req.ContentType = strings.TrimSpace(req.ContentType)
	req.MountKey = strings.TrimSpace(req.MountKey)
	req.Tags = normalizeResourceTags(req.Tags)
	req.Bindings = normalizeResourceBindings(req.Bindings)
	return req
}

func normalizeUpdateResourceRequest(req UpdateRequest) UpdateRequest {
	req.Title = strings.TrimSpace(req.Title)
	req.Description = normalizeOptionalResourceString(req.Description)
	req.Category = normalizeOptionalResourceString(req.Category)
	req.Tags = normalizeResourceTags(req.Tags)
	req.Bindings = normalizeResourceBindings(req.Bindings)
	return req
}

func normalizeListFilters(filters ListFilters) ListFilters {
	filters.Query = strings.TrimSpace(filters.Query)
	filters.Tag = strings.TrimSpace(filters.Tag)
	filters.BindingType = strings.TrimSpace(filters.BindingType)
	filters.BindingValue = strings.TrimSpace(filters.BindingValue)
	return filters
}

func normalizeOptionalResourceString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeResourceTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tags))
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
	}
	return normalized
}

func normalizeResourceBindings(bindings []Binding) []Binding {
	if len(bindings) == 0 {
		return nil
	}
	seen := make(map[Binding]struct{}, len(bindings))
	normalized := make([]Binding, 0, len(bindings))
	for _, binding := range bindings {
		item := Binding{
			Type:  strings.TrimSpace(binding.Type),
			Value: strings.TrimSpace(binding.Value),
		}
		if item.Type == "" || item.Value == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		normalized = append(normalized, item)
	}
	return normalized
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "_")
	return replacer.Replace(strings.TrimSpace(name))
}

func newResourceObjectKey(ownerUserID, filename string) (string, error) {
	objectID, err := id.New()
	if err != nil {
		return "", fmt.Errorf("generate resource object key: %w", err)
	}
	return fmt.Sprintf("resources/%s/%s-%s", sanitizeObjectKeySegment(ownerUserID), objectID, sanitizeFilename(filename)), nil
}

func sanitizeObjectKeySegment(value string) string {
	value = sanitizeFilename(value)
	if value == "" || value == "." || value == ".." {
		return "unknown"
	}
	return value
}
