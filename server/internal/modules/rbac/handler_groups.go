package rbac

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// handleListGroups 获取用户组列表
func (h *Handler) handleListGroups(c *gin.Context) {
	groups, err := h.service.ListGroups(c.Request.Context())
	if err != nil {
		logger.FromGin(c).Error("failed to list groups", zap.Error(err))
		response.InternalError(c, "failed to list groups")
		return
	}
	response.Success(c, groups)
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
	response.Created(c, group)
}

type updateGroupRequest struct {
	DisplayName *string `json:"displayName"`
	Description *string `json:"description"`
}

// handleUpdateGroup 更新用户组
func (h *Handler) handleUpdateGroup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("groupID"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid group ID")
		return
	}

	var req updateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	group, err := h.service.UpdateGroup(c.Request.Context(), id, UpdateGroupInput(req))
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			response.NotFound(c, "group not found", errs.ErrGroupNotFound)
			return
		}
		logger.FromGin(c).Error("failed to update group", zap.Error(err))
		response.InternalError(c, "failed to update group")
		return
	}
	response.Success(c, group)
}

// handleDeleteGroup 删除用户组
func (h *Handler) handleDeleteGroup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("groupID"), 10, 64)
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
	id, err := strconv.ParseInt(c.Param("groupID"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid group ID")
		return
	}

	members, err := h.service.GetGroupMembers(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			response.NotFound(c, "group not found", errs.ErrGroupNotFound)
			return
		}
		logger.FromGin(c).Error("failed to get group members", zap.Error(err))
		response.InternalError(c, "failed to get group members")
		return
	}
	response.Success(c, members)
}

type setGroupMembersRequest struct {
	UserIDs []int64 `json:"userIDs" binding:"required"`
}

// handleSetGroupMembers 设置用户组成员
func (h *Handler) handleSetGroupMembers(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("groupID"), 10, 64)
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
		if errors.Is(err, ErrUserSelectionInvalid) {
			response.BadRequest(c, "one or more selected users are invalid", errs.ErrUserSelectionInvalid)
			return
		}
		logger.FromGin(c).Error("failed to set group members", zap.Error(err))
		response.InternalError(c, "failed to set group members")
		return
	}
	response.Success(c, gin.H{"message": "group members updated"})
}

type setGroupPermissionsRequest struct {
	PermissionIDs []int64 `json:"permissionIDs" binding:"required"`
}

// handleSetGroupPermissions 设置用户组权限
func (h *Handler) handleSetGroupPermissions(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("groupID"), 10, 64)
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
		if errors.Is(err, ErrPermissionSelectionInvalid) {
			response.BadRequest(c, "one or more selected permissions are invalid", errs.ErrPermissionSelectionInvalid)
			return
		}
		logger.FromGin(c).Error("failed to set group permissions", zap.Error(err))
		response.InternalError(c, "failed to set group permissions")
		return
	}
	response.Success(c, gin.H{"message": "group permissions updated"})
}
