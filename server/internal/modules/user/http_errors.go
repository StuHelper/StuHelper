package user

import (
	"errors"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

var (
	submitIdentityErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrUserIDInvalid, 400, "user id is invalid", errs.ErrInvalidParam),
		response.MatchError(ErrIdentityAlreadyVerified, 409, "identity already verified", errs.ErrIdentityAlreadyVerified),
		response.MatchError(ErrIdentityAlreadyExists, 409, "identity already submitted", errs.ErrIdentityAlreadyExists),
		response.MatchError(ErrPhotoRequired, 400, "photo upload required for non-mainland documents", errs.ErrIdentityPhotoRequired),
		response.MatchError(ErrIdentityPhotoInvalidRef, 400, "invalid identity photo reference"),
		response.MatchError(ErrIdentityPhotoStoreDisabled, 503, "identity photo upload is not available", errs.ErrServiceUnavailable),
		response.MatchError(ErrIdentityPhotoStorageUnavailable, 503, "identity photo upload is not available", errs.ErrServiceUnavailable),
		response.MatchError(ErrIdentityPhotoStorageTemporaryUnavailable, 503, "identity photo storage is temporarily unavailable", errs.ErrServiceUnavailable),
		response.MatchError(ErrIdentityDocNumberInvalid, 400, "identity document number is invalid"),
		response.MatchError(ErrIdentityRealNameInvalid, 400, "identity real name is invalid"),
	}
	uploadIdentityPhotoErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrUserIDInvalid, 400, "user id is invalid", errs.ErrInvalidParam),
		response.MatchError(ErrIdentityPhotoStoreDisabled, 503, "identity photo upload is not available", errs.ErrServiceUnavailable),
		response.MatchError(ErrIdentityPhotoTooLarge, 400, "identity photo is too large"),
		response.MatchError(ErrIdentityPhotoInvalidType, 400, "identity photo content type is invalid"),
		response.MatchError(ErrIdentityPhotoInvalidData, 400, "identity photo data is invalid"),
		response.MatchError(ErrIdentityPhotoInvalidRef, 400, "identity photo data is invalid"),
		response.MatchError(ErrIdentityPhotoStorageUnavailable, 503, "identity photo upload is not available", errs.ErrServiceUnavailable),
		response.MatchError(ErrIdentityPhotoStorageTemporaryUnavailable, 503, "identity photo storage is temporarily unavailable", errs.ErrServiceUnavailable),
	}
	verifyStudentErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrUserIDInvalid, 400, "user id is invalid", errs.ErrInvalidParam),
		response.MatchError(ErrProfileAlreadyVerified, 409, "student profile already verified", errs.ErrProfileAlreadyVerified),
		response.MatchError(ErrProfilePendingReview, 409, "student profile is pending review", errs.ErrProfilePendingReview),
		response.MatchError(ErrSchoolNotFound, 404, "school not found", errs.ErrProfileSchoolNotFound),
		response.MatchError(ErrSchoolDisabled, 400, "school verification is not enabled", errs.ErrProfileSchoolDisabled),
		response.MatchError(ErrConsentRequired, 400, "consent is required for verification", errs.ErrProfileConsentRequired),
		response.MatchError(ErrStudentIDRequired, 400, "student ID is required for this verification method"),
		response.MatchError(ErrStudentIDInvalid, 400, "student ID is invalid"),
		response.MatchError(ErrStudentNameRequired, 400, "student name is required for this verification method"),
		response.MatchError(ErrStudentNameInvalid, 400, "student name is invalid"),
		response.MatchError(ErrStudentNameMismatch, 400, "student name does not match academic database"),
		response.MatchError(ErrStudentEmailDomainNotAllowed, 400, "student school email is not allowed"),
		response.MatchError(ErrStudentEmailOTPCooldown, 429, "please wait before requesting a new code"),
		response.MatchError(ErrStudentEmailOTPExpired, 400, "student email otp is invalid or expired"),
		response.MatchError(ErrStudentEmailOTPInvalid, 400, "student email otp is invalid or expired"),
		response.MatchError(ErrStudentEmailOTPMaxAttempts, 429, "too many failed attempts, please request a new code"),
		response.MatchError(ErrStudentEmailSenderUnavailable, 503, "student email verification is not configured", errs.ErrServiceUnavailable),
		response.MatchError(ErrStudentEmailRedisUnavailable, 503, "student email verification is unavailable", errs.ErrServiceUnavailable),
		response.MatchError(ErrPasswordRequired, 400, "password is required for this verification method"),
		response.MatchError(ErrManualFieldRequired, 400, "required form field is missing"),
		response.MatchError(ErrManualFieldInvalid, 400, "form field validation failed"),
		response.MatchError(ErrInvalidAcademicDBTable, 400, "school academic table configuration is invalid", errs.ErrProfileAcademicTable),
		response.MatchError(ErrAcademicTableNotConfigured, 400, "school academic table is not configured", errs.ErrAcademicTableNotConfigured),
		response.MatchError(ErrAcademicLookupUnavailable, 503, "academic student lookup is temporarily unavailable", errs.ErrServiceUnavailable),
		response.MatchError(ErrSchoolLDAPConfigMissing, 400, "school LDAP configuration is missing", errs.ErrSchoolLDAPConfigMissing),
		response.MatchError(ErrLDAPConfigInvalid, 400, "school LDAP configuration is invalid", errs.ErrLDAPConfigInvalid),
		response.MatchError(ErrLDAPFailed, 400, "LDAP verification failed, please check your credentials", errs.ErrProfileLDAPFailed),
	}
	bindPhoneErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrUserIDInvalid, 400, "user id is invalid", errs.ErrInvalidParam),
		response.MatchError(ErrInvalidPhoneFormat, 400, "invalid phone number format"),
		response.MatchError(ErrPhoneAlreadyBound, 409, "phone number already bound"),
		response.MatchError(ErrProfileNotFound, 404, "student profile not found", errs.ErrProfileNotFound),
	}
	academicInfoErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrSchoolNotFound, 404, "school configuration not found", errs.ErrProfileSchoolNotFound),
		response.MatchError(ErrSchoolDisabled, 400, "school verification channel disabled", errs.ErrProfileSchoolDisabled),
		response.MatchError(ErrAcademicTableNotConfigured, 400, "academic table is not configured", errs.ErrAcademicTableNotConfigured),
		response.MatchError(ErrInvalidAcademicDBTable, 400, "academic table configuration is invalid", errs.ErrProfileAcademicTable),
		response.MatchError(ErrAcademicLookupUnavailable, 503, "academic student lookup is temporarily unavailable", errs.ErrServiceUnavailable),
	}
	adminReviewIdentityErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrUserIDInvalid, 400, "user id is invalid", errs.ErrInvalidParam),
		response.MatchError(ErrIdentityNotFound, 404, "identity not found", errs.ErrIdentityNotFound),
	}
	adminReviewStudentVerificationErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrUserIDInvalid, 400, "user id is invalid", errs.ErrInvalidParam),
		response.MatchError(ErrProfileNotFound, 404, "student profile not found", errs.ErrProfileNotFound),
	}
	adminUpdateSchoolConfigErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrSchoolNotFound, 404, "school config not found", errs.ErrProfileSchoolNotFound),
		response.MatchError(ErrInvalidSchoolConfigValue, 400, "invalid school config value", errs.ErrInvalidParam),
		response.MatchError(ErrInvalidManualFieldConfig, 400, "invalid manual form field configuration"),
		response.MatchError(ErrAcademicTableNotConfigured, 400, "academic db table is required for enabled LDAP schools", errs.ErrAcademicTableNotConfigured),
		response.MatchError(ErrInvalidAcademicDBTable, 400, "invalid academic db table configuration", errs.ErrProfileAcademicTable),
		response.MatchError(ErrSchoolLDAPConfigMissing, 400, "school LDAP configuration is required for enabled LDAP schools", errs.ErrSchoolLDAPConfigMissing),
		response.MatchError(ErrLDAPConfigInvalid, 400, "school LDAP configuration is invalid", errs.ErrLDAPConfigInvalid),
	}
	adminUpdateSystemConfigErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrSystemConfigNotFound, 404, "system config not found", errs.ErrSystemConfigNotFound),
		response.MatchError(ErrInvalidSystemConfigValue, 400, "invalid system config value", errs.ErrInvalidParam),
	}
)

