package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/modules/rbac"
	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestStorageHandlers_RejectUnknownDriverAndCheckMountHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, config.ObjectStorageConfig{})
	svc.registry.drivers["s3"] = &fakeDriver{healthErr: errors.New("network timeout")}

	router := newStorageTestRouter(svc)
	createBody := marshalStorageRequest(t, CreateMountRequest{
		Key:      "bad-driver",
		Name:     "Bad Driver",
		Driver:   "webdav",
		BasePath: "resources",
		Enabled:  true,
	})

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/storage/mounts", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)
	require.Equal(t, http.StatusBadRequest, createResp.Code)

	created, err := svc.CreateMount(context.Background(), CreateMountRequest{
		Key:      "campus-share",
		Name:     "Campus Share",
		Driver:   "s3",
		BasePath: "resources",
		Enabled:  true,
	})
	require.NoError(t, err)

	healthReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/storage/mounts/"+strconv.FormatInt(created.ID, 10)+"/health-check", nil)
	healthResp := httptest.NewRecorder()
	router.ServeHTTP(healthResp, healthReq)
	require.Equal(t, http.StatusOK, healthResp.Code)

	var payload Mount
	decodeStorageData(t, healthResp.Body.Bytes(), &payload)
	require.NotNil(t, payload.LastHealthStatus)
	require.NotNil(t, payload.LastHealthError)
	assert.Equal(t, "unhealthy", *payload.LastHealthStatus)
	assert.Equal(t, "network timeout", *payload.LastHealthError)
}

func TestStorageHandlers_Return404ForMissingMountHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	svc := NewService(NewRepository(fixture.DB), config.ObjectStorageConfig{})
	router := newStorageTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/storage/mounts/999999/health-check", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
	var envelope response.Response
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &envelope))
	require.NotNil(t, envelope.Error)
	assert.Equal(t, "storage mount not found", envelope.Error.Message)
}

func TestStorageHandlers_Return400ForInvalidMountConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	svc := NewService(NewRepository(fixture.DB), config.ObjectStorageConfig{})
	router := newStorageTestRouter(svc)

	createBody := marshalStorageRequest(t, CreateMountRequest{
		Key:      " ",
		Name:     "Blank Key",
		Driver:   "s3",
		BasePath: "resources",
		Enabled:  true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/storage/mounts", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestStorageHandlers_Return409ForDuplicateMountKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	svc := NewService(NewRepository(fixture.DB), config.ObjectStorageConfig{})
	svc.registry.drivers["s3"] = &fakeDriver{}
	router := newStorageTestRouter(svc)

	createBody := marshalStorageRequest(t, CreateMountRequest{
		Key:     "duplicate-key",
		Name:    "Duplicate Key",
		Driver:  "s3",
		Enabled: true,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/storage/mounts", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusCreated, resp.Code)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/storage/mounts", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusConflict, resp.Code)
	var envelope response.Response
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &envelope))
	require.NotNil(t, envelope.Error)
	assert.Equal(t, "storage mount already exists", envelope.Error.Message)
}

func TestStorageAdminRoutesRequireGlobalSystemCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	NewHandler(&Service{}, WithAdminAuthorizers(storageAdminAuthorizers())).RegisterAdminRoutes(api, scopedSystemCapabilityMiddleware())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/storage/mounts", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusForbidden, resp.Code)
}

func TestNewHandlerPanicsWhenServiceNil(t *testing.T) {
	assert.PanicsWithValue(t, "storage.NewHandler: service must not be nil", func() {
		NewHandler(nil)
	})
}

func newStorageTestRouter(svc *Service) *gin.Engine {
	router := gin.New()
	api := router.Group("/api/v1")
	handler := NewHandler(svc, WithAdminAuthorizers(storageAdminAuthorizers()))
	handler.RegisterAdminRoutes(api, storageAdminMiddleware())
	return router
}

func storageAdminAuthorizers() AdminAuthorizers {
	return AdminAuthorizers{
		Read:   rbac.RequireGlobalCapability(capability.UserSystemRead),
		Update: rbac.RequireGlobalCapability(capability.UserSystemUpdate),
	}
}

func storageAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		snapshot := capability.BuildUserAccessSnapshot([]capability.Grant{
			{Name: capability.UserSystemRead},
			{Name: capability.UserSystemUpdate},
		})
		c.Set(middleware.CtxKeyUserID, "admin-user")
		c.Set(middleware.CtxKeyCapabilities, snapshot.Capabilities)
		c.Set(middleware.CtxKeyGlobalCapabilities, snapshot.GlobalCapabilities)
		c.Set(middleware.CtxKeyCapabilityGrants, snapshot.CapabilityGrants)
		c.Set(middleware.CtxKeyCapabilitySet, capabilitySet(snapshot.Capabilities))
		c.Next()
	}
}

func scopedSystemCapabilityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		snapshot := capability.BuildUserAccessSnapshot([]capability.Grant{{
			Name:           capability.UserSystemRead,
			ScopeSchoolIDs: []string{"4111010006"},
		}})
		c.Set(middleware.CtxKeyUserID, "scoped-admin")
		c.Set(middleware.CtxKeyCapabilities, snapshot.Capabilities)
		c.Set(middleware.CtxKeyGlobalCapabilities, snapshot.GlobalCapabilities)
		c.Set(middleware.CtxKeyCapabilityGrants, snapshot.CapabilityGrants)
		c.Set(middleware.CtxKeyCapabilitySet, capabilitySet(snapshot.Capabilities))
		c.Next()
	}
}

func capabilitySet(capabilities []string) map[string]struct{} {
	capSet := make(map[string]struct{}, len(capabilities))
	for _, capName := range capabilities {
		capSet[capName] = struct{}{}
	}
	return capSet
}

func marshalStorageRequest(t *testing.T, req CreateMountRequest) []byte {
	t.Helper()
	body, err := json.Marshal(req)
	require.NoError(t, err)
	return body
}

func decodeStorageData(t *testing.T, body []byte, target any) {
	t.Helper()
	var envelope response.Response
	require.NoError(t, json.Unmarshal(body, &envelope))
	raw, err := json.Marshal(envelope.Data)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, target))
}
