package user

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

func (h *Handler) handleAdminListSystemConfigs(c *gin.Context) {
	configs, err := h.service.ListSystemConfigs(c.Request.Context())
	if err != nil {
		logger.FromGin(c).Error("failed to list system configs", zap.Error(err))
		response.InternalError(c, "failed to list system configs")
		return
	}

	items := make([]systemConfigResponse, 0, len(configs))
	for i := range configs {
		items = append(items, systemConfigToJSON(&configs[i]))
	}

	response.Success(c, items)
}

type updateSystemConfigHTTPRequest struct {
	Value string `json:"value" binding:"required"`
}

func (h *Handler) handleAdminUpdateSystemConfig(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.BadRequest(c, "invalid config key")
		return
	}

	var req updateSystemConfigHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	if err := h.service.UpdateSystemConfig(c.Request.Context(), key, req.Value); err != nil {
		if respondAdminUpdateSystemConfigError(c, err) {
			logger.FromGin(c).Warn("system config update rejected",
				zap.String("config_key", key),
				zap.Error(err),
			)
			return
		}
		logger.FromGin(c).Error("failed to update system config",
			zap.String("config_key", key),
			zap.Error(err),
		)
		response.InternalError(c, "failed to update system config")
		return
	}

	audit.LogFromGin(c, audit.Event{
		Type:         audit.EventAdminConfigChange,
		Category:     "admin_operation",
		Resource:     "system_config",
		ResourceType: "system_config",
		ResourceID:   key,
		Action:       "update",
		Result:       "success",
		Details: map[string]any{
			"key": key,
		},
	})

	response.Success(c, messageResponse{Message: "system config updated"})
}
