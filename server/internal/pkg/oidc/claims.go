package oidc

import (
	"encoding/json"
	"fmt"
)

// Claims Zitadel OIDC ID Token 解析后的用户信息
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
}

// GetUserID 返回 Zitadel subject（唯一用户标识）
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

// ParseRolesFromRaw 从 gooidc ID Token 的原始 JSON 中提取 Zitadel 项目级角色。
// Zitadel 角色 claim 格式: "urn:zitadel:iam:org:project:{projectID}:roles"
// 值为 map[roleName]map[orgID]orgDomain
func ParseRolesFromRaw(rawJSON []byte, projectID string) ([]string, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal raw claims: %w", err)
	}

	claimKey := "urn:zitadel:iam:org:project:" + projectID + ":roles"
	roleData, ok := raw[claimKey]
	if !ok {
		return nil, nil
	}

	var roleMap map[string]any
	if err := json.Unmarshal(roleData, &roleMap); err != nil {
		return nil, fmt.Errorf("unmarshal roles claim %q: %w", claimKey, err)
	}

	roles := make([]string, 0, len(roleMap))
	for roleName := range roleMap {
		roles = append(roles, roleName)
	}
	return roles, nil
}
