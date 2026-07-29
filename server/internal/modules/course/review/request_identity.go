package review

import (
	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
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
