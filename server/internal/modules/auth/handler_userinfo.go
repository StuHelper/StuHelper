package auth

import (
	"strings"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

type currentUserPayload struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	DisplayName        string             `json:"displayName"`
	Avatar             *string            `json:"avatar,omitempty"`
	Email              string             `json:"email,omitempty"`
	Roles              []string           `json:"roles"`
	Capabilities       []string           `json:"capabilities"`
	GlobalCapabilities []string           `json:"globalCapabilities"`
	CapabilityGrants   []capability.Grant `json:"capabilityGrants"`
	IsPlatformAdmin    bool               `json:"isPlatformAdmin"`
	CanAccessAdmin     bool               `json:"canAccessAdmin"`
}

// GetCurrentUser 获取当前用户信息（从 AuthMiddleware 注入的 context 读取）
func (h *Handler) GetCurrentUser(c *gin.Context) {
	roles := middleware.GetRoles(c)
	capabilities := middleware.GetCapabilities(c)
	avatar := middleware.GetAvatar(c)
	response.Success(c, h.buildUserPayload(
		middleware.GetUserID(c),
		middleware.GetUsername(c),
		middleware.GetDisplayName(c),
		middleware.GetEmail(c),
		nullableString(avatar),
		roles,
		capabilities,
	))
}

func (h *Handler) buildUserPayload(
	userID, username, displayName, email string,
	avatar *string,
	roles, capabilities []string,
) currentUserPayload {
	snapshot := buildAccessSnapshot(capabilities)

	return currentUserPayload{
		ID:                 userID,
		Name:               username,
		DisplayName:        displayName,
		Avatar:             avatar,
		Email:              email,
		Roles:              roles,
		Capabilities:       snapshot.Capabilities,
		GlobalCapabilities: snapshot.GlobalCapabilities,
		CapabilityGrants:   snapshot.CapabilityGrants,
		IsPlatformAdmin:    hasRole(roles, "super_admin"),
		CanAccessAdmin:     capability.CanAccessAdmin(snapshot.Capabilities),
	}
}

func buildAccessSnapshot(capabilities []string) capability.UserAccessSnapshot {
	grants := make([]capability.Grant, 0, len(capabilities))
	for _, capName := range capabilities {
		grants = append(grants, capability.Grant{Name: capName})
	}
	return capability.BuildUserAccessSnapshot(grants)
}

func hasRole(roles []string, expected string) bool {
	for _, role := range roles {
		if role == expected {
			return true
		}
	}
	return false
}

func nullableString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
