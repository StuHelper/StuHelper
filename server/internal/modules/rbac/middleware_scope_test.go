package rbac

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appmiddleware "gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

func TestRequirePermission_UsesSchoolIDFromPathParam(t *testing.T) {
	var capturedSchoolID *string
	svc := &fakeHandlerService{
		onGetInternalUserID: func(_ context.Context, externalID string) (int64, error) {
			assert.Equal(t, "external-123", externalID)
			return 42, nil
		},
		onGetEffectivePerms: func(_ context.Context, userID int64) ([]EffectivePermission, error) {
			assert.Equal(t, int64(42), userID)
			return []EffectivePermission{
				{PermissionID: 1, Name: "admin.users.read", Granted: true},
			}, nil
		},
		onCheckPermissionScope: func(_ context.Context, ep EffectivePermission, userID int64, schoolID *string) (bool, error) {
			assert.Equal(t, int64(42), userID)
			assert.Equal(t, "admin.users.read", ep.Name)
			capturedSchoolID = schoolID
			return true, nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-123")
		c.Next()
	})
	r.GET("/guarded/:schoolID", RequirePermission(svc, "admin.users.read"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/guarded/10006", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	require.NotNil(t, capturedSchoolID)
	assert.Equal(t, "10006", *capturedSchoolID)
}

func TestRequirePermission_UsesSchoolIDFromJSONBodyAndKeepsBodyReadable(t *testing.T) {
	var capturedSchoolID *string
	svc := &fakeHandlerService{
		onGetInternalUserID: func(_ context.Context, externalID string) (int64, error) {
			assert.Equal(t, "external-123", externalID)
			return 42, nil
		},
		onGetEffectivePerms: func(_ context.Context, userID int64) ([]EffectivePermission, error) {
			assert.Equal(t, int64(42), userID)
			return []EffectivePermission{
				{PermissionID: 1, Name: "admin.users.read", Granted: true},
			}, nil
		},
		onCheckPermissionScope: func(_ context.Context, ep EffectivePermission, userID int64, schoolID *string) (bool, error) {
			assert.Equal(t, int64(42), userID)
			assert.Equal(t, "admin.users.read", ep.Name)
			capturedSchoolID = schoolID
			return true, nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-123")
		c.Next()
	})
	r.POST("/guarded", RequirePermission(svc, "admin.users.read"), func(c *gin.Context) {
		var payload struct {
			SchoolID string `json:"schoolID"`
		}
		err := c.ShouldBindJSON(&payload)
		require.NoError(t, err)
		assert.Equal(t, "10006", payload.SchoolID)
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/guarded", strings.NewReader(`{"schoolID":"10006"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	require.NotNil(t, capturedSchoolID)
	assert.Equal(t, "10006", *capturedSchoolID)
}

func TestRequirePermission_SchoolIDPriorityPathOverQueryAndBody(t *testing.T) {
	var capturedSchoolID *string
	svc := &fakeHandlerService{
		onGetInternalUserID: func(_ context.Context, _ string) (int64, error) { return 42, nil },
		onGetEffectivePerms: func(_ context.Context, _ int64) ([]EffectivePermission, error) {
			return []EffectivePermission{
				{PermissionID: 1, Name: "admin.users.read", Granted: true},
			}, nil
		},
		onCheckPermissionScope: func(_ context.Context, _ EffectivePermission, _ int64, schoolID *string) (bool, error) {
			capturedSchoolID = schoolID
			return true, nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-123")
		c.Next()
	})
	r.POST("/guarded/:schoolID", RequirePermission(svc, "admin.users.read"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/guarded/path-school?schoolID=query-school", strings.NewReader(`{"schoolID":"body-school"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	require.NotNil(t, capturedSchoolID)
	assert.Equal(t, "path-school", *capturedSchoolID)
}

func TestRequirePermission_SkipsLargeBodyExtractionAndKeepsBodyReadable(t *testing.T) {
	var capturedSchoolID *string
	svc := &fakeHandlerService{
		onGetInternalUserID: func(_ context.Context, _ string) (int64, error) { return 42, nil },
		onGetEffectivePerms: func(_ context.Context, _ int64) ([]EffectivePermission, error) {
			return []EffectivePermission{
				{PermissionID: 1, Name: "admin.users.read", Granted: true},
			}, nil
		},
		onCheckPermissionScope: func(_ context.Context, _ EffectivePermission, _ int64, schoolID *string) (bool, error) {
			capturedSchoolID = schoolID
			return true, nil
		},
	}

	largePadding := strings.Repeat("a", maxPermissionBodyRead+32)
	body := fmt.Sprintf(`{"schoolID":"10006","padding":%q}`, largePadding)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-123")
		c.Next()
	})
	r.POST("/guarded", RequirePermission(svc, "admin.users.read"), func(c *gin.Context) {
		var payload struct {
			SchoolID string `json:"schoolID"`
			Padding  string `json:"padding"`
		}
		err := c.ShouldBindJSON(&payload)
		require.NoError(t, err)
		assert.Equal(t, "10006", payload.SchoolID)
		assert.Equal(t, largePadding, payload.Padding)
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/guarded", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Nil(t, capturedSchoolID)
}

func TestRequirePermission_UsesTopLevelSchoolIDOnly(t *testing.T) {
	var capturedSchoolID *string
	svc := &fakeHandlerService{
		onGetInternalUserID: func(_ context.Context, _ string) (int64, error) { return 42, nil },
		onGetEffectivePerms: func(_ context.Context, _ int64) ([]EffectivePermission, error) {
			return []EffectivePermission{
				{PermissionID: 1, Name: "admin.users.read", Granted: true},
			}, nil
		},
		onCheckPermissionScope: func(_ context.Context, _ EffectivePermission, _ int64, schoolID *string) (bool, error) {
			capturedSchoolID = schoolID
			return true, nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-123")
		c.Next()
	})
	r.POST("/guarded", RequirePermission(svc, "admin.users.read"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/guarded", strings.NewReader(`{"schoolID":"top-level","nested":{"schoolID":"nested"}}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	require.NotNil(t, capturedSchoolID)
	assert.Equal(t, "top-level", *capturedSchoolID)
}
