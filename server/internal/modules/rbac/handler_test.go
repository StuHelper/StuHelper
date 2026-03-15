package rbac

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appmiddleware "gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type rbacHandlerTestResponse struct {
	Success bool               `json:"success"`
	Error   *response.APIError `json:"error,omitempty"`
}

type fakeHandlerService struct {
	onUpdateRole           func(ctx context.Context, id int64, input UpdateRoleInput) (*Role, error)
	onGetRolePermissionIDs func(ctx context.Context, roleID int64) ([]int64, error)
	onSetRolePermissions   func(ctx context.Context, roleID int64, permIDs []int64, clearAll bool) error
	onSetUserRoles         func(ctx context.Context, userID int64, roleIDs []int64) error
	onSetUserPermission    func(ctx context.Context, userID int64, permID int64, granted bool) error
	onUpdateGroup          func(ctx context.Context, id int64, input UpdateGroupInput) (*UserGroup, error)
	onSetGroupMembers      func(ctx context.Context, groupID int64, userIDs []int64) error
	onSetGroupPermissions  func(ctx context.Context, groupID int64, permIDs []int64) error
	onGetInternalUserID    func(ctx context.Context, externalID string) (int64, error)
	onCheckPermission      func(ctx context.Context, userID int64, permName string, schoolID *string) (bool, error)
}

func (f *fakeHandlerService) ListRoles(context.Context) ([]Role, error) { return nil, nil }
func (f *fakeHandlerService) CreateRole(context.Context, string, string, string) (*Role, error) {
	return nil, nil
}
func (f *fakeHandlerService) UpdateRole(ctx context.Context, id int64, input UpdateRoleInput) (*Role, error) {
	if f.onUpdateRole != nil {
		return f.onUpdateRole(ctx, id, input)
	}
	return &Role{}, nil
}
func (f *fakeHandlerService) DeleteRole(context.Context, int64) error { return nil }
func (f *fakeHandlerService) GetRolePermissionIDs(ctx context.Context, roleID int64) ([]int64, error) {
	if f.onGetRolePermissionIDs != nil {
		return f.onGetRolePermissionIDs(ctx, roleID)
	}
	return nil, nil
}
func (f *fakeHandlerService) SetRolePermissions(ctx context.Context, roleID int64, permIDs []int64, clearAll bool) error {
	if f.onSetRolePermissions != nil {
		return f.onSetRolePermissions(ctx, roleID, permIDs, clearAll)
	}
	return nil
}
func (f *fakeHandlerService) ListPermissions(context.Context, string) ([]Permission, error) {
	return nil, nil
}
func (f *fakeHandlerService) GetUserRoles(context.Context, int64) ([]Role, error) { return nil, nil }
func (f *fakeHandlerService) SetUserRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	if f.onSetUserRoles != nil {
		return f.onSetUserRoles(ctx, userID, roleIDs)
	}
	return nil
}
func (f *fakeHandlerService) GetEffectivePermissions(context.Context, int64) ([]EffectivePermission, error) {
	return nil, nil
}
func (f *fakeHandlerService) SetUserPermission(ctx context.Context, userID int64, permID int64, granted bool) error {
	if f.onSetUserPermission != nil {
		return f.onSetUserPermission(ctx, userID, permID, granted)
	}
	return nil
}
func (f *fakeHandlerService) ListGroups(context.Context) ([]UserGroup, error) { return nil, nil }
func (f *fakeHandlerService) CreateGroup(context.Context, string, string, string, int64) (*UserGroup, error) {
	return nil, nil
}
func (f *fakeHandlerService) UpdateGroup(ctx context.Context, id int64, input UpdateGroupInput) (*UserGroup, error) {
	if f.onUpdateGroup != nil {
		return f.onUpdateGroup(ctx, id, input)
	}
	return &UserGroup{}, nil
}
func (f *fakeHandlerService) DeleteGroup(context.Context, int64) error { return nil }
func (f *fakeHandlerService) GetGroupMembers(context.Context, int64) ([]GroupMember, error) {
	return nil, nil
}
func (f *fakeHandlerService) SetGroupMembers(ctx context.Context, groupID int64, userIDs []int64) error {
	if f.onSetGroupMembers != nil {
		return f.onSetGroupMembers(ctx, groupID, userIDs)
	}
	return nil
}
func (f *fakeHandlerService) SetGroupPermissions(ctx context.Context, groupID int64, permIDs []int64) error {
	if f.onSetGroupPermissions != nil {
		return f.onSetGroupPermissions(ctx, groupID, permIDs)
	}
	return nil
}
func (f *fakeHandlerService) GetInternalUserID(ctx context.Context, externalID string) (int64, error) {
	if f.onGetInternalUserID != nil {
		return f.onGetInternalUserID(ctx, externalID)
	}
	return 0, nil
}
func (f *fakeHandlerService) CheckPermission(ctx context.Context, userID int64, permName string, schoolID *string) (bool, error) {
	if f.onCheckPermission != nil {
		return f.onCheckPermission(ctx, userID, permName, schoolID)
	}
	return true, nil
}

