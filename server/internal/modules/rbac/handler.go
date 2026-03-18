package rbac

import (
	"context"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
)

// HandlerService defines the RBAC service methods used by HTTP handlers.
type HandlerService interface {
	ListRoles(ctx context.Context) ([]Role, error)
	CreateRole(ctx context.Context, name, displayName, description string) (*Role, error)
	UpdateRole(ctx context.Context, id int64, input UpdateRoleInput) (*Role, error)
	DeleteRole(ctx context.Context, id int64) error
	GetRolePermissionIDs(ctx context.Context, roleID int64) ([]int64, error)
	SetRolePermissions(ctx context.Context, roleID int64, permIDs []int64, clearAll bool) error
	ListPermissions(ctx context.Context, module string) ([]Permission, error)
	GetUserRoles(ctx context.Context, userID int64) ([]Role, error)
	SetUserRoles(ctx context.Context, userID int64, roleIDs []int64) error
	GetEffectivePermissions(ctx context.Context, userID int64) ([]EffectivePermission, error)
	SetUserPermission(ctx context.Context, userID int64, permID int64, granted bool) error
	ListGroups(ctx context.Context) ([]UserGroup, error)
	CreateGroup(ctx context.Context, name, displayName, desc string, createdBy int64) (*UserGroup, error)
	UpdateGroup(ctx context.Context, id int64, input UpdateGroupInput) (*UserGroup, error)
	DeleteGroup(ctx context.Context, id int64) error
	GetGroupMembers(ctx context.Context, groupID int64) ([]GroupMember, error)
	SetGroupMembers(ctx context.Context, groupID int64, userIDs []int64) error
	SetGroupPermissions(ctx context.Context, groupID int64, permIDs []int64) error
	GetInternalUserID(ctx context.Context, externalID string) (int64, error)
}

// Handler RBAC HTTP 处理器
type Handler struct {
	service HandlerService
}

// NewHandler 创建 RBAC 处理器
func NewHandler(service HandlerService) *Handler {
	return &Handler{service: service}
}

// RegisterAdminRoutes 注册 RBAC 管理路由（调用方负责挂载到 /admin 组并添加鉴权中间件）
func (h *Handler) RegisterAdminRoutes(admin *gin.RouterGroup, rbacService PermissionService) {
	admin.GET("/roles", RequirePermission(rbacService, capability.RBACRoleRead), h.handleListRoles)
	admin.POST("/roles", RequirePermission(rbacService, capability.RBACRoleCreate), h.handleCreateRole)
	admin.PUT("/roles/:roleID", RequirePermission(rbacService, capability.RBACRoleUpdate), h.handleUpdateRole)
	admin.DELETE("/roles/:roleID", RequirePermission(rbacService, capability.RBACRoleDelete), h.handleDeleteRole)
	admin.GET("/roles/:roleID/permissions", RequirePermission(rbacService, capability.RBACRoleRead), h.handleGetRolePermissions)
	admin.PUT("/roles/:roleID/permissions", RequirePermission(rbacService, capability.RBACRoleUpdate), h.handleSetRolePermissions)

	admin.GET("/permissions", RequirePermission(rbacService, capability.RBACPermissionRead), h.handleListPermissions)

	admin.GET("/users/:userID/roles", RequirePermission(rbacService, capability.RBACUserRead), h.handleGetUserRoles)
	admin.PUT("/users/:userID/roles", RequirePermission(rbacService, capability.RBACUserUpdate), h.handleSetUserRoles)
	admin.GET("/users/:userID/permissions", RequirePermission(rbacService, capability.RBACUserRead), h.handleGetUserPermissions)
	admin.PUT("/users/:userID/permissions", RequirePermission(rbacService, capability.RBACUserUpdate), h.handleSetUserPermission)

	admin.GET("/groups", RequirePermission(rbacService, capability.RBACGroupRead), h.handleListGroups)
	admin.POST("/groups", RequirePermission(rbacService, capability.RBACGroupCreate), h.handleCreateGroup)
	admin.PUT("/groups/:groupID", RequirePermission(rbacService, capability.RBACGroupUpdate), h.handleUpdateGroup)
	admin.DELETE("/groups/:groupID", RequirePermission(rbacService, capability.RBACGroupDelete), h.handleDeleteGroup)
	admin.GET("/groups/:groupID/members", RequirePermission(rbacService, capability.RBACGroupRead), h.handleGetGroupMembers)
	admin.PUT("/groups/:groupID/members", RequirePermission(rbacService, capability.RBACGroupUpdate), h.handleSetGroupMembers)
	admin.PUT("/groups/:groupID/permissions", RequirePermission(rbacService, capability.RBACGroupUpdate), h.handleSetGroupPermissions)
}
