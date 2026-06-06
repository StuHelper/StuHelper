package user

import (
	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

var (
	qqBindingErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrUserIDInvalid, 400, "user id is invalid", errs.ErrInvalidParam),
		response.MatchError(ErrQQBindingAlreadyExists, 409, "qq binding already exists", errs.ErrConflict),
	}
	qqBindingCodeErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrUserIDInvalid, 400, "user id is invalid", errs.ErrInvalidParam),
		response.MatchError(ErrQQBindingCodeTTLInvalid, 400, "qq binding code ttl is invalid", errs.ErrInvalidParam),
		response.MatchError(ErrQQBindingAlreadyExists, 409, "qq binding already exists", errs.ErrConflict),
	}
	qqBindingConsumeErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrQQIDRequired, 400, "qq id is required", errs.ErrInvalidParam),
		response.MatchError(ErrQQBindingCodeInvalid, 400, "qq binding code is invalid", errs.ErrInvalidParam),
		response.MatchError(ErrQQBindingCodeExpired, 400, "qq binding code has expired", errs.ErrInvalidParam),
		response.MatchError(ErrQQBindingQQAlreadyBound, 409, "qq account already bound to another user", errs.ErrConflict),
		response.MatchError(ErrQQBindingUserConflict, 409, "user already bound to another qq account", errs.ErrConflict),
	}
	qqVerificationErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrQQIDRequired, 400, "qq id is required", errs.ErrInvalidParam),
	}
)

func respondQQBindingError(c *gin.Context, err error) bool {
	return response.RespondMappedError(c, err, qqBindingErrorMappings...)
}

func respondQQBindingCodeError(c *gin.Context, err error) bool {
	return response.RespondMappedError(c, err, qqBindingCodeErrorMappings...)
}

func respondQQBindingConsumeError(c *gin.Context, err error) bool {
	return response.RespondMappedError(c, err, qqBindingConsumeErrorMappings...)
}

func respondQQVerificationError(c *gin.Context, err error) bool {
	return response.RespondMappedError(c, err, qqVerificationErrorMappings...)
}
