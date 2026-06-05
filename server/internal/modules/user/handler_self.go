package user

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

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

	response.Success(c, userSurfaceResponse{
		DisplayName:        surface.DisplayName,
		AvatarURL:          surface.AvatarURL,
		IdentityStatus:     surface.IdentityStatus,
		VerificationStatus: surface.VerificationStatus,
		PhoneBound:         surface.PhoneBound,
		Capabilities:       nonNilStrings(surface.Capabilities),
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
		if respondSubmitIdentityError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to submit identity", zap.Error(err))
		response.InternalError(c, "failed to submit identity")
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
		if errors.Is(err, ErrIdentityPhotoStoreDisabled) {
			logger.FromGin(c).Error("identity photo storage is not configured", zap.Error(err))
		}
		if respondUploadIdentityPhotoError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to upload identity photo", zap.Error(err))
		response.InternalError(c, "failed to upload identity photo")
		return
	}

	response.Created(c, uploadIdentityPhotoResponse{Key: key})
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
	SchoolCode     string         `json:"schoolCode" binding:"required,numeric,len=10"`
	StudentID      string         `json:"studentID" binding:"omitempty,max=50"`
	Password       string         `json:"password" binding:"omitempty,max=200"`
	ManualFormData map[string]any `json:"manualFormData"`
	Consent        bool           `json:"consent"`
}

type studentEmailOTPHTTPRequest struct {
	SchoolCode  string `json:"schoolCode" binding:"required,numeric,len=10"`
	Email       string `json:"email" binding:"omitempty,max=320"`
	StudentID   string `json:"studentID" binding:"omitempty,max=50"`
	StudentName string `json:"studentName" binding:"omitempty,max=80"`
}

type studentEmailAcademicMatchHTTPRequest struct {
	SchoolCode  string `json:"schoolCode" binding:"required,numeric,len=10"`
	StudentID   string `json:"studentID" binding:"required,max=50"`
	StudentName string `json:"studentName" binding:"required,max=80"`
}

type studentEmailOTPVerifyHTTPRequest struct {
	SchoolCode string `json:"schoolCode" binding:"required,numeric,len=10"`
	Email      string `json:"email" binding:"omitempty,max=320"`
	Code       string `json:"code" binding:"required,min=4,max=12"`
	Consent    bool   `json:"consent"`
}

func (h *Handler) handleVerifyStudent(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}

	var req verifyStudentHTTPRequest
	if !bindPublicSchoolCodeJSON(c, &req) {
		return
	}
	schoolID, ok := h.resolvePublicSchoolID(c, req.SchoolCode)
	if !ok {
		return
	}

	profile, err := h.service.VerifyStudent(c.Request.Context(), userID, VerifyStudentRequest{
		SchoolID:       schoolID,
		StudentID:      req.StudentID,
		Password:       req.Password,
		ManualFormData: req.ManualFormData,
		Consent:        req.Consent,
	})
	if err != nil {
		if respondVerifyStudentError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to verify student", zap.Error(err))
		response.InternalError(c, "failed to verify student")
		return
	}

	response.Success(c, profileToJSON(profile))
}

func (h *Handler) handleMatchStudentEmailAcademicStudent(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}

	var req studentEmailAcademicMatchHTTPRequest
	if !bindPublicSchoolCodeJSON(c, &req) {
		return
	}
	schoolID, ok := h.resolvePublicSchoolID(c, req.SchoolCode)
	if !ok {
		return
	}
	result, err := h.service.MatchStudentEmailAcademicStudent(c.Request.Context(), StudentEmailAcademicMatchInput{
		UserID:      userID,
		SchoolID:    schoolID,
		StudentID:   req.StudentID,
		StudentName: req.StudentName,
	})
	if err != nil {
		if respondVerifyStudentError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to match student email academic identity", zap.Error(err))
		response.InternalError(c, "failed to match student academic identity")
		return
	}
	response.Success(c, result)
}

func (h *Handler) handleRequestStudentEmailOTP(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}

	var req studentEmailOTPHTTPRequest
	if !bindPublicSchoolCodeJSON(c, &req) {
		return
	}
	schoolID, ok := h.resolvePublicSchoolID(c, req.SchoolCode)
	if !ok {
		return
	}
	result, err := h.service.RequestStudentEmailOTP(c.Request.Context(), StudentEmailOTPInput{
		UserID:      userID,
		SchoolID:    schoolID,
		Email:       req.Email,
		StudentID:   req.StudentID,
		StudentName: req.StudentName,
	})
	if err != nil {
		if respondVerifyStudentError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to request student email otp", zap.Error(err))
		response.InternalError(c, "failed to request student email otp")
		return
	}
	response.Success(c, result)
}

