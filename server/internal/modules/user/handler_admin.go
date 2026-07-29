package user

import (
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
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

	items := make([]identityReviewItemResponse, 0, len(list))
	for i := range list {
		items = append(items, identityReviewItemToJSON(&list[i]))
	}

	response.Success(c, pagedListResponse[identityReviewItemResponse]{List: items, Total: total})
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
		if respondAdminReviewIdentityError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to review identity",
			zap.Int64("target_user_id", userID),
			zap.Error(err),
		)
		response.InternalError(c, "failed to review identity")
		return
	}

	audit.LogFromGin(c, audit.Event{
		Type:         audit.EventDataUpdate,
		Category:     "admin_operation",
		Resource:     "identity_review",
		ResourceType: "identity_review",
		ResourceID:   strconv.FormatInt(userID, 10),
		Action:       map[bool]string{true: "approve", false: "reject"}[*req.Approved],
		Result:       "success",
		Details: map[string]any{
			"target_user_id":   userID,
			"approved":         *req.Approved,
			"rejection_reason": derefOptionalString(req.RejectionReason),
		},
	})

	response.Success(c, messageResponse{Message: "identity reviewed"})
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
	resolvedSchoolID, ok := resolveScopedAdminSchoolID(c, capability.UserStudentRead, schoolID)
	if !ok {
		return
	}
	page, pageSize := httputil.ParsePage(c)

	list, total, err := h.service.ListProfiles(c.Request.Context(), status, resolvedSchoolID, page, pageSize)
	if err != nil {
		logger.FromGin(c).Error("failed to list student verifications", zap.Error(err))
		response.InternalError(c, "failed to list student verifications")
		return
	}

	items := make([]adminStudentVerificationResponse, 0, len(list))
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

	response.Success(c, pagedListResponse[adminStudentVerificationResponse]{List: items, Total: total})
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
	profileSchoolID, err := h.service.GetProfileSchoolID(c.Request.Context(), userID)
	if err != nil {
		if respondAdminReviewStudentVerificationError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to resolve student profile school scope",
			zap.Int64("target_user_id", userID),
			zap.Error(err),
		)
		response.InternalError(c, "failed to review student verification")
		return
	}
	if !ensureAdminSchoolAccess(c, capability.UserStudentReview, profileSchoolID) {
		return
	}

	err = h.service.ReviewStudentVerification(c.Request.Context(), userID, *req.Approved, derefOptionalString(req.RejectionReason))
	if err != nil {
		if respondAdminReviewStudentVerificationError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to review student verification",
			zap.Int64("target_user_id", userID),
			zap.Error(err),
		)
		response.InternalError(c, "failed to review student verification")
		return
	}

	audit.LogFromGin(c, audit.Event{
		Type:         audit.EventDataUpdate,
		Category:     "admin_operation",
		Resource:     "student_verification_review",
		ResourceType: "student_verification_review",
		ResourceID:   strconv.FormatInt(userID, 10),
		Action:       map[bool]string{true: "approve", false: "reject"}[*req.Approved],
		Result:       "success",
		Details: map[string]any{
			"target_user_id":   userID,
			"approved":         *req.Approved,
			"rejection_reason": derefOptionalString(req.RejectionReason),
		},
	})

	response.Success(c, messageResponse{Message: "student verification reviewed"})
}

