package studentverification

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

const (
	manualJSONBodyLimit    int64 = 128 * 1024
	manualCaptureBodyLimit int64 = 30 * 1024 * 1024
)

type upsertManualReviewHTTPRequest struct {
	MaterialType         ManualMaterialType `json:"materialType"`
	FormValues           map[string]string  `json:"formValues"`
	PrivacyNoticeVersion string             `json:"privacyNoticeVersion"`
	SensitiveDataConsent bool               `json:"sensitiveDataConsent"`
}

type manualCameraCaptureHTTPRequest struct {
	ContentType         string `json:"contentType"`
	ImageBase64         string `json:"imageBase64"`
	CaptureSource       string `json:"captureSource"`
	RequestedFacingMode string `json:"requestedFacingMode"`
}

type submitManualReviewHTTPRequest struct {
	ConfirmMaterialUse bool `json:"confirmMaterialUse"`
}

type manualCameraContinuationHTTPRequest struct {
	ContinueOn string `json:"continueOn"`
}

type createSchoolVerificationSuggestionHTTPRequest struct {
	SchoolName     string `json:"schoolName"`
	SchoolLocation string `json:"schoolLocation"`
}

type adminManualReviewDecisionHTTPRequest struct {
	Action            ManualReviewDecisionAction `json:"action"`
	UserVisibleReason string                     `json:"userVisibleReason"`
	InternalRiskNote  string                     `json:"internalRiskNote"`
	ExpiresInDays     *int                       `json:"expiresInDays"`
}

func (h *Handler) handleUpsertManualReview(c *gin.Context) {
	userID, applicationID, ok := h.userAndUUIDParam(c, "applicationID")
	if !ok {
		return
	}
	var request upsertManualReviewHTTPRequest
	if !bindManualJSON(c, &request, manualJSONBodyLimit) {
		return
	}
	reviewCase, err := h.service.UpsertManualReview(c.Request.Context(), UpsertManualReviewInput{
		UserID: userID, ApplicationID: applicationID,
		MaterialType: request.MaterialType, FormValues: request.FormValues,
		PrivacyNoticeVersion: request.PrivacyNoticeVersion,
		SensitiveDataConsent: request.SensitiveDataConsent,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, reviewCase)
}

func (h *Handler) handleGetManualReview(c *gin.Context) {
	userID, applicationID, ok := h.userAndUUIDParam(c, "applicationID")
	if !ok {
		return
	}
	reviewCase, err := h.service.GetManualReview(c.Request.Context(), userID, applicationID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, reviewCase)
}

func (h *Handler) handleUploadManualCameraCapture(c *gin.Context) {
	userID, applicationID, ok := h.userAndUUIDParam(c, "applicationID")
	if !ok {
		return
	}
	var request manualCameraCaptureHTTPRequest
	if !bindManualJSON(c, &request, manualCaptureBodyLimit) {
		return
	}
	reviewCase, err := h.service.UploadManualCameraCapture(c.Request.Context(), ManualCameraCaptureInput{
		UserID: userID, ApplicationID: applicationID,
		ContentType: request.ContentType, ImageBase64: request.ImageBase64,
		CaptureSource: request.CaptureSource, RequestedFacingMode: request.RequestedFacingMode,
	})
	request.ImageBase64 = ""
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, reviewCase)
}

func (h *Handler) handleCreateManualCameraHandoff(c *gin.Context) {
	userID, applicationID, ok := h.userAndUUIDParam(c, "applicationID")
	if !ok {
		return
	}
	handoff, err := h.service.CreateManualCameraHandoff(c.Request.Context(), userID, applicationID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, handoff)
}

func (h *Handler) handleGetManualCameraHandoff(c *gin.Context) {
	userID, applicationID, ok := h.userAndUUIDParam(c, "applicationID")
	if !ok {
		return
	}
	handoffID, ok := parseUUIDParam(c, "handoffID")
	if !ok {
		return
	}
	handoff, err := h.service.GetManualCameraHandoff(
		c.Request.Context(), userID, applicationID, handoffID,
	)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, handoff)
}

func (h *Handler) handleSubmitManualReview(c *gin.Context) {
	userID, applicationID, ok := h.userAndUUIDParam(c, "applicationID")
	if !ok {
		return
	}
	var request submitManualReviewHTTPRequest
	if !bindManualJSON(c, &request, manualJSONBodyLimit) {
		return
	}
	reviewCase, err := h.service.SubmitManualReview(
		c.Request.Context(), userID, applicationID, request.ConfirmMaterialUse,
	)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, reviewCase)
}

func (h *Handler) handleRequestManualReviewEmailOTP(c *gin.Context) {
	userID, applicationID, ok := h.userAndUUIDParam(c, "applicationID")
	if !ok {
		return
	}
	challenge, err := h.service.RequestManualReviewEmailOTP(
		c.Request.Context(), userID, applicationID,
	)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, challenge)
}

func (h *Handler) handleVerifyManualReviewEmailOTP(c *gin.Context) {
	userID, applicationID, ok := h.userAndUUIDParam(c, "applicationID")
	if !ok {
		return
	}
	var request verifyStudentEmailOTPHTTPRequest
	if !bindManualJSON(c, &request, manualJSONBodyLimit) {
		return
	}
	reviewCase, err := h.service.VerifyManualReviewEmailOTP(
		c.Request.Context(), userID, applicationID, request.Code,
	)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, reviewCase)
}

func (h *Handler) handleCreateSchoolVerificationSuggestion(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	var request createSchoolVerificationSuggestionHTTPRequest
	if !bindManualJSON(c, &request, manualJSONBodyLimit) {
		return
	}
	suggestion, err := h.service.CreateSchoolVerificationSuggestion(
		c.Request.Context(), userID, request.SchoolName, request.SchoolLocation,
	)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, suggestion)
}

