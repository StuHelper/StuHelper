package app

import (
	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/modules/academics"
	"github.com/StuHelper/StuHelper/server/internal/modules/admission"
	"github.com/StuHelper/StuHelper/server/internal/modules/auth"
	authorizationmodule "github.com/StuHelper/StuHelper/server/internal/modules/authorization"
	"github.com/StuHelper/StuHelper/server/internal/modules/course/review"
	"github.com/StuHelper/StuHelper/server/internal/modules/openplatform"
	"github.com/StuHelper/StuHelper/server/internal/modules/rbac"
	"github.com/StuHelper/StuHelper/server/internal/modules/storage"
	"github.com/StuHelper/StuHelper/server/internal/modules/user"
	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
)

func adminEntryAuthorizer() gin.HandlerFunc {
	return rbac.RequireAnyCapability(capability.AdminEntryCapabilities...)
}

func authAdminAuthorizers() auth.AdminAuthorizers {
	return auth.AdminAuthorizers{
		AccountLockUpdate: rbac.RequireGlobalCapability(capability.UserSystemUpdate),
		StepUpMFA:         rbac.RequireStepUpMFA(),
	}
}

func authorizationAdminAuthorizers() authorizationmodule.AdminAuthorizers {
	return authorizationmodule.AdminAuthorizers{
		Manage:    rbac.RequireGlobalCapability(capability.IAMGrantsManage),
		StepUpMFA: rbac.RequireStepUpMFA(),
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
		StepUpMFA:      rbac.RequireStepUpMFA(),
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
		ReviewsModerate:      rbac.RequireCapability(capability.AdminReviewsManage),
		ReviewsEditContent:   rbac.RequireCapability(capability.AdminReviewsEditContent),
		ReportsManage:        rbac.RequireCapability(capability.AdminReportsManage),
		TeachersManage:       rbac.RequireGlobalCapability(capability.AdminTeachersManage),
		SensitiveWordsManage: rbac.RequireGlobalCapability(capability.AdminSensitiveWordsManage),
		StepUpVerified:       rbac.EnsureStepUpMFA,
	}
}

func openPlatformAdminAuthorizers() openplatform.AdminAuthorizers {
	return openplatform.AdminAuthorizers{
		Manage: rbac.RequireGlobalCapability(capability.OpenPlatformManage),
	}
}