func (h *Handler) handleAdminListSchoolConfigs(c *gin.Context) {
	configs, err := h.service.ListAllSchoolConfigs(c.Request.Context())
	if err != nil {
		logger.FromGin(c).Error("failed to list school configs", zap.Error(err))
		response.InternalError(c, "failed to list school configs")
		return
	}

	items := make([]adminSchoolConfigResponse, 0, len(configs))
	for i := range configs {
		if !middleware.HasGlobalCapability(c, capability.UserSchoolRead) &&
			!middleware.HasCapabilityInSchool(c, capability.UserSchoolRead, strconv.FormatInt(configs[i].SchoolID, 10)) {
			continue
		}
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
	if !ensureAdminSchoolAccess(c, capability.UserSchoolUpdate, &schoolID) {
		return
	}

	var req updateSchoolConfigHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	input := UpdateSchoolConfigInput(req)

	if err := h.service.UpdateSchoolConfig(c.Request.Context(), schoolID, input); err != nil {
		if shouldWarnAdminSchoolConfigError(err) {
			logger.FromGin(c).Warn("invalid school config update", zap.Error(err))
		}
		if respondAdminUpdateSchoolConfigError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to update school config",
			zap.Int64("school_id", schoolID),
			zap.Error(err),
		)
		response.InternalError(c, "failed to update school config")
		return
	}

	audit.LogFromGin(c, audit.Event{
		Type:         audit.EventAdminConfigChange,
		Category:     "admin_operation",
		Resource:     "school_config",
		ResourceType: "school_config",
		ResourceID:   strconv.FormatInt(schoolID, 10),
		Action:       "update",
		Result:       "success",
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

	response.Success(c, messageResponse{Message: "school config updated"})
}

func derefOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func resolveScopedAdminSchoolID(c *gin.Context, capabilityName string, requested *int64) (*int64, bool) {
	if middleware.HasGlobalCapability(c, capabilityName) {
		return requested, true
	}

	allowedSchoolIDs := scopedSchoolIDsForCapability(c, capabilityName)
	if len(allowedSchoolIDs) == 0 {
		response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
		return nil, false
	}

	if requested != nil {
		if ensureAdminSchoolAccess(c, capabilityName, requested) {
			return requested, true
		}
		return nil, false
	}

	if len(allowedSchoolIDs) == 1 {
		schoolID := allowedSchoolIDs[0]
		return &schoolID, true
	}

	response.BadRequest(c, "schoolID is required for scoped admin access")
	return nil, false
}

func ensureAdminSchoolAccess(c *gin.Context, capabilityName string, schoolID *int64) bool {
	if middleware.HasGlobalCapability(c, capabilityName) {
		return true
	}
	if schoolID == nil || *schoolID <= 0 {
		response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
		return false
	}
	if !middleware.HasCapabilityInSchool(c, capabilityName, strconv.FormatInt(*schoolID, 10)) {
		response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
		return false
	}
	return true
}

func scopedSchoolIDsForCapability(c *gin.Context, capabilityName string) []int64 {
	seen := make(map[int64]struct{})
	allowed := make([]int64, 0)
	for _, grant := range middleware.GetCapabilityGrants(c) {
		if grant.Name != capabilityName || grant.Global {
			continue
		}
		for _, rawSchoolID := range grant.ScopeSchoolIDs {
			parsed, err := strconv.ParseInt(rawSchoolID, 10, 64)
			if err != nil || parsed <= 0 {
				continue
			}
			if _, exists := seen[parsed]; exists {
				continue
			}
			seen[parsed] = struct{}{}
			allowed = append(allowed, parsed)
		}
	}
	sort.Slice(allowed, func(i, j int) bool { return allowed[i] < allowed[j] })
	return allowed
}

func (h *Handler) handleAdminListSystemConfigs(c *gin.Context) {
	configs, err := h.service.ListSystemConfigs(c.Request.Context())
	if err != nil {
		logger.FromGin(c).Error("failed to list system configs", zap.Error(err))
		response.InternalError(c, "failed to list system configs")
		return
	}

	items := make([]systemConfigResponse, 0, len(configs))
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
		if respondAdminUpdateSystemConfigError(c, err) {
			logger.FromGin(c).Warn("system config update rejected",
				zap.String("config_key", key),
				zap.Error(err),
			)
			return
		}
		logger.FromGin(c).Error("failed to update system config",
			zap.String("config_key", key),
			zap.Error(err),
		)
		response.InternalError(c, "failed to update system config")
		return
	}

	audit.LogFromGin(c, audit.Event{
		Type:         audit.EventAdminConfigChange,
		Category:     "admin_operation",
		Resource:     "system_config",
		ResourceType: "system_config",
		ResourceID:   key,
		Action:       "update",
		Result:       "success",
		Details: map[string]any{
			"key": key,
		},
	})

	response.Success(c, messageResponse{Message: "system config updated"})
}
