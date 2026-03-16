package user

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
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

	identity, err := h.service.SubmitIdentity(c.Request.Context(), userID, SubmitIdentityRequest{
		DocType:        req.DocType,
		DocNumber:      req.DocNumber,
		RealName:       req.RealName,
		DocPhotoFront:  req.DocPhotoFront,
		DocPhotoBack:   req.DocPhotoBack,
		DocPhotoSelfie: req.DocPhotoSelfie,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrIdentityAlreadyVerified):
			response.Conflict(c, "identity already verified", errs.ErrIdentityAlreadyVerified)
		case errors.Is(err, ErrIdentityAlreadyExists):
			response.Conflict(c, "identity already submitted", errs.ErrIdentityAlreadyExists)
		case errors.Is(err, ErrPhotoRequired):
			response.BadRequest(c, "photo upload required for non-mainland documents", errs.ErrIdentityPhotoRequired)
		default:
			logger.FromGin(c).Error("failed to submit identity", zap.Error(err))
			response.InternalError(c, "failed to submit identity")
		}
		return
	}

	response.Created(c, identityStatusToJSON(identity))
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
	SchoolID       string         `json:"schoolID" binding:"required,max=10"`
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

	profile, err := h.service.VerifyStudent(c.Request.Context(), userID, VerifyStudentRequest{
		SchoolID:       req.SchoolID,
		StudentID:      req.StudentID,
		Password:       req.Password,
		ManualFormData: req.ManualFormData,
		Consent:        req.Consent,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrIdentityRequired):
			response.Forbidden(c, "identity verification required before student verification", errs.ErrForbidden)
		case errors.Is(err, ErrProfileAlreadyVerified):
			response.Conflict(c, "student profile already verified", errs.ErrProfileAlreadyVerified)
		case errors.Is(err, ErrSchoolNotFound):
			response.NotFound(c, "school not found", errs.ErrProfileSchoolNotFound)
		case errors.Is(err, ErrSchoolDisabled):
			response.BadRequest(c, "school verification is not enabled", errs.ErrProfileSchoolDisabled)
		case errors.Is(err, ErrConsentRequired):
			response.BadRequest(c, "consent is required for verification", errs.ErrProfileConsentRequired)
		case errors.Is(err, ErrStudentIDRequired), errors.Is(err, ErrPasswordRequired),
			errors.Is(err, ErrManualFieldRequired), errors.Is(err, ErrManualFieldInvalid):
			response.BadRequest(c, err.Error())
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
	Phone string `json:"phone" binding:"required,max=20"`
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

	err := h.service.BindPhone(c.Request.Context(), userID, req.Phone)
	if err != nil {
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

	student, err := h.service.GetAcademicInfo(c.Request.Context(), *profile.ActiveStudentID)
	if err != nil {
		logger.FromGin(c).Error("failed to get academic info", zap.Error(err))
		response.InternalError(c, "failed to get academic information")
		return
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
				zap.String("school_id", schools[i].SchoolID),
				zap.Error(err),
			)
			response.InternalError(c, "failed to list schools")
			return
		}
		list = append(list, item)
	}

	response.Success(c, list)
}
