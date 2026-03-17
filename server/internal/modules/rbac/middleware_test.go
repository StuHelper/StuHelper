package rbac

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	appmiddleware "gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

// ---------------------------------------------------------------------------
// RequireAnyPermission
// ---------------------------------------------------------------------------

func TestRequireAnyPermission_AllowsWhenUserHasAdminCapability(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-user-123")
		c.Next()
	})
	r.GET("/admin", RequireAnyPermission(&fakeHandlerService{
		onGetInternalUserID: func(_ context.Context, externalID string) (int64, error) {
			assert.Equal(t, "external-user-123", externalID)
			return 42, nil
		},
		onGetEffectivePerms: func(_ context.Context, userID int64) ([]EffectivePermission, error) {
			assert.Equal(t, int64(42), userID)
			return []EffectivePermission{
				{Name: capability.AdminLogsView, Granted: true},
			}, nil
		},
	}, capability.AdminEntryCapabilities...), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestRequireAnyPermission_RejectsWhenUserHasNoAdminCapability(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-user-123")
		c.Next()
	})
	r.GET("/admin", RequireAnyPermission(&fakeHandlerService{
		onGetInternalUserID: func(_ context.Context, _ string) (int64, error) { return 42, nil },
		onGetEffectivePerms: func(_ context.Context, _ int64) ([]EffectivePermission, error) {
			return []EffectivePermission{
				{Name: "review:create", Granted: true},
			}, nil
		},
	}, capability.AdminEntryCapabilities...), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireAnyPermission_Returns500WhenLookupFails(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-user-123")
		c.Next()
	})
	r.GET("/admin", RequireAnyPermission(&fakeHandlerService{
		onGetInternalUserID: func(_ context.Context, _ string) (int64, error) { return 42, nil },
		onGetEffectivePerms: func(_ context.Context, _ int64) ([]EffectivePermission, error) {
			return nil, errors.New("lookup failed")
		},
	}, capability.AdminEntryCapabilities...), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// RequirePermission — 复用 RequireAnyPermission 缓存
// ---------------------------------------------------------------------------

func TestRequirePermission_ReusesCache(t *testing.T) {
	idCalls := 0
	permsCalls := 0

	svc := &fakeHandlerService{
		onGetInternalUserID: func(_ context.Context, _ string) (int64, error) {
			idCalls++
			return 42, nil
		},
		onGetEffectivePerms: func(_ context.Context, _ int64) ([]EffectivePermission, error) {
			permsCalls++
			return []EffectivePermission{
				{PermissionID: 1, Name: capability.RBACRoleRead, Granted: true},
			}, nil
		},
		onCheckPermissionScope: func(_ context.Context, ep EffectivePermission, _ int64, _ *string) (bool, error) {
			assert.Equal(t, int64(1), ep.PermissionID)
			return true, nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-user-123")
		c.Next()
	})
	r.GET("/admin/roles",
		RequireAnyPermission(svc, capability.AdminEntryCapabilities...),
		RequirePermission(svc, capability.RBACRoleRead),
		func(c *gin.Context) { c.Status(http.StatusNoContent) },
	)

	req := httptest.NewRequest(http.MethodGet, "/admin/roles", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	// 关键断言：GetInternalUserID 和 GetEffectivePermissions 各只调用一次
	assert.Equal(t, 1, idCalls, "GetInternalUserID 应只调用一次")
	assert.Equal(t, 1, permsCalls, "GetEffectivePermissions 应只调用一次")
}

func TestRequirePermission_DeniesWhenPermissionNotGranted(t *testing.T) {
	svc := &fakeHandlerService{
		onGetInternalUserID: func(_ context.Context, _ string) (int64, error) { return 42, nil },
		onGetEffectivePerms: func(_ context.Context, _ int64) ([]EffectivePermission, error) {
			return []EffectivePermission{
				{Name: capability.RBACRoleRead, Granted: true},
			}, nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-user-123")
		c.Next()
	})
	r.DELETE("/admin/roles/1",
		RequirePermission(svc, capability.RBACRoleDelete),
		func(c *gin.Context) { c.Status(http.StatusNoContent) },
	)

	req := httptest.NewRequest(http.MethodDelete, "/admin/roles/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_Returns500WhenScopeCheckFails(t *testing.T) {
	svc := &fakeHandlerService{
		onGetInternalUserID: func(_ context.Context, _ string) (int64, error) { return 42, nil },
		onGetEffectivePerms: func(_ context.Context, _ int64) ([]EffectivePermission, error) {
			return []EffectivePermission{
				{PermissionID: 1, Name: capability.RBACRoleRead, Granted: true},
			}, nil
		},
		onCheckPermissionScope: func(_ context.Context, _ EffectivePermission, _ int64, _ *string) (bool, error) {
			return false, errors.New("repository down")
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-user-123")
		c.Next()
	})
	r.GET("/admin/roles",
		RequirePermission(svc, capability.RBACRoleRead),
		func(c *gin.Context) { c.Status(http.StatusNoContent) },
	)

	req := httptest.NewRequest(http.MethodGet, "/admin/roles", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
