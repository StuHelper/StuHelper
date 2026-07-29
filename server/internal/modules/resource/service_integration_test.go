package resource

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/modules/storage"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

type fakeObjectStore struct {
	mountID     int64
	downloadURL string
	putKeys     []string
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
	s.putKeys = append(s.putKeys, objectKey)
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

	items, total, err = svc.ListResources(ctx, ListFilters{
		Query:    "Algorithms",
		Page:     0,
		PageSize: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, created.ID, items[0].ID)

	url, err := svc.GetDownloadURL(ctx, created.ID, "")
	require.NoError(t, err)
	assert.Equal(t, store.downloadURL, url)
}

func TestListResourcesQueryMatchesVisibleMetadata(t *testing.T) {
	ctx, _, _, svc, _ := setupResourceService(t)

	created, err := svc.CreateResource(ctx, "oidc-user-1", CreateRequest{
		Title:       "期末复习资料",
		Description: ptr("积分专题"),
		Category:    ptr("数学资料"),
		Visibility:  "public",
		Tags:        []string{"高数", "期末"},
		Filename:    "course-8-final-review-guide.md",
		ContentType: "text/plain",
		DataBase64:  base64.StdEncoding.EncodeToString([]byte("calculus final review")),
	})
	require.NoError(t, err)

	for _, tc := range []struct {
		name  string
		query string
	}{
		{name: "tag", query: "高数"},
		{name: "category", query: "数学资料"},
		{name: "filename", query: "course-8-final"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items, total, err := svc.ListResources(ctx, ListFilters{
				Query:    tc.query,
				Page:     1,
				PageSize: 20,
			})
			require.NoError(t, err)
			assert.Equal(t, 1, total)
			require.Len(t, items, 1)
			assert.Equal(t, created.ID, items[0].ID)

			owned, ownedTotal, err := svc.ListMyResources(ctx, "oidc-user-1", ListFilters{
				Query:    tc.query,
				Page:     1,
				PageSize: 20,
			})
			require.NoError(t, err)
			assert.Equal(t, 1, ownedTotal)
			require.Len(t, owned, 1)
			assert.Equal(t, created.ID, owned[0].ID)
		})
	}
}

