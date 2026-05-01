package oidc

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Claims OIDC ID Token 解析后的用户信息
// PreferredUsername 对齐标准字段 preferred_username。
//
//nolint:revive // 保持与 OIDC 标准 claims 命名一致
type Claims struct {
	Sub               string `json:"sub"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	Picture           string `json:"picture"`

	// 解析后的角色列表（如 ["school_admin", "verified_student"]）
	Roles []string

	// OrgScopedRoles 保留 legacy 角色的组织作用域：roleName → 授予该角色的 orgID 列表。
	// 用于多租户授权判定（resource.school_id 必须匹配 token 中对应 role 的 scope）。
	// 例：{"school_admin": ["1001", "1002"]} 表示该用户在 org 1001 和 1002 上是 school_admin。
	OrgScopedRoles map[string][]string
}

// GetUserID 返回 OIDC subject（唯一用户标识）
func (c *Claims) GetUserID() string {
	return c.Sub
}

// GetUsername 返回用户名
func (c *Claims) GetUsername() string {
	if c.PreferredUsername != "" {
		return c.PreferredUsername
	}
	return c.Name
}

// GetDisplayName 返回显示名称
func (c *Claims) GetDisplayName() string {
	return c.Name
}

// GetEmail 返回邮箱
func (c *Claims) GetEmail() string {
	return c.Email
}

// GetAvatar 返回头像 URL，空值返回 nil
func (c *Claims) GetAvatar() *string {
	if c.Picture == "" {
		return nil
	}
	return &c.Picture
}

// HasRoleInOrg 检查用户是否在指定 orgID 上拥有指定角色。
// 用于多租户授权：例如 school_admin 必须对请求资源所属 school_id 有对应 scope。
// 若 orgID 为空，等价于"该用户是否在任意 org 上拥有此角色"。
func (c *Claims) HasRoleInOrg(role, orgID string) bool {
	if c == nil || c.OrgScopedRoles == nil {
		return false
	}
	orgs, ok := c.OrgScopedRoles[role]
	if !ok {
		return false
	}
	if orgID == "" {
		return len(orgs) > 0
	}
	for _, o := range orgs {
		if o == orgID {
			return true
		}
	}
	return false
}

func defaultRolesClaim(value string) string {
	if value == "" {
		return "roles"
	}
	return value
}

// ParseProviderRolesFromRaw extracts roles from the provider-neutral flat roles claim.
// Casdoor role projection is intentionally flat; resource scope is resolved from DB/OpenFGA.
func ParseProviderRolesFromRaw(rawJSON []byte, rolesClaim string) ([]string, map[string][]string, error) {
	return parseFlatRolesClaim(rawJSON, defaultRolesClaim(rolesClaim))
}

func parseFlatRolesClaim(rawJSON []byte, rolesClaim string) ([]string, map[string][]string, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		return nil, nil, fmt.Errorf("unmarshal raw claims: %w", err)
	}
	roleData, ok := raw[rolesClaim]
	if !ok {
		return nil, nil, nil
	}
	roles, err := parseRoleNames(roleData, rolesClaim)
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(roles)
	return roles, nil, nil
}

func parseRoleNames(roleData json.RawMessage, rolesClaim string) ([]string, error) {
	var list []string
	if err := json.Unmarshal(roleData, &list); err == nil {
		return dedupeRoles(list), nil
	}

	var anyList []any
	if err := json.Unmarshal(roleData, &anyList); err == nil {
		roles := make([]string, 0, len(anyList))
		for _, item := range anyList {
			role, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("roles claim %q contains non-string role", rolesClaim)
			}
			roles = append(roles, role)
		}
		return dedupeRoles(roles), nil
	}

	var roleMap map[string]any
	if err := json.Unmarshal(roleData, &roleMap); err == nil {
		roles := make([]string, 0, len(roleMap))
		for role := range roleMap {
			roles = append(roles, role)
		}
		return dedupeRoles(roles), nil
	}

	return nil, fmt.Errorf("unmarshal roles claim %q", rolesClaim)
}

func dedupeRoles(roles []string) []string {
	seen := make(map[string]struct{}, len(roles))
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		if role == "" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		out = append(out, role)
	}
	return out
}
