package middleware

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
)

// 上下文键名常量
const (
	CtxKeyUserID             = "user_id"
	CtxKeyAppID              = "app_id"
	CtxKeyUsername           = "username"
	CtxKeyEmail              = "email"
	CtxKeyDisplayName        = "display_name"
	CtxKeyAvatar             = "avatar"
	CtxKeyRoles              = "roles"
	CtxKeyTokenScopes        = "token_scopes"
	CtxKeyOrgScopedRoles     = "org_scoped_roles" // map[string][]string — StuHelper scoped role grants
	CtxKeyCapabilities       = "capabilities"
	CtxKeyGlobalCapabilities = "global_capabilities"
	CtxKeyCapabilityGrants   = "capability_grants"
	CtxKeyCapabilitySet      = "capability_set"       // map[string]struct{} — O(1) 查找
	CtxKeyAuthBackendFailure = "auth_backend_failure" // OptionalAuth 后端故障诊断标记
	CtxKeyAuthenticationTime = "authentication_time"
)

// authResult 认证解析结果
type authResult struct {
	userID, appID, username, email, displayName string
	avatar                                      *string
	tokenScopes                                 []string
	roles                                       []string
	orgScopedRoles                              map[string][]string
	authTime                                    time.Time
	mfaProofAt                                  time.Time
}

func withResolvedRoleScopes(ctx context.Context, auth *authResult, resolver RoleScopeResolver) (*authResult, error) {
	if auth == nil || resolver == nil {
		return auth, nil
	}
	scopes, err := resolver.ResolveRoleScopes(ctx, auth.userID, auth.roles)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errRoleScopeUnavailable, err)
	}
	if len(scopes) == 0 {
		return auth, nil
	}
	resolved := *auth
	resolved.orgScopedRoles = mergeRoleScopes(auth.orgScopedRoles, scopes)
	return &resolved, nil
}

func mergeRoleScopes(base, overlay map[string][]string) map[string][]string {
	merged := make(map[string][]string, len(base)+len(overlay))
	for role, values := range base {
		merged[role] = append([]string(nil), values...)
	}
	for role, values := range overlay {
		merged[role] = append(merged[role], values...)
	}
	return normalizeOrgScopedRoles(merged)
}

func normalizeOrgScopedRoles(scopedRoles map[string][]string) map[string][]string {
	if len(scopedRoles) == 0 {
		return nil
	}

	normalized := make(map[string][]string, len(scopedRoles))
	seenByRole := make(map[string]map[string]struct{}, len(scopedRoles))
	for rawRole, values := range scopedRoles {
		role := strings.TrimSpace(rawRole)
		if role == "" {
			continue
		}
		seen := seenByRole[role]
		if seen == nil {
			seen = make(map[string]struct{}, len(values))
			seenByRole[role] = seen
		}
		for _, rawValue := range values {
			value := strings.TrimSpace(rawValue)
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			normalized[role] = append(normalized[role], value)
		}
	}
	if len(normalized) == 0 {
		return nil
	}
	for role, values := range normalized {
		sort.Strings(values)
		normalized[role] = values
	}
	return normalized
}

