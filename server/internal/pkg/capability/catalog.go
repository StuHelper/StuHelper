package capability

const (
	AdminDashboardView        = "admin:dashboard:view"
	AdminReviewsManage        = "admin:reviews:manage"
	AdminReviewsEditContent   = "admin:reviews:edit_content"
	AdminReportsManage        = "admin:reports:manage"
	AdminTeachersManage       = "admin:teachers:manage"
	AdminSensitiveWordsManage = "admin:sensitive_words:manage"
	AdminLogsView             = "admin:logs:view"

	UserSchoolRead                  = "user:school:read"
	UserSchoolUpdate                = "user:school:update"
	UserSystemRead                  = "user:system:read"
	UserSystemUpdate                = "user:system:update"
	IAMGrantsManage                 = "iam:grants:manage"
	StudentRosterRead               = "student:roster:read"
	StudentRosterActivate           = "student:roster:activate"
	StudentRosterDecryptPII         = "student:roster:decrypt_pii"
	StudentManualReviewRead         = "student:manual_review:read"
	StudentManualReviewDecide       = "student:manual_review:decide"
	StudentManualMaterialAccess     = "student:manual_material:access"
	StudentVerificationConfigRead   = "student:verification_config:read"
	StudentVerificationConfigUpdate = "student:verification_config:update"
	StudentCredentialRead           = "student:credential:read"
	StudentCredentialRevoke         = "student:credential:revoke"
	StudentSubjectConflictRead      = "student:subject_conflict:read"
	StudentSubjectConflictResolve   = "student:subject_conflict:resolve"
	CampusConnectorHealthRead       = "campus_connector:health:read"
	CampusConnectorManage           = "campus_connector:manage"

	AdmissionPolicyRead    = "admission:policy:read"
	AdmissionPolicyUpdate  = "admission:policy:update"
	AdmissionSessionRead   = "admission:session:read"
	AdmissionSessionManage = "admission:session:manage"
	MemberBlacklistRead    = "member_blacklist:read"
	MemberBlacklistManage  = "member_blacklist:manage"
	OpenPlatformRead       = "open_platform:read"
	OpenPlatformManage     = "open_platform:manage"

	ReviewListFull  = "review:list:full"
	ReviewCreate    = "review:create"
	ReviewEditOwn   = "review:edit:own"
	ReviewDeleteOwn = "review:delete:own"
	ReviewListBrief = "review:list:brief"
)

var roleCapabilities = map[string][]string{
	"super_admin": {
		AdminDashboardView, AdminReviewsManage, AdminReviewsEditContent, AdminReportsManage,
		AdminTeachersManage, AdminSensitiveWordsManage, AdminLogsView,
		UserSchoolRead, UserSchoolUpdate,
		UserSystemRead, UserSystemUpdate,
		IAMGrantsManage,
		StudentRosterRead, StudentRosterActivate, StudentRosterDecryptPII,
		StudentManualReviewRead, StudentManualReviewDecide, StudentManualMaterialAccess,
		StudentVerificationConfigRead, StudentVerificationConfigUpdate,
		StudentCredentialRead, StudentCredentialRevoke,
		StudentSubjectConflictRead, StudentSubjectConflictResolve,
		CampusConnectorHealthRead, CampusConnectorManage,
		AdmissionPolicyRead, AdmissionPolicyUpdate,
		AdmissionSessionRead, AdmissionSessionManage,
		MemberBlacklistRead, MemberBlacklistManage,
		OpenPlatformRead, OpenPlatformManage,
	},
	"school_admin": {
		AdminReviewsManage, AdminReviewsEditContent, AdminReportsManage,
		UserSchoolRead, UserSchoolUpdate,
		StudentRosterRead, StudentRosterActivate,
		StudentManualReviewRead, StudentManualReviewDecide, StudentManualMaterialAccess,
		StudentVerificationConfigRead, StudentVerificationConfigUpdate,
		StudentCredentialRead, StudentCredentialRevoke,
		StudentSubjectConflictRead, StudentSubjectConflictResolve,
		CampusConnectorHealthRead,
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
	AdminReviewsEditContent,
	AdminReportsManage,
	AdminTeachersManage,
	AdminSensitiveWordsManage,
	AdminLogsView,
	UserSchoolRead,
	UserSchoolUpdate,
	UserSystemRead,
	UserSystemUpdate,
	IAMGrantsManage,
	StudentRosterRead,
	StudentRosterActivate,
	StudentRosterDecryptPII,
	StudentManualReviewRead,
	StudentManualReviewDecide,
	StudentManualMaterialAccess,
	StudentVerificationConfigRead,
	StudentVerificationConfigUpdate,
	StudentCredentialRead,
	StudentCredentialRevoke,
	StudentSubjectConflictRead,
	StudentSubjectConflictResolve,
	CampusConnectorHealthRead,
	CampusConnectorManage,
	AdmissionPolicyRead,
	AdmissionPolicyUpdate,
	AdmissionSessionRead,
	AdmissionSessionManage,
	MemberBlacklistRead,
	MemberBlacklistManage,
	OpenPlatformRead,
	OpenPlatformManage,
}
