package resource

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/storage"
)

const maxResourceUploadSize = 10 * 1024 * 1024

type objectStore interface {
	Put(ctx context.Context, mountKey, objectKey string, content []byte, contentType string) (*storage.Mount, *storage.StoredObject, error)
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
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Filename) == "" {
		return nil, errors.New("title and filename are required")
	}
	req.Visibility = normalizeVisibility(req.Visibility)
	objectKey := fmt.Sprintf("resources/%s/%d-%s", ownerUserID, time.Now().UnixNano(), sanitizeFilename(req.Filename))
	mount, stored, err := s.storage.Put(ctx, req.MountKey, objectKey, content, detectedType)
	if err != nil {
		return nil, err
	}
	item, err := s.repo.CreateResource(ctx, ownerUserID, req, mount.ID, stored)
	if err != nil {
		if cleanupErr := s.storage.Delete(ctx, mount.ID, stored.ObjectKey); cleanupErr != nil {
			return nil, errors.Join(err, fmt.Errorf("cleanup uploaded resource object: %w", cleanupErr))
		}
		return nil, err
	}
	return item, nil
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
		return nil, errors.New("title is required")
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
		return nil, "", errors.New("dataBase64 is required")
	}
	if idx := strings.Index(raw, ","); strings.HasPrefix(raw, "data:") && idx > 0 {
		raw = raw[idx+1:]
	}
	content, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, "", errors.New("invalid base64 payload")
	}
	if len(content) == 0 || len(content) > maxResourceUploadSize {
		return nil, "", errors.New("resource payload size is invalid")
	}
	detectedType := http.DetectContentType(content)
	if strings.TrimSpace(contentType) != "" && contentType != detectedType {
		return nil, "", errors.New("contentType does not match payload")
	}
	return content, detectedType, nil
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
