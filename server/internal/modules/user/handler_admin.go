package user

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

func (h *Handler) handleAdminListIdentities(c *gin.Context) {
	status, ok := normalizeAdminReviewStatus(c.Query("status"))
	if !ok {
		response.BadRequest(c, "invalid status")
		return
	}
	page, pageSize := httputil.ParsePage(c)

	list, total, err := h.service.ListIdentities(c.Request.Context(), status, page, pageSize)
	if err != nil {
		logger.FromGin(c).Error("failed to list identities", zap.Error(err))
		response.InternalError(c, "failed to list identities")
		return
	}

	items := make([]gin.H, 0, len(list))
	for i := range list {
		items = append(items, identityReviewItemToJSON(&list[i]))
	}

	response.Success(c, gin.H{"list": items, "total": total})
}

type reviewIdentityHTTPRequest struct {
	Approved        *bool  `json:"approved" binding:"required"`
	RejectionReason string `json:"rejectionReason"`
}

func (h *Handler) handleAdminReviewIdentity(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userID"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "invalid user ID")
		return
	}

	var req reviewIdentityHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	err = h.service.ReviewIdentity(c.Request.Context(), userID, *req.Approved, req.RejectionReason)
	if err != nil {
		switch {
		case errors.Is(err, ErrIdentityNotFound):
			response.NotFound(c, "identity not found", errs.ErrIdentityNotFound)
		case errors.Is(err, ErrRejectionReasonRequired):
			response.BadRequest(c, "rejection reason is required when rejecting")
		default:
			logger.FromGin(c).Error("failed to review identity",
				zap.Int64("target_user_id", userID),
				zap.Error(err),
			)
			response.InternalError(c, "failed to review identity")
		}
		return
	}

	response.Success(c, gin.H{"message": "identity reviewed"})
}

func (h *Handler) handleAdminListStudentVerifications(c *gin.Context) {
	status, ok := normalizeAdminReviewStatus(c.Query("status"))
	if !ok {
		response.BadRequest(c, "invalid status")
		return
	}
	schoolID := c.Query("schoolID")
	page, pageSize := httputil.ParsePage(c)

	list, total, err := h.service.ListProfiles(c.Request.Context(), status, schoolID, page, pageSize)
	if err != nil {
		logger.FromGin(c).Error("failed to list student verifications", zap.Error(err))
		response.InternalError(c, "failed to list student verifications")
		return
	}

	items := make([]gin.H, 0, len(list))
	for i := range list {
		item, err := adminStudentVerificationToJSON(&list[i])
		if err != nil {
			logger.FromGin(c).Error("failed to serialize student verification",
				zap.Int64("user_id", list[i].UserID),
				zap.Error(err),
			)
			response.InternalError(c, "failed to list student verifications")
			return
		}
		items = append(items, item)
	}

	response.Success(c, gin.H{"list": items, "total": total})
}

type reviewStudentVerificationHTTPRequest struct {
	Approved        *bool  `json:"approved" binding:"required"`
	RejectionReason string `json:"rejectionReason"`
}

func (h *Handler) handleAdminReviewStudentVerification(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userID"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "invalid user ID")
		return
	}

	var req reviewStudentVerificationHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	err = h.service.ReviewStudentVerification(c.Request.Context(), userID, *req.Approved, req.RejectionReason)
	if err != nil {
		switch {
		case errors.Is(err, ErrProfileNotFound):
			response.NotFound(c, "student profile not found", errs.ErrProfileNotFound)
		case errors.Is(err, ErrRejectionReasonRequired):
			response.BadRequest(c, "rejection reason is required when rejecting")
		default:
			logger.FromGin(c).Error("failed to review student verification",
				zap.Int64("target_user_id", userID),
				zap.Error(err),
			)
			response.InternalError(c, "failed to review student verification")
		}
		return
	}

	response.Success(c, gin.H{"message": "student verification reviewed"})
}

func (h *Handler) handleAdminListSchoolConfigs(c *gin.Context) {
	configs, err := h.service.ListAllSchoolConfigs(c.Request.Context())
	if err != nil {
		logger.FromGin(c).Error("failed to list school configs", zap.Error(err))
		response.InternalError(c, "failed to list school configs")
		return
	}

	items := make([]gin.H, 0, len(configs))
	for i := range configs {
		item, err := adminSchoolConfigToJSON(&configs[i])
		if err != nil {
			logger.FromGin(c).Error("failed to serialize school config",
				zap.String("school_id", configs[i].SchoolID),
				zap.Error(err),
			)
			response.InternalError(c, "failed to list school configs")
			return
		}
		items = append(items, item)
	}

	response.Success(c, items)
}

type updateSchoolConfigHTTPRequest struct {
	SchoolName         *string                  `json:"schoolName" binding:"omitempty,max=100"`
	VerificationMethod *string                  `json:"verificationMethod" binding:"omitempty,oneof=ldap manual"`
	LDAPConfig         *map[string]any          `json:"ldapConfig"`
	AcademicDBTable    *string                  `json:"academicDbTable" binding:"omitempty,max=100"`
	ConsentText        *string                  `json:"consentText"`
	ManualFormFields   *[]ManualFieldDescriptor `json:"manualFormFields"`
	Enabled            *bool                    `json:"enabled"`
}

func (h *Handler) handleAdminUpdateSchoolConfig(c *gin.Context) {
	schoolID := c.Param("schoolID")
	if schoolID == "" {
		response.BadRequest(c, "invalid school ID")
		return
	}

	var req updateSchoolConfigHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	if err := h.service.UpdateSchoolConfig(c.Request.Context(), schoolID, UpdateSchoolConfigInput{
		SchoolName:         req.SchoolName,
		VerificationMethod: req.VerificationMethod,
		LDAPConfig:         req.LDAPConfig,
		AcademicDBTable:    req.AcademicDBTable,
		ConsentText:        req.ConsentText,
		ManualFormFields:   req.ManualFormFields,
		Enabled:            req.Enabled,
	}); err != nil {
		if errors.Is(err, ErrSchoolNotFound) {
			response.NotFound(c, "school config not found", errs.ErrProfileSchoolNotFound)
			return
		}
		if errors.Is(err, ErrInvalidManualFieldConfig) {
			response.BadRequest(c, err.Error())
			return
		}
		logger.FromGin(c).Error("failed to update school config",
			zap.String("school_id", schoolID),
			zap.Error(err),
		)
		response.InternalError(c, "failed to update school config")
		return
	}

	response.Success(c, gin.H{"message": "school config updated"})
}

func (h *Handler) handleAdminListSystemConfigs(c *gin.Context) {
	configs, err := h.service.ListSystemConfigs(c.Request.Context())
	if err != nil {
		logger.FromGin(c).Error("failed to list system configs", zap.Error(err))
		response.InternalError(c, "failed to list system configs")
		return
	}

	items := make([]gin.H, 0, len(configs))
	for i := range configs {
		items = append(items, systemConfigToJSON(&configs[i]))
	}

	response.Success(c, items)
}

type updateSystemConfigHTTPRequest struct {
	Value string `json:"value" binding:"required"`
}

func (h *Handler) handleAdminUpdateSystemConfig(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.BadRequest(c, "invalid config key")
		return
	}

	var req updateSystemConfigHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	if err := h.service.UpdateSystemConfig(c.Request.Context(), key, req.Value); err != nil {
		logger.FromGin(c).Error("failed to update system config",
			zap.String("config_key", key),
			zap.Error(err),
		)
		response.InternalError(c, "failed to update system config")
		return
	}

	response.Success(c, gin.H{"message": "system config updated"})
}
