package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleAdminListStudentVerifications_IncludesReviewMeta(t *testing.T) {
	reviewedAt := time.Date(2026, 3, 17, 15, 20, 0, 0, time.UTC)
	rejectionReason := "学籍材料不完整"

	repo := &mockRepo{
		onListProfilesByStatus: func(_ context.Context, status string, schoolID *int64, _, _ int) ([]Profile, int, error) {
			assert.Equal(t, StatusPending, status)
			assert.Nil(t, schoolID)
			return []Profile{
				{
					UserID:             101,
					SchoolID:           ptr(int64(4111010006)),
					VerificationStatus: StatusRejected,
					VerificationMethod: ptr(VerifyMethodManual),
					ManualFormData:     json.RawMessage(`{"studentID":"20240001"}`),
					RejectionReason:    &rejectionReason,
					ReviewedAt:         &reviewedAt,
				},
			}, 1, nil
		},
	}

	router := setupAdminHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/student-verifications", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)

	var responseBody map[string]any
	err := json.Unmarshal(recorder.Body.Bytes(), &responseBody)
	require.NoError(t, err)

	data := responseBody["data"].(map[string]any)
	list := data["list"].([]any)
	require.Len(t, list, 1)
	item := list[0].(map[string]any)

	assert.Equal(t, rejectionReason, item["rejectionReason"])
	assert.Equal(t, reviewedAt.Format(time.RFC3339), item["reviewedAt"])
}

func TestHandleAdminListIdentitiesRequiresStepUpProof(t *testing.T) {
	router := setupAdminHandlerTestRouterWithRoleAndProof(
		t,
		&mockRepo{},
		[]string{"super_admin"},
		nil,
		time.Time{},
	)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/identities", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusPreconditionRequired, recorder.Code)
}

func TestHandleAdminListIdentities_IncludesReviewMeta(t *testing.T) {
	reviewedAt := time.Date(2026, 3, 17, 16, 40, 0, 0, time.UTC)
	photoFront := "identity/front.png"
	photoBack := "identity/back.png"

	repo := &mockRepo{
		onListIdentityReviewItems: func(_ context.Context, status string, _, _ int) ([]IdentityReviewItem, int, error) {
			assert.Equal(t, StatusPending, status)
			return []IdentityReviewItem{
				{
					UserID:        202,
					DocType:       DocTypeMainlandID,
					RealName:      "李四",
					ReviewedAt:    &reviewedAt,
					DocPhotoFront: &photoFront,
					DocPhotoBack:  &photoBack,
				},
			}, 1, nil
		},
	}

	router := setupAdminHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/identities", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)

	var responseBody map[string]any
	err := json.Unmarshal(recorder.Body.Bytes(), &responseBody)
	require.NoError(t, err)

	data := responseBody["data"].(map[string]any)
	list := data["list"].([]any)
	require.Len(t, list, 1)
	item := list[0].(map[string]any)

	assert.Equal(t, reviewedAt.Format(time.RFC3339), item["reviewedAt"])
	assert.NotContains(t, item, "docPhotoFront")
	assert.NotContains(t, item, "docPhotoBack")
	assert.NotContains(t, item, "docPhotoSelfie")
}
