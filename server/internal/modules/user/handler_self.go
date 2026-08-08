package user

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

func (h *Handler) handleGetUserSurface(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}

	surface, err := h.service.GetUserSurface(
		c.Request.Context(),
		userID,
		middleware.GetDisplayName(c),
		middleware.GetAvatar(c),
		middleware.GetCapabilities(c),
	)
	if err != nil {
		logger.FromGin(c).Error("failed to get user surface", zap.Error(err))
		response.InternalError(c, "failed to get user information")
		return
	}

	response.Success(c, userSurfaceResponse{
		DisplayName:               surface.DisplayName,
		AvatarURL:                 surface.AvatarURL,
		Phone:                     surface.Phone,
		StudentVerificationStatus: surface.StudentVerificationStatus,
		PhoneBound:                surface.PhoneBound,
		Capabilities:              nonNilStrings(surface.Capabilities),
	})
}
