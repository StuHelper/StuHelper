package admission

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestAdmissionLinkHandlerAllowsSameUserToResumeConsumedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "handler-link-resume")
	router := newAdmissionHandlerTestRouter(t, svc, userID)
	tokenPath := url.PathEscape(created.Token)

	first := performAdmissionHandlerRequest(
		router,
		http.MethodPost,
		"/api/v1/admission/sessions/"+tokenPath+"/link?qq=10001",
	)
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())
	firstBody := decodeAdmissionSessionResponse(t, first)
	assert.True(t, firstBody.Success)
	assert.Equal(t, StatusLinked, firstBody.Data.Status)
	assert.Equal(t, fmt.Sprint(userID), firstBody.Data.UserID)

	preview := performAdmissionHandlerRequest(
		router,
		http.MethodGet,
		"/api/v1/admission/sessions/"+tokenPath+"?qq=10001",
	)
	require.Equal(t, http.StatusConflict, preview.Code, preview.Body.String())
	previewBody := decodeAdmissionErrorResponse(t, preview)
	assert.False(t, previewBody.Success)
	assert.Equal(t, "admission.token_consumed", previewBody.Error.Code)

	resumed := performAdmissionHandlerRequest(
		router,
		http.MethodPost,
		"/api/v1/admission/sessions/"+tokenPath+"/link?qq=10001",
	)
	require.Equal(t, http.StatusOK, resumed.Code, resumed.Body.String())
	resumedBody := decodeAdmissionSessionResponse(t, resumed)
	assert.True(t, resumedBody.Success)
	assert.Equal(t, firstBody.Data.ID, resumedBody.Data.ID)
	assert.Equal(t, StatusLinked, resumedBody.Data.Status)
	assert.Equal(t, fmt.Sprint(userID), resumedBody.Data.UserID)
}

func TestAdmissionPreviewHandlerReturnsDomainCodeForMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	userID := seedAdmissionUser(t, fixture, "handler-preview-missing")
	router := newAdmissionHandlerTestRouter(t, svc, userID)

	response := performAdmissionHandlerRequest(
		router,
		http.MethodGet,
		"/api/v1/admission/sessions/missing-token?qq=10001",
	)
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
	body := decodeAdmissionErrorResponse(t, response)
	assert.False(t, body.Success)
	assert.Equal(t, "admission.token_not_found", body.Error.Code)
}

func TestFreshmanApplicationHandlerReusesPendingApplicationOnDuplicatePost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	enableBUAAAdmissionSchool(t, fixture)
	created := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "handler-freshman-reuse")
	_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:   created.Token,
		QQQuery: "10001",
		UserID:  userID,
	})
	require.NoError(t, err)
	router := newAdmissionHandlerTestRouter(t, svc, userID)
	body := `{"schoolCode":"4111010006","applicantName":"张三","materialType":"admission_notice"}`

	first := performAdmissionHandlerJSONRequest(
		router,
		http.MethodPost,
		"/api/v1/admission/freshman/applications",
		body,
	)
	require.Equal(t, http.StatusCreated, first.Code, first.Body.String())
	firstBody := decodeFreshmanApplicationResponse(t, first)
	assert.True(t, firstBody.Success)
	assert.NotEmpty(t, firstBody.Data.ID)
	assert.Equal(t, "pending", firstBody.Data.Status)

	second := performAdmissionHandlerJSONRequest(
		router,
		http.MethodPost,
		"/api/v1/admission/freshman/applications",
		body,
	)
	require.Equal(t, http.StatusCreated, second.Code, second.Body.String())
	secondBody := decodeFreshmanApplicationResponse(t, second)
	assert.True(t, secondBody.Success)
	assert.Equal(t, firstBody.Data.ID, secondBody.Data.ID)
	assert.Equal(t, "pending", secondBody.Data.Status)
}

func TestAdmissionPublicSchoolRequestsRejectSchoolIDField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	userID := seedAdmissionUser(t, fixture, "handler-school-id-rejected")
	router := newAdmissionHandlerTestRouter(t, svc, userID)

	cases := []struct {
		name string
		path string
		body string
	}{
		{
			name: "freshman application",
			path: "/api/v1/admission/freshman/applications",
			body: `{"schoolCode":"4111010006","schoolID":4111019999,"applicantName":"张三","materialType":"admission_notice"}`,
		},
		{
			name: "request school email otp",
			path: "/api/v1/admission/school-email/request-otp",
			body: `{"schoolCode":"4111010006","schoolID":4111019999,"studentID":"20240001","studentName":"张三"}`,
		},
		{
			name: "match school email academic student",
			path: "/api/v1/admission/school-email/academic-match",
			body: `{"schoolCode":"4111010006","schoolID":4111019999,"studentID":"20240001","studentName":"张三"}`,
		},
		{
			name: "verify school email otp",
			path: "/api/v1/admission/school-email/verify-otp",
			body: `{"schoolCode":"4111010006","schoolID":4111019999,"email":"20240001@buaa.edu.cn","code":"123456"}`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			response := performAdmissionHandlerJSONRequest(
				router,
				http.MethodPost,
				tt.path,
				tt.body,
			)
			require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
			assert.Contains(t, response.Body.String(), "schoolID is not accepted; use schoolCode")
		})
	}
}

func newAdmissionHandlerTestRouter(t *testing.T, svc *Service, userID int64) *gin.Engine {
	t.Helper()

	const casdoorSubject = "casdoor-handler-test-user"
	handler := NewHandler(
		svc,
		func(_ context.Context, subject string) (int64, error) {
			require.Equal(t, casdoorSubject, subject)
			return userID, nil
		},
		nil,
	)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"), func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, casdoorSubject)
		c.Next()
	})
	return router
}

func performAdmissionHandlerRequest(router http.Handler, method string, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func performAdmissionHandlerJSONRequest(
	router http.Handler,
	method string,
	target string,
	body string,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func decodeAdmissionSessionResponse(t *testing.T, recorder *httptest.ResponseRecorder) struct {
	Success bool                        `json:"success"`
	Data    admissionSessionHTTPPayload `json:"data"`
} {
	t.Helper()
	var body struct {
		Success bool                        `json:"success"`
		Data    admissionSessionHTTPPayload `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body), recorder.Body.String())
	return body
}

type admissionSessionHTTPPayload struct {
	ID     string                 `json:"id"`
	Status AdmissionSessionStatus `json:"status"`
	UserID string                 `json:"userID"`
}

func decodeFreshmanApplicationResponse(t *testing.T, recorder *httptest.ResponseRecorder) struct {
	Success bool                           `json:"success"`
	Data    freshmanApplicationHTTPPayload `json:"data"`
} {
	t.Helper()
	var body struct {
		Success bool                           `json:"success"`
		Data    freshmanApplicationHTTPPayload `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body), recorder.Body.String())
	return body
}

type freshmanApplicationHTTPPayload struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func decodeAdmissionErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder) struct {
	Success bool `json:"success"`
	Error   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
} {
	t.Helper()
	var body struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body), recorder.Body.String())
	return body
}

func enableBUAAAdmissionSchool(t *testing.T, fixture *postgresfixture.Fixture) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE school_configs
		SET enabled = true, updated_at = NOW()
		WHERE school_id = 4111010006
	`)
	require.NoError(t, err)
}
