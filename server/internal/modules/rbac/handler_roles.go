package rbac

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// handleListRoles 获取角色列表
func (h *Handler) handleListRoles(c *gin.Context) {
	roles, err := h.service.ListRoles(c.Request.Context())
	if err != nil {
		logger.FromGin(c).Error("failed to list roles", zap.Error(err))
		response.InternalError(c, "failed to list roles")
		return
	}
	response.Success(c, roles)
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
	response.Created(c, role)
}

type updateRoleRequest struct {
	DisplayName *string `json:"displayName"`
	Description *string `json:"description"`
}

// handleUpdateRole 更新角色
func (h *Handler) handleUpdateRole(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("roleID"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid role ID")
		return
	}

	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	role, err := h.service.UpdateRole(c.Request.Context(), id, UpdateRoleInput(req))
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
	response.Success(c, role)
}

// handleDeleteRole 删除角色
func (h *Handler) handleDeleteRole(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("roleID"), 10, 64)
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
	PermissionIDs *[]int64 `json:"permissionIDs"`
	ClearAll      bool     `json:"clearAll"`
}

// handleGetRolePermissions 获取角色已分配权限 ID 列表
func (h *Handler) handleGetRolePermissions(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("roleID"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid role ID")
		return
	}

	permissionIDs, err := h.service.GetRolePermissionIDs(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			response.NotFound(c, "role not found", errs.ErrRoleNotFound)
			return
		}
		logger.FromGin(c).Error("failed to get role permissions", zap.Error(err))
		response.InternalError(c, "failed to get role permissions")
		return
	}

	response.Success(c, gin.H{"permissionIDs": permissionIDs})
}

// handleSetRolePermissions 设置角色权限
func (h *Handler) handleSetRolePermissions(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("roleID"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid role ID")
		return
	}

	var req setRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	if req.PermissionIDs == nil {
		response.BadRequest(c, "permissionIDs is required")
		return
	}

	err = h.service.SetRolePermissions(c.Request.Context(), id, *req.PermissionIDs, req.ClearAll)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			response.NotFound(c, "role not found", errs.ErrRoleNotFound)
			return
		}
		if errors.Is(err, ErrRoleIsSystem) {
			response.Forbidden(c, "system role cannot be modified", errs.ErrRoleIsSystem)
			return
		}
		if errors.Is(err, ErrPermissionSelectionInvalid) {
			response.BadRequest(c, "one or more selected permissions are invalid", errs.ErrPermissionSelectionInvalid)
			return
		}
		if errors.Is(err, ErrRolePermissionClearConfirmRequired) {
			response.BadRequest(c, "clearing all permissions requires explicit confirmation", errs.ErrRolePermissionClearConfirm)
			return
		}
		logger.FromGin(c).Error("failed to set role permissions", zap.Error(err))
		response.InternalError(c, "failed to set role permissions")
		return
	}
	response.Success(c, gin.H{"message": "role permissions updated"})
}

// handleListPermissions 获取权限列表
func (h *Handler) handleListPermissions(c *gin.Context) {
	module := c.Query("module")
	perms, err := h.service.ListPermissions(c.Request.Context(), module)
	if err != nil {
		logger.FromGin(c).Error("failed to list permissions", zap.Error(err))
		response.InternalError(c, "failed to list permissions")
		return
	}
	response.Success(c, perms)
}
