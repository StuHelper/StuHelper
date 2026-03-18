package rbac

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
)

func TestHandleSetUserRoles_Returns404WhenUserNotFound(t *testing.T) {
	r := setupRBACAdminRouter(&fakeHandlerService{
		onSetUserRoles: func(_ context.Context, _ int64, _ []int64) error {
			return ErrUserNotFound
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/12/roles", strings.NewReader(`{"roleIDs":[5]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp rbacHandlerTestResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, string(errs.ErrUserNotFound), resp.Error.Code)
}

func TestHandleSetUserRoles_Returns400WhenRoleSelectionInvalid(t *testing.T) {
	r := setupRBACAdminRouter(&fakeHandlerService{
		onSetUserRoles: func(_ context.Context, _ int64, _ []int64) error {
			return ErrRoleSelectionInvalid
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/12/roles", strings.NewReader(`{"roleIDs":[0]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp rbacHandlerTestResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, string(errs.ErrRoleSelectionInvalid), resp.Error.Code)
}

func TestHandleSetGroupMembers_Returns400WhenUserSelectionInvalid(t *testing.T) {
	r := setupRBACAdminRouter(&fakeHandlerService{
		onSetGroupMembers: func(_ context.Context, _ int64, _ []int64) error {
			return ErrUserSelectionInvalid
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/groups/21/members", strings.NewReader(`{"userIDs":[-1]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp rbacHandlerTestResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, string(errs.ErrUserSelectionInvalid), resp.Error.Code)
}

func TestHandleSetGroupPermissions_Returns400WhenPermissionSelectionInvalid(t *testing.T) {
	r := setupRBACAdminRouter(&fakeHandlerService{
		onSetGroupPermissions: func(_ context.Context, _ int64, _ []int64) error {
			return ErrPermissionSelectionInvalid
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/groups/3/permissions", strings.NewReader(`{"permissionIDs":[0]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp rbacHandlerTestResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, string(errs.ErrPermissionSelectionInvalid), resp.Error.Code)
}

func TestHandleGetUserRoles_Returns404WhenUserNotFound(t *testing.T) {
	r := setupRBACAdminRouter(&fakeHandlerService{
		onGetUserRoles: func(_ context.Context, _ int64) ([]Role, error) {
			return nil, ErrUserNotFound
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/12/roles", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp rbacHandlerTestResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, string(errs.ErrUserNotFound), resp.Error.Code)
}

func TestHandleGetUserPermissions_Returns404WhenUserNotFound(t *testing.T) {
	r := setupRBACAdminRouter(&fakeHandlerService{
		onGetEffectivePerms: func(_ context.Context, _ int64) ([]EffectivePermission, error) {
			return nil, ErrUserNotFound
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/12/permissions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp rbacHandlerTestResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, string(errs.ErrUserNotFound), resp.Error.Code)
}

func TestHandleGetGroupMembers_Returns404WhenGroupNotFound(t *testing.T) {
	r := setupRBACAdminRouter(&fakeHandlerService{
		onGetGroupMembers: func(_ context.Context, _ int64) ([]GroupMember, error) {
			return nil, ErrGroupNotFound
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups/21/members", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp rbacHandlerTestResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, string(errs.ErrGroupNotFound), resp.Error.Code)
}
