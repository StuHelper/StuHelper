package user

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/auth"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

func (h *Handler) handleGetIdentity(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}

	identity, err := h.service.GetIdentity(c.Request.Context(), userID)
	if err != nil {
		logger.FromGin(c).Error("failed to get identity", zap.Error(err))
		response.InternalError(c, "failed to get identity information")
		return
	}
	if identity == nil {
		response.NotFound(c, "identity not found", errs.ErrIdentityNotFound)
		return
	}

	response.Success(c, identityStatusToJSON(identity))
}

func (h *Handler) handleGetUserSurface(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}

	displayName := middleware.GetDisplayName(c)
	avatarURL := middleware.GetAvatar(c)
	capabilities := middleware.GetCapabilities(c)

	surface, err := h.service.GetUserSurface(c.Request.Context(), userID, displayName, avatarURL, capabilities)
	if err != nil {
		logger.FromGin(c).Error("failed to get user surface", zap.Error(err))
		response.InternalError(c, "failed to get user information")
		return
	}

	response.Success(c, gin.H{
		"displayName":        surface.DisplayName,
		"avatarURL":          surface.AvatarURL,
		"identityStatus":     surface.IdentityStatus,
		"verificationStatus": surface.VerificationStatus,
		"phoneBound":         surface.PhoneBound,
		"capabilities":       surface.Capabilities,
	})
}

type submitIdentityHTTPRequest struct {
	DocType        string  `json:"docType" binding:"required,oneof=MAINLAND_ID HK_MACAU TW PASSPORT"`
	DocNumber      string  `json:"docNumber" binding:"required,max=50"`
	RealName       string  `json:"realName" binding:"required,max=100"`
	DocPhotoFront  *string `json:"docPhotoFront"`
	DocPhotoBack   *string `json:"docPhotoBack"`
	DocPhotoSelfie *string `json:"docPhotoSelfie"`
}

func (h *Handler) handleSubmitIdentity(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}

	var req submitIdentityHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	identity, err := h.service.SubmitIdentity(c.Request.Context(), userID, SubmitIdentityRequest(req))
	if err != nil {
		switch {
		case errors.Is(err, ErrIdentityAlreadyVerified):
			response.Conflict(c, "identity already verified", errs.ErrIdentityAlreadyVerified)
		case errors.Is(err, ErrIdentityAlreadyExists):
			response.Conflict(c, "identity already submitted", errs.ErrIdentityAlreadyExists)
		case errors.Is(err, ErrPhotoRequired):
			response.BadRequest(c, "photo upload required for non-mainland documents", errs.ErrIdentityPhotoRequired)
		case errors.Is(err, ErrIdentityPhotoInvalidRef):
			response.BadRequest(c, "invalid identity photo reference")
		default:
			logger.FromGin(c).Error("failed to submit identity", zap.Error(err))
			response.InternalError(c, "failed to submit identity")
		}
		return
	}

	response.Created(c, identityStatusToJSON(identity))
}

type uploadIdentityPhotoHTTPRequest struct {
	Slot        string `json:"slot" binding:"required,oneof=front back selfie"`
	Filename    string `json:"filename" binding:"required,max=255"`
	ContentType string `json:"contentType" binding:"required,oneof=image/jpeg image/png image/webp"`
	DataBase64  string `json:"dataBase64" binding:"required"`
}

func (h *Handler) handleUploadIdentityPhoto(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}

	var req uploadIdentityPhotoHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	key, err := h.service.UploadIdentityPhoto(c.Request.Context(), userID, UploadIdentityPhotoRequest(req))
	if err != nil {
		switch {
		case errors.Is(err, ErrIdentityPhotoStoreDisabled):
			logger.FromGin(c).Error("identity photo storage is not configured", zap.Error(err))
			response.InternalError(c, "identity photo upload is not available")
		case errors.Is(err, ErrIdentityPhotoTooLarge):
			response.BadRequest(c, "identity photo is too large")
		case errors.Is(err, ErrIdentityPhotoInvalidType):
			response.BadRequest(c, "identity photo content type is invalid")
		case errors.Is(err, ErrIdentityPhotoInvalidData), errors.Is(err, ErrIdentityPhotoInvalidRef):
			response.BadRequest(c, "identity photo data is invalid")
		default:
			logger.FromGin(c).Error("failed to upload identity photo", zap.Error(err))
			response.InternalError(c, "failed to upload identity photo")
		}
		return
	}

	response.Created(c, gin.H{"key": key})
}

func (h *Handler) handleGetProfile(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}

	profile, err := h.service.GetProfile(c.Request.Context(), userID)
	if err != nil {
		logger.FromGin(c).Error("failed to get profile", zap.Error(err))
		response.InternalError(c, "failed to get profile information")
		return
	}
	if profile == nil {
		response.NotFound(c, "student profile not found", errs.ErrProfileNotFound)
		return
	}

	response.Success(c, profileToJSON(profile))
}

