package user

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
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
		resolved, err := h.service.ResolveIdentityReviewItemAssets(c.Request.Context(), &list[i])
		if err != nil {
			logger.FromGin(c).Error("failed to resolve identity photo URLs",
				zap.Int64("user_id", list[i].UserID),
				zap.Error(err),
			)
			response.InternalError(c, "failed to list identities")
			return
		}
		items = append(items, identityReviewItemToJSON(resolved))
	}

	response.Success(c, gin.H{"list": items, "total": total})
}

type reviewIdentityHTTPRequest struct {
	Approved        *bool   `json:"approved" binding:"required"`
	RejectionReason *string `json:"rejectionReason"`
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

	err = h.service.ReviewIdentity(c.Request.Context(), userID, *req.Approved, derefOptionalString(req.RejectionReason))
	if err != nil {
		switch {
		case errors.Is(err, ErrIdentityNotFound):
			response.NotFound(c, "identity not found", errs.ErrIdentityNotFound)
		default:
			logger.FromGin(c).Error("failed to review identity",
				zap.Int64("target_user_id", userID),
				zap.Error(err),
			)
			response.InternalError(c, "failed to review identity")
		}
		return
	}

	audit.Log(audit.Event{
		Type:      audit.EventDataUpdate,
		UserID:    middleware.GetUserID(c),
		Username:  middleware.GetUsername(c),
		IP:        c.ClientIP(),
		UserAgent: httputil.TruncateUserAgent(c.GetHeader("User-Agent")),
		RequestID: middleware.GetRequestID(c),
		Resource:  "identity_review",
		Action:    map[bool]string{true: "approve", false: "reject"}[*req.Approved],
		Result:    "success",
		Details: map[string]any{
			"target_user_id":   userID,
			"approved":         *req.Approved,
			"rejection_reason": derefOptionalString(req.RejectionReason),
		},
	})

	response.Success(c, gin.H{"message": "identity reviewed"})
}

func (h *Handler) handleAdminListStudentVerifications(c *gin.Context) {
	status, ok := normalizeAdminReviewStatus(c.Query("status"))
	if !ok {
		response.BadRequest(c, "invalid status")
		return
	}
	var schoolID *int64
	if raw := strings.TrimSpace(c.Query("schoolID")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			response.BadRequest(c, "invalid school ID")
			return
		}
		schoolID = &parsed
	}
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
	Approved        *bool   `json:"approved" binding:"required"`
	RejectionReason *string `json:"rejectionReason"`
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

	err = h.service.ReviewStudentVerification(c.Request.Context(), userID, *req.Approved, derefOptionalString(req.RejectionReason))
	if err != nil {
		switch {
		case errors.Is(err, ErrProfileNotFound):
			response.NotFound(c, "student profile not found", errs.ErrProfileNotFound)
		default:
			logger.FromGin(c).Error("failed to review student verification",
				zap.Int64("target_user_id", userID),
				zap.Error(err),
			)
			response.InternalError(c, "failed to review student verification")
		}
		return
	}

	audit.Log(audit.Event{
		Type:      audit.EventDataUpdate,
		UserID:    middleware.GetUserID(c),
		Username:  middleware.GetUsername(c),
		IP:        c.ClientIP(),
		UserAgent: httputil.TruncateUserAgent(c.GetHeader("User-Agent")),
		RequestID: middleware.GetRequestID(c),
		Resource:  "student_verification_review",
		Action:    map[bool]string{true: "approve", false: "reject"}[*req.Approved],
		Result:    "success",
		Details: map[string]any{
			"target_user_id":   userID,
			"approved":         *req.Approved,
			"rejection_reason": derefOptionalString(req.RejectionReason),
		},
	})

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
				zap.Int64("school_id", configs[i].SchoolID),
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
	ApprovalPolicy     *string                  `json:"approvalPolicy" binding:"omitempty,oneof=auto manual"`
	LDAPConfig         *SchoolLDAPConfigInput   `json:"ldapConfig"`
	AcademicDBTable    *string                  `json:"academicDbTable" binding:"omitempty,max=100"`
	ConsentText        *string                  `json:"consentText"`
	ManualFormFields   *[]ManualFieldDescriptor `json:"manualFormFields"`
	Enabled            *bool                    `json:"enabled"`
}

