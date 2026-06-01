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
	ErrCodeAdmissionMemberBlacklisted errs.ErrorCode = "admission.member_blacklisted"
	ErrCodeAdmissionQQMismatch        errs.ErrorCode = "admission.qq_mismatch"
	ErrCodeAdmissionTokenConsumed     errs.ErrorCode = "admission.token_consumed"
	ErrCodeAdmissionTokenExpired      errs.ErrorCode = "admission.token_expired"
)

func respondAdmissionError(c *gin.Context, err error) {
	if respondAdmissionSessionError(c, err) {
		return
	}
	if respondMemberBlacklistError(c, err) {
		return
	}
	if respondFreshmanAdmissionError(c, err) {
		return
	}
	if respondAdmissionVerificationError(c, err) {
		return
	}
	if respondAdmissionDependencyError(c, err) {
		return
	}
	response.InternalError(c, "admission request failed")
}

func respondAdmissionSessionError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, ErrAdmissionQQMismatch):
		response.BadRequest(c, "admission link qq parameter mismatch", ErrCodeAdmissionQQMismatch)
	case errors.Is(err, ErrAdmissionTokenConsumed):
		response.Conflict(c, "admission token already consumed", ErrCodeAdmissionTokenConsumed)
	case errors.Is(err, ErrAdmissionTokenExpired):
		response.Error(c, http.StatusGone, ErrCodeAdmissionTokenExpired, "admission token expired")
	case errors.Is(err, ErrAdmissionTokenNotFound):
		response.NotFound(c, "admission token not found")
	case errors.Is(err, ErrAdmissionSessionNotFound):
		response.NotFound(c, "admission session not found")
	case errors.Is(err, ErrAdmissionInvalidInput):
		response.BadRequest(c, "admission input invalid")
	case errors.Is(err, ErrAdmissionInvalidStatus):
		response.Conflict(c, "admission session status invalid")
	case errors.Is(err, ErrAdmissionApplicationNotFound):
		response.NotFound(c, "admission application not found")
	case errors.Is(err, ErrAdmissionSchoolNotFound):
		response.NotFound(c, "admission school not found")
	case errors.Is(err, ErrAdmissionSchoolDisabled):
		response.BadRequest(c, "admission school disabled")
	case errors.Is(err, ErrAdmissionLinkedSessionRequired):
		response.Conflict(c, "admission linked session required")
	case errors.Is(err, ErrAdmissionPolicyNotFound):
		response.NotFound(c, "admission policy not found")
	case errors.Is(err, ErrAdmissionBlacklistNotFound):
		response.NotFound(c, "admission blacklist not found")
	case errors.Is(err, ErrAdmissionPendingActionFilterInvalid):
		response.BadRequest(c, "admission pending action filter invalid")
	case errors.Is(err, ErrMemberBlacklisted):
		response.Conflict(c, "member is blacklisted", ErrCodeAdmissionMemberBlacklisted)
	case errors.Is(err, ErrMemberBlacklistNotFound):
		response.NotFound(c, "member blacklist not found")
	case errors.Is(err, ErrMemberBlacklistInvalidInput):
		response.BadRequest(c, "invalid member blacklist request")
	case errors.Is(err, ErrMemberBlacklistSourceForbidden):
		response.Forbidden(c, "member blacklist source forbidden")
	case errors.Is(err, ErrAdmissionOperatorUnbound),
		errors.Is(err, ErrAdmissionOperatorForbidden),
		errors.Is(err, ErrAdmissionManagementGuildForbidden):
		response.Forbidden(c, "forbidden")
	case errors.Is(err, ErrAdmissionReviewExtensionTooLong):
		response.BadRequest(c, "freshman credential extension exceeds policy limit")
	default:
		return false
	}
	return true
}

func respondMemberBlacklistError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, ErrMemberBlacklistInvalidInput):
		response.BadRequest(c, "member blacklist input invalid")
	case errors.Is(err, ErrMemberBlacklistNotFound):
		response.NotFound(c, "member blacklist not found")
	default:
		return false
	}
	return true
}

