package capability

import (
	"sort"
	"strings"
)

const (
	AdminDashboardView        = "admin:dashboard:view"
	AdminReviewsManage        = "admin:reviews:manage"
	AdminReportsManage        = "admin:reports:manage"
	AdminTeachersManage       = "admin:teachers:manage"
	AdminSensitiveWordsManage = "admin:sensitive_words:manage"
	AdminLogsView             = "admin:logs:view"

	UserIdentityRead   = "user:identity:read"
	UserIdentityReview = "user:identity:review"
	UserStudentRead    = "user:student:read"
	UserStudentReview  = "user:student:review"
	UserSchoolRead     = "user:school:read"
	UserSchoolUpdate   = "user:school:update"
	UserSystemRead     = "user:system:read"
	UserSystemUpdate   = "user:system:update"

	// 评课相关能力
	ReviewListFull  = "review:list:full"
	ReviewCreate    = "review:create"
	ReviewEditOwn   = "review:edit:own"
	ReviewDeleteOwn = "review:delete:own"
	ReviewListBrief = "review:list:brief"
)

// RoleCapabilities 角色 → 能力静态映射。
// 这是权限模型的核心配置，修改需要 code review。
var RoleCapabilities = map[string][]string{
	"super_admin": {
		AdminDashboardView, AdminReviewsManage, AdminReportsManage,
		AdminTeachersManage, AdminSensitiveWordsManage, AdminLogsView,
		UserIdentityRead, UserIdentityReview,
		UserStudentRead, UserStudentReview,
		UserSchoolRead, UserSchoolUpdate,
		UserSystemRead, UserSystemUpdate,
	},
	"school_admin": {
		AdminDashboardView, AdminReviewsManage, AdminReportsManage,
		AdminTeachersManage, AdminSensitiveWordsManage, AdminLogsView,
		UserIdentityRead, UserIdentityReview,
		UserStudentRead, UserStudentReview,
		UserSchoolRead, UserSystemRead,
	},
	"moderator": {
		AdminReviewsManage, AdminReportsManage, AdminTeachersManage,
	},
	"verified_student": {
		ReviewListFull, ReviewCreate, ReviewEditOwn, ReviewDeleteOwn,
	},
	"user": {
		ReviewListBrief,
	},
}

// ExpandRoles 将角色列表展开为去重排序的能力列表（纯内存操作，零 DB 查询）
func ExpandRoles(roles []string) []string {
	var all []string
	for _, role := range roles {
		if caps, ok := RoleCapabilities[role]; ok {
			all = append(all, caps...)
		}
	}
	return Normalize(all)
}

// AdminEntryCapabilities 拥有任一即可进入管理后台
var AdminEntryCapabilities = []string{
	AdminDashboardView,
	AdminReviewsManage,
	AdminReportsManage,
	AdminTeachersManage,
	AdminSensitiveWordsManage,
	AdminLogsView,
	UserIdentityRead,
	UserIdentityReview,
	UserStudentRead,
	UserStudentReview,
	UserSchoolRead,
	UserSchoolUpdate,
	UserSystemRead,
	UserSystemUpdate,
}

type Grant struct {
	Name           string   `json:"name"`
	ScopeSchoolIDs []string `json:"scopeSchoolIDs,omitempty"`
	ScopeRoles     []string `json:"scopeRoles,omitempty"`
	Global         bool     `json:"global"`
}

type UserAccessSnapshot struct {
	Capabilities       []string `json:"capabilities"`
	GlobalCapabilities []string `json:"globalCapabilities"`
	CapabilityGrants   []Grant  `json:"capabilityGrants"`
}

func Has(capabilities []string, expected string) bool {
	for _, capability := range capabilities {
		if capability == expected {
			return true
		}
	}
	return false
}

func HasAny(capabilities []string, expected ...string) bool {
	for _, candidate := range expected {
		if Has(capabilities, candidate) {
			return true
		}
	}
	return false
}

func Normalize(capabilities []string) []string {
	seen := make(map[string]struct{}, len(capabilities))
	result := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		if capability == "" {
			continue
		}
		if _, ok := seen[capability]; ok {
			continue
		}
		seen[capability] = struct{}{}
		result = append(result, capability)
	}
	sort.Strings(result)
	return result
}

func normalizeScope(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	sort.Strings(result)
	return result
}

func NormalizeGrant(grant Grant) Grant {
	grant.Name = strings.TrimSpace(grant.Name)
	grant.ScopeSchoolIDs = normalizeScope(grant.ScopeSchoolIDs)
	grant.ScopeRoles = normalizeScope(grant.ScopeRoles)
	grant.Global = len(grant.ScopeSchoolIDs) == 0 && len(grant.ScopeRoles) == 0
	return grant
}

func BuildUserAccessSnapshot(grants []Grant) UserAccessSnapshot {
	normalizedGrants := make([]Grant, 0, len(grants))
	grantSeen := make(map[string]struct{}, len(grants))
	capabilityNames := make([]string, 0, len(grants))
	globalNames := make([]string, 0, len(grants))

	for _, grant := range grants {
		normalized := NormalizeGrant(grant)
		if normalized.Name == "" {
			continue
		}

		key := normalized.Name + "|" + strings.Join(normalized.ScopeSchoolIDs, ",") + "|" + strings.Join(normalized.ScopeRoles, ",")
		if _, ok := grantSeen[key]; ok {
			continue
		}
		grantSeen[key] = struct{}{}
		normalizedGrants = append(normalizedGrants, normalized)
		capabilityNames = append(capabilityNames, normalized.Name)
		if normalized.Global {
			globalNames = append(globalNames, normalized.Name)
		}
	}

	sort.Slice(normalizedGrants, func(i, j int) bool {
		left := normalizedGrants[i]
		right := normalizedGrants[j]
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		leftSchools := strings.Join(left.ScopeSchoolIDs, ",")
		rightSchools := strings.Join(right.ScopeSchoolIDs, ",")
		if leftSchools != rightSchools {
			return leftSchools < rightSchools
		}
		return strings.Join(left.ScopeRoles, ",") < strings.Join(right.ScopeRoles, ",")
	})

	return UserAccessSnapshot{
		Capabilities:       Normalize(capabilityNames),
		GlobalCapabilities: Normalize(globalNames),
		CapabilityGrants:   normalizedGrants,
	}
}

func CanAccessAdmin(capabilities []string) bool {
	return HasAny(capabilities, AdminEntryCapabilities...)
}
