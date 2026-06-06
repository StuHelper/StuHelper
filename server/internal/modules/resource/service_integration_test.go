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
	mountID     int64
	downloadURL string
	deletedKeys []string
	putErr      error
	putMissing  bool
	putObject   *StoredObject
	deleteErr   error
	downloadErr error
	onPut       func()
}

func (s *fakeObjectStore) Put(_ context.Context, _ string, objectKey string, content []byte, contentType string) (int64, *StoredObject, error) {
	if s.onPut != nil {
		s.onPut()
	}
	if s.putErr != nil {
		return 0, nil, s.putErr
	}
	if s.putMissing {
		return s.mountID, nil, nil
	}
	if s.putObject != nil {
		return s.mountID, s.putObject, nil
	}
	return s.mountID, &StoredObject{
		ObjectKey:   objectKey,
		SizeBytes:   int64(len(content)),
		ContentType: contentType,
	}, nil
}

func (s *fakeObjectStore) Delete(ctx context.Context, _ int64, objectKey string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	if err := ctx.Err(); err != nil {
		return err
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

func TestCreateResource_AcceptsPlainTextWithoutCharsetParameter(t *testing.T) {
	ctx, _, _, svc, _ := setupResourceService(t)

	created, err := svc.CreateResource(ctx, "oidc-user-1", CreateRequest{
		Title:       "Plain Text",
		Visibility:  "public",
		Filename:    "plain.txt",
		ContentType: "text/plain",
		DataBase64:  base64.StdEncoding.EncodeToString([]byte("plain text body")),
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, "Plain Text", created.Title)
}

func TestCreateResource_RejectsMissingStorageMetadata(t *testing.T) {
	ctx, _, repo, svc, store := setupResourceService(t)
	store.putMissing = true

	_, err := svc.CreateResource(ctx, "oidc-user-1", CreateRequest{
		Title:       "Broken Upload",
		Visibility:  "public",
		Filename:    "broken.txt",
		ContentType: "text/plain",
		DataBase64:  base64.StdEncoding.EncodeToString([]byte("missing metadata")),
	})
	require.ErrorIs(t, err, ErrResourceStoredObjectMissing)
	assert.Len(t, store.deletedKeys, 1)
	assert.Contains(t, store.deletedKeys[0], "broken.txt")

	var count int
	require.NoError(t, repo.db.QueryRow(ctx, `SELECT COUNT(*) FROM resource_items`).Scan(&count))
	assert.Zero(t, count)
}

func TestCreateResource_RejectsInvalidStorageMetadata(t *testing.T) {
	for _, tc := range []struct {
		name       string
		stored     *StoredObject
		cleanupKey string
	}{
		{
			name:       "blank object key",
			stored:     &StoredObject{ObjectKey: " ", SizeBytes: 16, ContentType: "text/plain"},
			cleanupKey: "",
		},
		{
			name:       "blank content type",
			stored:     &StoredObject{ObjectKey: "resources/oidc-user-1/actual-upload-key.txt", SizeBytes: 16, ContentType: " "},
			cleanupKey: "resources/oidc-user-1/actual-upload-key.txt",
		},
		{
			name:       "zero size",
			stored:     &StoredObject{ObjectKey: "resources/oidc-user-1/actual-upload-key.txt", SizeBytes: 0, ContentType: "text/plain"},
			cleanupKey: "resources/oidc-user-1/actual-upload-key.txt",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _, repo, svc, store := setupResourceService(t)
			store.putObject = tc.stored

			_, err := svc.CreateResource(ctx, "oidc-user-1", CreateRequest{
				Title:       "Broken Upload",
				Visibility:  "public",
				Filename:    "broken.txt",
				ContentType: "text/plain",
				DataBase64:  base64.StdEncoding.EncodeToString([]byte("missing metadata")),
			})
			require.ErrorIs(t, err, ErrResourceStoredObjectInvalid)
			assert.Len(t, store.deletedKeys, 1)
			if tc.cleanupKey != "" {
				assert.Equal(t, tc.cleanupKey, store.deletedKeys[0])
			} else {
				assert.Contains(t, store.deletedKeys[0], "broken.txt")
			}

			var count int
			require.NoError(t, repo.db.QueryRow(ctx, `SELECT COUNT(*) FROM resource_items`).Scan(&count))
			assert.Zero(t, count)
		})
	}
}

func TestCreateResource_CleanupSurvivesRequestCancellationAfterUpload(t *testing.T) {
	ctx, _, repo, svc, store := setupResourceService(t)
	ctx, cancel := context.WithCancel(ctx)
	store.onPut = cancel

	_, err := svc.CreateResource(ctx, "oidc-user-1", CreateRequest{
		Title:       "Cancelled Upload",
		Visibility:  "public",
		Filename:    "cancelled.txt",
		ContentType: "text/plain",
		DataBase64:  base64.StdEncoding.EncodeToString([]byte("uploaded but request cancelled")),
	})
	require.Error(t, err)
	require.Len(t, store.deletedKeys, 1)
	assert.Contains(t, store.deletedKeys[0], "cancelled.txt")

	var count int
	require.NoError(t, repo.db.QueryRow(context.Background(), `SELECT COUNT(*) FROM resource_items`).Scan(&count))
	assert.Zero(t, count)
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

	_, err = svc.UpdateResource(ctx, created.ID, "oidc-user-2", UpdateRequest{
		Title:      "Not Mine",
		Visibility: "public",
	})
	require.ErrorIs(t, err, ErrResourceForbidden)

	_, err = svc.UpdateResource(ctx, 999999, "oidc-user-1", UpdateRequest{
		Title:      "Missing",
		Visibility: "public",
	})
	require.ErrorIs(t, err, ErrResourceNotFound)

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

func TestDeleteResource_EnqueuesCleanupForAllVersions(t *testing.T) {
	ctx, mount, _, svc, store := setupResourceService(t)
	created := createSampleResource(t, ctx, svc)

	secondObjectKey := "resources/oidc-user-1/version-2.txt"
	_, err := svc.repo.db.Exec(ctx, `
		INSERT INTO resource_versions (
			resource_id, version_no, mount_id, object_key, filename, content_type, size_bytes
		) VALUES ($1, 2, $2, $3, 'version-2.txt', 'text/plain', 12)
	`, created.ID, mount.ID, secondObjectKey)
	require.NoError(t, err)

	err = svc.DeleteResource(ctx, created.ID, "oidc-user-1")
	require.NoError(t, err)
	assert.Empty(t, store.deletedKeys)

	err = svc.processCleanupBatch(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{created.LatestVersion.ObjectKey, secondObjectKey}, store.deletedKeys)
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

func TestStartBackgroundJobsRequiresStarter(t *testing.T) {
	_, _, _, svc, _ := setupResourceService(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.Panics(t, func() {
		svc.StartBackgroundJobs(ctx, nil)
	})
}

func setupResourceService(t *testing.T) (context.Context, *storage.Mount, *Repository, *Service, *fakeObjectStore) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()
	mount, err := storage.NewRepository(fixture.DB).GetMountByKey(ctx, "default-s3")
	require.NoError(t, err)

	store := &fakeObjectStore{
		mountID:     mount.ID,
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
