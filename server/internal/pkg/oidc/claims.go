package oidc

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Claims OIDC ID Token 解析后的用户信息
// PreferredUsername 对齐标准字段 preferred_username。
//
//nolint:revive // 保持与 OIDC 标准 claims 命名一致
type Claims struct {
	Sub               string   `json:"sub"`
	Name              string   `json:"name"`
	PreferredUsername string   `json:"preferred_username"`
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"email_verified"`
	Picture           string   `json:"picture"`
	AMR               []string `json:"amr,omitempty"`
	AuthTime          int64    `json:"auth_time,omitempty"`
	AppID             string   `json:"-"`

	// 解析后的角色列表（如 ["school_admin", "verified_student"]）
	Roles []string `json:"-"`

	// OrgScopedRoles 是 StuHelper 内部的 school-scope grant 载体：roleName → schoolID 列表。
	// Casdoor roles claim 保持扁平；学校作用域必须来自 DB/OpenFGA 投影，不能从 role 名嵌入解析。
	// 例：{"school_admin": ["4111010001", "4111010002"]} 表示该用户在学校 4111010001 和 4111010002 上拥有 school_admin grant。
	OrgScopedRoles map[string][]string `json:"-"`
}

// GetUserID 返回 OIDC subject（唯一用户标识）
func (c *Claims) GetUserID() string {
	return c.Sub
}

func (c *Claims) GetAppID() string {
	if c == nil {
		return ""
	}
	return c.AppID
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

func (c *Claims) MFAProofVerifiedAt() time.Time {
	if c == nil {
		return time.Time{}
	}
	return MFAProofVerifiedAt(c.AMR, c.AuthTime)
}

func defaultRolesClaim(value string) string {
	if value == "" {
		return "roles"
	}
	return value
}

// ParseProviderRolesFromRaw extracts roles from the provider-neutral flat roles claim.
// Casdoor role projection is intentionally flat; resource scope is resolved from DB/OpenFGA.
func ParseProviderRolesFromRaw(rawJSON []byte, rolesClaim string) ([]string, error) {
	return parseFlatRolesClaim(rawJSON, defaultRolesClaim(rolesClaim))
}

func parseFlatRolesClaim(rawJSON []byte, rolesClaim string) ([]string, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal raw claims: %w", err)
	}
	roleData, ok := raw[rolesClaim]
	if !ok {
		return nil, nil
	}
	roles, err := parseRoleNames(roleData, rolesClaim)
	if err != nil {
		return nil, err
	}
	sort.Strings(roles)
	return roles, nil
}

func parseRoleNames(roleData json.RawMessage, rolesClaim string) ([]string, error) {
	var list []string
	if err := json.Unmarshal(roleData, &list); err == nil {
		return dedupeRoles(list), nil
	}

	var anyList []any
	if err := json.Unmarshal(roleData, &anyList); err == nil {
		roles, err := rolesFromClaimItems(anyList, rolesClaim)
		if err != nil {
			return nil, err
		}
		return dedupeRoles(roles), nil
	}

	return nil, fmt.Errorf("roles claim %q must be a string array or role object array", rolesClaim)
}

func rolesFromClaimItems(items []any, rolesClaim string) ([]string, error) {
	roles := make([]string, 0, len(items))
	for _, item := range items {
		switch value := item.(type) {
		case string:
			roles = append(roles, value)
		case map[string]any:
			role, err := roleNameFromClaimObject(value, rolesClaim)
			if err != nil {
				return nil, err
			}
			roles = append(roles, role)
		default:
			return nil, fmt.Errorf("roles claim %q contains unsupported role item", rolesClaim)
		}
	}
	return roles, nil
}

func roleNameFromClaimObject(value map[string]any, rolesClaim string) (string, error) {
	name, ok := value["name"].(string)
	trimmed := strings.TrimSpace(name)
	if !ok || trimmed == "" {
		return "", fmt.Errorf("roles claim %q contains role object without string name", rolesClaim)
	}
	return trimmed, nil
}

func dedupeRoles(roles []string) []string {
	seen := make(map[string]struct{}, len(roles))
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		trimmed := strings.TrimSpace(role)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func appIDFromRawClaims(rawJSON []byte) string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		return ""
	}
	if appID := authorizedPartyFromRaw(raw); appID != "" {
		return appID
	}
	audiences := claimStringList(raw["aud"])
	if len(audiences) == 1 {
		return audiences[0]
	}
	return ""
}

func authorizedPartyFromRawClaims(rawJSON []byte) string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		return ""
	}
	return authorizedPartyFromRaw(raw)
}

func authorizedPartyFromRaw(raw map[string]json.RawMessage) string {
	if appID := claimString(raw["azp"]); appID != "" {
		return appID
	}
	return claimString(raw["client_id"])
}

func claimString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func claimStringList(raw json.RawMessage) []string {
	if single := claimString(raw); single != "" {
		return []string{single}
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil
	}
	return nonEmptyStrings(values)
}

func nonEmptyStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

const (
	amrMFA      = "mfa"
	amrOTP      = "otp"
	amrSMS      = "sms"
	amrPhone    = "phone"
	amrApp      = "app"
	amrTOTP     = "totp"
	amrWebAuthn = "webauthn"
	amrFIDO     = "fido"
	amrFIDO2    = "fido2"
	amrHWK      = "hwk"
)

var mfaAMRMethods = map[string]struct{}{
	amrMFA:      {},
	amrOTP:      {},
	amrSMS:      {},
	amrPhone:    {},
	amrApp:      {},
	amrTOTP:     {},
	amrWebAuthn: {},
	amrFIDO:     {},
	amrFIDO2:    {},
	amrHWK:      {},
}

func MFAProofVerifiedAt(amr []string, authTime int64) time.Time {
	if authTime <= 0 || !HasMFAAMR(amr) {
		return time.Time{}
	}
	return time.Unix(authTime, 0).UTC()
}

func HasMFAAMR(amr []string) bool {
	for _, method := range amr {
		normalized := strings.ToLower(strings.TrimSpace(method))
		if _, ok := mfaAMRMethods[normalized]; ok {
			return true
		}
	}
	return false
}
