package user

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

type fakeIdentityPhotoStore struct {
	uploadErr    error
	presignErr   error
	presignURL   string
	uploadedKey  string
	uploadedType string
	uploadedData []byte
	presignedKey string
}

func (f *fakeIdentityPhotoStore) Upload(_ context.Context, key string, content []byte, contentType string) error {
	f.uploadedKey = key
	f.uploadedType = contentType
	f.uploadedData = append([]byte(nil), content...)
	return f.uploadErr
}

func (f *fakeIdentityPhotoStore) PresignGetURL(_ context.Context, key string) (string, error) {
	f.presignedKey = key
	if f.presignErr != nil {
		return "", f.presignErr
	}
	return f.presignURL, nil
}

func TestUploadIdentityPhoto_UsesConfiguredStore(t *testing.T) {
	store := &fakeIdentityPhotoStore{}
	svc, err := NewService(
		&mockRepo{},
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithIdentityPhotoStore(store),
	)
	require.NoError(t, err)

	key, err := svc.UploadIdentityPhoto(context.Background(), 42, UploadIdentityPhotoRequest{
		Slot:        IdentityPhotoSlotFront,
		Filename:    "identity.png",
		ContentType: "image/png",
		DataBase64:  base64.StdEncoding.EncodeToString(validPNGBytes(t)),
	})
	require.NoError(t, err)

	assert.Equal(t, key, store.uploadedKey)
	assert.Equal(t, "image/png", store.uploadedType)
	assert.NotEmpty(t, store.uploadedData)
	assert.True(t, strings.HasPrefix(key, "identities/42/"))
	assert.True(t, strings.HasSuffix(key, "-front.png"))
}

func TestResolveIdentityReviewItemAssets_PresignsStoredKeys(t *testing.T) {
	store := &fakeIdentityPhotoStore{presignURL: "https://storage.example.test/identity/front.png"}
	svc, err := NewService(
		&mockRepo{},
		[]byte("test-hmac-key-at-least-32-chars!"),
		&fakeEncryptor{},
		WithIdentityPhotoStore(store),
	)
	require.NoError(t, err)

	raw := "identities/42/2026/04/front.png"
	resolved, err := svc.ResolveIdentityReviewItemAssets(context.Background(), &IdentityReviewItem{
		UserID:        42,
		DocPhotoFront: &raw,
	})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	require.NotNil(t, resolved.DocPhotoFront)
	assert.Equal(t, raw, store.presignedKey)
	assert.Equal(t, store.presignURL, *resolved.DocPhotoFront)
}

func TestHandleUploadIdentityPhoto_Returns503WhenStoreUnavailable(t *testing.T) {
	repo := &mockRepo{
		onGetInternalUserID: func(_ context.Context, casdoorSubject string) (int64, error) {
			assert.Equal(t, "external-user-123", casdoorSubject)
			return 42, nil
		},
	}
	router := setupUserHandlerTestRouterWithServiceOptions(t, repo)
	body := bytes.NewBufferString(`{"slot":"front","filename":"identity.png","contentType":"image/png","dataBase64":"` + base64.StdEncoding.EncodeToString(validPNGBytes(t)) + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/identity/uploads", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	assert.Equal(t, "identity photo upload is not available", decodeUserErrorMessage(t, resp.Body.Bytes()))
}

func TestHandleUploadIdentityPhoto_MapsTemporaryStorageDomainErrorTo503(t *testing.T) {
	repo := &mockRepo{
		onGetInternalUserID: func(_ context.Context, casdoorSubject string) (int64, error) {
			assert.Equal(t, "external-user-123", casdoorSubject)
			return 42, nil
		},
	}
	store := &fakeIdentityPhotoStore{
		uploadErr: ErrIdentityPhotoStorageTemporaryUnavailable,
	}
	router := setupUserHandlerTestRouterWithServiceOptions(t, repo, WithIdentityPhotoStore(store))
	body := bytes.NewBufferString(`{"slot":"front","filename":"identity.png","contentType":"image/png","dataBase64":"` + base64.StdEncoding.EncodeToString(validPNGBytes(t)) + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/identity/uploads", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	assert.Equal(t, "identity photo storage is temporarily unavailable", decodeUserErrorMessage(t, resp.Body.Bytes()))
}

func TestHandleUploadIdentityPhoto_MapsUnavailableStorageDomainErrorTo503(t *testing.T) {
	repo := &mockRepo{
		onGetInternalUserID: func(_ context.Context, _ string) (int64, error) {
			return 42, nil
		},
	}
	store := &fakeIdentityPhotoStore{uploadErr: ErrIdentityPhotoStorageUnavailable}
	router := setupUserHandlerTestRouterWithServiceOptions(t, repo, WithIdentityPhotoStore(store))
	body := bytes.NewBufferString(`{"slot":"front","filename":"identity.png","contentType":"image/png","dataBase64":"` + base64.StdEncoding.EncodeToString(validPNGBytes(t)) + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/identity/uploads", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	assert.Equal(t, "identity photo upload is not available", decodeUserErrorMessage(t, resp.Body.Bytes()))
}

func decodeUserErrorMessage(t *testing.T, body []byte) string {
	t.Helper()
	var envelope response.Response
	require.NoError(t, json.Unmarshal(body, &envelope))
	require.NotNil(t, envelope.Error)
	return envelope.Error.Message
}

func validPNGBytes(t *testing.T) []byte {
	t.Helper()
	const raw = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+nX1cAAAAASUVORK5CYII="
	content, err := base64.StdEncoding.DecodeString(raw)
	require.NoError(t, err)
	return content
}
