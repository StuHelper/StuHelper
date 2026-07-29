package review

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

// CreateTeacherRequest 创建教师请求
type CreateTeacherRequest struct {
	Name         string `json:"name" binding:"required,min=1,max=100"`
	DepartmentID *int64 `json:"departmentID" binding:"required"`
}

// UpdateTeacherRequest 更新教师请求
type UpdateTeacherRequest struct {
	Name         string `json:"name" binding:"required,min=1,max=100"`
	DepartmentID *int64 `json:"departmentID"`
}

// ListAdminTeachers 获取教师列表（管理员）
func (h *Handler) ListAdminTeachers(c *gin.Context) {
	search := c.Query("search")
	departmentID, ok := httputil.ParseOptionalInt64Query(c, "departmentID")
	if !ok {
		response.BadRequest(c, "invalid departmentID")
		return
	}
	page, pageSize := httputil.ParsePage(c)
	offset := httputil.SafeOffset(page, pageSize)

	list, total, err := h.service.ListAdminTeachers(c.Request.Context(), search, departmentID, pageSize, offset)
	if err != nil {
		logger.FromGin(c).Error("failed to list teachers", zap.Error(err))
		response.InternalError(c, "failed to list teachers")
		return
	}

	response.Success(c, gin.H{"list": list, "total": total})
}

// CreateTeacher 新增教师
func (h *Handler) CreateTeacher(c *gin.Context) {
	var req CreateTeacherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	teacher, err := h.service.CreateTeacher(c.Request.Context(), req.Name, req.DepartmentID)
	if err != nil {
		if respondTeacherAdminError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to create teacher", zap.Error(err))
		response.InternalError(c, "failed to create teacher")
		return
	}

	// 记录操作日志
	h.logAdminOp(c, "create_teacher", "teacher", strconv.FormatInt(teacher.ID, 10),
		nil,
		map[string]interface{}{"name": req.Name, "departmentID": req.DepartmentID})

	response.Created(c, teacher)
}

// UpdateTeacher 更新教师
func (h *Handler) UpdateTeacher(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("teacherID"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid teacher ID")
		return
	}

	var req UpdateTeacherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	if err := h.service.UpdateTeacher(c.Request.Context(), id, req.Name, req.DepartmentID); err != nil {
		if respondTeacherAdminError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to update teacher", zap.Error(err))
		response.InternalError(c, "failed to update teacher")
		return
	}

	h.logAdminOp(c, "update_teacher", "teacher", strconv.FormatInt(id, 10),
		nil,
		map[string]interface{}{"name": req.Name, "departmentID": req.DepartmentID})

	response.Success(c, gin.H{"message": "teacher updated successfully"})
}

// DeleteTeacher 删除教师
func (h *Handler) DeleteTeacher(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("teacherID"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid teacher ID")
		return
	}

	if err := h.service.DeleteTeacher(c.Request.Context(), id); err != nil {
		if respondTeacherAdminError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to delete teacher", zap.Error(err))
		response.InternalError(c, "failed to delete teacher")
		return
	}

	h.logAdminOp(c, "delete_teacher", "teacher", strconv.FormatInt(id, 10), nil, nil)

	response.Success(c, gin.H{"message": "teacher deleted successfully"})
}
