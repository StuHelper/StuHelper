package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/rbac"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	appmiddleware "git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type handlerTestResponse struct {
	Success bool               `json:"success"`
	Error   *response.APIError `json:"error,omitempty"`
}

func ptr[T any](value T) *T {
	return &value
}

func setupAdminHandlerTestRouterWithRepo(t *testing.T, repo *mockRepo) *gin.Engine {
	t.Helper()
	return setupAdminHandlerTestRouterWithRole(t, repo, []string{"super_admin"}, nil)
}

func setupAdminHandlerTestRouterWithRole(
	t *testing.T,
	repo *mockRepo,
	roles []string,
	orgScopedRoles map[string][]string,
) *gin.Engine {
	t.Helper()
	return setupAdminHandlerTestRouterWithRoleAndProof(t, repo, roles, orgScopedRoles, time.Now())
}

func setupAdminHandlerTestRouterWithRoleAndProof(
	t *testing.T,
	repo *mockRepo,
	roles []string,
	orgScopedRoles map[string][]string,
	proofAt time.Time,
) *gin.Engine {
	t.Helper()

	if repo == nil {
		repo = &mockRepo{}
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	h := NewHandler(svc, nil, nil, nil, WithAdminAuthorizers(userAdminAuthorizers()))
	r := gin.New()
	api := r.Group("/api/v1")
	admin := api.Group("/admin")
	// 模拟 AuthMiddleware 注入用户信息和能力快照
	admin.Use(func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-user-123")
		snapshot := capability.BuildUserAccessSnapshot(capability.ExpandRoleGrants(roles, orgScopedRoles))
		c.Set(appmiddleware.CtxKeyRoles, roles)
		if orgScopedRoles != nil {
			c.Set(appmiddleware.CtxKeyOrgScopedRoles, orgScopedRoles)
		}
		c.Set(appmiddleware.CtxKeyCapabilities, snapshot.Capabilities)
		c.Set(appmiddleware.CtxKeyGlobalCapabilities, snapshot.GlobalCapabilities)
		c.Set(appmiddleware.CtxKeyCapabilityGrants, snapshot.CapabilityGrants)
		appmiddleware.SetMFAContext(c, appmiddleware.MFAContext{
			EnrollmentActive: true,
			ProofVerifiedAt:  proofAt,
		})
		// HasCapability 从 CtxKeyCapabilitySet (map) 做 O(1) 查找
		capSet := make(map[string]struct{}, len(snapshot.Capabilities))
		for _, cap := range snapshot.Capabilities {
			capSet[cap] = struct{}{}
		}
		c.Set(appmiddleware.CtxKeyCapabilitySet, capSet)
		c.Next()
	})
	h.RegisterAdminRoutes(admin)

	return r
}

func userAdminAuthorizers() AdminAuthorizers {
	return AdminAuthorizers{
		IdentityRead:   rbac.RequireGlobalCapability(capability.UserIdentityRead),
		IdentityReview: rbac.RequireGlobalCapability(capability.UserIdentityReview),
		StudentRead:    rbac.RequireCapability(capability.UserStudentRead),
		StudentReview:  rbac.RequireCapability(capability.UserStudentReview),
		SchoolRead:     rbac.RequireCapability(capability.UserSchoolRead),
		SchoolUpdate:   rbac.RequireCapability(capability.UserSchoolUpdate),
		SystemRead:     rbac.RequireGlobalCapability(capability.UserSystemRead),
		SystemUpdate:   rbac.RequireGlobalCapability(capability.UserSystemUpdate),
		StepUpMFA:      rbac.RequireStepUpMFA(),
	}
}

func setupAdminHandlerTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	return setupAdminHandlerTestRouterWithRepo(t, nil)
}

func setupUserHandlerTestRouterWithRepo(t *testing.T, repo *mockRepo) *gin.Engine {
	t.Helper()
	return setupUserHandlerTestRouterWithServiceOptions(t, repo)
}

