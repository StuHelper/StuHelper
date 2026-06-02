package admission

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

type freshmanApplicationCreateHTTPRequest struct {
	SchoolCode         string               `json:"schoolCode" binding:"omitempty,numeric,len=10"`
	SchoolID           int64                `json:"schoolID" binding:"omitempty,min=1"`
	AdmissionSessionID string               `json:"admissionSessionID"`
	ApplicantName      string               `json:"applicantName" binding:"required"`
	DepartmentOrMajor  *string              `json:"departmentOrMajor"`
	MaterialType       FreshmanMaterialType `json:"materialType" binding:"required"`
}

type cameraCaptureHTTPRequest struct {
	ContentType string `json:"contentType" binding:"required"`
	ImageBase64 string `json:"imageBase64" binding:"required"`
}

type cameraHandoffContinuationHTTPRequest struct {
	ContinueOn FreshmanCameraContinuation `json:"continueOn" binding:"required"`
}

type schoolEmailOTPHTTPRequest struct {
	SchoolCode         string `json:"schoolCode" binding:"omitempty,numeric,len=10"`
	SchoolID           int64  `json:"schoolID" binding:"omitempty"`
	AdmissionSessionID string `json:"admissionSessionID"`
	Email              string `json:"email"`
	StudentID          string `json:"studentID"`
	StudentName        string `json:"studentName"`
}

type schoolEmailAcademicMatchHTTPRequest struct {
	SchoolCode         string `json:"schoolCode" binding:"required,numeric,len=10"`
	AdmissionSessionID string `json:"admissionSessionID"`
	StudentID          string `json:"studentID" binding:"required,max=50"`
	StudentName        string `json:"studentName" binding:"required,max=80"`
}

type schoolEmailOTPVerifyHTTPRequest struct {
	SchoolCode         string `json:"schoolCode" binding:"omitempty,numeric,len=10"`
	SchoolID           int64  `json:"schoolID" binding:"omitempty"`
	AdmissionSessionID string `json:"admissionSessionID"`
	Email              string `json:"email" binding:"required"`
	Code               string `json:"code" binding:"required"`
}

func (h *Handler) handlePreviewAdmissionSession(c *gin.Context) {
	session, err := h.service.PreviewToken(c.Request.Context(), c.Param("token"), c.Query("qq"))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, session)
}

func (h *Handler) handleLinkAdmissionSession(c *gin.Context) {
	userID, ok := middleware.ResolveRequiredInternalUserID(
		c,
		h.internalUserIDResolver,
		"failed to resolve admission user",
	)
	if !ok {
		return
	}
	session, err := h.service.LinkTokenToUser(c.Request.Context(), AdmissionTokenLinkInput{
		Token:   c.Param("token"),
		QQQuery: c.Query("qq"),
		UserID:  userID,
	})
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, session)
}