func respondFreshmanAdmissionError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, ErrAdmissionFreshmanChannelClosed):
		response.Conflict(c, "freshman admission channel closed")
	case errors.Is(err, ErrAdmissionFreshmanPendingExists):
		response.Conflict(c, "freshman application pending already exists")
	case errors.Is(err, ErrAdmissionMaterialInvalidType):
		response.Error(c, http.StatusUnsupportedMediaType, errs.ErrUnsupportedMedia, "admission material content type invalid")
	case errors.Is(err, ErrAdmissionMaterialInvalidData):
		response.BadRequest(c, "admission material image data invalid")
	case errors.Is(err, ErrAdmissionMaterialTooLarge):
		response.Error(c, http.StatusRequestEntityTooLarge, errs.ErrPayloadTooLarge, "admission material too large")
	case errors.Is(err, ErrAdmissionCameraHandoffNotFound):
		response.NotFound(c, "admission camera handoff not found")
	case errors.Is(err, ErrAdmissionCameraHandoffExpired):
		response.Error(c, http.StatusGone, errs.ErrInvalidParam, "admission camera handoff expired")
	case errors.Is(err, ErrAdmissionCameraHandoffLocked):
		response.Conflict(c, "admission camera handoff locked")
	case errors.Is(err, ErrAdmissionCameraHandoffInvalidChoice):
		response.BadRequest(c, "admission camera handoff continuation invalid")
	default:
		return false
	}
	return true
}

func respondAdmissionVerificationError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, ErrAdmissionEmailDomainNotAllowed):
		response.BadRequest(c, "admission school email domain not allowed")
	case errors.Is(err, ErrAdmissionStudentIDRequired):
		response.BadRequest(c, "admission student id required")
	case errors.Is(err, ErrAdmissionStudentNameRequired):
		response.BadRequest(c, "admission student name required")
	case errors.Is(err, ErrAdmissionStudentRecordNotFound):
		response.BadRequest(c, "admission student record not found")
	case errors.Is(err, ErrAdmissionStudentNameMismatch):
		response.BadRequest(c, "admission student name mismatch")
	case errors.Is(err, ErrAdmissionOTPCooldown):
		response.RateLimitExceeded(c, "please wait before requesting a new code")
	case errors.Is(err, ErrAdmissionOTPExpired), errors.Is(err, ErrAdmissionOTPInvalid):
		response.BadRequest(c, "admission email otp invalid")
	case errors.Is(err, ErrAdmissionOTPMaxAttempts):
		response.RateLimitExceeded(c, "too many failed attempts, please request a new code")
	case errors.Is(err, ErrAdmissionSSONotConfigured):
		response.ServiceUnavailable(c, "admission school sso not configured")
	case errors.Is(err, ErrAdmissionSSOStateInvalid):
		response.BadRequest(c, "invalid or expired school sso state")
	case errors.Is(err, ErrAdmissionSSOExchangeFailed):
		response.Unauthorized(c, "school sso verification failed", errs.ErrOAuthCodeInvalid)
	case errors.Is(err, ErrAdmissionSSOIdentityInvalid):
		response.Unauthorized(c, "school sso identity invalid", errs.ErrOAuthFailed)
	case errors.Is(err, ErrAdmissionSSOProviderUnavailable):
		response.ServiceUnavailable(c, "school sso provider unavailable")
	case errors.Is(err, ErrAdmissionReturnURLNotAllowed):
		response.BadRequest(c, "admission return url not allowed")
	default:
		return false
	}
	return true
}

func respondAdmissionDependencyError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, ErrAdmissionOperatorAccessUnavailable):
		response.ServiceUnavailable(c, "operator access verification unavailable")
	case errors.Is(err, ErrAdmissionProjectionUnavailable),
		errors.Is(err, ErrAdmissionMaterialStoreUnavailable),
		errors.Is(err, ErrAdmissionEmailSenderUnavailable),
		errors.Is(err, ErrAdmissionAcademicLookupUnavailable),
		errors.Is(err, ErrAdmissionRedisUnavailable):
		response.ServiceUnavailable(c, "admission dependency unavailable")
	case errors.Is(err, user.ErrQQBindingQQAlreadyBound), errors.Is(err, user.ErrQQBindingUserConflict):
		response.Conflict(c, "qq binding conflict")
	default:
		return false
	}
	return true
}