func setupUserHandlerTestRouterWithServiceOptions(t *testing.T, repo *mockRepo, opts ...ServiceOption) *gin.Engine {
	t.Helper()

	if repo == nil {
		repo = &mockRepo{}
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{}, opts...)
	require.NoError(t, err)

	h := NewHandler(svc, nil, nil, nil)
	r := gin.New()
	api := r.Group("/api/v1")
	authMW := func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-user-123")
		c.Next()
	}
	h.RegisterRoutes(api, authMW)

	return r
}

func setupUserHandlerTestRouterWithRuntimeDeps(
	t *testing.T,
	repo *mockRepo,
	rdb *redis.Client,
	otpService OTPGenerator,
	smsService SMSSender,
	opts ...ServiceOption,
) *gin.Engine {
	t.Helper()

	if repo == nil {
		repo = &mockRepo{}
	}

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{}, opts...)
	require.NoError(t, err)

	h := NewHandler(svc, rdb, otpService, smsService)
	r := gin.New()
	api := r.Group("/api/v1")
	authMW := func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-user-123")
		c.Next()
	}
	h.RegisterRoutes(api, authMW)
	return r
}

type stubBindPhoneOTPGenerator struct {
	issueCalls   int
	checkCalls   int
	consumeCalls int
	consumePhone string
	consumeCode  string
	checkErr     error
	consumeErr   error
}

func (s *stubBindPhoneOTPGenerator) IssueCode(_ context.Context, _ string, _ SMSSender) error {
	s.issueCalls++
	return nil
}

func (s *stubBindPhoneOTPGenerator) CooldownSeconds() int {
	return 60
}

func (s *stubBindPhoneOTPGenerator) Check(context.Context, string, string) error {
	s.checkCalls++
	return s.checkErr
}

func (s *stubBindPhoneOTPGenerator) Consume(_ context.Context, phone, code string) error {
	s.consumeCalls++
	s.consumePhone = phone
	s.consumeCode = code
	return s.consumeErr
}

type stubBindPhoneSMSSender struct{}

func (stubBindPhoneSMSSender) Send(context.Context, string, string) error {
	return nil
}