// setClaimsToContext 将用户信息、角色和能力集合注入 Gin context。
// 同时构建 capability set（map）供 HasCapability 进行 O(1) 查找。
func setClaimsToContext(c *gin.Context, auth *authResult) {
	orgScopedRoles := normalizeOrgScopedRoles(auth.orgScopedRoles)
	grants := capability.ExpandRoleGrants(auth.roles, orgScopedRoles)
	snapshot := capability.BuildUserAccessSnapshot(grants)
	capSet := make(map[string]struct{}, len(snapshot.Capabilities))
	for _, cap := range snapshot.Capabilities {
		capSet[cap] = struct{}{}
	}

	c.Set(CtxKeyUserID, auth.userID)
	c.Set(CtxKeyAppID, auth.appID)
	c.Set(CtxKeyUsername, auth.username)
	c.Set(CtxKeyEmail, auth.email)
	c.Set(CtxKeyDisplayName, auth.displayName)
	setAvatarContext(c, auth.avatar)
	c.Set(CtxKeyTokenScopes, append([]string(nil), auth.tokenScopes...))
	c.Set(CtxKeyRoles, auth.roles)
	if orgScopedRoles != nil {
		c.Set(CtxKeyOrgScopedRoles, orgScopedRoles)
	}
	c.Set(CtxKeyCapabilities, snapshot.Capabilities)
	c.Set(CtxKeyGlobalCapabilities, snapshot.GlobalCapabilities)
	c.Set(CtxKeyCapabilityGrants, snapshot.CapabilityGrants)
	c.Set(CtxKeyCapabilitySet, capSet)
	SetAuthenticationTime(c, auth.authTime)
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

func GetAppID(c *gin.Context) string {
	return getContextString(c, CtxKeyAppID)
}

func GetTokenScopes(c *gin.Context) []string {
	if val, exists := getContextValue(c, CtxKeyTokenScopes); exists {
		if scopes, ok := val.([]string); ok {
			return append([]string(nil), scopes...)
		}
	}
	return nil
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
	if val, exists := getContextValue(c, CtxKeyRoles); exists {
		if roles, ok := val.([]string); ok {
			return append([]string(nil), roles...)
		}
	}
	return nil
}

// GetCapabilities 从上下文获取能力列表（slice 形式，用于序列化）
func GetCapabilities(c *gin.Context) []string {
	if val, exists := getContextValue(c, CtxKeyCapabilities); exists {
		if caps, ok := val.([]string); ok {
			return append([]string(nil), caps...)
		}
	}
	return nil
}

func GetGlobalCapabilities(c *gin.Context) []string {
	if val, exists := getContextValue(c, CtxKeyGlobalCapabilities); exists {
		if caps, ok := val.([]string); ok {
			return append([]string(nil), caps...)
		}
	}
	return nil
}

func GetCapabilityGrants(c *gin.Context) []capability.Grant {
	if val, exists := getContextValue(c, CtxKeyCapabilityGrants); exists {
		if grants, ok := val.([]capability.Grant); ok {
			return cloneCapabilityGrants(grants)
		}
	}
	return nil
}

// HasCapability 检查当前用户是否具有指定能力（O(1) map 查找）
func HasCapability(c *gin.Context, capabilityName string) bool {
	if val, exists := getContextValue(c, CtxKeyCapabilitySet); exists {
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

// IsAuthenticated 检查当前请求是否已认证
func IsAuthenticated(c *gin.Context) bool {
	return GetUserID(c) != ""
}

func SetAuthenticationTime(c *gin.Context, authTime time.Time) {
	if authTime.IsZero() {
		return
	}
	c.Set(CtxKeyAuthenticationTime, authTime.UTC())
}

func GetAuthenticationTime(c *gin.Context) time.Time {
	value, exists := getContextValue(c, CtxKeyAuthenticationTime)
	if !exists {
		return time.Time{}
	}
	authTime, ok := value.(time.Time)
	if !ok {
		return time.Time{}
	}
	return authTime
}

func getContextValue(c *gin.Context, key string) (any, bool) {
	if c == nil {
		return nil, false
	}
	return c.Get(key)
}

func getContextString(c *gin.Context, key string) string {
	if val, exists := getContextValue(c, key); exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func cloneCapabilityGrants(grants []capability.Grant) []capability.Grant {
	if grants == nil {
		return nil
	}
	out := make([]capability.Grant, len(grants))
	for i, grant := range grants {
		out[i] = grant
		out[i].ScopeSchoolIDs = append([]string(nil), grant.ScopeSchoolIDs...)
		out[i].ScopeSectionIDs = append([]string(nil), grant.ScopeSectionIDs...)
		out[i].ScopeRoles = append([]string(nil), grant.ScopeRoles...)
	}
	return out
}
