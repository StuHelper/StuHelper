package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	appmiddleware "gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type handlerTestResponse struct {
	Success bool               `json:"success"`
	Error   *response.APIError `json:"error,omitempty"`
}

func setupAdminHandlerTestRouterWithRepo(t *testing.T, repo *mockRepo) *gin.Engine {
	t.Helper()

	if repo == nil {
		repo = &mockRepo{}
	}

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	h := NewHandler(svc)
	r := gin.New()
	api := r.Group("/api/v1")
	admin := api.Group("/admin")
	h.RegisterAdminRoutes(admin)

	return r
}

func setupAdminHandlerTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	return setupAdminHandlerTestRouterWithRepo(t, nil)
}

func setupUserHandlerTestRouterWithRepo(t *testing.T, repo *mockRepo) *gin.Engine {
	t.Helper()

	if repo == nil {
		repo = &mockRepo{}
	}

	svc, err := NewService(repo, nil, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	h := NewHandler(svc)
	r := gin.New()
	api := r.Group("/api/v1")
	authMW := func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-user-123")
		c.Next()
	}
	h.RegisterRoutes(api, authMW)

	return r
}

func TestHandleAdminReviewIdentity_RejectionReasonRequiredReturns400(t *testing.T) {
	r := setupAdminHandlerTestRouter(t)

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/identities/123",
		strings.NewReader(`{"approved":false,"rejectionReason":"   "}`),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp handlerTestResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)
	assert.Equal(t, string(errs.ErrBadRequest), resp.Error.Code)
	assert.Equal(t, "rejection reason is required when rejecting", resp.Error.Message)
}

func TestHandleAdminReviewStudentVerification_RejectionReasonRequiredReturns400(t *testing.T) {
	r := setupAdminHandlerTestRouter(t)

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/student-verifications/123",
		strings.NewReader(`{"approved":false,"rejectionReason":"   "}`),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp handlerTestResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)
	assert.Equal(t, string(errs.ErrBadRequest), resp.Error.Code)
	assert.Equal(t, "rejection reason is required when rejecting", resp.Error.Message)
}

func TestHandleAdminReviewIdentity_MissingApprovedReturns400(t *testing.T) {
	r := setupAdminHandlerTestRouter(t)

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/identities/123",
		strings.NewReader(`{"rejectionReason":"need review"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAdminReviewStudentVerification_MissingApprovedReturns400(t *testing.T) {
	r := setupAdminHandlerTestRouter(t)

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/student-verifications/123",
		strings.NewReader(`{"rejectionReason":"need review"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleGetIdentity_PublicPayloadOmitsSensitiveFields(t *testing.T) {
	repo := &mockRepo{
		onGetInternalUserID: func(_ context.Context, externalID string) (int64, error) {
			assert.Equal(t, "external-user-123", externalID)
			return 42, nil
		},
		onGetIdentityStatusByUserID: func(_ context.Context, userID int64) (*IdentityStatus, error) {
			assert.Equal(t, int64(42), userID)
			return &IdentityStatus{
				UserID:   42,
				DocType:  DocTypeMainlandID,
				RealName: "张三",
				Verified: false,
			}, nil
		},
	}

	r := setupUserHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/identity", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	data := resp["data"].(map[string]any)
	assert.NotContains(t, data, "personUID")
	assert.NotContains(t, data, "docPhotoFront")
	assert.NotContains(t, data, "docPhotoBack")
	assert.NotContains(t, data, "docPhotoSelfie")
}

func TestHandleAdminListIdentities_DefaultsStatusToPending(t *testing.T) {
	var capturedStatus string
	repo := &mockRepo{
		onListIdentityReviewItems: func(_ context.Context, status string, _, _ int) ([]IdentityReviewItem, int, error) {
			capturedStatus = status
			return []IdentityReviewItem{}, 0, nil
		},
	}

	r := setupAdminHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/identities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, StatusPending, capturedStatus)
}

func TestHandleAdminListStudentVerifications_DefaultsStatusToPending(t *testing.T) {
	var capturedStatus string
	repo := &mockRepo{
		onListProfilesByStatus: func(_ context.Context, status, schoolID string, _, _ int) ([]Profile, int, error) {
			capturedStatus = status
			assert.Empty(t, schoolID)
			return []Profile{}, 0, nil
		},
	}

	r := setupAdminHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/student-verifications", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, StatusPending, capturedStatus)
}

func TestHandleAdminListSchoolConfigs_MapsToSpecShape(t *testing.T) {
	now := time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		onListAllSchoolConfigs: func(_ context.Context) ([]SchoolConfig, error) {
			academicTable := "academic.buaa_students"
			consentText := "授权说明"
			return []SchoolConfig{{
				SchoolID:           "10006",
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodLDAP,
				LDAPConfig:         json.RawMessage(`{"host":"ldap.example","port":636}`),
				AcademicDBTable:    &academicTable,
				ConsentText:        &consentText,
				ManualFormFields:   json.RawMessage(`{"fields":["studentID"]}`),
				Enabled:            true,
				CreatedAt:          now,
			}}, nil
		},
	}

	r := setupAdminHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/school-configs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	items := resp["data"].([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	assert.Equal(t, "10006", item["schoolID"])
	assert.Equal(t, "北航", item["schoolName"])
	assert.NotContains(t, item, "SchoolID")
	assert.NotContains(t, item, "LDAPConfig")
	ldapConfig := item["ldapConfig"].(map[string]any)
	assert.Equal(t, "ldap.example", ldapConfig["host"])
	manualFormFields := item["manualFormFields"].(map[string]any)
	assert.Equal(t, "studentID", manualFormFields["fields"].([]any)[0])
}

func TestHandleAdminListSystemConfigs_MapsToSpecShape(t *testing.T) {
	now := time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		onListSystemConfigs: func(_ context.Context) ([]SystemConfig, error) {
			description := "演示配置"
			return []SystemConfig{{
				Key:         "feature.demo",
				Value:       "enabled",
				Description: &description,
				UpdatedAt:   now,
			}}, nil
		},
	}

	r := setupAdminHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/system-configs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	items := resp["data"].([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	assert.Equal(t, "feature.demo", item["key"])
	assert.Equal(t, "enabled", item["value"])
	assert.Equal(t, "演示配置", item["description"])
	assert.NotContains(t, item, "Key")
	assert.NotContains(t, item, "UpdatedAt")
}
