package resource

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/objectstorage"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

func TestResourceHandlers_CreatePrivateResourceAndProtectDownload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, _, _, svc, store := setupResourceService(t)

	router := newResourceTestRouter(svc)
	body := marshalResourceRequest(t, CreateRequest{
		Title:       "Private Notes",
		Visibility:  "private",
		Tags:        []string{"private"},
		Bindings:    []Binding{{Type: "course", Value: "CS101"}},
		Filename:    "private.txt",
		ContentType: "text/plain; charset=utf-8",
		DataBase64:  base64.StdEncoding.EncodeToString([]byte("secret resource")),
	})

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("X-User-ID", "oidc-user-1")
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)
	require.Equal(t, http.StatusCreated, createResp.Code)

	created := decodeResourceItemResponse(t, createResp.Body.Bytes())
	assert.Equal(t, "private", created.Visibility)

	downloadReq := httptest.NewRequest(http.MethodGet, "/api/v1/resources/"+strconv.FormatInt(created.ID, 10)+"/download-url", nil)
	downloadReq.Header.Set("X-User-ID", "oidc-user-1")
	downloadResp := httptest.NewRecorder()
	router.ServeHTTP(downloadResp, downloadReq)
	require.Equal(t, http.StatusOK, downloadResp.Code)
	assert.Equal(t, store.downloadURL, decodeURLResponse(t, downloadResp.Body.Bytes()))

	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/v1/resources/"+strconv.FormatInt(created.ID, 10)+"/download-url", nil)
	forbiddenReq.Header.Set("X-User-ID", "oidc-user-2")
	forbiddenResp := httptest.NewRecorder()
	router.ServeHTTP(forbiddenResp, forbiddenReq)
	require.Equal(t, http.StatusNotFound, forbiddenResp.Code)
}

func TestResourceHandlers_MapObjectStorageNetworkFailureTo503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, _, _, svc, store := setupResourceService(t)
	store.putErr = &objectstorage.StoreError{
		Kind:     objectstorage.ErrorKindNetwork,
		Op:       "upload",
		Resource: "resources/oidc-user-1/network.txt",
		Err:      context.DeadlineExceeded,
	}

	router := newResourceTestRouter(svc)
	body := marshalResourceRequest(t, CreateRequest{
		Title:       "Broken Upload",
		Visibility:  "public",
		Filename:    "network.txt",
		ContentType: "text/plain; charset=utf-8",
		DataBase64:  base64.StdEncoding.EncodeToString([]byte("payload")),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/resources", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "oidc-user-1")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	parsed := decodeEnvelope(t, resp.Body.Bytes())
	require.NotNil(t, parsed.Error)
	assert.Equal(t, "resource storage is temporarily unavailable", parsed.Error.Message)
}

func TestResourceHandlers_MapUnknownCreateFailureTo500WithoutLeakingDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, _, _, svc, store := setupResourceService(t)
	store.putErr = errors.New("storage driver exploded")

	router := newResourceTestRouter(svc)
	body := marshalResourceRequest(t, CreateRequest{
		Title:       "Broken Upload",
		Visibility:  "public",
		Filename:    "broken.txt",
		ContentType: "text/plain",
		DataBase64:  base64.StdEncoding.EncodeToString([]byte("payload")),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/resources", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "oidc-user-1")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
	parsed := decodeEnvelope(t, resp.Body.Bytes())
	require.NotNil(t, parsed.Error)
	assert.Equal(t, "failed to create resource", parsed.Error.Message)
	assert.NotContains(t, resp.Body.String(), "storage driver exploded")
}

func TestResourceHandlers_OptionalAuthBackendFailureReturns503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, _, _, svc, _ := setupResourceService(t)

	router := gin.New()
	api := router.Group("/api/v1")
	handler := NewHandler(svc)
	handler.RegisterRoutes(api, resourceAuthMiddleware(), func(c *gin.Context) {
		c.Set(middleware.CtxKeyAuthBackendFailure, true)
		c.Next()
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	assert.Contains(t, resp.Body.String(), string(errs.ErrServiceUnavailable))
}

func newResourceTestRouter(svc *Service) *gin.Engine {
	router := gin.New()
	api := router.Group("/api/v1")
	handler := NewHandler(svc)
	handler.RegisterRoutes(api, resourceAuthMiddleware(), resourceOptionalAuthMiddleware())
	return router
}

func resourceAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Set(middleware.CtxKeyUserID, userID)
		c.Next()
	}
}

func resourceOptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if userID := c.GetHeader("X-User-ID"); userID != "" {
			c.Set(middleware.CtxKeyUserID, userID)
		}
		c.Next()
	}
}

func marshalResourceRequest(t *testing.T, req CreateRequest) []byte {
	t.Helper()
	body, err := json.Marshal(req)
	require.NoError(t, err)
	return body
}

func decodeResourceItemResponse(t *testing.T, body []byte) Item {
	t.Helper()
	var envelope response.Response
	require.NoError(t, json.Unmarshal(body, &envelope))
	raw, err := json.Marshal(envelope.Data)
	require.NoError(t, err)
	var item Item
	require.NoError(t, json.Unmarshal(raw, &item))
	return item
}

func decodeURLResponse(t *testing.T, body []byte) string {
	t.Helper()
	var envelope response.Response
	require.NoError(t, json.Unmarshal(body, &envelope))
	raw, err := json.Marshal(envelope.Data)
	require.NoError(t, err)
	var payload struct {
		URL string `json:"url"`
	}
	require.NoError(t, json.Unmarshal(raw, &payload))
	return payload.URL
}

func decodeEnvelope(t *testing.T, body []byte) response.Response {
	t.Helper()
	var envelope response.Response
	require.NoError(t, json.Unmarshal(body, &envelope))
	return envelope
}