func (h *Handler) handleVerifyStudentEmailOTP(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}

	var req studentEmailOTPVerifyHTTPRequest
	if !bindPublicSchoolCodeJSON(c, &req) {
		return
	}
	schoolID, ok := h.resolvePublicSchoolID(c, req.SchoolCode)
	if !ok {
		return
	}
	profile, err := h.service.VerifyStudentEmailOTP(c.Request.Context(), StudentEmailOTPVerifyInput{
		UserID:   userID,
		SchoolID: schoolID,
		Email:    req.Email,
		Code:     req.Code,
		Consent:  req.Consent,
	})
	if err != nil {
		if respondVerifyStudentError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to verify student email otp", zap.Error(err))
		response.InternalError(c, "failed to verify student email otp")
		return
	}
	response.Success(c, profileToJSON(profile))
}

func (h *Handler) resolvePublicSchoolID(c *gin.Context, schoolCode string) (int64, bool) {
	code := strings.TrimSpace(schoolCode)
	if code == "" {
		response.BadRequest(c, "schoolCode is required")
		return 0, false
	}
	resolvedSchoolID, err := h.service.ResolveEnabledSchoolIDByCode(c.Request.Context(), code)
	if err != nil {
		if respondVerifyStudentError(c, err) {
			return 0, false
		}
		logger.FromGin(c).Error("failed to resolve school code", zap.Error(err))
		response.InternalError(c, "failed to resolve school")
		return 0, false
	}
	return resolvedSchoolID, true
}

func bindPublicSchoolCodeJSON(c *gin.Context, target any) bool {
	raw, err := c.GetRawData()
	if err != nil {
		response.BadRequest(c, "invalid request parameters")
		return false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return false
	}
	if _, ok := fields["schoolID"]; ok {
		response.BadRequest(c, "schoolID is not accepted; use schoolCode")
		return false
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))
	if err := c.ShouldBindJSON(target); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return false
	}
	return true
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

	if err := h.otpService.IssueCode(c.Request.Context(), phone, h.smsService); err != nil {
		if errors.Is(err, ErrBindPhoneOTPPhoneRateLimited) {
			response.RateLimitExceeded(c, "too many verification code requests for this phone number")
			return
		}
		if errors.Is(err, ErrBindPhoneOTPCooldown) {
			response.RateLimitExceeded(c, "please wait before requesting a new code")
			return
		}
		logger.FromGin(c).Error("failed to send bind phone OTP", zap.Error(err))
		response.InternalError(c, "failed to send verification code")
		return
	}

	response.Success(c, bindPhoneOTPResponse{
		Message:  "verification code sent",
		Cooldown: h.otpService.CooldownSeconds(),
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

	if err := h.otpService.Check(c.Request.Context(), phone, req.OTPCode); err != nil {
		switch {
		case errors.Is(err, ErrBindPhoneOTPExpired):
			response.Unauthorized(c, "verification code expired", errs.ErrPhoneOTPExpired)
			return
		case errors.Is(err, ErrBindPhoneOTPMaxAttempts):
			response.RateLimitExceeded(c, "too many failed attempts, please request a new code")
			return
		case errors.Is(err, ErrBindPhoneOTPInvalidCode):
			response.Unauthorized(c, "invalid verification code", errs.ErrPhoneOTPFailed)
			return
		}
		logger.FromGin(c).Error("failed to check bind phone OTP", zap.Error(err))
		response.InternalError(c, "verification failed")
		return
	}

	err := h.service.BindPhone(c.Request.Context(), userID, phone)
	if err != nil {
		if respondBindPhoneError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to bind phone", zap.Error(err))
		response.InternalError(c, "failed to bind phone")
		return
	}
	if err := h.otpService.Consume(c.Request.Context(), phone, req.OTPCode); err != nil {
		logger.FromGin(c).Error("failed to consume bind phone OTP", zap.Error(err))
		response.InternalError(c, "failed to finalize phone binding")
		return
	}

	response.Success(c, messageResponse{Message: "phone number updated"})
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
		if respondAcademicInfoError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to get academic info", zap.Error(err))
		response.InternalError(c, "failed to get academic information")
		return
	}
	if student == nil {
		response.NotFound(c, "academic record not found")
		return
	}

	response.Success(c, academicInfoResponse{
		XH:     student.XH,
		XM:     student.XM,
		YXDM:   student.YXDM,
		ZYDM:   student.ZYDM,
		BJDM:   student.BJDM,
		XZNJ:   student.XZNJ,
		RXNJ:   student.RXNJ,
		PYCCDM: student.PYCCDM,
		SJH:    student.SJH,
		DZXX:   student.DZXX,
	})
}

func (h *Handler) handleListSchools(c *gin.Context) {
	schools, err := h.service.ListSchools(c.Request.Context())
	if err != nil {
		logger.FromGin(c).Error("failed to list schools", zap.Error(err))
		response.InternalError(c, "failed to list schools")
		return
	}

	list := make([]schoolConfigPublicResponse, 0, len(schools))
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
