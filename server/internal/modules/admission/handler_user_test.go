package admission

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
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
		"/api/v1/admission/sessions/"+tokenPath+"/link",
	)
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())
	firstBody := decodeAdmissionSessionResponse(t, first)
	assert.True(t, firstBody.Success)
	assert.Equal(t, StatusLinked, firstBody.Data.Status)
	assert.Equal(t, fmt.Sprint(userID), firstBody.Data.UserID)

	preview := performAdmissionHandlerRequest(
		router,
		http.MethodGet,
		"/api/v1/admission/sessions/"+tokenPath,
	)
	require.Equal(t, http.StatusConflict, preview.Code, preview.Body.String())
	previewBody := decodeAdmissionErrorResponse(t, preview)
	assert.False(t, previewBody.Success)
	assert.Equal(t, "admission.token_consumed", previewBody.Error.Code)

	resumed := performAdmissionHandlerRequest(
		router,
		http.MethodPost,
		"/api/v1/admission/sessions/"+tokenPath+"/link",
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
		"/api/v1/admission/sessions/missing-token",
	)
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
	body := decodeAdmissionErrorResponse(t, response)
	assert.False(t, body.Success)
	assert.Equal(t, "admission.token_not_found", body.Error.Code)
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
