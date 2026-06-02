package academics

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestAcademicsHandlers_TriggerImportAndExposeReadModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	svc := NewService(NewRepository(fixture.DB), NewRegistry())
	router := newAcademicsTestRouter(svc)

	triggerReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/academics/import-jobs", bytes.NewBufferString(`{"sourceKey":"buaa-fixture"}`))
	triggerReq.Header.Set("Content-Type", "application/json")
	triggerResp := httptest.NewRecorder()
	router.ServeHTTP(triggerResp, triggerReq)
	require.Equal(t, http.StatusCreated, triggerResp.Code)

	var job ImportJob
	decodeAcademicsData(t, triggerResp.Body.Bytes(), &job)
	assert.Equal(t, "succeeded", job.Status)

	termsResp := executeAcademicsRequest(t, router, http.MethodGet, "/api/v1/academics/terms", "")
	require.Equal(t, http.StatusOK, termsResp.Code)
	var terms struct {
		Items []Term `json:"items"`
	}
	decodeAcademicsData(t, termsResp.Body.Bytes(), &terms)
	require.NotEmpty(t, terms.Items)
	assert.Equal(t, "2026-SPRING", terms.Items[0].Code)

	offeringsResp := executeAcademicsRequest(t, router, http.MethodGet, "/api/v1/academics/offerings?termCode=2026-SPRING", "")
	require.Equal(t, http.StatusOK, offeringsResp.Code)
	var offerings struct {
		Items []Offering `json:"items"`
		Total int        `json:"total"`
	}
	decodeAcademicsData(t, offeringsResp.Body.Bytes(), &offerings)
	require.NotEmpty(t, offerings.Items)
	assert.GreaterOrEqual(t, offerings.Total, 1)

	myCoursesResp := executeAcademicsRequest(t, router, http.MethodGet, "/api/v1/academics/me/courses?termCode=2026-SPRING", "oidc-user-1")
	require.Equal(t, http.StatusOK, myCoursesResp.Code)
	var myCourses struct {
		Items []Offering `json:"items"`
	}
	decodeAcademicsData(t, myCoursesResp.Body.Bytes(), &myCourses)
	assert.Len(t, myCourses.Items, 2)

	myScheduleResp := executeAcademicsRequest(t, router, http.MethodGet, "/api/v1/academics/me/schedule?termCode=2026-SPRING", "oidc-user-1")
	require.Equal(t, http.StatusOK, myScheduleResp.Code)
	var mySchedule struct {
		Items []Offering `json:"items"`
	}
	decodeAcademicsData(t, myScheduleResp.Body.Bytes(), &mySchedule)
	assert.Len(t, mySchedule.Items, 2)
}

func TestAcademicsHandlers_RejectDisabledSourceImport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	svc := NewService(NewRepository(fixture.DB), NewRegistry())
	router := newAcademicsTestRouter(svc)

	_, err := fixture.DB.Exec(
		httptest.NewRequest(http.MethodGet, "/", nil).Context(),
		`UPDATE academic_sources SET enabled = FALSE WHERE key = $1`,
		"buaa-fixture",
	)
	require.NoError(t, err)

	triggerReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/academics/import-jobs", bytes.NewBufferString(`{"sourceKey":"buaa-fixture"}`))
	triggerReq.Header.Set("Content-Type", "application/json")
	triggerResp := httptest.NewRecorder()
	router.ServeHTTP(triggerResp, triggerReq)

	require.Equal(t, http.StatusNotFound, triggerResp.Code)
}

func TestAcademicsHandlers_ListSourcesMatchesContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	svc := NewService(NewRepository(fixture.DB), NewRegistry())
	router := newAcademicsTestRouter(svc)

	resp := executeAcademicsRequest(t, router, http.MethodGet, "/api/v1/admin/academics/sources", "")
	require.Equal(t, http.StatusOK, resp.Code)

	var envelope response.Response
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &envelope))

	data, ok := envelope.Data.(map[string]any)
	require.True(t, ok)
	items, ok := data["items"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, items)

	first, ok := items[0].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, first, "id")
	assert.Contains(t, first, "key")
	assert.Contains(t, first, "name")
	assert.Contains(t, first, "provider")
	assert.Contains(t, first, "enabled")
	assert.NotContains(t, first, "Config")
	assert.NotContains(t, first, "config")
	assert.NotContains(t, first, "ID")
	assert.NotContains(t, first, "Key")
	assert.NotContains(t, first, "Name")
	assert.NotContains(t, first, "Provider")
	assert.NotContains(t, first, "Enabled")
}

func TestAcademicsHandlers_AdminRoutesRequireGlobalCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	svc := NewService(NewRepository(fixture.DB), NewRegistry())
	router := newAcademicsScopedTestRouter(svc, []capability.Grant{{
		Name:           capability.UserSchoolRead,
		ScopeSchoolIDs: []string{"4111010006"},
	}})

	resp := executeAcademicsRequest(t, router, http.MethodGet, "/api/v1/admin/academics/sources", "")
	require.Equal(t, http.StatusForbidden, resp.Code)
}

func newAcademicsTestRouter(svc *Service) *gin.Engine {
	return newAcademicsScopedTestRouter(svc, []capability.Grant{
		{Name: capability.UserSchoolRead},
		{Name: capability.UserSchoolUpdate},
	})
}

func newAcademicsScopedTestRouter(svc *Service, grants []capability.Grant) *gin.Engine {
	router := gin.New()
	api := router.Group("/api/v1")
	handler := NewHandler(svc)
	handler.RegisterRoutes(api, academicsAuthMiddleware(grants))
	return router
}

func academicsAuthMiddleware(grants []capability.Grant) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			userID = "admin-user"
		}
		snapshot := capability.BuildUserAccessSnapshot(grants)
		capSet := make(map[string]struct{}, len(snapshot.Capabilities))
		for _, capName := range snapshot.Capabilities {
			capSet[capName] = struct{}{}
		}
		c.Set(middleware.CtxKeyUserID, userID)
		c.Set(middleware.CtxKeyCapabilities, snapshot.Capabilities)
		c.Set(middleware.CtxKeyGlobalCapabilities, snapshot.GlobalCapabilities)
		c.Set(middleware.CtxKeyCapabilityGrants, snapshot.CapabilityGrants)
		c.Set(middleware.CtxKeyCapabilitySet, capSet)
		c.Next()
	}
}

func executeAcademicsRequest(t *testing.T, router *gin.Engine, method, path, userID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if userID != "" {
		req.Header.Set("X-User-ID", userID)
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func decodeAcademicsData(t *testing.T, body []byte, target any) {
	t.Helper()
	var envelope response.Response
	require.NoError(t, json.Unmarshal(body, &envelope))
	raw, err := json.Marshal(envelope.Data)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, target))
}
