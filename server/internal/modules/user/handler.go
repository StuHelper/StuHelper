package user

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// Handler 用户模块 HTTP 处理器
type Handler struct {
	service *Service
}

// NewHandler 创建用户处理器
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes 注册用户中心路由（挂载到 /api/v1 级别）
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMW gin.HandlerFunc) {
	user := rg.Group("/user")
	user.Use(authMW)
	{
		user.GET("/identity", h.handleGetIdentity)
		user.POST("/identity", h.handleSubmitIdentity)
		user.GET("/profile", h.handleGetProfile)
		user.POST("/profile/verify", h.handleVerifyStudent)
		user.POST("/profile/bind-phone", h.handleBindPhone)
		user.GET("/profile/academic-info", h.handleGetAcademicInfo)
	}
	// 学校列表无需认证
	rg.GET("/user/schools", h.handleListSchools)
}

// RegisterAdminRoutes 注册管理后台路由（调用方负责挂载到 /admin 组并添加鉴权中间件）
func (h *Handler) RegisterAdminRoutes(admin *gin.RouterGroup) {
	admin.GET("/identities", h.handleAdminListIdentities)
	admin.PUT("/identities/:userID", h.handleAdminReviewIdentity)
	admin.GET("/student-verifications", h.handleAdminListStudentVerifications)
	admin.PUT("/student-verifications/:userID", h.handleAdminReviewStudentVerification)
	admin.GET("/school-configs", h.handleAdminListSchoolConfigs)
	admin.PUT("/school-configs/:schoolID", h.handleAdminUpdateSchoolConfig)
	admin.GET("/system-configs", h.handleAdminListSystemConfigs)
	admin.PUT("/system-configs/:key", h.handleAdminUpdateSystemConfig)
}

// ---------------------------------------------------------------------------
// User-facing: Identity
// ---------------------------------------------------------------------------

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

	response.Success(c, gin.H{"identity": identityToJSON(identity)})
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

	response.Created(c, gin.H{"identity": identityToJSON(identity)})
}

// ---------------------------------------------------------------------------
// User-facing: Profile
// ---------------------------------------------------------------------------

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

	response.Success(c, gin.H{"profile": profileToJSON(profile)})
}

type verifyStudentHTTPRequest struct {
	SchoolID  string `json:"schoolID" binding:"required,max=10"`
	StudentID string `json:"studentID" binding:"required,max=50"`
	Password  string `json:"password" binding:"required,max=200"`
	Consent   bool   `json:"consent"`
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
		SchoolID:  req.SchoolID,
		StudentID: req.StudentID,
		Password:  req.Password,
		Consent:   req.Consent,
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
		case errors.Is(err, ErrLDAPFailed):
			response.BadRequest(c, "LDAP verification failed, please check your credentials", errs.ErrProfileLDAPFailed)
		default:
			logger.FromGin(c).Error("failed to verify student", zap.Error(err))
			response.InternalError(c, "failed to verify student")
		}
		return
	}

	response.Success(c, gin.H{"profile": profileToJSON(profile)})
}

type bindPhoneHTTPRequest struct {
	Phone string `json:"phone" binding:"required,max=20"`
	Code  string `json:"code"`
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

	err := h.service.BindPhone(c.Request.Context(), userID, BindPhoneRequest{
		Phone: req.Phone,
		Code:  req.Code,
	})
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

	// 获取用户 profile，确认已认证且有 activeStudentID
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

	response.Success(c, gin.H{"student": gin.H{
		"xh":   student.XH,
		"xm":   student.XM,
		"yxdm": student.YXDM,
		"zydm": student.ZYDM,
		"bjdm": student.BJDM,
		"xznj": student.XZNJ,
		"rxnj": student.RXNJ,
	}})
}

func (h *Handler) handleListSchools(c *gin.Context) {
	schools, err := h.service.ListSchools(c.Request.Context())
	if err != nil {
		logger.FromGin(c).Error("failed to list schools", zap.Error(err))
		response.InternalError(c, "failed to list schools")
		return
	}

	list := make([]gin.H, 0, len(schools))
	for _, s := range schools {
		list = append(list, gin.H{
			"schoolID":           s.SchoolID,
			"schoolName":         s.SchoolName,
			"verificationMethod": s.VerificationMethod,
			"consentText":        s.ConsentText,
			"enabled":            s.Enabled,
		})
	}

	response.Success(c, gin.H{"schools": list})
}

// ---------------------------------------------------------------------------
// Admin: Identity review
// ---------------------------------------------------------------------------

func (h *Handler) handleAdminListIdentities(c *gin.Context) {
	status := c.DefaultQuery("status", "all")
	if status == "all" {
		status = ""
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
		items = append(items, identityToJSON(&list[i]))
	}

	response.Success(c, gin.H{"list": items, "total": total})
}

type reviewIdentityHTTPRequest struct {
	Approved        bool   `json:"approved"`
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

	err = h.service.ReviewIdentity(c.Request.Context(), userID, req.Approved, req.RejectionReason)
	if err != nil {
		if errors.Is(err, ErrIdentityNotFound) {
			response.NotFound(c, "identity not found", errs.ErrIdentityNotFound)
			return
		}
		logger.FromGin(c).Error("failed to review identity",
			zap.Int64("target_user_id", userID),
			zap.Error(err),
		)
		response.InternalError(c, "failed to review identity")
		return
	}

	response.Success(c, gin.H{"message": "identity reviewed"})
}