func (h *Handler) handleAdmissionMe(c *gin.Context) {
	userID, ok := middleware.ResolveRequiredInternalUserID(
		c,
		h.internalUserIDResolver,
		"failed to resolve admission user",
	)
	if !ok {
		return
	}
	me, err := h.service.GetAdmissionMe(c.Request.Context(), userID, c.Query("admissionSessionID"))
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
	schoolID, ok := h.resolveAdmissionRequestSchoolID(c, req.SchoolCode, req.SchoolID)
	if !ok {
		return
	}
	app, err := h.service.CreateFreshmanApplication(c.Request.Context(), freshmanApplicationCreateInput(userID, schoolID, req))
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

func (h *Handler) handleCreateFreshmanCameraHandoff(c *gin.Context) {
	userID, ok := h.resolveAdmissionUser(c)
	if !ok {
		return
	}
	handoff, err := h.service.CreateFreshmanCameraHandoff(c.Request.Context(), FreshmanCameraHandoffCreateInput{
		UserID:        userID,
		ApplicationID: c.Param("id"),
	})
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Created(c, handoff)
}

func (h *Handler) handleGetFreshmanCameraHandoff(c *gin.Context) {
	userID, ok := h.resolveAdmissionUser(c)
	if !ok {
		return
	}
	handoff, err := h.service.GetFreshmanCameraHandoffForUser(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, handoff)
}

func (h *Handler) handleWatchFreshmanCameraHandoff(c *gin.Context) {
	userID, ok := h.resolveAdmissionUser(c)
	if !ok {
		return
	}
	handoffID := c.Param("id")
	handoff, err := h.service.GetFreshmanCameraHandoffForUser(c.Request.Context(), userID, handoffID)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}

	headers := c.Writer.Header()
	headers.Set("Content-Type", "text/event-stream")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("Connection", "keep-alive")
	headers.Set("X-Accel-Buffering", "no")

	if !writeFreshmanCameraHandoffEvent(c, handoff) {
		return
	}
	if isFreshmanCameraHandoffTerminal(handoff) {
		return
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()
	deadline := time.NewTimer(10 * time.Minute)
	defer deadline.Stop()
	lastSignature := freshmanCameraHandoffSignature(handoff)

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-deadline.C:
			c.SSEvent("end", "timeout")
			c.Writer.Flush()
			return
		case <-keepalive.C:
			c.SSEvent("keepalive", time.Now().UTC().Format(time.RFC3339))
			c.Writer.Flush()
		case <-ticker.C:
			next, err := h.service.GetFreshmanCameraHandoffForUser(c.Request.Context(), userID, handoffID)
			if err != nil {
				c.SSEvent("error", "admission camera handoff status unavailable")
				c.Writer.Flush()
				return
			}
			signature := freshmanCameraHandoffSignature(next)
			if signature == lastSignature {
				continue
			}
			lastSignature = signature
			if !writeFreshmanCameraHandoffEvent(c, next) {
				return
			}
			if isFreshmanCameraHandoffTerminal(next) {
				return
			}
		}
	}
}

func (h *Handler) handlePreviewFreshmanCameraHandoff(c *gin.Context) {
	handoff, err := h.service.PreviewFreshmanCameraHandoff(c.Request.Context(), c.Param("token"))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, handoff)
}

func writeFreshmanCameraHandoffEvent(c *gin.Context, handoff *FreshmanCameraHandoff) bool {
	if handoff == nil {
		c.SSEvent("error", "admission camera handoff status unavailable")
		c.Writer.Flush()
		return false
	}
	c.SSEvent("handoff", handoff)
	c.Writer.Flush()
	return true
}

func freshmanCameraHandoffSignature(handoff *FreshmanCameraHandoff) string {
	if handoff == nil {
		return ""
	}
	sum := sha256.Sum256([]byte(strings.Join([]string{
		string(handoff.Status),
		freshmanCameraHandoffContinuationString(handoff.ContinueOn),
		formatOptionalTime(handoff.UploadedAt),
		formatOptionalTime(handoff.ChosenAt),
		handoff.ExpiresAt.UTC().Format(time.RFC3339Nano),
	}, "|")))
	return hex.EncodeToString(sum[:])
}

func freshmanCameraHandoffContinuationString(value *FreshmanCameraContinuation) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func isFreshmanCameraHandoffTerminal(handoff *FreshmanCameraHandoff) bool {
	return handoff != nil && (handoff.Status == FreshmanCameraHandoffLocked || handoff.Status == FreshmanCameraHandoffExpired)
}

func (h *Handler) handleUploadFreshmanCameraHandoffCapture(c *gin.Context) {
	var req cameraCaptureHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	handoff, err := h.service.SubmitFreshmanCameraHandoffCapture(c.Request.Context(), FreshmanCameraHandoffCaptureInput{
		Token:       c.Param("token"),
		ContentType: req.ContentType,
		ImageBase64: req.ImageBase64,
	})
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, handoff)
}