type verifyStudentHTTPRequest struct {
	SchoolID       int64          `json:"schoolID" binding:"required,min=1"`
	StudentID      string         `json:"studentID" binding:"omitempty,max=50"`
	Password       string         `json:"password" binding:"omitempty,max=200"`
	ManualFormData map[string]any `json:"manualFormData"`
	Consent        bool           `json:"consent"`
}

func (h *Handler) handleVerifyStudent(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}

	var req verifyStudentHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	profile, err := h.service.VerifyStudent(c.Request.Context(), userID, VerifyStudentRequest(req))
	if err != nil {
		switch {
		case errors.Is(err, ErrIdentityRequired):
			response.Forbidden(c, "identity verification required before student verification", errs.ErrForbidden)
		case errors.Is(err, ErrProfileAlreadyVerified):
			response.Conflict(c, "student profile already verified", errs.ErrProfileAlreadyVerified)
		case errors.Is(err, ErrProfilePendingReview):
			response.Conflict(c, "student profile is pending review", errs.ErrProfilePendingReview)
		case errors.Is(err, ErrSchoolNotFound):
			response.NotFound(c, "school not found", errs.ErrProfileSchoolNotFound)
		case errors.Is(err, ErrSchoolDisabled):
			response.BadRequest(c, "school verification is not enabled", errs.ErrProfileSchoolDisabled)
		case errors.Is(err, ErrConsentRequired):
			response.BadRequest(c, "consent is required for verification", errs.ErrProfileConsentRequired)
		case errors.Is(err, ErrStudentIDRequired):
			response.BadRequest(c, "student ID is required for this verification method")
		case errors.Is(err, ErrPasswordRequired):
			response.BadRequest(c, "password is required for this verification method")
		case errors.Is(err, ErrManualFieldRequired):
			response.BadRequest(c, "required form field is missing")
		case errors.Is(err, ErrManualFieldInvalid):
			response.BadRequest(c, "form field validation failed")
		case errors.Is(err, ErrInvalidAcademicDBTable):
			response.BadRequest(c, "school academic table configuration is invalid", errs.ErrProfileAcademicTable)
		case errors.Is(err, ErrAcademicTableNotConfigured):
			response.BadRequest(c, "school academic table is not configured", errs.ErrAcademicTableNotConfigured)
		case errors.Is(err, ErrSchoolLDAPConfigMissing):
			response.BadRequest(c, "school LDAP configuration is missing", errs.ErrSchoolLDAPConfigMissing)
		case errors.Is(err, ErrLDAPConfigInvalid):
			response.BadRequest(c, "school LDAP configuration is invalid", errs.ErrLDAPConfigInvalid)
		case errors.Is(err, ErrLDAPFailed):
			response.BadRequest(c, "LDAP verification failed, please check your credentials", errs.ErrProfileLDAPFailed)
		default:
			logger.FromGin(c).Error("failed to verify student", zap.Error(err))
			response.InternalError(c, "failed to verify student")
		}
		return
	}

	response.Success(c, profileToJSON(profile))
}

type bindPhoneHTTPRequest struct {
	Phone   string `json:"phone" binding:"required,max=20"`
	OTPCode string `json:"otpCode" binding:"required,len=6"`
}

type requestBindPhoneOTPRequest struct {
	Phone string `json:"phone" binding:"required"`
}

func (h *Handler) handleRequestBindPhoneOTP(c *gin.Context) {
	if _, ok := h.resolveCurrentUser(c); !ok {
		return
	}
	if h.otpService == nil || h.smsService == nil {
		response.ServiceUnavailable(c, "phone binding is not configured")
		return
	}

	var req requestBindPhoneOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	phone := strings.TrimSpace(req.Phone)
	if !phoneutil.IsValidMainlandPhone(phone) {
		response.BadRequest(c, "invalid phone number format")
		return
	}

	if err := h.otpService.CheckPhoneRateLimit(c.Request.Context(), phone); err != nil {
		if errors.Is(err, auth.ErrOTPPhoneRateLimited) {
			response.RateLimitExceeded(c, "too many verification code requests for this phone number")
			return
		}
		logger.FromGin(c).Error("failed to check bind phone OTP rate limit", zap.Error(err))
		response.InternalError(c, "failed to send verification code")
		return
	}

	code, err := h.otpService.Generate(c.Request.Context(), phone)
	if err != nil {
		if errors.Is(err, auth.ErrOTPCooldown) {
			response.RateLimitExceeded(c, "please wait before requesting a new code")
			return
		}
		logger.FromGin(c).Error("failed to generate OTP for bind phone", zap.Error(err))
		response.InternalError(c, "failed to send verification code")
		return
	}

	internationalPhone := "+86" + phone
	if err := h.smsService.Send(c.Request.Context(), internationalPhone, code); err != nil {
		if cleanupErr := h.otpService.CleanupCodeOnly(c.Request.Context(), phone); cleanupErr != nil {
			logger.FromGin(c).Warn("failed to cleanup OTP after SMS send failure", zap.Error(cleanupErr))
		}
		logger.FromGin(c).Error("failed to send SMS for bind phone", zap.Error(err))
		response.InternalError(c, "failed to send verification code")
		return
	}

	response.Success(c, gin.H{
		"message":  "verification code sent",
		"cooldown": int(auth.OTPCooldownSeconds()),
	})
}