func setupRBACAdminRouter(service HandlerService) *gin.Engine {
	r := gin.New()
	api := r.Group("/api/v1")
	admin := api.Group("/admin")
	admin.Use(func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-user-123")
		c.Next()
	})
	permissionService, ok := service.(PermissionService)
	if !ok {
		panic("test handler service must implement PermissionService")
	}
	NewHandler(service).RegisterAdminRoutes(admin, permissionService)
	return r
}

func TestHandleSetRolePermissions_AcceptsPermissionIDs(t *testing.T) {
	var (
		captured []int64
		clearAll bool
	)
	r := setupRBACAdminRouter(&fakeHandlerService{
		onSetRolePermissions: func(_ context.Context, roleID int64, permIDs []int64, reqClearAll bool) error {
			assert.Equal(t, int64(7), roleID)
			captured = append([]int64(nil), permIDs...)
			clearAll = reqClearAll
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/roles/7/permissions", strings.NewReader(`{"permissionIDs":[1,2,3]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []int64{1, 2, 3}, captured)
	assert.False(t, clearAll)
}

func TestHandleGetRolePermissions_ReturnsPermissionIDs(t *testing.T) {
	r := setupRBACAdminRouter(&fakeHandlerService{
		onGetRolePermissionIDs: func(_ context.Context, roleID int64) ([]int64, error) {
			assert.Equal(t, int64(7), roleID)
			return []int64{2, 4, 8}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/roles/7/permissions", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"success":true,"data":{"permissionIDs":[2,4,8]}}`, w.Body.String())
}

func TestHandleSetRolePermissions_RequiresExplicitClearAllConfirmation(t *testing.T) {
	r := setupRBACAdminRouter(&fakeHandlerService{
		onSetRolePermissions: func(_ context.Context, roleID int64, permIDs []int64, clearAll bool) error {
			assert.Equal(t, int64(7), roleID)
			assert.Empty(t, permIDs)
			assert.False(t, clearAll)
			return ErrRolePermissionClearConfirmRequired
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/roles/7/permissions", strings.NewReader(`{"permissionIDs":[]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "clearing all permissions requires explicit confirmation")
}

func TestHandleSetRolePermissions_RequiresPermissionIDsField(t *testing.T) {
	r := setupRBACAdminRouter(&fakeHandlerService{})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/roles/7/permissions", strings.NewReader(`{"clearAll":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "permissionIDs is required")
}

func TestHandleSetRolePermissions_RejectsSystemRole(t *testing.T) {
	r := setupRBACAdminRouter(&fakeHandlerService{
		onSetRolePermissions: func(_ context.Context, roleID int64, permIDs []int64, clearAll bool) error {
			assert.Equal(t, int64(7), roleID)
			assert.Equal(t, []int64{1, 2}, permIDs)
			assert.False(t, clearAll)
			return ErrRoleIsSystem
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/roles/7/permissions", strings.NewReader(`{"permissionIDs":[1,2]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "system role cannot be modified")
}

func TestHandleUpdateRole_AllowsDescriptionOnlyPartialUpdate(t *testing.T) {
	var captured UpdateRoleInput
	r := setupRBACAdminRouter(&fakeHandlerService{
		onUpdateRole: func(_ context.Context, id int64, input UpdateRoleInput) (*Role, error) {
			assert.Equal(t, int64(7), id)
			captured = input
			return &Role{
				ID:          id,
				Name:        "moderator",
				DisplayName: "Moderator",
				IsSystem:    false,
			}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/roles/7", strings.NewReader(`{"description":"updated desc"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, captured.DisplayName)
	require.NotNil(t, captured.Description)
	assert.Equal(t, "updated desc", *captured.Description)
}

func TestHandleSetUserRoles_AcceptsRoleIDs(t *testing.T) {
	var captured []int64
	r := setupRBACAdminRouter(&fakeHandlerService{
		onSetUserRoles: func(_ context.Context, userID int64, roleIDs []int64) error {
			assert.Equal(t, int64(12), userID)
			captured = append([]int64(nil), roleIDs...)
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/12/roles", strings.NewReader(`{"roleIDs":[5,6]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []int64{5, 6}, captured)
}

func TestHandleSetUserPermission_AcceptsPermissionIDAndFalseGranted(t *testing.T) {
	var (
		capturedPermID  int64
		capturedGranted bool
	)
	r := setupRBACAdminRouter(&fakeHandlerService{
		onSetUserPermission: func(_ context.Context, userID int64, permID int64, granted bool) error {
			assert.Equal(t, int64(9), userID)
			capturedPermID = permID
			capturedGranted = granted
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/9/permissions", strings.NewReader(`{"permissionID":88,"granted":false}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, int64(88), capturedPermID)
	assert.False(t, capturedGranted)
}

func TestHandleSetGroupMembers_AcceptsUserIDs(t *testing.T) {
	var captured []int64
	r := setupRBACAdminRouter(&fakeHandlerService{
		onSetGroupMembers: func(_ context.Context, groupID int64, userIDs []int64) error {
			assert.Equal(t, int64(21), groupID)
			captured = append([]int64(nil), userIDs...)
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/groups/21/members", strings.NewReader(`{"userIDs":[10,11]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []int64{10, 11}, captured)
}

func TestHandleSetGroupPermissions_AcceptsPermissionIDs(t *testing.T) {
	var captured []int64
	r := setupRBACAdminRouter(&fakeHandlerService{
		onSetGroupPermissions: func(_ context.Context, groupID int64, permIDs []int64) error {
			assert.Equal(t, int64(3), groupID)
			captured = append([]int64(nil), permIDs...)
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/groups/3/permissions", strings.NewReader(`{"permissionIDs":[100]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []int64{100}, captured)
}

func TestHandleUpdateGroup_AllowsDescriptionOnlyPartialUpdate(t *testing.T) {
	var captured UpdateGroupInput
	r := setupRBACAdminRouter(&fakeHandlerService{
		onUpdateGroup: func(_ context.Context, id int64, input UpdateGroupInput) (*UserGroup, error) {
			assert.Equal(t, int64(21), id)
			captured = input
			return &UserGroup{
				ID:          id,
				Name:        "reviewers",
				DisplayName: "Reviewers",
			}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/groups/21", strings.NewReader(`{"description":"updated desc"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, captured.DisplayName)
	require.NotNil(t, captured.Description)
	assert.Equal(t, "updated desc", *captured.Description)
}

func TestRequirePermission_UsesSchoolIDQuery(t *testing.T) {
	var capturedSchoolID *string
	svc := &fakeHandlerService{
		onGetInternalUserID: func(_ context.Context, externalID string) (int64, error) {
			assert.Equal(t, "external-123", externalID)
			return 42, nil
		},
		onCheckPermission: func(_ context.Context, userID int64, permName string, schoolID *string) (bool, error) {
			assert.Equal(t, int64(42), userID)
			assert.Equal(t, "admin.users.read", permName)
			capturedSchoolID = schoolID
			return true, nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-123")
		c.Next()
	})
	r.GET("/guarded", RequirePermission(svc, "admin.users.read"), func(c *gin.Context) {
		response.Success(c, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/guarded?schoolID=CS001", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedSchoolID)
	assert.Equal(t, "CS001", *capturedSchoolID)
}

func TestHandleSetUserPermission_MissingGrantedReturns400(t *testing.T) {
	r := setupRBACAdminRouter(&fakeHandlerService{})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/9/permissions", strings.NewReader(`{"permissionID":88}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp rbacHandlerTestResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)
}
