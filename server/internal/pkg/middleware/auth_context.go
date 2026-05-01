package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
)

// 上下文键名常量
const (
	CtxKeyUserID             = "user_id"
	CtxKeyUsername           = "username"
	CtxKeyEmail              = "email"
	CtxKeyDisplayName        = "display_name"
	CtxKeyAvatar             = "avatar"
	CtxKeyRoles              = "roles"
	CtxKeyOrgScopedRoles     = "org_scoped_roles" // map[string][]string — provider-scoped legacy roles
	CtxKeyCapabilities       = "capabilities"
	CtxKeyGlobalCapabilities = "global_capabilities"
	CtxKeyCapabilityGrants   = "capability_grants"
	CtxKeyCapabilitySet      = "capability_set"       // map[string]struct{} — O(1) 查找
	CtxKeyAuthBackendFailure = "auth_backend_failure" // OptionalAuth 后端故障诊断标记
)

// authResult 认证解析结果
type authResult struct {
	userID, username, email, displayName string
	avatar                               *string
	roles                                []string
	orgScopedRoles                       map[string][]string
	mfaProofAt                           time.Time
}

// setClaimsToContext 将用户信息、角色和能力集合注入 Gin context。
// 同时构建 capability set（map）供 HasCapability 进行 O(1) 查找。
func setClaimsToContext(c *gin.Context, auth *authResult) {
	grants := capability.ExpandRoleGrants(auth.roles, auth.orgScopedRoles)
	snapshot := capability.BuildUserAccessSnapshot(grants)
	capSet := make(map[string]struct{}, len(snapshot.Capabilities))
	for _, cap := range snapshot.Capabilities {
		capSet[cap] = struct{}{}
	}

	c.Set(CtxKeyUserID, auth.userID)
	c.Set(CtxKeyUsername, auth.username)
	c.Set(CtxKeyEmail, auth.email)
	c.Set(CtxKeyDisplayName, auth.displayName)
	setAvatarContext(c, auth.avatar)
	c.Set(CtxKeyRoles, auth.roles)
	if auth.orgScopedRoles != nil {
		c.Set(CtxKeyOrgScopedRoles, auth.orgScopedRoles)
	}
	c.Set(CtxKeyCapabilities, snapshot.Capabilities)
	c.Set(CtxKeyGlobalCapabilities, snapshot.GlobalCapabilities)
	c.Set(CtxKeyCapabilityGrants, snapshot.CapabilityGrants)
	c.Set(CtxKeyCapabilitySet, capSet)
	SetMFAProofVerifiedAt(c, auth.mfaProofAt)
}

func setAvatarContext(c *gin.Context, avatar *string) {
	if avatar != nil {
		c.Set(CtxKeyAvatar, *avatar)
		return
	}
	c.Set(CtxKeyAvatar, "")
}

// GetUserID 从上下文获取用户 ID
func GetUserID(c *gin.Context) string {
	return getContextString(c, CtxKeyUserID)
}

// GetUsername 从上下文获取用户名
func GetUsername(c *gin.Context) string {
	return getContextString(c, CtxKeyUsername)
}

// GetEmail 从上下文获取邮箱
func GetEmail(c *gin.Context) string {
	return getContextString(c, CtxKeyEmail)
}

// GetDisplayName 从上下文获取显示名称
func GetDisplayName(c *gin.Context) string {
	return getContextString(c, CtxKeyDisplayName)
}

// GetAvatar 从上下文获取头像地址
func GetAvatar(c *gin.Context) string {
	return getContextString(c, CtxKeyAvatar)
}

// GetRoles 从上下文获取角色列表
func GetRoles(c *gin.Context) []string {
	if val, exists := c.Get(CtxKeyRoles); exists {
		if roles, ok := val.([]string); ok {
			return roles
		}
	}
	return nil
}

// GetCapabilities 从上下文获取能力列表（slice 形式，用于序列化）
func GetCapabilities(c *gin.Context) []string {
	if val, exists := c.Get(CtxKeyCapabilities); exists {
		if caps, ok := val.([]string); ok {
			return caps
		}
	}
	return nil
}

func GetGlobalCapabilities(c *gin.Context) []string {
	if val, exists := c.Get(CtxKeyGlobalCapabilities); exists {
		if caps, ok := val.([]string); ok {
			return caps
		}
	}
	return nil
}

func GetCapabilityGrants(c *gin.Context) []capability.Grant {
	if val, exists := c.Get(CtxKeyCapabilityGrants); exists {
		if grants, ok := val.([]capability.Grant); ok {
			return grants
		}
	}
	return nil
}

// HasCapability 检查当前用户是否具有指定能力（O(1) map 查找）
func HasCapability(c *gin.Context, capabilityName string) bool {
	if val, exists := c.Get(CtxKeyCapabilitySet); exists {
		if set, ok := val.(map[string]struct{}); ok {
			_, found := set[capabilityName]
			return found
		}
	}
	return false
}

func HasGlobalCapability(c *gin.Context, capabilityName string) bool {
	return capability.HasGlobalGrant(GetCapabilityGrants(c), capabilityName)
}

func HasCapabilityInSchool(c *gin.Context, capabilityName, schoolID string) bool {
	return capability.HasGrantInSchool(GetCapabilityGrants(c), capabilityName, schoolID)
}

// HasRoleInOrg 检查当前用户是否在指定 orgID 上拥有指定角色（provider-scoped legacy roles）。
// 仅 cookie-OIDC 登录路径填充 scope；手机登录与 Bearer introspection 返回
// false。orgID 为空时判定"是否在任意 org 拥有此角色"。
func HasRoleInOrg(c *gin.Context, role, orgID string) bool {
	if val, exists := c.Get(CtxKeyOrgScopedRoles); exists {
		if scoped, ok := val.(map[string][]string); ok {
			return hasScopedRole(scoped, role, orgID)
		}
	}
	return false
}

func hasScopedRole(scoped map[string][]string, role, orgID string) bool {
	orgs, has := scoped[role]
	if !has {
		return false
	}
	if orgID == "" {
		return len(orgs) > 0
	}
	for _, org := range orgs {
		if org == orgID {
			return true
		}
	}
	return false
}

// IsAuthenticated 检查当前请求是否已认证
func IsAuthenticated(c *gin.Context) bool {
	return GetUserID(c) != ""
}

func getContextString(c *gin.Context, key string) string {
	if val, exists := c.Get(key); exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}
