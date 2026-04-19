package resource

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/storage"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

type fakeObjectStore struct {
	mount       *storage.Mount
	downloadURL string
	deletedKeys []string
	putErr      error
	deleteErr   error
	downloadErr error
}

func (s *fakeObjectStore) Put(_ context.Context, _ string, objectKey string, content []byte, contentType string) (*storage.Mount, *storage.StoredObject, error) {
	if s.putErr != nil {
		return nil, nil, s.putErr
	}
	return s.mount, &storage.StoredObject{
		ObjectKey:   objectKey,
		SizeBytes:   int64(len(content)),
		ContentType: contentType,
	}, nil
}

func (s *fakeObjectStore) Delete(_ context.Context, _ int64, objectKey string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedKeys = append(s.deletedKeys, objectKey)
	return nil
}

func (s *fakeObjectStore) GetDownloadURL(_ context.Context, _ int64, _ string) (string, error) {
	if s.downloadErr != nil {
		return "", s.downloadErr
	}
	return s.downloadURL, nil
}

func TestCreateAndQueryResource(t *testing.T) {
	ctx, mount, _, svc, store := setupResourceService(t)

	created := createSampleResource(t, ctx, svc)
	assert.Equal(t, mount.ID, created.LatestVersion.MountID)

	items, total, err := svc.ListResources(ctx, ListFilters{
		Query:        "Algorithms",
		Tag:          "algorithm",
		BindingType:  "course",
		BindingValue: "CS101",
		Page:         1,
		PageSize:     20,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, created.ID, items[0].ID)

	url, err := svc.GetDownloadURL(ctx, created.ID, "")
	require.NoError(t, err)
	assert.Equal(t, store.downloadURL, url)
}

func TestUpdateAndDeletePrivateResource(t *testing.T) {
	ctx, _, repo, svc, store := setupResourceService(t)
	created := createSampleResource(t, ctx, svc)

	updated, err := svc.UpdateResource(ctx, created.ID, "oidc-user-1", UpdateRequest{
		Title:       "Algorithms Notes v2",
		Description: ptr("Private version"),
		Category:    ptr("notes"),
		Visibility:  "private",
		Tags:        []string{"algorithm", "private"},
		Bindings:    []Binding{{Type: "course", Value: "CS101"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "private", updated.Visibility)
	assert.Equal(t, []string{"algorithm", "private"}, updated.Tags)

	ownerView, err := svc.GetResource(ctx, created.ID, "oidc-user-1")
	require.NoError(t, err)
	assert.Equal(t, updated.ID, ownerView.ID)

	_, err = svc.GetResource(ctx, created.ID, "oidc-user-2")
	require.ErrorIs(t, err, ErrResourceNotFound)

	err = svc.DeleteResource(ctx, created.ID, "oidc-user-1")
	require.NoError(t, err)
	assert.Empty(t, store.deletedKeys)

	_, err = repo.GetResourceByID(ctx, created.ID)
	require.ErrorIs(t, err, ErrResourceNotFound)

	err = svc.processCleanupBatch(ctx)
	require.NoError(t, err)
	assert.Len(t, store.deletedKeys, 1)
}

func TestDeleteResource_EnqueuesRetryWhenStorageCleanupFails(t *testing.T) {
	ctx, _, repo, svc, store := setupResourceService(t)
	created := createSampleResource(t, ctx, svc)
	store.deleteErr = errors.New("storage unavailable")

	err := svc.DeleteResource(ctx, created.ID, "oidc-user-1")
	require.NoError(t, err)

	err = svc.processCleanupBatch(ctx)
	require.NoError(t, err)

	var (
		status       string
		attemptCount int
		lastError    string
	)
	require.NoError(t, repo.db.QueryRow(ctx, `
		SELECT status, attempt_count, last_error
		FROM domain_event_outbox
		WHERE stream = $1
	`, resourceCleanupOutboxStream).Scan(&status, &attemptCount, &lastError))
	assert.Equal(t, "failed", status)
	assert.Equal(t, 1, attemptCount)
	assert.Contains(t, lastError, "storage unavailable")
}

func setupResourceService(t *testing.T) (context.Context, *storage.Mount, *Repository, *Service, *fakeObjectStore) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	mount, err := storage.NewRepository(fixture.DB).GetMountByKey(ctx, "default-s3")
	require.NoError(t, err)

	store := &fakeObjectStore{
		mount:       mount,
		downloadURL: "https://storage.example.test/download/resource-1",
	}
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, store)
	return ctx, mount, repo, svc, store
}

func createSampleResource(t *testing.T, ctx context.Context, svc *Service) *Item {
	created, err := svc.CreateResource(ctx, "oidc-user-1", CreateRequest{
		Title:       "Algorithms Notes",
		Description: ptr("Midterm summary"),
		Category:    ptr("notes"),
		Visibility:  "public",
		Tags:        []string{"algorithm", "midterm"},
		Bindings: []Binding{
			{Type: "course", Value: "CS101"},
			{Type: "term", Value: "2026-SPRING"},
		},
		Filename:    "algo notes.txt",
		ContentType: "text/plain; charset=utf-8",
		DataBase64:  base64.StdEncoding.EncodeToString([]byte("hello, resource")),
	})
	require.NoError(t, err)
	assert.Equal(t, "Algorithms Notes", created.Title)
	assert.Equal(t, "public", created.Visibility)
	assert.Equal(t, []string{"algorithm", "midterm"}, created.Tags)
	return created
}

func ptr(value string) *string {
	return &value
}
