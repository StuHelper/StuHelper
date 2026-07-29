package rbac

import (
	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/platform/authorization"
)

func subjectFromGin(c *gin.Context) authorization.Subject {
	return authorization.Subject{
		UserID:              middleware.GetUserID(c),
		AppID:               middleware.GetAppID(c),
		Roles:               middleware.GetRoles(c),
		Capabilities:        middleware.GetCapabilities(c),
		CapabilityGrants:    middleware.GetCapabilityGrants(c),
		GlobalCapabilities:  middleware.GetGlobalCapabilities(c),
		MFAEnrollmentActive: middleware.GetMFAEnrollmentActive(c),
		MFAProofVerifiedAt:  middleware.GetMFAProofVerifiedAt(c),
	}
}
