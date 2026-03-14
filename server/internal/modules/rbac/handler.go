package rbac

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// Handler RBAC HTTP 处理器
type Handler struct {
	service *Service
}

// NewHandler 创建 RBAC 处理器
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterAdminRoutes 注册 RBAC 管理路由（调用方负责挂载到 /admin 组并添加鉴权中间件）
func (h *Handler) RegisterAdminRoutes(admin *gin.RouterGroup) {
	// 角色管理
	admin.GET("/roles", h.handleListRoles)
	admin.POST("/roles", h.handleCreateRole)
	admin.PUT("/roles/:id", h.handleUpdateRole)
	admin.DELETE("/roles/:id", h.handleDeleteRole)
	admin.PUT("/roles/:id/permissions", h.handleSetRolePermissions)

	// 权限列表
	admin.GET("/permissions", h.handleListPermissions)

	// 用户角色/权限管理
	admin.GET("/users/:userId/roles", h.handleGetUserRoles)
	admin.PUT("/users/:userId/roles", h.handleSetUserRoles)
	admin.GET("/users/:userId/permissions", h.handleGetUserPermissions)
	admin.PUT("/users/:userId/permissions", h.handleSetUserPermission)

	// 用户组管理
	admin.GET("/groups", h.handleListGroups)
	admin.POST("/groups", h.handleCreateGroup)
	admin.PUT("/groups/:id", h.handleUpdateGroup)
	admin.DELETE("/groups/:id", h.handleDeleteGroup)
	admin.GET("/groups/:id/members", h.handleGetGroupMembers)
	admin.PUT("/groups/:id/members", h.handleSetGroupMembers)
	admin.PUT("/groups/:id/permissions", h.handleSetGroupPermissions)
}

// ---------------------------------------------------------------------------
// Role endpoints
// ---------------------------------------------------------------------------

// handleListRoles 获取角色列表
func (h *Handler) handleListRoles(c *gin.Context) {
	roles, err := h.service.ListRoles(c.Request.Context())
	if err != nil {
		logger.FromGin(c).Error("failed to list roles", zap.Error(err))
		response.InternalError(c, "failed to list roles")
		return
	}
	response.Success(c, gin.H{"roles": roles})
}

type createRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"displayName" binding:"required"`
	Description string `json:"description"`
}

// handleCreateRole 创建角色
func (h *Handler) handleCreateRole(c *gin.Context) {
	var req createRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	role, err := h.service.CreateRole(c.Request.Context(), req.Name, req.DisplayName, req.Description)
	if err != nil {
		if errors.Is(err, ErrRoleNameTaken) {
			response.Conflict(c, "role name already taken", errs.ErrRoleNameTaken)
			return
		}
		logger.FromGin(c).Error("failed to create role", zap.Error(err))
		response.InternalError(c, "failed to create role")
		return
	}
	response.Created(c, gin.H{"role": role})
}

type updateRoleRequest struct {
	DisplayName string `json:"displayName" binding:"required"`
	Description string `json:"description"`
}

// handleUpdateRole 更新角色
func (h *Handler) handleUpdateRole(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid role ID")
		return
	}

	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	role, err := h.service.UpdateRole(c.Request.Context(), id, req.DisplayName, req.Description)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			response.NotFound(c, "role not found", errs.ErrRoleNotFound)
			return
		}
		if errors.Is(err, ErrRoleIsSystem) {
			response.Forbidden(c, "system role cannot be modified", errs.ErrRoleIsSystem)
			return
		}
		logger.FromGin(c).Error("failed to update role", zap.Error(err))
		response.InternalError(c, "failed to update role")
		return
	}
	response.Success(c, gin.H{"role": role})
}

// handleDeleteRole 删除角色
func (h *Handler) handleDeleteRole(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid role ID")
		return
	}

	err = h.service.DeleteRole(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			response.NotFound(c, "role not found", errs.ErrRoleNotFound)
			return
		}
		if errors.Is(err, ErrRoleIsSystem) {
			response.Forbidden(c, "system role cannot be deleted", errs.ErrRoleIsSystem)
			return
		}
		logger.FromGin(c).Error("failed to delete role", zap.Error(err))
		response.InternalError(c, "failed to delete role")
		return
	}
	response.Success(c, gin.H{"message": "role deleted"})
}

type setRolePermissionsRequest struct {
	PermissionIDs []int64 `json:"permissionIds" binding:"required"`
}

// handleSetRolePermissions 设置角色权限
func (h *Handler) handleSetRolePermissions(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid role ID")
		return
	}

	var req setRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	err = h.service.SetRolePermissions(c.Request.Context(), id, req.PermissionIDs)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			response.NotFound(c, "role not found", errs.ErrRoleNotFound)
			return
		}
		logger.FromGin(c).Error("failed to set role permissions", zap.Error(err))
		response.InternalError(c, "failed to set role permissions")
		return
	}
	response.Success(c, gin.H{"message": "role permissions updated"})
}

// ---------------------------------------------------------------------------
// Permission endpoints
// ---------------------------------------------------------------------------

// handleListPermissions 获取权限列表
func (h *Handler) handleListPermissions(c *gin.Context) {
	module := c.Query("module")
	perms, err := h.service.ListPermissions(c.Request.Context(), module)
	if err != nil {
		logger.FromGin(c).Error("failed to list permissions", zap.Error(err))
		response.InternalError(c, "failed to list permissions")
		return
	}
	response.Success(c, gin.H{"permissions": perms})
}

// ---------------------------------------------------------------------------
// User role/permission endpoints
// ---------------------------------------------------------------------------

// handleGetUserRoles 获取用户角色
func (h *Handler) handleGetUserRoles(c *gin.Context) {
	userID, err := h.resolveUserID(c, "userId")
	if err != nil {
		return // response already sent
	}

	roles, err := h.service.GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		logger.FromGin(c).Error("failed to get user roles", zap.Error(err))
		response.InternalError(c, "failed to get user roles")
		return
	}
	response.Success(c, gin.H{"roles": roles})
}

type setUserRolesRequest struct {
	RoleIDs []int64 `json:"roleIds" binding:"required"`
}

// handleSetUserRoles 设置用户角色
func (h *Handler) handleSetUserRoles(c *gin.Context) {
	userID, err := h.resolveUserID(c, "userId")
	if err != nil {
		return
	}

	var req setUserRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	err = h.service.SetUserRoles(c.Request.Context(), userID, req.RoleIDs)
	if err != nil {
		logger.FromGin(c).Error("failed to set user roles", zap.Error(err))
		response.InternalError(c, "failed to set user roles")
		return
	}
	response.Success(c, gin.H{"message": "user roles updated"})
}

// handleGetUserPermissions 获取用户最终生效权限
func (h *Handler) handleGetUserPermissions(c *gin.Context) {
	userID, err := h.resolveUserID(c, "userId")
	if err != nil {
		return
	}

	perms, err := h.service.GetEffectivePermissions(c.Request.Context(), userID)
	if err != nil {
		logger.FromGin(c).Error("failed to get user permissions", zap.Error(err))
		response.InternalError(c, "failed to get user permissions")
		return
	}
	response.Success(c, gin.H{"permissions": perms})
}

type setUserPermissionRequest struct {
	PermissionID int64 `json:"permissionId" binding:"required"`
	Granted      bool  `json:"granted"`
}

// handleSetUserPermission 设置用户个人权限覆盖
func (h *Handler) handleSetUserPermission(c *gin.Context) {
	userID, err := h.resolveUserID(c, "userId")
	if err != nil {
		return
	}

	var req setUserPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	err = h.service.SetUserPermission(c.Request.Context(), userID, req.PermissionID, req.Granted)
	if err != nil {
		if errors.Is(err, ErrPermNotFound) {
			response.NotFound(c, "permission not found", errs.ErrPermissionNotFound)
			return
		}
		logger.FromGin(c).Error("failed to set user permission", zap.Error(err))
		response.InternalError(c, "failed to set user permission")
		return
	}
	response.Success(c, gin.H{"message": "user permission updated"})
}

// ---------------------------------------------------------------------------
// Group endpoints
// ---------------------------------------------------------------------------

// handleListGroups 获取用户组列表
func (h *Handler) handleListGroups(c *gin.Context) {
	groups, err := h.service.ListGroups(c.Request.Context())
	if err != nil {
		logger.FromGin(c).Error("failed to list groups", zap.Error(err))
		response.InternalError(c, "failed to list groups")
		return
	}
	response.Success(c, gin.H{"groups": groups})
}

type createGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"displayName" binding:"required"`
	Description string `json:"description"`
}

