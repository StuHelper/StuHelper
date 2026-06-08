package app

import (
	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/academics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/admission"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/auth"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/course/review"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/openplatform"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/rbac"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/storage"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
)

func adminEntryAuthorizer() gin.HandlerFunc {
	return rbac.RequireAnyCapability(capability.AdminEntryCapabilities...)
}

func adminStepUpBypass() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func adminStepUpVerifiedBypass(*gin.Context) bool {
	return true
}

func authAdminAuthorizers() auth.AdminAuthorizers {
	return auth.AdminAuthorizers{
		AccountLockUpdate: rbac.RequireGlobalCapability(capability.UserSystemUpdate),
		StepUpMFA:         adminStepUpBypass(),
	}
}

func academicsAdminAuthorizers() academics.AdminAuthorizers {
	return academics.AdminAuthorizers{
		Read:   rbac.RequireGlobalCapability(capability.UserSchoolRead),
		Update: rbac.RequireGlobalCapability(capability.UserSchoolUpdate),
	}
}

func storageAdminAuthorizers() storage.AdminAuthorizers {
	return storage.AdminAuthorizers{
		Read:   rbac.RequireGlobalCapability(capability.UserSystemRead),
		Update: rbac.RequireGlobalCapability(capability.UserSystemUpdate),
	}
}

func userAdminAuthorizers() user.AdminAuthorizers {
	return user.AdminAuthorizers{
		IdentityRead:   rbac.RequireGlobalCapability(capability.UserIdentityRead),
		IdentityReview: rbac.RequireGlobalCapability(capability.UserIdentityReview),
		StudentRead:    rbac.RequireCapability(capability.UserStudentRead),
		StudentReview:  rbac.RequireCapability(capability.UserStudentReview),
		SchoolRead:     rbac.RequireCapability(capability.UserSchoolRead),
		SchoolUpdate:   rbac.RequireCapability(capability.UserSchoolUpdate),
		SystemRead:     rbac.RequireGlobalCapability(capability.UserSystemRead),
		SystemUpdate:   rbac.RequireGlobalCapability(capability.UserSystemUpdate),
		StepUpMFA:      adminStepUpBypass(),
	}
}

func admissionAdminAuthorizers() admission.AdminAuthorizers {
	return admission.AdminAuthorizers{
		AdmissionPolicyRead:     rbac.RequireCapability(capability.AdmissionPolicyRead),
		AdmissionPolicyUpdate:   rbac.RequireCapability(capability.AdmissionPolicyUpdate),
		AdmissionSessionRead:    rbac.RequireCapability(capability.AdmissionSessionRead),
		AdmissionSessionManage:  rbac.RequireCapability(capability.AdmissionSessionManage),
		AdmissionFreshmanRead:   rbac.RequireCapability(capability.AdmissionFreshmanRead),
		AdmissionFreshmanReview: rbac.RequireCapability(capability.AdmissionFreshmanReview),
		MemberBlacklistRead:     rbac.RequireCapability(capability.MemberBlacklistRead),
		MemberBlacklistManage:   rbac.RequireCapability(capability.MemberBlacklistManage),
	}
}

func reviewAdminAuthorizers() review.AdminAuthorizers {
	return review.AdminAuthorizers{
		Entry:                adminEntryAuthorizer(),
		DashboardView:        rbac.RequireGlobalCapability(capability.AdminDashboardView),
		LogsView:             rbac.RequireGlobalCapability(capability.AdminLogsView),
		ReviewsManage:        rbac.RequireGlobalCapability(capability.AdminReviewsManage),
		TeachersManage:       rbac.RequireGlobalCapability(capability.AdminTeachersManage),
		SensitiveWordsManage: rbac.RequireGlobalCapability(capability.AdminSensitiveWordsManage),
		StepUpVerified:       adminStepUpVerifiedBypass,
	}
}

func openPlatformAdminAuthorizers() openplatform.AdminAuthorizers {
	return openplatform.AdminAuthorizers{
		Manage: rbac.RequireGlobalCapability(capability.OpenPlatformManage),
	}
}