func TestHandleAdminReviewIdentity_AllowsBlankRejectionReason(t *testing.T) {
	repo := &mockRepo{
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 123, Verified: true}, nil
		},
		onUpdateIdentityReviewStatus: func(_ context.Context, _ int64, approved bool, verifyMethod *string, _ *time.Time, _ *time.Time, rejectionReason *string) error {
			assert.False(t, approved)
			assert.Nil(t, verifyMethod)
			assert.Nil(t, rejectionReason)
			return nil
		},
	}
	r := setupAdminHandlerTestRouterWithRepo(t, repo)

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/identities/123",
		strings.NewReader(`{"approved":false,"rejectionReason":"   "}`),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp handlerTestResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandleAdminReviewStudentVerification_AllowsBlankRejectionReason(t *testing.T) {
	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
			return &Profile{UserID: 123, VerificationStatus: StatusPending}, nil
		},
		onUpdateProfile: func(_ context.Context, profile *Profile) error {
			assert.Equal(t, StatusRejected, profile.VerificationStatus)
			assert.Nil(t, profile.VerifiedAt)
			assert.Nil(t, profile.RejectionReason)
			return nil
		},
	}
	r := setupAdminHandlerTestRouterWithRepo(t, repo)

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/student-verifications/123",
		strings.NewReader(`{"approved":false,"rejectionReason":"   "}`),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp handlerTestResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandleAdminReviewStudentVerification_AllowsNullRejectionReason(t *testing.T) {
	repo := &mockRepo{
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
			return &Profile{UserID: 123, VerificationStatus: StatusPending}, nil
		},
		onUpdateProfile: func(_ context.Context, profile *Profile) error {
			assert.Equal(t, StatusRejected, profile.VerificationStatus)
			assert.Nil(t, profile.RejectionReason)
			return nil
		},
	}
	r := setupAdminHandlerTestRouterWithRepo(t, repo)

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/student-verifications/123",
		strings.NewReader(`{"approved":false,"rejectionReason":null}`),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp handlerTestResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
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
		onGetInternalUserID: func(_ context.Context, casdoorSubject string) (int64, error) {
			assert.Equal(t, "external-user-123", casdoorSubject)
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
		onListProfilesByStatus: func(_ context.Context, status string, schoolID *int64, _, _ int) ([]Profile, int, error) {
			capturedStatus = status
			assert.Nil(t, schoolID)
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

func TestHandleAdminListStudentVerifications_ScopedAdminRequiresSchoolIDForMultipleScopes(t *testing.T) {
	repo := &mockRepo{}
	r := setupAdminHandlerTestRouterWithRole(t, repo, []string{"school_admin"}, map[string][]string{
		"school_admin": {"4111010006", "4111010007"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/student-verifications", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAdminListStudentVerifications_ScopedAdminAllowsInScopeSchool(t *testing.T) {
	var capturedSchoolID *int64
	repo := &mockRepo{
		onListProfilesByStatus: func(_ context.Context, status string, schoolID *int64, _, _ int) ([]Profile, int, error) {
			capturedSchoolID = schoolID
			return []Profile{}, 0, nil
		},
	}
	r := setupAdminHandlerTestRouterWithRole(t, repo, []string{"school_admin"}, map[string][]string{
		"school_admin": {"4111010006"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/student-verifications?schoolID=4111010006", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedSchoolID)
	assert.Equal(t, int64(4111010006), *capturedSchoolID)
}

func TestHandleAdminListIdentities_AllStatusClearsRepositoryFilter(t *testing.T) {
	var capturedStatus string
	repo := &mockRepo{
		onListIdentityReviewItems: func(_ context.Context, status string, _, _ int) ([]IdentityReviewItem, int, error) {
			capturedStatus = status
			return []IdentityReviewItem{}, 0, nil
		},
	}

	r := setupAdminHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/identities?status=all", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, capturedStatus)
}

func TestHandleAdminListStudentVerifications_AllStatusClearsRepositoryFilter(t *testing.T) {
	var capturedStatus string
	repo := &mockRepo{
		onListProfilesByStatus: func(_ context.Context, status string, schoolID *int64, _, _ int) ([]Profile, int, error) {
			capturedStatus = status
			assert.Nil(t, schoolID)
			return []Profile{}, 0, nil
		},
	}

	r := setupAdminHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/student-verifications?status=all", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, capturedStatus)
}

func TestHandleAdminListIdentities_RejectsUnverifiedStatus(t *testing.T) {
	r := setupAdminHandlerTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/identities?status=unverified", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAdminListStudentVerifications_IncludesManualFormData(t *testing.T) {
	repo := &mockRepo{
		onListProfilesByStatus: func(_ context.Context, status string, schoolID *int64, _, _ int) ([]Profile, int, error) {
			assert.Equal(t, StatusPending, status)
			return []Profile{{
				UserID:             7,
				SchoolID:           ptr(int64(4111010006)),
				VerificationStatus: StatusPending,
				VerificationMethod: ptr(VerifyMethodManual),
				ManualFormData:     json.RawMessage(`{"studentID":"20240001","department":"计算机学院"}`),
			}}, 1, nil
		},
	}

	r := setupAdminHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/student-verifications", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	data := resp["data"].(map[string]any)
	list := data["list"].([]any)
	require.Len(t, list, 1)
	item := list[0].(map[string]any)
	manualFormData := item["manualFormData"].(map[string]any)
	assert.Equal(t, "20240001", manualFormData["studentID"])
	assert.Equal(t, "计算机学院", manualFormData["department"])
}

func TestHandleVerifyStudent_ManualAllowsEmptyCredentials(t *testing.T) {
	var createdProfile *Profile
	repo := &mockRepo{
		onGetInternalUserID: func(_ context.Context, casdoorSubject string) (int64, error) {
			assert.Equal(t, "external-user-123", casdoorSubject)
			return 42, nil
		},
		onGetIdentityStatusByUserID: func(_ context.Context, userID int64) (*IdentityStatus, error) {
			assert.Equal(t, int64(42), userID)
			return &IdentityStatus{UserID: userID, Verified: true}, nil
		},
		onListSchoolConfigs: func(_ context.Context) ([]SchoolConfig, error) {
			return []SchoolConfig{{
				SchoolID:           4111010006,
				SchoolCode:         "4111010006",
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodManual,
				Enabled:            true,
			}}, nil
		},
		onGetSchoolConfig: func(_ context.Context, schoolID int64) (*SchoolConfig, error) {
			assert.Equal(t, int64(4111010006), schoolID)
			return &SchoolConfig{
				SchoolID:           4111010006,
				SchoolCode:         "4111010006",
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodManual,
				ApprovalPolicy:     "manual",
				ManualFormFields:   json.RawMessage(`[{"key":"studentID","label":"学号","type":"text","required":true}]`),
				Enabled:            true,
			}, nil
		},
		onCreateProfile: func(_ context.Context, profile *Profile) error {
			assert.Equal(t, StatusPending, profile.VerificationStatus)
			assert.Equal(t, []string{"20240001"}, profile.StudentIDs)
			copied := *profile
			createdProfile = &copied
			return nil
		},
		onGetProfileByUserID: func(_ context.Context, _ int64) (*Profile, error) {
			if createdProfile == nil {
				return nil, nil
			}
			schoolID := int64(4111010006)
			method := VerifyMethodManual
			activeStudentID := "20240001"
			return &Profile{
				UserID:             42,
				SchoolID:           &schoolID,
				StudentIDs:         []string{"20240001"},
				ActiveStudentID:    &activeStudentID,
				VerificationStatus: StatusPending,
				VerificationMethod: &method,
			}, nil
		},
	}

	r := setupUserHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/user/profile/verify",
		strings.NewReader(`{"schoolCode":"4111010006","manualFormData":{"studentID":"20240001"},"consent":true}`),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	data := resp["data"].(map[string]any)
	assert.Equal(t, "pending", data["verificationStatus"])
}

func TestHandleVerifyStudent_RejectsSchoolIDOnlyPublicRequest(t *testing.T) {
	repo := &mockRepo{
		onGetInternalUserID: func(_ context.Context, _ string) (int64, error) {
			return 42, nil
		},
	}

	r := setupUserHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/user/profile/verify",
		strings.NewReader(`{"schoolID":4111010006,"manualFormData":{"studentID":"20240001"},"consent":true}`),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "schoolID is not accepted; use schoolCode")
}

func TestPublicStudentVerificationEndpointsRequireSchoolCode(t *testing.T) {
	cases := []struct {
		name string
		path string
		body string
	}{
		{
			name: "manual verification",
			path: "/api/v1/user/profile/verify",
			body: `{"manualFormData":{"studentID":"20240001"},"consent":true}`,
		},
		{
			name: "request school email otp",
			path: "/api/v1/user/profile/school-email/request-otp",
			body: `{"studentID":"20240001","studentName":"张三"}`,
		},
		{
			name: "match school email academic student",
			path: "/api/v1/user/profile/school-email/academic-match",
			body: `{"studentID":"20240001","studentName":"张三"}`,
		},
		{
			name: "verify school email otp",
			path: "/api/v1/user/profile/school-email/verify-otp",
			body: `{"email":"20240001@buaa.edu.cn","code":"123456","consent":true}`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{
				onGetInternalUserID: func(_ context.Context, _ string) (int64, error) {
					return 42, nil
				},
				onListSchoolConfigs: func(context.Context) ([]SchoolConfig, error) {
					t.Fatal("missing schoolCode public request must fail before school lookup")
					return nil, nil
				},
			}

			r := setupUserHandlerTestRouterWithRepo(t, repo)
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), "invalid request parameters")
		})
	}
}

func TestPublicStudentVerificationEndpointsRejectSchoolIDFieldEvenWithSchoolCode(t *testing.T) {
	cases := []struct {
		name string
		path string
		body string
	}{
		{
			name: "manual verification",
			path: "/api/v1/user/profile/verify",
			body: `{"schoolCode":"4111010006","schoolID":4111019999,"manualFormData":{"studentID":"20240001"},"consent":true}`,
		},
		{
			name: "request school email otp",
			path: "/api/v1/user/profile/school-email/request-otp",
			body: `{"schoolCode":"4111010006","schoolID":4111019999,"studentID":"20240001","studentName":"张三"}`,
		},
		{
			name: "match school email academic student",
			path: "/api/v1/user/profile/school-email/academic-match",
			body: `{"schoolCode":"4111010006","schoolID":4111019999,"studentID":"20240001","studentName":"张三"}`,
		},
		{
			name: "verify school email otp",
			path: "/api/v1/user/profile/school-email/verify-otp",
			body: `{"schoolCode":"4111010006","schoolID":4111019999,"email":"20240001@buaa.edu.cn","code":"123456","consent":true}`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{
				onGetInternalUserID: func(_ context.Context, _ string) (int64, error) {
					return 42, nil
				},
				onListSchoolConfigs: func(context.Context) ([]SchoolConfig, error) {
					t.Fatal("schoolID field must fail before school lookup")
					return nil, nil
				},
				onGetSchoolConfig: func(context.Context, int64) (*SchoolConfig, error) {
					t.Fatal("schoolID field must fail before verification service")
					return nil, nil
				},
			}

			r := setupUserHandlerTestRouterWithRepo(t, repo)
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), "schoolID is not accepted; use schoolCode")
		})
	}
}

func TestHandleVerifyStudent_LDAPMissingStudentIDReturns400(t *testing.T) {
	repo := &mockRepo{
		onGetInternalUserID: func(_ context.Context, _ string) (int64, error) {
			return 42, nil
		},
		onGetIdentityStatusByUserID: func(_ context.Context, _ int64) (*IdentityStatus, error) {
			return &IdentityStatus{UserID: 42, Verified: true}, nil
		},
		onGetSchoolConfig: func(_ context.Context, _ int64) (*SchoolConfig, error) {
			return &SchoolConfig{
				SchoolID:           4111010001,
				SchoolName:         "LDAP 学校",
				VerificationMethod: VerifyMethodLDAP,
				Enabled:            true,
			}, nil
		},
	}

	r := setupUserHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/user/profile/verify",
		strings.NewReader(`{"schoolID":"ldap","password":"secret","consent":true}`),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleGetUserSurface_NormalizesNilCapabilitiesToEmptySlice(t *testing.T) {
	repo := &mockRepo{
		onGetInternalUserID: func(_ context.Context, casdoorSubject string) (int64, error) {
			assert.Equal(t, "external-user-123", casdoorSubject)
			return 42, nil
		},
	}

	r := setupUserHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	data := resp["data"].(map[string]any)
	capabilities := data["capabilities"].([]any)
	assert.Empty(t, capabilities)
}

func TestHandleGetProfile_NormalizesNilStudentIDsToEmptySlice(t *testing.T) {
	repo := &mockRepo{
		onGetInternalUserID: func(_ context.Context, _ string) (int64, error) {
			return 42, nil
		},
		onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
			assert.Equal(t, int64(42), userID)
			return &Profile{
				UserID:             userID,
				VerificationStatus: StatusPending,
			}, nil
		},
	}

	r := setupUserHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/profile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	data := resp["data"].(map[string]any)
	studentIDs := data["studentIDs"].([]any)
	assert.Empty(t, studentIDs)
}

func TestHandleRequestBindPhoneOTP_IsRateLimitedPerUserEndpoint(t *testing.T) {
	fixture := redisfixture.Start(t)
	otp := &stubBindPhoneOTPGenerator{}
	repo := &mockRepo{
		onGetInternalUserID: func(_ context.Context, _ string) (int64, error) {
			return 42, nil
		},
	}

	r := setupUserHandlerTestRouterWithRuntimeDeps(
		t,
		repo,
		fixture.Client,
		otp,
		stubBindPhoneSMSSender{},
	)

	body := strings.NewReader(`{"phone":"13800138000"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/profile/bind-phone/otp", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, otp.issueCalls)

	for i := 0; i < bindPhoneOTPUserLimitPerMinute-1; i++ {
		body = strings.NewReader(`{"phone":"13900139000"}`)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/user/profile/bind-phone/otp", body)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	}

	body = strings.NewReader(`{"phone":"13700137000"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/user/profile/bind-phone/otp", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, bindPhoneOTPUserLimitPerMinute, otp.issueCalls)
}

func TestHandleBindPhoneConsumesOTPOnlyAfterSuccessfulBind(t *testing.T) {
	otp := &stubBindPhoneOTPGenerator{}
	repo := &mockRepo{
		onGetInternalUserID: func(_ context.Context, _ string) (int64, error) {
			return 42, nil
		},
		onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
			assert.Equal(t, int64(42), userID)
			return &Profile{UserID: userID}, nil
		},
		onGetCasdoorSubject: func(_ context.Context, userID int64) (string, error) {
			assert.Equal(t, int64(42), userID)
			return "casdoor-subject-42", nil
		},
	}
	r := setupUserHandlerTestRouterWithRuntimeDeps(
		t,
		repo,
		nil,
		otp,
		stubBindPhoneSMSSender{},
		WithProfileIdentitySyncGateway(profileIdentitySyncFunc(func(context.Context, string, string) error {
			return nil
		})),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/user/profile/bind-phone",
		strings.NewReader(`{"phone":"13800138000","otpCode":"123456"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, otp.checkCalls)
	assert.Equal(t, 1, otp.consumeCalls)
	assert.Equal(t, "13800138000", otp.consumePhone)
	assert.Equal(t, "123456", otp.consumeCode)
}

func TestHandleBindPhoneKeepsOTPWhenBindFails(t *testing.T) {
	otp := &stubBindPhoneOTPGenerator{}
	setPhoneErr := errors.New("phone projection unavailable")
	repo := &mockRepo{
		onGetInternalUserID: func(_ context.Context, _ string) (int64, error) {
			return 42, nil
		},
		onGetProfileByUserID: func(_ context.Context, userID int64) (*Profile, error) {
			assert.Equal(t, int64(42), userID)
			return &Profile{UserID: userID}, nil
		},
		onGetCasdoorSubject: func(_ context.Context, userID int64) (string, error) {
			assert.Equal(t, int64(42), userID)
			return "casdoor-subject-42", nil
		},
		onSetUserPhone: func(_ context.Context, userID int64, _ []byte, _ string) error {
			assert.Equal(t, int64(42), userID)
			return setPhoneErr
		},
	}
	r := setupUserHandlerTestRouterWithRuntimeDeps(
		t,
		repo,
		nil,
		otp,
		stubBindPhoneSMSSender{},
		WithProfileIdentitySyncGateway(profileIdentitySyncFunc(func(context.Context, string, string) error {
			return nil
		})),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/user/profile/bind-phone",
		strings.NewReader(`{"phone":"13800138000","otpCode":"123456"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, 1, otp.checkCalls)
	assert.Equal(t, 0, otp.consumeCalls)
}

func TestHandleAdminListSchoolConfigs_MapsToSpecShape(t *testing.T) {
	now := time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		onListAllSchoolConfigs: func(_ context.Context) ([]SchoolConfig, error) {
			academicTable := "academic.buaa_students"
			consentText := "授权说明"
			return []SchoolConfig{{
				SchoolID:           4111010006,
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodLDAP,
				LDAPConfig:         json.RawMessage(`{"url":"ldaps://ldap.example:636","baseDN":"ou=users,dc=example,dc=com","systemBindDN":"cn=system,dc=example,dc=com","systemBindPassword":"secret","useTLS":true,"insecureSkipVerify":false}`),
				AcademicDBTable:    &academicTable,
				ConsentText:        &consentText,
				ManualFormFields:   json.RawMessage(`[{"key":"studentID","label":"学号","type":"text","required":true}]`),
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
	assert.Equal(t, float64(4111010006), item["schoolID"])
	assert.Equal(t, "北航", item["schoolName"])
	assert.NotContains(t, item, "SchoolID")
	assert.NotContains(t, item, "LDAPConfig")
	ldapConfig := item["ldapConfig"].(map[string]any)
	assert.Equal(t, "ldaps://ldap.example:636", ldapConfig["url"])
	assert.Equal(t, true, ldapConfig["hasSystemBindPassword"])
	assert.NotContains(t, ldapConfig, "systemBindPassword")
	manualFormFields := item["manualFormFields"].([]any)
	require.Len(t, manualFormFields, 1)
	firstField := manualFormFields[0].(map[string]any)
	assert.Equal(t, "studentID", firstField["key"])
	assert.Equal(t, "学号", firstField["label"])
}

func TestHandleAdminListSchoolConfigs_NormalizesNilManualFormFieldsToEmptySlice(t *testing.T) {
	repo := &mockRepo{
		onListAllSchoolConfigs: func(_ context.Context) ([]SchoolConfig, error) {
			return []SchoolConfig{{
				SchoolID:           4111010006,
				SchoolName:         "北航",
				VerificationMethod: VerifyMethodManual,
				ApprovalPolicy:     "manual",
				Enabled:            true,
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
	manualFormFields := item["manualFormFields"].([]any)
	assert.Empty(t, manualFormFields)
}

func TestHandleAdminUpdateSchoolConfig_InvalidAcademicTableReturnsBusinessCode(t *testing.T) {
	existing := &SchoolConfig{
		SchoolID:           4111010006,
		SchoolName:         "北航",
		VerificationMethod: VerifyMethodLDAP,
		Enabled:            true,
	}
	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, schoolID int64) (*SchoolConfig, error) {
			assert.Equal(t, int64(4111010006), schoolID)
			copied := *existing
			return &copied, nil
		},
	}

	r := setupAdminHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/school-configs/4111010006",
		strings.NewReader(`{"academicDbTable":"academic.students;drop table users;"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp handlerTestResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.NotNil(t, resp.Error)
	assert.Equal(t, string(errs.ErrProfileAcademicTable), resp.Error.Code)
}

func TestHandleAdminUpdateSchoolConfig_MissingLDAPConfigReturnsBusinessCode(t *testing.T) {
	existing := &SchoolConfig{
		SchoolID:           4111010006,
		SchoolName:         "北航",
		VerificationMethod: VerifyMethodManual,
		Enabled:            false,
	}
	repo := &mockRepo{
		onGetSchoolConfig: func(_ context.Context, schoolID int64) (*SchoolConfig, error) {
			assert.Equal(t, int64(4111010006), schoolID)
			copied := *existing
			return &copied, nil
		},
	}

	r := setupAdminHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/school-configs/4111010006",
		strings.NewReader(`{"verificationMethod":"ldap","enabled":true,"academicDbTable":"academic.buaa_students"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp handlerTestResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.NotNil(t, resp.Error)
	assert.Equal(t, string(errs.ErrSchoolLDAPConfigMissing), resp.Error.Code)
}

func TestHandleAdminUpdateSchoolConfig_ScopedAdminOutOfScopeReturns403(t *testing.T) {
	repo := &mockRepo{}
	r := setupAdminHandlerTestRouterWithRole(t, repo, []string{"school_admin"}, map[string][]string{
		"school_admin": {"4111010006"},
	})

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/school-configs/4111010007",
		strings.NewReader(`{"enabled":true}`),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleListSchools_ManualIncludesManualFormFields(t *testing.T) {
	repo := &mockRepo{
		onListSchoolConfigs: func(_ context.Context) ([]SchoolConfig, error) {
			return []SchoolConfig{{
				SchoolID:           4111010002,
				SchoolName:         "人工审核学校",
				VerificationMethod: VerifyMethodManual,
				ManualFormFields:   json.RawMessage(`[{"key":"studentID","label":"学号","type":"text","required":true}]`),
				Enabled:            true,
			}}, nil
		},
	}

	r := setupUserHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/schools", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	list := resp["data"].([]any)
	require.Len(t, list, 1)
	item := list[0].(map[string]any)
	manualFormFields := item["manualFormFields"].([]any)
	require.Len(t, manualFormFields, 1)
	firstField := manualFormFields[0].(map[string]any)
	assert.Equal(t, "studentID", firstField["key"])
	assert.Equal(t, false, item["schoolSsoEnabled"])
	assert.Equal(t, false, item["schoolEmailOtpEnabled"])
}

func TestHandleListSchools_AdmissionCapabilitiesFromSchoolConfig(t *testing.T) {
	repo := &mockRepo{
		onListSchoolConfigs: func(_ context.Context) ([]SchoolConfig, error) {
			return []SchoolConfig{{
				SchoolID:           4111010006,
				SchoolName:         "北京航空航天大学",
				VerificationMethod: VerifyMethodLDAP,
				ManualFormFields: json.RawMessage(
					`{"admission":{"emailDomains":["buaa.edu.cn"],"ssoLoginURL":"https://sso.school.example/login"}}`,
				),
				Enabled: true,
			}}, nil
		},
	}

	r := setupUserHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/schools", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	list := resp["data"].([]any)
	require.Len(t, list, 1)
	item := list[0].(map[string]any)
	assert.Equal(t, true, item["schoolSsoEnabled"])
	assert.Equal(t, true, item["schoolEmailOtpEnabled"])
	assert.NotContains(t, item, "manualFormFields")
}

func TestHandleListSchools_ManualWithoutFieldsReturnsEmptySlice(t *testing.T) {
	repo := &mockRepo{
		onListSchoolConfigs: func(_ context.Context) ([]SchoolConfig, error) {
			return []SchoolConfig{{
				SchoolID:           4111010002,
				SchoolName:         "人工审核学校",
				VerificationMethod: VerifyMethodManual,
				Enabled:            true,
			}}, nil
		},
	}

	r := setupUserHandlerTestRouterWithRepo(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/schools", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	list := resp["data"].([]any)
	require.Len(t, list, 1)
	item := list[0].(map[string]any)
	manualFormFields := item["manualFormFields"].([]any)
	assert.Empty(t, manualFormFields)
	assert.Equal(t, false, item["schoolSsoEnabled"])
	assert.Equal(t, false, item["schoolEmailOtpEnabled"])
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

func TestHandleAdminListSystemConfigs_ScopedAdminForbidden(t *testing.T) {
	r := setupAdminHandlerTestRouterWithRole(t, nil, []string{"school_admin"}, map[string][]string{
		"school_admin": {"4111010006"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/system-configs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleAdminUpdateSystemConfig_InvalidReviewPreviewPercentReturns400(t *testing.T) {
	r := setupAdminHandlerTestRouter(t)

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/system-configs/review_preview_content_percent",
		strings.NewReader(`{"value":"120"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp handlerTestResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.NotNil(t, resp.Error)
	assert.Equal(t, string(errs.ErrInvalidParam), resp.Error.Code)
}

func TestHandleAdminUpdateSystemConfig_NotFoundReturns404(t *testing.T) {
	repo := &mockRepo{
		onUpdateSystemConfig: func(_ context.Context, key, value string) error {
			assert.Equal(t, "feature.missing", key)
			assert.Equal(t, "enabled", value)
			return ErrSystemConfigNotFound
		},
	}
	r := setupAdminHandlerTestRouterWithRepo(t, repo)

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/system-configs/feature.missing",
		strings.NewReader(`{"value":"enabled"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)

	var resp handlerTestResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.NotNil(t, resp.Error)
	assert.Equal(t, string(errs.ErrSystemConfigNotFound), resp.Error.Code)
}