func (h *Handler) handleAdminUpdateSchoolConfig(c *gin.Context) {
	schoolID, parseErr := strconv.ParseInt(c.Param("schoolID"), 10, 64)
	if parseErr != nil || schoolID <= 0 {
		response.BadRequest(c, "invalid school ID")
		return
	}

	var req updateSchoolConfigHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	input := UpdateSchoolConfigInput(req)

	if err := h.service.UpdateSchoolConfig(c.Request.Context(), schoolID, input); err != nil {
		if errors.Is(err, ErrSchoolNotFound) {
			response.NotFound(c, "school config not found", errs.ErrProfileSchoolNotFound)
			return
		}
		if errors.Is(err, ErrInvalidManualFieldConfig) {
			logger.FromGin(c).Warn("invalid manual field config", zap.Error(err))
			response.BadRequest(c, "invalid manual form field configuration")
			return
		}
		if errors.Is(err, ErrAcademicTableNotConfigured) {
			logger.FromGin(c).Warn("academic db table not configured", zap.Error(err))
			response.BadRequest(c, "academic db table is required for enabled LDAP schools", errs.ErrAcademicTableNotConfigured)
			return
		}
		if errors.Is(err, ErrInvalidAcademicDBTable) {
			logger.FromGin(c).Warn("invalid academic db table config", zap.Error(err))
			response.BadRequest(c, "invalid academic db table configuration", errs.ErrProfileAcademicTable)
			return
		}
		if errors.Is(err, ErrSchoolLDAPConfigMissing) {
			logger.FromGin(c).Warn("missing school ldap config", zap.Error(err))
			response.BadRequest(c, "school LDAP configuration is required for enabled LDAP schools", errs.ErrSchoolLDAPConfigMissing)
			return
		}
		if errors.Is(err, ErrLDAPConfigInvalid) {
			logger.FromGin(c).Warn("invalid school ldap config", zap.Error(err))
			response.BadRequest(c, "school LDAP configuration is invalid", errs.ErrLDAPConfigInvalid)
			return
		}
		logger.FromGin(c).Error("failed to update school config",
			zap.Int64("school_id", schoolID),
			zap.Error(err),
		)
		response.InternalError(c, "failed to update school config")
		return
	}

	audit.Log(audit.Event{
		Type:      audit.EventAdminConfigChange,
		UserID:    middleware.GetUserID(c),
		Username:  middleware.GetUsername(c),
		IP:        c.ClientIP(),
		UserAgent: httputil.TruncateUserAgent(c.GetHeader("User-Agent")),
		RequestID: middleware.GetRequestID(c),
		Resource:  "school_config",
		Action:    "update",
		Result:    "success",
		Details: map[string]any{
			"school_id": schoolID,
			"fields": map[string]bool{
				"schoolName":         req.SchoolName != nil,
				"verificationMethod": req.VerificationMethod != nil,
				"approvalPolicy":     req.ApprovalPolicy != nil,
				"ldapConfig":         req.LDAPConfig != nil,
				"academicDbTable":    req.AcademicDBTable != nil,
				"consentText":        req.ConsentText != nil,
				"manualFormFields":   req.ManualFormFields != nil,
				"enabled":            req.Enabled != nil,
			},
		},
	})

	response.Success(c, gin.H{"message": "school config updated"})
}

func derefOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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
		if errors.Is(err, ErrSystemConfigNotFound) {
			logger.FromGin(c).Warn("system config not found",
				zap.String("config_key", key),
				zap.Error(err),
			)
			response.NotFound(c, "system config not found", errs.ErrSystemConfigNotFound)
			return
		}
		if errors.Is(err, ErrInvalidSystemConfigValue) {
			logger.FromGin(c).Warn("invalid system config value",
				zap.String("config_key", key),
				zap.Error(err),
			)
			response.BadRequest(c, "invalid system config value", errs.ErrInvalidParam)
			return
		}
		logger.FromGin(c).Error("failed to update system config",
			zap.String("config_key", key),
			zap.Error(err),
		)
		response.InternalError(c, "failed to update system config")
		return
	}

	audit.Log(audit.Event{
		Type:      audit.EventAdminConfigChange,
		UserID:    middleware.GetUserID(c),
		Username:  middleware.GetUsername(c),
		IP:        c.ClientIP(),
		UserAgent: httputil.TruncateUserAgent(c.GetHeader("User-Agent")),
		RequestID: middleware.GetRequestID(c),
		Resource:  "system_config",
		Action:    "update",
		Result:    "success",
		Details: map[string]any{
			"key": key,
		},
	})

	response.Success(c, gin.H{"message": "system config updated"})
}