// ---------------------------------------------------------------------------
// Admin: Student verification review
// ---------------------------------------------------------------------------

func (h *Handler) handleAdminListStudentVerifications(c *gin.Context) {
	status := c.DefaultQuery("status", "all")
	if status == "all" {
		status = ""
	}
	schoolID := c.Query("schoolId")
	page, pageSize := httputil.ParsePage(c)

	list, total, err := h.service.ListProfiles(c.Request.Context(), status, schoolID, page, pageSize)
	if err != nil {
		logger.FromGin(c).Error("failed to list student verifications", zap.Error(err))
		response.InternalError(c, "failed to list student verifications")
		return
	}

	items := make([]gin.H, 0, len(list))
	for i := range list {
		items = append(items, profileToJSON(&list[i]))
	}

	response.Success(c, gin.H{"list": items, "total": total})
}

type reviewStudentVerificationHTTPRequest struct {
	Status          string `json:"status" binding:"required,oneof=verified rejected"`
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

	err = h.service.ReviewStudentVerification(c.Request.Context(), userID, req.Status, req.RejectionReason)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			response.NotFound(c, "student profile not found", errs.ErrProfileNotFound)
			return
		}
		logger.FromGin(c).Error("failed to review student verification",
			zap.Int64("target_user_id", userID),
			zap.Error(err),
		)
		response.InternalError(c, "failed to review student verification")
		return
	}

	response.Success(c, gin.H{"message": "student verification reviewed"})
}

// ---------------------------------------------------------------------------
// Admin: School configs
// ---------------------------------------------------------------------------

func (h *Handler) handleAdminListSchoolConfigs(c *gin.Context) {
	configs, err := h.service.ListAllSchoolConfigs(c.Request.Context())
	if err != nil {
		logger.FromGin(c).Error("failed to list school configs", zap.Error(err))
		response.InternalError(c, "failed to list school configs")
		return
	}

	response.Success(c, gin.H{"configs": configs})
}

type updateSchoolConfigHTTPRequest struct {
	SchoolName         string `json:"schoolName" binding:"required"`
	VerificationMethod string `json:"verificationMethod" binding:"required,oneof=ldap manual"`
	ConsentText        string `json:"consentText"`
	Enabled            bool   `json:"enabled"`
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

	consentText := req.ConsentText
	config := &SchoolConfig{
		SchoolID:           schoolID,
		SchoolName:         req.SchoolName,
		VerificationMethod: req.VerificationMethod,
		ConsentText:        &consentText,
		Enabled:            req.Enabled,
	}

	if err := h.service.UpdateSchoolConfig(c.Request.Context(), config); err != nil {
		logger.FromGin(c).Error("failed to update school config",
			zap.String("school_id", schoolID),
			zap.Error(err),
		)
		response.InternalError(c, "failed to update school config")
		return
	}

	response.Success(c, gin.H{"message": "school config updated"})
}

// ---------------------------------------------------------------------------
// Admin: System configs
// ---------------------------------------------------------------------------

func (h *Handler) handleAdminListSystemConfigs(c *gin.Context) {
	configs, err := h.service.ListSystemConfigs(c.Request.Context())
	if err != nil {
		logger.FromGin(c).Error("failed to list system configs", zap.Error(err))
		response.InternalError(c, "failed to list system configs")
		return
	}

	response.Success(c, gin.H{"configs": configs})
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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// resolveCurrentUser 从请求上下文解析当前用户内部 ID
func (h *Handler) resolveCurrentUser(c *gin.Context) (int64, bool) {
	externalID := middleware.GetUserID(c)
	if externalID == "" {
		response.Unauthorized(c, "authentication required", errs.ErrLoginRequired)
		return 0, false
	}

	userID, err := h.service.GetInternalUserID(c.Request.Context(), externalID)
	if err != nil {
		logger.FromGin(c).Error("failed to resolve user ID",
			zap.String("external_id", externalID),
			zap.Error(err),
		)
		response.InternalError(c, "failed to resolve user")
		return 0, false
	}

	return userID, true
}

func identityToJSON(i *Identity) gin.H {
	return gin.H{
		"userID":          i.UserID,
		"docType":         i.DocType,
		"realName":        i.RealName,
		"verified":        i.Verified,
		"verifyMethod":    i.VerifyMethod,
		"verifiedAt":      i.VerifiedAt,
		"rejectionReason": i.RejectionReason,
		"createdAt":       i.CreatedAt,
		"updatedAt":       i.UpdatedAt,
	}
}

func profileToJSON(p *Profile) gin.H {
	return gin.H{
		"userID":             p.UserID,
		"schoolID":           p.SchoolID,
		"studentIDs":         p.StudentIDs,
		"activeStudentID":    p.ActiveStudentID,
		"verificationStatus": p.VerificationStatus,
		"verificationMethod": p.VerificationMethod,
		"phone":              p.Phone,
		"phoneVerified":      p.PhoneVerified,
		"consentGivenAt":     p.ConsentGivenAt,
		"verifiedAt":         p.VerifiedAt,
		"createdAt":          p.CreatedAt,
		"updatedAt":          p.UpdatedAt,
	}
}
