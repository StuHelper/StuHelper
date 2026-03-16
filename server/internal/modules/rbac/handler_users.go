package rbac

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// handleGetUserRoles 获取用户角色
func (h *Handler) handleGetUserRoles(c *gin.Context) {
	userID, err := h.resolveUserID(c, "userID")
	if err != nil {
		return
	}

	roles, err := h.service.GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		logger.FromGin(c).Error("failed to get user roles", zap.Error(err))
		response.InternalError(c, "failed to get user roles")
		return
	}
	response.Success(c, roles)
}

type setUserRolesRequest struct {
	RoleIDs []int64 `json:"roleIDs" binding:"required"`
}

// handleSetUserRoles 设置用户角色
func (h *Handler) handleSetUserRoles(c *gin.Context) {
	userID, err := h.resolveUserID(c, "userID")
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
	userID, err := h.resolveUserID(c, "userID")
	if err != nil {
		return
	}

	perms, err := h.service.GetEffectivePermissions(c.Request.Context(), userID)
	if err != nil {
		logger.FromGin(c).Error("failed to get user permissions", zap.Error(err))
		response.InternalError(c, "failed to get user permissions")
		return
	}
	response.Success(c, perms)
}

type setUserPermissionRequest struct {
	PermissionID int64 `json:"permissionID" binding:"required"`
	Granted      *bool `json:"granted" binding:"required"`
}

// handleSetUserPermission 设置用户个人权限覆盖
func (h *Handler) handleSetUserPermission(c *gin.Context) {
	userID, err := h.resolveUserID(c, "userID")
	if err != nil {
		return
	}

	var req setUserPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	err = h.service.SetUserPermission(c.Request.Context(), userID, req.PermissionID, *req.Granted)
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
