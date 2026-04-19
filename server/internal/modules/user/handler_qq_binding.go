package user

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

const qqBindingCodeTTL = 10 * time.Minute

func (h *Handler) handleGetQQBinding(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}

	binding, err := h.service.GetQQBinding(c.Request.Context(), userID)
	if err != nil {
		if respondQQBindingError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to get qq binding", zap.Error(err))
		response.InternalError(c, "failed to get qq binding")
		return
	}
	if binding == nil {
		response.NotFound(c, "qq binding not found", errs.ErrNotFound)
		return
	}

	response.Success(c, qqBindingToJSON(binding))
}

func (h *Handler) handleCreateQQBindingCode(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}

	code, err := h.service.GenerateQQBindingCode(c.Request.Context(), userID, qqBindingCodeTTL)
	if err != nil {
		if respondQQBindingCodeError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to create qq binding code", zap.Error(err))
		response.InternalError(c, "failed to create qq binding code")
		return
	}

	response.Created(c, qqBindingCodeResponse{
		Code:      code.Code,
		ExpiresAt: code.ExpiresAt,
	})
}
