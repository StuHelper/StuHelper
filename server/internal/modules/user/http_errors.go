package user

import (
	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

var adminUpdateSystemConfigErrorMappings = []response.ErrorMapping{
	response.MatchError(ErrSystemConfigNotFound, 404, "system config not found", errs.ErrSystemConfigNotFound),
	response.MatchError(ErrInvalidSystemConfigValue, 400, "invalid system config value", errs.ErrInvalidParam),
}

func respondAdminUpdateSystemConfigError(c *gin.Context, err error) bool {
	return response.RespondMappedError(c, err, adminUpdateSystemConfigErrorMappings...)
}
