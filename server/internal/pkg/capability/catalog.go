package capability

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

	AdmissionPolicyRead     = "admission:policy:read"
	AdmissionPolicyUpdate   = "admission:policy:update"
	AdmissionFreshmanRead   = "admission:freshman:read"
	AdmissionFreshmanReview = "admission:freshman:review"
	AdmissionSessionRead    = "admission:session:read"
	MemberBlacklistRead     = "member_blacklist:read"
	MemberBlacklistManage   = "member_blacklist:manage"
	OpenPlatformRead        = "open_platform:read"
	OpenPlatformManage      = "open_platform:manage"

	ReviewListFull  = "review:list:full"
	ReviewCreate    = "review:create"
	ReviewEditOwn   = "review:edit:own"
	ReviewDeleteOwn = "review:delete:own"
	ReviewListBrief = "review:list:brief"
)

var roleCapabilities = map[string][]string{
	"super_admin": {
		AdminDashboardView, AdminReviewsManage, AdminReportsManage,
		AdminTeachersManage, AdminSensitiveWordsManage, AdminLogsView,
		UserIdentityRead, UserIdentityReview,
		UserStudentRead, UserStudentReview,
		UserSchoolRead, UserSchoolUpdate,
		UserSystemRead, UserSystemUpdate,
		AdmissionPolicyRead, AdmissionPolicyUpdate,
		AdmissionFreshmanRead, AdmissionFreshmanReview,
		AdmissionSessionRead, MemberBlacklistRead, MemberBlacklistManage,
		OpenPlatformRead, OpenPlatformManage,
	},
	"school_admin": {
		AdminReviewsManage, AdminReportsManage,
		UserStudentRead, UserStudentReview,
		UserSchoolRead, UserSchoolUpdate,
	},
	"section_admin": {
		AdminReviewsManage, AdminReportsManage,
	},
	"section_moderator": {
		AdminReviewsManage, AdminReportsManage,
	},
	"section_reviewer": {},
	"verified_student": {
		ReviewListFull, ReviewCreate, ReviewEditOwn, ReviewDeleteOwn,
	},
	"freshman_provisional": {
		ReviewListFull, ReviewCreate, ReviewEditOwn, ReviewDeleteOwn,
	},
	"user": {
		ReviewListBrief,
	},
}

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
	AdmissionPolicyRead,
	AdmissionPolicyUpdate,
	AdmissionFreshmanRead,
	AdmissionFreshmanReview,
	AdmissionSessionRead,
	MemberBlacklistRead,
	MemberBlacklistManage,
	OpenPlatformRead,
	OpenPlatformManage,
}
