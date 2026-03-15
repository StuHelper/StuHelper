package capability

import "sort"

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

	RBACRoleRead       = "rbac:role:read"
	RBACRoleCreate     = "rbac:role:create"
	RBACRoleUpdate     = "rbac:role:update"
	RBACRoleDelete     = "rbac:role:delete"
	RBACPermissionRead = "rbac:permission:read"
	RBACUserRead       = "rbac:user:read"
	RBACUserUpdate     = "rbac:user:update"
	RBACGroupRead      = "rbac:group:read"
	RBACGroupCreate    = "rbac:group:create"
	RBACGroupUpdate    = "rbac:group:update"
	RBACGroupDelete    = "rbac:group:delete"
)

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
	RBACRoleRead,
	RBACRoleCreate,
	RBACRoleUpdate,
	RBACRoleDelete,
	RBACPermissionRead,
	RBACUserRead,
	RBACUserUpdate,
	RBACGroupRead,
	RBACGroupCreate,
	RBACGroupUpdate,
	RBACGroupDelete,
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

func CanAccessAdmin(capabilities []string) bool {
	return HasAny(capabilities, AdminEntryCapabilities...)
}