func TestListMyResourcesIncludesPrivateAndExcludesOtherOwners(t *testing.T) {
	ctx, _, _, svc, _ := setupResourceService(t)

	publicOwned, err := svc.CreateResource(ctx, "oidc-user-1", CreateRequest{
		Title:       "Owned Public Resource",
		Visibility:  "public",
		Tags:        []string{"owned"},
		Bindings:    []Binding{{Type: "course", Value: "CS101"}},
		Filename:    "owned-public.txt",
		ContentType: "text/plain",
		DataBase64:  base64.StdEncoding.EncodeToString([]byte("owned public")),
	})
	require.NoError(t, err)
	privateOwned, err := svc.CreateResource(ctx, "oidc-user-1", CreateRequest{
		Title:       "Owned Private Resource",
		Visibility:  "private",
		Tags:        []string{"owned"},
		Bindings:    []Binding{{Type: "course", Value: "CS101"}},
		Filename:    "owned-private.txt",
		ContentType: "text/plain",
		DataBase64:  base64.StdEncoding.EncodeToString([]byte("owned private")),
	})
	require.NoError(t, err)
	otherOwned, err := svc.CreateResource(ctx, "oidc-user-2", CreateRequest{
		Title:       "Other User Resource",
		Visibility:  "public",
		Tags:        []string{"owned"},
		Bindings:    []Binding{{Type: "course", Value: "CS101"}},
		Filename:    "other-public.txt",
		ContentType: "text/plain",
		DataBase64:  base64.StdEncoding.EncodeToString([]byte("other public")),
	})
	require.NoError(t, err)

	items, total, err := svc.ListMyResources(ctx, " oidc-user-1 ", ListFilters{
		Query:        "Resource",
		Tag:          "owned",
		BindingType:  "course",
		BindingValue: "CS101",
		Page:         1,
		PageSize:     20,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.ElementsMatch(t, []int64{publicOwned.ID, privateOwned.ID}, resourceItemIDs(items))

	items, total, err = svc.ListMyResources(ctx, "oidc-user-1", ListFilters{
		Visibility: "private",
		Page:       1,
		PageSize:   20,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, privateOwned.ID, items[0].ID)

	items, total, err = svc.ListResources(ctx, ListFilters{
		Query:    "Resource",
		Page:     1,
		PageSize: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.ElementsMatch(t, []int64{publicOwned.ID, otherOwned.ID}, resourceItemIDs(items))

	_, _, err = svc.ListMyResources(ctx, "   ", ListFilters{Page: 1, PageSize: 20})
	require.ErrorIs(t, err, ErrResourceOwnerRequired)
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

func TestCreateResource_RejectsMissingOwnerBeforeUpload(t *testing.T) {
	ctx, _, repo, svc, store := setupResourceService(t)

	_, err := svc.CreateResource(ctx, "   ", CreateRequest{
		Title:       "Missing Owner",
		Visibility:  "public",
		Filename:    "missing-owner.txt",
		ContentType: "text/plain",
		DataBase64:  base64.StdEncoding.EncodeToString([]byte("owner is required")),
	})

	require.ErrorIs(t, err, ErrResourceOwnerRequired)
	assert.Empty(t, store.putKeys)
	assert.Empty(t, store.deletedKeys)

	var count int
	require.NoError(t, repo.db.QueryRow(ctx, `SELECT COUNT(*) FROM resource_items`).Scan(&count))
	assert.Zero(t, count)
}

func TestCreateResource_NormalizesTagsAndBindingsBeforePersist(t *testing.T) {
	ctx, _, _, svc, _ := setupResourceService(t)

	created, err := svc.CreateResource(ctx, "oidc-user-1", CreateRequest{
		Title:       "  Algorithms Notes  ",
		Description: ptr("  Midterm summary  "),
		Category:    ptr("  notes  "),
		Visibility:  "public",
		Tags:        []string{" algorithm ", "algorithm", "", " midterm "},
		Bindings: []Binding{
			{Type: " course ", Value: " CS101 "},
			{Type: "course", Value: "CS101"},
			{Type: "", Value: "ignored"},
			{Type: "term", Value: " 2026-SPRING "},
		},
		Filename:    "  algo notes.txt  ",
		ContentType: " text/plain; charset=utf-8 ",
		DataBase64:  base64.StdEncoding.EncodeToString([]byte("hello, normalized resource")),
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, "Algorithms Notes", created.Title)
	require.NotNil(t, created.Description)
	assert.Equal(t, "Midterm summary", *created.Description)
	require.NotNil(t, created.Category)
	assert.Equal(t, "notes", *created.Category)
	assert.Equal(t, []string{"algorithm", "midterm"}, created.Tags)
	assert.Equal(t, []Binding{
		{Type: "course", Value: "CS101"},
		{Type: "term", Value: "2026-SPRING"},
	}, created.Bindings)
	assert.Equal(t, "algo notes.txt", created.LatestVersion.Filename)
}

func TestCreateResource_NormalizesStoredObjectMetadataBeforePersist(t *testing.T) {
	ctx, _, _, svc, store := setupResourceService(t)
	store.putObject = &StoredObject{
		ObjectKey:   " resources/oidc-user-1/actual-upload-key.txt ",
		SizeBytes:   16,
		ContentType: " text/plain ",
	}

	created, err := svc.CreateResource(ctx, "oidc-user-1", CreateRequest{
		Title:       "Stored Metadata",
		Visibility:  "public",
		Filename:    "stored.txt",
		ContentType: "text/plain",
		DataBase64:  base64.StdEncoding.EncodeToString([]byte("stored metadata")),
	})

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, "resources/oidc-user-1/actual-upload-key.txt", created.LatestVersion.ObjectKey)
	assert.Equal(t, "text/plain", created.LatestVersion.ContentType)
	assert.Empty(t, store.deletedKeys)
}

func TestUpdateResource_NormalizesTagsAndBindingsBeforePersist(t *testing.T) {
	ctx, _, _, svc, _ := setupResourceService(t)
	created := createSampleResource(t, ctx, svc)

	updated, err := svc.UpdateResource(ctx, created.ID, " oidc-user-1 ", UpdateRequest{
		Title:       "  Algorithms Notes v2  ",
		Description: ptr("   "),
		Category:    ptr(" updated "),
		Visibility:  "private",
		Tags:        []string{" updated ", "updated", "", "resource"},
		Bindings: []Binding{
			{Type: " course ", Value: " CS101 "},
			{Type: "course", Value: "CS101"},
			{Type: "term", Value: " 2026-SPRING "},
			{Type: "term", Value: ""},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "Algorithms Notes v2", updated.Title)
	assert.Nil(t, updated.Description)
	require.NotNil(t, updated.Category)
	assert.Equal(t, "updated", *updated.Category)
	assert.Equal(t, []string{"resource", "updated"}, updated.Tags)
	assert.Equal(t, []Binding{
		{Type: "course", Value: "CS101"},
		{Type: "term", Value: "2026-SPRING"},
	}, updated.Bindings)
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

	_, err = svc.UpdateResource(ctx, created.ID, " ", UpdateRequest{
		Title:      "Missing Owner",
		Visibility: "public",
	})
	require.ErrorIs(t, err, ErrResourceOwnerRequired)

	_, err = svc.GetResource(ctx, created.ID, "oidc-user-2")
	require.ErrorIs(t, err, ErrResourceNotFound)

	err = svc.DeleteResource(ctx, created.ID, " ")
	require.ErrorIs(t, err, ErrResourceOwnerRequired)

	err = svc.DeleteResource(ctx, created.ID, " oidc-user-1 ")
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

func resourceItemIDs(items []Item) []int64 {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func ptr(value string) *string {
	return &value
}
