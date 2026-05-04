package admission

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

type freshmanApplicationCreateHTTPRequest struct {
	SchoolID          int64                `json:"schoolID" binding:"required"`
	ApplicantName     string               `json:"applicantName" binding:"required"`
	DepartmentOrMajor *string              `json:"departmentOrMajor"`
	MaterialType      FreshmanMaterialType `json:"materialType" binding:"required"`
}

type cameraCaptureHTTPRequest struct {
	ContentType string `json:"contentType" binding:"required"`
	ImageBase64 string `json:"imageBase64" binding:"required"`
}

type schoolEmailOTPHTTPRequest struct {
	SchoolID int64  `json:"schoolID" binding:"required"`
	Email    string `json:"email" binding:"required"`
}

type schoolEmailOTPVerifyHTTPRequest struct {
	SchoolID int64  `json:"schoolID" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

func (h *Handler) handlePreviewAdmissionSession(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	session, err := h.service.PreviewToken(c.Request.Context(), c.Param("token"), c.Query("qq"))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, session)
}

func (h *Handler) handleLinkAdmissionSession(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	userID, ok := middleware.ResolveRequiredInternalUserID(
		c,
		h.internalUserIDResolver,
		"failed to resolve admission user",
	)
	if !ok {
		return
	}
	session, err := h.service.LinkTokenToUser(c.Request.Context(), c.Param("token"), c.Query("qq"), userID)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, session)
}

func (h *Handler) handleAdmissionMe(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	userID, ok := middleware.ResolveRequiredInternalUserID(
		c,
		h.internalUserIDResolver,
		"failed to resolve admission user",
	)
	if !ok {
		return
	}
	me, err := h.service.GetAdmissionMe(c.Request.Context(), userID)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, me)
}

func (h *Handler) handleCreateFreshmanApplication(c *gin.Context) {
	userID, ok := h.resolveAdmissionUser(c)
	if !ok {
		return
	}
	var req freshmanApplicationCreateHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	app, err := h.service.CreateFreshmanApplication(c.Request.Context(), freshmanApplicationCreateInput(userID, req))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Created(c, app)
}

func (h *Handler) handleUploadFreshmanCameraCapture(c *gin.Context) {
	userID, ok := h.resolveAdmissionUser(c)
	if !ok {
		return
	}
	var req cameraCaptureHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	app, err := h.service.SubmitCameraCapture(c.Request.Context(), CameraCaptureInput{
		UserID:        userID,
		ApplicationID: c.Param("id"),
		ContentType:   req.ContentType,
		ImageBase64:   req.ImageBase64,
	})
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, app)
}

func (h *Handler) handleRequestSchoolEmailOTP(c *gin.Context) {
	userID, ok := h.resolveAdmissionUser(c)
	if !ok {
		return
	}
	var req schoolEmailOTPHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	result, err := h.service.RequestSchoolEmailOTP(c.Request.Context(), schoolEmailOTPInput(userID, req))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) handleVerifySchoolEmailOTP(c *gin.Context) {
	userID, ok := h.resolveAdmissionUser(c)
	if !ok {
		return
	}
	var req schoolEmailOTPVerifyHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	if _, err := h.service.VerifySchoolEmailOTP(c.Request.Context(), schoolEmailOTPVerifyInput(userID, req)); err != nil {
		respondAdmissionError(c, err)
		return
	}
	h.handleAdmissionMe(c)
}

func (h *Handler) handleStartSchoolSSO(c *gin.Context) {
	userID, schoolID, ok := h.resolveAdmissionUserAndSchool(c)
	if !ok {
		return
	}
	result, err := h.service.StartSchoolSSO(c.Request.Context(), SchoolSSOStartInput{
		UserID:    userID,
		SchoolID:  schoolID,
		ReturnURL: c.Query("return"),
	})
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	c.Redirect(http.StatusFound, result.RedirectURL)
}

func (h *Handler) handleCompleteSchoolSSO(c *gin.Context) {
	userID, schoolID, ok := h.resolveAdmissionUserAndSchool(c)
	if !ok {
		return
	}
	input, ok := schoolSSOCompleteInputFromQuery(c, userID, schoolID)
	if !ok {
		return
	}
	result, err := h.service.CompleteSchoolSSO(c.Request.Context(), input)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	c.Redirect(http.StatusFound, result.ReturnURL)
}

func (h *Handler) ready(c *gin.Context) bool {
	if h.service != nil {
		return true
	}
	notImplemented(c)
	return false
}

func (h *Handler) resolveAdmissionUser(c *gin.Context) (int64, bool) {
	if !h.ready(c) {
		return 0, false
	}
	return middleware.ResolveRequiredInternalUserID(
		c,
		h.internalUserIDResolver,
		"failed to resolve admission user",
	)
}

func (h *Handler) resolveAdmissionUserAndSchool(c *gin.Context) (int64, int64, bool) {
	userID, ok := h.resolveAdmissionUser(c)
	if !ok {
		return 0, 0, false
	}
	schoolID, ok := parseAdmissionSchoolIDParam(c)
	return userID, schoolID, ok
}

func parseAdmissionSchoolIDParam(c *gin.Context) (int64, bool) {
	schoolID, err := strconv.ParseInt(c.Param("schoolID"), 10, 64)
	if err != nil || schoolID <= 0 {
		response.BadRequest(c, "invalid school ID")
		return 0, false
	}
	return schoolID, true
}

func freshmanApplicationCreateInput(
	userID int64,
	req freshmanApplicationCreateHTTPRequest,
) FreshmanApplicationCreateInput {
	return FreshmanApplicationCreateInput{
		UserID:            userID,
		SchoolID:          req.SchoolID,
		ApplicantName:     req.ApplicantName,
		DepartmentOrMajor: req.DepartmentOrMajor,
		MaterialType:      req.MaterialType,
	}
}

func schoolEmailOTPInput(userID int64, req schoolEmailOTPHTTPRequest) SchoolEmailOTPInput {
	return SchoolEmailOTPInput{UserID: userID, SchoolID: req.SchoolID, Email: req.Email}
}

func schoolEmailOTPVerifyInput(userID int64, req schoolEmailOTPVerifyHTTPRequest) SchoolEmailOTPVerifyInput {
	return SchoolEmailOTPVerifyInput{UserID: userID, SchoolID: req.SchoolID, Email: req.Email, Code: req.Code}
}

func schoolSSOCompleteInputFromQuery(c *gin.Context, userID, schoolID int64) (SchoolSSOCompleteInput, bool) {
	code := strings.TrimSpace(c.Query("code"))
	state := strings.TrimSpace(c.Query("state"))
	if code == "" || state == "" {
		response.BadRequest(c, "missing school sso callback parameters")
		return SchoolSSOCompleteInput{}, false
	}
	return SchoolSSOCompleteInput{
		SchoolID: schoolID,
		State:    state,
		UserID:   userID,
		Code:     code,
	}, true
}