// handleCreateGroup 创建用户组
func (h *Handler) handleCreateGroup(c *gin.Context) {
	var req createGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	// 获取管理员内部用户 ID
	adminExternalID := middleware.GetUserID(c)
	adminUserID, err := h.service.GetInternalUserID(c.Request.Context(), adminExternalID)
	if err != nil {
		logger.FromGin(c).Error("failed to resolve admin user ID", zap.Error(err))
		response.InternalError(c, "failed to create group")
		return
	}

	group, err := h.service.CreateGroup(c.Request.Context(), req.Name, req.DisplayName, req.Description, adminUserID)
	if err != nil {
		if errors.Is(err, ErrGroupNameTaken) {
			response.Conflict(c, "group name already taken", errs.ErrGroupNameTaken)
			return
		}
		logger.FromGin(c).Error("failed to create group", zap.Error(err))
		response.InternalError(c, "failed to create group")
		return
	}
	response.Created(c, gin.H{"group": group})
}

type updateGroupRequest struct {
	DisplayName string `json:"displayName" binding:"required"`
	Description string `json:"description"`
}

// handleUpdateGroup 更新用户组
func (h *Handler) handleUpdateGroup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid group ID")
		return
	}

	var req updateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	group, err := h.service.UpdateGroup(c.Request.Context(), id, req.DisplayName, req.Description)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			response.NotFound(c, "group not found", errs.ErrGroupNotFound)
			return
		}
		logger.FromGin(c).Error("failed to update group", zap.Error(err))
		response.InternalError(c, "failed to update group")
		return
	}
	response.Success(c, gin.H{"group": group})
}

// handleDeleteGroup 删除用户组
func (h *Handler) handleDeleteGroup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid group ID")
		return
	}

	err = h.service.DeleteGroup(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			response.NotFound(c, "group not found", errs.ErrGroupNotFound)
			return
		}
		logger.FromGin(c).Error("failed to delete group", zap.Error(err))
		response.InternalError(c, "failed to delete group")
		return
	}
	response.Success(c, gin.H{"message": "group deleted"})
}

// handleGetGroupMembers 获取用户组成员
func (h *Handler) handleGetGroupMembers(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid group ID")
		return
	}

	members, err := h.service.GetGroupMembers(c.Request.Context(), id)
	if err != nil {
		logger.FromGin(c).Error("failed to get group members", zap.Error(err))
		response.InternalError(c, "failed to get group members")
		return
	}
	response.Success(c, gin.H{"memberIds": members})
}

type setGroupMembersRequest struct {
	UserIDs []int64 `json:"userIds" binding:"required"`
}

// handleSetGroupMembers 设置用户组成员
func (h *Handler) handleSetGroupMembers(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid group ID")
		return
	}

	var req setGroupMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	err = h.service.SetGroupMembers(c.Request.Context(), id, req.UserIDs)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			response.NotFound(c, "group not found", errs.ErrGroupNotFound)
			return
		}
		logger.FromGin(c).Error("failed to set group members", zap.Error(err))
		response.InternalError(c, "failed to set group members")
		return
	}
	response.Success(c, gin.H{"message": "group members updated"})
}

type setGroupPermissionsRequest struct {
	PermissionIDs []int64 `json:"permissionIds" binding:"required"`
}

// handleSetGroupPermissions 设置用户组权限
func (h *Handler) handleSetGroupPermissions(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid group ID")
		return
	}

	var req setGroupPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	err = h.service.SetGroupPermissions(c.Request.Context(), id, req.PermissionIDs)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			response.NotFound(c, "group not found", errs.ErrGroupNotFound)
			return
		}
		logger.FromGin(c).Error("failed to set group permissions", zap.Error(err))
		response.InternalError(c, "failed to set group permissions")
		return
	}
	response.Success(c, gin.H{"message": "group permissions updated"})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// resolveUserID 从路由参数解析内部用户 ID（参数值为内部 BIGINT user ID）
func (h *Handler) resolveUserID(c *gin.Context, paramName string) (int64, error) {
	userID, err := strconv.ParseInt(c.Param(paramName), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "invalid user ID")
		return 0, fmt.Errorf("invalid user ID param: %s", c.Param(paramName))
	}
	return userID, nil
}
