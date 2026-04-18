package review

import (
	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

func (h *Handler) resolveRequiredUserHash(c *gin.Context) (string, string, bool) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c, "missing authentication token", errs.ErrTokenMissing)
		return "", "", false
	}
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return "", "", false
	}
	return userID, userHash, true
}

func (h *Handler) resolveOptionalUserHash(c *gin.Context) (string, bool) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return "", true
	}
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return "", false
	}
	return userHash, true
}
