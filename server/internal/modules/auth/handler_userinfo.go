package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

const casdoorAccountSettingsPath = "/account"

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
	AccountSettingsURL string             `json:"accountSettingsUrl,omitempty"`
}

// GetCurrentUser 获取当前用户信息（从 AuthMiddleware 注入的 context 读取）
func (h *Handler) GetCurrentUser(c *gin.Context) {
	roles := middleware.GetRoles(c)
	avatar := middleware.GetAvatar(c)
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)
	displayName := middleware.GetDisplayName(c)
	email := middleware.GetEmail(c)
	avatarURL := nullableString(avatar)
	if !h.syncCurrentUser(c, userID, username, email, avatarURL, roles) {
		return
	}
	response.Success(c, h.buildUserPayload(
		userID,
		username,
		displayName,
		email,
		avatarURL,
		roles,
		middleware.GetCapabilityGrants(c),
	))
}

func (h *Handler) syncCurrentUser(
	c *gin.Context,
	userID,
	username,
	email string,
	avatarURL *string,
	roles []string,
) bool {
	if err := h.svc.SyncOIDCUser(c.Request.Context(), UserSyncInput{
		CasdoorSubject: userID,
		Username:       username,
		Email:          email,
		AvatarURL:      avatarURL,
		Roles:          roles,
		// /auth/me 可能由较旧的 access token 驱动，只同步 shadow profile，
		// 不得据此重新授予或撤销平台级 OpenFGA role tuple。
		RolesAuthoritative: false,
	}); err != nil {
		logger.FromGin(c).Error("failed to sync current user",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		response.InternalError(c, "failed to synchronize user")
		return false
	}
	return true
}

func (h *Handler) buildUserPayload(
	userID, username, displayName, email string,
	avatar *string,
	roles []string,
	grants []capability.Grant,
) currentUserPayload {
	roles = nonNilStringSlice(roles)
	grants = nonNilCapabilityGrants(grants)
	snapshot := capability.BuildUserAccessSnapshot(grants)

	return currentUserPayload{
		ID:                 userID,
		Name:               username,
		DisplayName:        displayName,
		Avatar:             avatar,
		Email:              email,
		Roles:              roles,
		Capabilities:       nonNilStringSlice(snapshot.Capabilities),
		GlobalCapabilities: nonNilStringSlice(snapshot.GlobalCapabilities),
		CapabilityGrants:   nonNilCapabilityGrants(snapshot.CapabilityGrants),
		IsPlatformAdmin:    hasRole(roles, "super_admin"),
		CanAccessAdmin:     capability.CanAccessAdmin(snapshot.Capabilities),
		AccountSettingsURL: buildAccountSettingsURL(h.accountSettingsURLBase()),
	}
}

func (h *Handler) accountSettingsURLBase() string {
	if h == nil {
		return ""
	}
	if strings.TrimSpace(h.accountSettingsBaseURL) != "" {
		return h.accountSettingsBaseURL
	}
	return h.oidcIssuer
}

func buildAccountSettingsURL(issuer string) string {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	if issuer == "" {
		return ""
	}
	return issuer + casdoorAccountSettingsPath
}

func hasRole(roles []string, expected string) bool {
	for _, role := range roles {
		if role == expected {
			return true
		}
	}
	return false
}

func buildAccessSnapshotForRoles(roles []string, orgScopedRoles map[string][]string) capability.UserAccessSnapshot {
	return capability.BuildUserAccessSnapshot(capability.ExpandRoleGrants(roles, orgScopedRoles))
}

func nullableString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func nonNilStringSlice(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilCapabilityGrants(values []capability.Grant) []capability.Grant {
	if values == nil {
		return []capability.Grant{}
	}
	return values
}
