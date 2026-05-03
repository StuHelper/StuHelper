package admission

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

const (
	ErrCodeAdmissionQQMismatch    errs.ErrorCode = "admission.qq_mismatch"
	ErrCodeAdmissionTokenConsumed errs.ErrorCode = "admission.token_consumed"
	ErrCodeAdmissionTokenExpired  errs.ErrorCode = "admission.token_expired"
)

func respondAdmissionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrAdmissionQQMismatch):
		response.BadRequest(c, "admission link qq parameter mismatch", ErrCodeAdmissionQQMismatch)
	case errors.Is(err, ErrAdmissionTokenConsumed):
		response.Conflict(c, "admission token already consumed", ErrCodeAdmissionTokenConsumed)
	case errors.Is(err, ErrAdmissionTokenExpired):
		response.Error(c, http.StatusGone, ErrCodeAdmissionTokenExpired, "admission token expired")
	case errors.Is(err, ErrAdmissionTokenNotFound):
		response.NotFound(c, "admission token not found")
	case errors.Is(err, ErrAdmissionInvalidStatus):
		response.Conflict(c, "admission session status invalid")
	case errors.Is(err, user.ErrQQBindingQQAlreadyBound), errors.Is(err, user.ErrQQBindingUserConflict):
		response.Conflict(c, "qq binding conflict")
	default:
		response.InternalError(c, "admission request failed")
	}
}
