package review

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// SaveDraftRequest 保存草稿请求
type SaveDraftRequest struct {
	CourseID  int64         `json:"courseID" binding:"required,gt=0"`
	TeacherID *int64        `json:"teacherID" binding:"omitempty,gt=0"`
	TermID    string        `json:"termID" binding:"omitempty,max=20"`
	Title     string        `json:"title" binding:"max=200"`
	Content   string        `json:"content" binding:"max=5000"`
	Grade     string        `json:"grade" binding:"omitempty,oneof=A+ A A- B+ B B- C+ C C- D F"`
	Ratings   ReviewRatings `json:"ratings"`
}

// SaveDraft 保存草稿
func (h *Handler) SaveDraft(c *gin.Context) {
	var req SaveDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	if req.Content != "" && strings.TrimSpace(req.Content) == "" {
		response.BadRequest(c, "content must contain at least 1 non-whitespace character")
		return
	}

	userID := middleware.GetUserID(c)
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return
	}

	draft, err := h.service.SaveDraft(c.Request.Context(), SaveDraftParams{
		UserHash:  userHash,
		CourseID:  req.CourseID,
		TeacherID: req.TeacherID,
		TermID:    req.TermID,
		Title:     req.Title,
		Content:   req.Content,
		Grade:     req.Grade,
		Ratings:   req.Ratings,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrCourseNotFound):
			response.NotFound(c, "course not found", errs.ErrCourseNotFound)
		case errors.Is(err, ErrDangerousContent):
			response.BadRequest(c, "content contains potentially dangerous elements")
		default:
			logger.FromGin(c).Error("failed to save draft", zap.Error(err))
			response.InternalError(c, "failed to save draft")
		}
		return
	}

	response.Success(c, draft)
}

// GetDraft 获取草稿
func (h *Handler) GetDraft(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "courseID")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	userID := middleware.GetUserID(c)
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return
	}

	draft, err := h.service.GetDraft(c.Request.Context(), userHash, courseID)
	if err != nil {
		if errors.Is(err, ErrDraftNotFound) {
			response.NotFound(c, "draft not found", errs.ErrDraftNotFound)
		} else {
			logger.FromGin(c).Error("failed to get draft", zap.Error(err))
			response.InternalError(c, "failed to get draft")
		}
		return
	}

	response.Success(c, draft)
}

// DeleteDraft 删除草稿
func (h *Handler) DeleteDraft(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "courseID")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	userID := middleware.GetUserID(c)
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return
	}

	err = h.service.DeleteDraft(c.Request.Context(), userHash, courseID)
	if err != nil {
		logger.FromGin(c).Error("failed to delete draft", zap.Error(err))
		response.InternalError(c, "failed to delete draft")
		return
	}

	response.Success(c, gin.H{"message": "draft deleted successfully"})
}