func respondSubmitIdentityError(c *gin.Context, err error) bool {
	return response.RespondMappedError(c, err, submitIdentityErrorMappings...)
}

func respondUploadIdentityPhotoError(c *gin.Context, err error) bool {
	return response.RespondMappedError(c, err, uploadIdentityPhotoErrorMappings...)
}

func respondVerifyStudentError(c *gin.Context, err error) bool {
	return response.RespondMappedError(c, err, verifyStudentErrorMappings...)
}

func respondBindPhoneError(c *gin.Context, err error) bool {
	return response.RespondMappedError(c, err, bindPhoneErrorMappings...)
}

func respondAcademicInfoError(c *gin.Context, err error) bool {
	return response.RespondMappedError(c, err, academicInfoErrorMappings...)
}

func respondAdminReviewIdentityError(c *gin.Context, err error) bool {
	return response.RespondMappedError(c, err, adminReviewIdentityErrorMappings...)
}

func respondAdminReviewStudentVerificationError(c *gin.Context, err error) bool {
	return response.RespondMappedError(c, err, adminReviewStudentVerificationErrorMappings...)
}

func respondAdminUpdateSchoolConfigError(c *gin.Context, err error) bool {
	return response.RespondMappedError(c, err, adminUpdateSchoolConfigErrorMappings...)
}

func respondAdminUpdateSystemConfigError(c *gin.Context, err error) bool {
	return response.RespondMappedError(c, err, adminUpdateSystemConfigErrorMappings...)
}

func shouldWarnAdminSchoolConfigError(err error) bool {
	return errors.Is(err, ErrInvalidSchoolConfigValue) ||
		errors.Is(err, ErrInvalidManualFieldConfig) ||
		errors.Is(err, ErrAcademicTableNotConfigured) ||
		errors.Is(err, ErrInvalidAcademicDBTable) ||
		errors.Is(err, ErrSchoolLDAPConfigMissing) ||
		errors.Is(err, ErrLDAPConfigInvalid)
}