func (h *Handler) handleBindPhone(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}

	var req bindPhoneHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	phone := strings.TrimSpace(req.Phone)
	if !phoneutil.IsValidMainlandPhone(phone) {
		response.BadRequest(c, "invalid phone number format")
		return
	}

	if h.otpService == nil {
		response.ServiceUnavailable(c, "phone binding is not configured")
		return
	}

	if err := h.otpService.Verify(c.Request.Context(), phone, req.OTPCode); err != nil {
		switch {
		case errors.Is(err, auth.ErrOTPExpired):
			response.Unauthorized(c, "verification code expired", errs.ErrPhoneOTPExpired)
			return
		case errors.Is(err, auth.ErrOTPMaxAttempts):
			response.RateLimitExceeded(c, "too many failed attempts, please request a new code")
			return
		case errors.Is(err, auth.ErrOTPInvalidCode):
			response.Unauthorized(c, "invalid verification code", errs.ErrPhoneOTPFailed)
			return
		}
		logger.FromGin(c).Error("failed to verify bind phone OTP", zap.Error(err))
		response.InternalError(c, "verification failed")
		return
	}

	err := h.service.BindPhone(c.Request.Context(), userID, phone)
	if err != nil {
		if errors.Is(err, ErrInvalidPhoneFormat) {
			response.BadRequest(c, "invalid phone number format")
			return
		}
		if errors.Is(err, ErrPhoneAlreadyBound) {
			response.Conflict(c, "phone number already bound")
			return
		}
		if errors.Is(err, ErrProfileNotFound) {
			response.NotFound(c, "student profile not found", errs.ErrProfileNotFound)
			return
		}
		logger.FromGin(c).Error("failed to bind phone", zap.Error(err))
		response.InternalError(c, "failed to bind phone")
		return
	}

	response.Success(c, gin.H{"message": "phone number updated"})
}

func (h *Handler) handleGetAcademicInfo(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}

	profile, err := h.service.GetProfile(c.Request.Context(), userID)
	if err != nil {
		logger.FromGin(c).Error("failed to get profile for academic info", zap.Error(err))
		response.InternalError(c, "failed to get academic information")
		return
	}
	if profile == nil || profile.VerificationStatus != StatusVerified || profile.ActiveStudentID == nil {
		response.Forbidden(c, "student verification required", errs.ErrForbidden)
		return
	}

	var schoolID int64
	if profile.SchoolID != nil {
		schoolID = *profile.SchoolID
	}

	student, err := h.service.GetAcademicInfo(c.Request.Context(), schoolID, *profile.ActiveStudentID)
	if err != nil {
		switch {
		case errors.Is(err, ErrSchoolNotFound):
			response.NotFound(c, "school configuration not found", errs.ErrProfileSchoolNotFound)
			return
		case errors.Is(err, ErrSchoolDisabled):
			response.BadRequest(c, "school verification channel disabled", errs.ErrProfileSchoolDisabled)
			return
		case errors.Is(err, ErrAcademicTableNotConfigured):
			response.BadRequest(c, "academic table is not configured", errs.ErrAcademicTableNotConfigured)
			return
		case errors.Is(err, ErrInvalidAcademicDBTable):
			response.BadRequest(c, "academic table configuration is invalid", errs.ErrProfileAcademicTable)
			return
		default:
			logger.FromGin(c).Error("failed to get academic info", zap.Error(err))
			response.InternalError(c, "failed to get academic information")
			return
		}
	}
	if student == nil {
		response.NotFound(c, "academic record not found")
		return
	}

	response.Success(c, gin.H{
		"xh":     student.XH,
		"xm":     student.XM,
		"yxdm":   student.YXDM,
		"zydm":   student.ZYDM,
		"bjdm":   student.BJDM,
		"xznj":   student.XZNJ,
		"rxnj":   student.RXNJ,
		"pyccdm": student.PYCCDM,
		"sjh":    student.SJH,
		"dzxx":   student.DZXX,
	})
}

func (h *Handler) handleListSchools(c *gin.Context) {
	schools, err := h.service.ListSchools(c.Request.Context())
	if err != nil {
		logger.FromGin(c).Error("failed to list schools", zap.Error(err))
		response.InternalError(c, "failed to list schools")
		return
	}

	list := make([]gin.H, 0, len(schools))
	for i := range schools {
		item, err := schoolConfigPublicToJSON(&schools[i])
		if err != nil {
			logger.FromGin(c).Error("failed to serialize school config",
				zap.Int64("school_id", schools[i].SchoolID),
				zap.Error(err),
			)
			response.InternalError(c, "failed to list schools")
			return
		}
		list = append(list, item)
	}

	response.Success(c, list)
}