func (h *Handler) handlePreviewManualCameraHandoff(c *gin.Context) {
	handoff, err := h.service.PreviewManualCameraHandoff(c.Request.Context(), c.Param("token"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, handoff)
}

func (h *Handler) handleUploadManualHandoffCameraCapture(c *gin.Context) {
	var request manualCameraCaptureHTTPRequest
	if !bindManualJSON(c, &request, manualCaptureBodyLimit) {
		return
	}
	handoff, err := h.service.UploadManualHandoffCameraCapture(
		c.Request.Context(),
		ManualCameraCaptureInput{
			Token: c.Param("token"), ContentType: request.ContentType,
			ImageBase64: request.ImageBase64, CaptureSource: request.CaptureSource,
			RequestedFacingMode: request.RequestedFacingMode,
		},
	)
	request.ImageBase64 = ""
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, handoff)
}

func (h *Handler) handleChooseManualCameraContinuation(c *gin.Context) {
	var request manualCameraContinuationHTTPRequest
	if !bindManualJSON(c, &request, manualJSONBodyLimit) {
		return
	}
	handoff, err := h.service.ChooseManualCameraContinuation(
		c.Request.Context(), c.Param("token"), request.ContinueOn,
	)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, handoff)
}

func (h *Handler) handleResumeManualCameraHandoff(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	application, err := h.service.ResumeManualCameraHandoff(
		c.Request.Context(), userID, c.Param("token"),
	)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, application)
}

func (h *Handler) handleListAdminManualReviews(c *gin.Context) {
	schoolCode := strings.TrimSpace(c.Query("schoolCode"))
	if !schoolCodePattern.MatchString(schoolCode) {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return
	}
	if !middleware.HasCapabilityInSchool(c, capability.StudentManualReviewRead, schoolCode) {
		response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
		return
	}
	limit, ok := parseStrictBoundedQueryInteger(c, "limit", 50, 1, 100)
	if !ok {
		return
	}
	offset, ok := parseStrictBoundedQueryInteger(c, "offset", 0, 0, 1_000_000)
	if !ok {
		return
	}
	cases, err := h.service.ListAdminManualReviews(
		c.Request.Context(), schoolCode, normalizeManualReviewStatus(c.Query("status")), limit, offset,
	)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, cases)
}

func (h *Handler) handleGetAdminManualReview(c *gin.Context) {
	caseID, ok := parseUUIDParam(c, "caseID")
	if !ok || !h.authorizeManualReviewCase(c, caseID, capability.StudentManualReviewRead) {
		return
	}
	detail, err := h.service.GetAdminManualReview(c.Request.Context(), caseID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, detail)
}

func (h *Handler) handleGetAdminManualMaterialAccess(c *gin.Context) {
	caseID, ok := parseUUIDParam(c, "caseID")
	if !ok || !h.authorizeManualReviewCase(c, caseID, capability.StudentManualMaterialAccess) {
		return
	}
	materialID, ok := parseUUIDParam(c, "materialID")
	if !ok {
		return
	}
	actorUserID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	access, _, err := h.service.GetManualMaterialAccess(
		c.Request.Context(), caseID, materialID, actorUserID,
	)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, access)
}

func (h *Handler) handleDecideAdminManualReview(c *gin.Context) {
	caseID, ok := parseUUIDParam(c, "caseID")
	if !ok || !h.authorizeManualReviewCase(c, caseID, capability.StudentManualReviewDecide) {
		return
	}
	reviewerUserID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	var request adminManualReviewDecisionHTTPRequest
	if !bindManualJSON(c, &request, manualJSONBodyLimit) {
		return
	}
	reviewCase, err := h.service.DecideManualReview(c.Request.Context(), ManualReviewDecisionInput{
		CaseID: caseID, ReviewerUserID: reviewerUserID, Action: request.Action,
		UserVisibleReason: request.UserVisibleReason, InternalRiskNote: request.InternalRiskNote,
		ExpiresInDays: request.ExpiresInDays,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, reviewCase)
}

func (h *Handler) authorizeManualReviewCase(c *gin.Context, caseID string, capabilityName string) bool {
	schoolCode, err := h.service.ManualReviewCaseSchoolCode(c.Request.Context(), caseID)
	if err != nil {
		respondError(c, err)
		return false
	}
	if !middleware.HasCapabilityInSchool(c, capabilityName, schoolCode) {
		if capabilityName == capability.StudentManualMaterialAccess {
			actorUserID, ok := h.resolveCurrentUser(c)
			if !ok {
				return false
			}
			if auditErr := h.service.RecordManualMaterialAccessDenied(
				context.WithoutCancel(c.Request.Context()), caseID, actorUserID,
			); auditErr != nil {
				logger.FromGin(c).Warn("failed to record denied student review material access", zap.Error(auditErr))
			}
		}
		response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
		return false
	}
	return true
}

func bindManualJSON(c *gin.Context, destination any, maximumBytes int64) bool {
	if c.Request == nil || c.Request.Body == nil {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return false
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maximumBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		var maximumError *http.MaxBytesError
		if errors.As(err, &maximumError) {
			response.Error(
				c, http.StatusRequestEntityTooLarge,
				errs.ErrStudentVerificationManualMaterialTooLarge,
				"request body is too large",
			)
			return false
		}
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return false
	}
	return true
}

func parseStrictBoundedQueryInteger(
	c *gin.Context,
	name string,
	fallback int,
	minimum int,
	maximum int,
) (int, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return 0, false
	}
	return value, true
}
