package authorization

import (
	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

func SubjectFromGin(c *gin.Context) Subject {
	return Subject{
		UserID:             middleware.GetUserID(c),
		Roles:              middleware.GetRoles(c),
		Capabilities:       middleware.GetCapabilities(c),
		CapabilityGrants:   middleware.GetCapabilityGrants(c),
		GlobalCapabilities: middleware.GetGlobalCapabilities(c),
	}
}
