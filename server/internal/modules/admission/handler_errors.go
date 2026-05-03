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
	case errors.Is(err, ErrAdmissionApplicationNotFound):
		response.NotFound(c, "admission application not found")
	case errors.Is(err, ErrAdmissionOperatorUnbound):
		response.Forbidden(c, "operator qq is not bound to a StuHelper admin account")
	case errors.Is(err, ErrAdmissionOperatorForbidden):
		response.Forbidden(c, "operator does not have admission review permission")
	case errors.Is(err, ErrAdmissionManagementGuildForbidden):
		response.Forbidden(c, "management group is not allowed for this admission policy")
	case errors.Is(err, ErrAdmissionReviewExtensionTooLong):
		response.BadRequest(c, "freshman credential extension exceeds policy limit")
	case errors.Is(err, ErrAdmissionOperatorAccessUnavailable):
		response.ServiceUnavailable(c, "operator access verification unavailable")
	case errors.Is(err, user.ErrQQBindingQQAlreadyBound), errors.Is(err, user.ErrQQBindingUserConflict):
		response.Conflict(c, "qq binding conflict")
	default:
		response.InternalError(c, "admission request failed")
	}
}
