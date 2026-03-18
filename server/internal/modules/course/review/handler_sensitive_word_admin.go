package review

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// CreateSensitiveWordRequest 创建敏感词请求
type CreateSensitiveWordRequest struct {
	Word     string `json:"word" binding:"required,min=1,max=100"`
	Category string `json:"category"`
	Level    string `json:"level"`
}

// UpdateSensitiveWordRequest 更新敏感词请求
type UpdateSensitiveWordRequest struct {
	Word     *string `json:"word"`
	Category *string `json:"category"`
	Level    *string `json:"level"`
	IsActive *bool   `json:"isActive"`
}

// ListSensitiveWords 获取敏感词列表
func (h *Handler) ListSensitiveWords(c *gin.Context) {
	category := c.Query("category")
	level := c.Query("level")
	page, pageSize := httputil.ParsePage(c)
	offset := httputil.SafeOffset(page, pageSize)

	list, total, err := h.service.ListSensitiveWords(c.Request.Context(), category, level, pageSize, offset)
	if err != nil {
		logger.FromGin(c).Error("failed to list sensitive words", zap.Error(err))
		response.InternalError(c, "failed to list sensitive words")
		return
	}

	response.Success(c, gin.H{"list": list, "total": total})
}

// CreateSensitiveWord 新增敏感词
func (h *Handler) CreateSensitiveWord(c *gin.Context) {
	var req CreateSensitiveWordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	w, err := h.service.CreateSensitiveWord(c.Request.Context(), req.Word, req.Category, req.Level)
	if err != nil {
		logger.FromGin(c).Error("failed to create sensitive word", zap.Error(err))
		response.InternalError(c, "failed to create sensitive word")
		return
	}

	h.logAdminOp(c, "create_sensitive_word", "sensitive_word", w.ID,
		nil,
		map[string]interface{}{"word": req.Word, "category": req.Category, "level": req.Level})

	response.Created(c, w)
}

// UpdateSensitiveWord 更新敏感词
func (h *Handler) UpdateSensitiveWord(c *gin.Context) {
	wordID := c.Param("id")
	if wordID == "" {
		response.BadRequest(c, "invalid ID")
		return
	}

	var req UpdateSensitiveWordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	if err := h.service.UpdateSensitiveWord(c.Request.Context(), wordID, req.Word, req.Category, req.Level, req.IsActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFound(c, "sensitive word not found", errs.ErrSensitiveWordNotFound)
			return
		}
		logger.FromGin(c).Error("failed to update sensitive word", zap.Error(err))
		response.InternalError(c, "failed to update sensitive word")
		return
	}

	h.logAdminOp(c, "update_sensitive_word", "sensitive_word", wordID, nil, nil)

	response.Success(c, gin.H{"message": "sensitive word updated"})
}

// DeleteSensitiveWord 删除敏感词
func (h *Handler) DeleteSensitiveWord(c *gin.Context) {
	wordID := c.Param("id")
	if wordID == "" {
		response.BadRequest(c, "invalid ID")
		return
	}

	if err := h.service.DeleteSensitiveWord(c.Request.Context(), wordID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFound(c, "sensitive word not found", errs.ErrSensitiveWordNotFound)
			return
		}
		logger.FromGin(c).Error("failed to delete sensitive word", zap.Error(err))
		response.InternalError(c, "failed to delete sensitive word")
		return
	}

	h.logAdminOp(c, "delete_sensitive_word", "sensitive_word", wordID, nil, nil)

	response.Success(c, gin.H{"message": "sensitive word deleted"})
}
