package openplatform

import "strings"

type ScopeDefinition struct {
	Scope       string   `json:"scope"`
	DisplayName string   `json:"displayName"`
	Sensitivity string   `json:"sensitivity"`
	Fields      []string `json:"fields"`
}

var scopeCatalog = map[string]ScopeDefinition{
	ScopeProfileBasicRead: {
		Scope: ScopeProfileBasicRead, DisplayName: "用户基本信息", Sensitivity: "low",
		Fields: []string{"用户名", "用户昵称", "头像地址"},
	},
	ScopeEmailRead: {
		Scope: ScopeEmailRead, DisplayName: "邮箱", Sensitivity: "medium",
		Fields: []string{"邮箱地址"},
	},
	ScopePhoneRead: {
		Scope: ScopePhoneRead, DisplayName: "手机号", Sensitivity: "high",
		Fields: []string{"已绑定手机号", "手机号验证状态"},
	},
	ScopeIdentityStatusRead: {
		Scope: ScopeIdentityStatusRead, DisplayName: "实名认证状态", Sensitivity: "high",
		Fields: []string{"是否已完成实名认证"},
	},
	ScopeIdentityTypeRead: {
		Scope: ScopeIdentityTypeRead, DisplayName: "身份类型", Sensitivity: "high",
		Fields: []string{"身份类型"},
	},
	ScopeStudentStatusRead: {
		Scope: ScopeStudentStatusRead, DisplayName: "学生认证状态", Sensitivity: "high",
		Fields: []string{"是否已完成学生认证"},
	},
	ScopeStudentSchoolRead: {
		Scope: ScopeStudentSchoolRead, DisplayName: "学校信息", Sensitivity: "high",
		Fields: []string{"学校 ID", "学校名称"},
	},
	ScopeResourceRead: {
		Scope: ScopeResourceRead, DisplayName: "授权资源读取", Sensitivity: "high",
		Fields: []string{"用户授权的资源读取权限"},
	},
	ScopeResourceWrite: {
		Scope: ScopeResourceWrite, DisplayName: "授权资源写入", Sensitivity: "very_high",
		Fields: []string{"用户授权的资源写入权限"},
	},
}

func NormalizeScopes(scopes []string) ([]string, error) {
	seen := make(map[string]struct{}, len(scopes))
	result := make([]string, 0, len(scopes))
	for _, raw := range scopes {
		scope := strings.TrimSpace(raw)
		if scope == "" {
			continue
		}
		if _, ok := scopeCatalog[scope]; !ok {
			return nil, ErrInvalidScope
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		result = append(result, scope)
	}
	if len(result) == 0 {
		return nil, ErrInvalidScope
	}
	return result, nil
}

func ScopeDefinitions(scopes []string) []ScopeDefinition {
	result := make([]ScopeDefinition, 0, len(scopes))
	for _, scope := range scopes {
		if def, ok := scopeCatalog[scope]; ok {
			result = append(result, def)
		}
	}
	return result
}