func (h *Handler) handleChooseFreshmanCameraHandoffContinuation(c *gin.Context) {
	var req cameraHandoffContinuationHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	handoff, err := h.service.ChooseFreshmanCameraHandoffContinuation(c.Request.Context(), FreshmanCameraHandoffContinuationInput{
		Token:      c.Param("token"),
		ContinueOn: req.ContinueOn,
	})
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, handoff)
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
	schoolID, ok := h.resolveAdmissionRequestSchoolID(c, req.SchoolCode, req.SchoolID)
	if !ok {
		return
	}
	result, err := h.service.RequestSchoolEmailOTP(c.Request.Context(), schoolEmailOTPInput(userID, schoolID, req))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) handleMatchSchoolEmailAcademicStudent(c *gin.Context) {
	userID, ok := h.resolveAdmissionUser(c)
	if !ok {
		return
	}
	var req schoolEmailAcademicMatchHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	schoolID, ok := h.resolveAdmissionRequestSchoolID(c, req.SchoolCode, 0)
	if !ok {
		return
	}
	result, err := h.service.MatchSchoolEmailAcademicStudent(c.Request.Context(), SchoolEmailAcademicMatchInput{
		UserID:             userID,
		SchoolID:           schoolID,
		AdmissionSessionID: strings.TrimSpace(req.AdmissionSessionID),
		StudentID:          req.StudentID,
		StudentName:        req.StudentName,
	})
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
	schoolID, ok := h.resolveAdmissionRequestSchoolID(c, req.SchoolCode, req.SchoolID)
	if !ok {
		return
	}
	input := schoolEmailOTPVerifyInput(userID, schoolID, req)
	if _, err := h.service.VerifySchoolEmailOTP(c.Request.Context(), input); err != nil {
		respondAdmissionError(c, err)
		return
	}
	me, err := h.service.GetAdmissionMe(c.Request.Context(), userID, input.AdmissionSessionID)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, me)
}

func (h *Handler) handleStartSchoolSSO(c *gin.Context) {
	userID, schoolID, ok := h.resolveAdmissionUserAndSchool(c)
	if !ok {
		return
	}
	result, err := h.service.StartSchoolSSO(c.Request.Context(), SchoolSSOStartInput{
		UserID:             userID,
		SchoolID:           schoolID,
		ReturnURL:          c.Query("return"),
		AdmissionSessionID: c.Query("admissionSessionID"),
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

func (h *Handler) resolveAdmissionUser(c *gin.Context) (int64, bool) {
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
	schoolID, ok := h.resolveAdmissionRequestSchoolID(c, c.Param("schoolCode"), 0)
	return userID, schoolID, ok
}

func (h *Handler) resolveAdmissionRequestSchoolID(c *gin.Context, schoolCode string, schoolID int64) (int64, bool) {
	code := strings.TrimSpace(schoolCode)
	if code == "" {
		response.BadRequest(c, "schoolCode is required")
		return 0, false
	}
	resolvedSchoolID, err := h.service.ResolveSchoolIDByCode(c.Request.Context(), code)
	if err != nil {
		respondAdmissionError(c, err)
		return 0, false
	}
	if schoolID > 0 && schoolID != resolvedSchoolID {
		response.BadRequest(c, "schoolCode and schoolID mismatch")
		return 0, false
	}
	return resolvedSchoolID, true
}

func freshmanApplicationCreateInput(
	userID int64,
	schoolID int64,
	req freshmanApplicationCreateHTTPRequest,
) FreshmanApplicationCreateInput {
	return FreshmanApplicationCreateInput{
		UserID:             userID,
		SchoolID:           schoolID,
		AdmissionSessionID: strings.TrimSpace(req.AdmissionSessionID),
		ApplicantName:      req.ApplicantName,
		DepartmentOrMajor:  req.DepartmentOrMajor,
		MaterialType:       req.MaterialType,
	}
}

func schoolEmailOTPInput(userID int64, schoolID int64, req schoolEmailOTPHTTPRequest) SchoolEmailOTPInput {
	return SchoolEmailOTPInput{
		UserID:             userID,
		SchoolID:           schoolID,
		AdmissionSessionID: strings.TrimSpace(req.AdmissionSessionID),
		Email:              req.Email,
		StudentID:          req.StudentID,
		StudentName:        req.StudentName,
	}
}

func schoolEmailOTPVerifyInput(userID int64, schoolID int64, req schoolEmailOTPVerifyHTTPRequest) SchoolEmailOTPVerifyInput {
	return SchoolEmailOTPVerifyInput{
		UserID:             userID,
		SchoolID:           schoolID,
		AdmissionSessionID: strings.TrimSpace(req.AdmissionSessionID),
		Email:              req.Email,
		Code:               req.Code,
	}
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
