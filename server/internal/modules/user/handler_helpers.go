package user

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
)

func (h *Handler) resolveCurrentUser(c *gin.Context) (int64, bool) {
	return middleware.ResolveRequiredInternalUserID(c, h.service.GetInternalUserID, "failed to resolve user")
}

type messageResponse struct {
	Message string `json:"message"`
}

type systemConfigResponse struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description *string   `json:"description"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type userSurfaceResponse struct {
	DisplayName               string   `json:"displayName"`
	AvatarURL                 string   `json:"avatarURL,omitempty"`
	Phone                     *string  `json:"phone,omitempty"`
	StudentVerificationStatus string   `json:"studentVerificationStatus"`
	PhoneBound                bool     `json:"phoneBound"`
	Capabilities              []string `json:"capabilities"`
}

func systemConfigToJSON(c *SystemConfig) systemConfigResponse {
	return systemConfigResponse{
		Key:         c.Key,
		Value:       c.Value,
		Description: c.Description,
		UpdatedAt:   c.UpdatedAt,
	}
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
