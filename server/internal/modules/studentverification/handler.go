package studentverification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
	"github.com/StuHelper/StuHelper/server/internal/platform/serviceaccount"
)

const (
	studentVerificationSensitiveLimit = 5
	internalEligibilityReadScope      = "student.eligibility.read"
	internalPhoneGateReadScope        = "phone.gate.read"
	bearerPrefix                      = "Bearer "
)

type ServiceCredentialVerifier interface {
	Verify(ctx context.Context, rawToken, audience, scope string) error
}

type Handler struct {
	service                *Service
	internalUserIDResolver middleware.InternalUserIDResolver
	serviceVerifier        ServiceCredentialVerifier
	sensitiveLimiter       *middleware.RedisRateLimiter
	adminAuthorizers       AdminAuthorizers
	inboundWebhookVerifier InboundEmailWebhookVerifier
	rosterSyncCoordinator  AdminRosterSyncCoordinator
}

type AdminAuthorizers struct {
	RosterRead             gin.HandlerFunc
	RosterActivate         gin.HandlerFunc
	ManualReviewRead       gin.HandlerFunc
	ManualReviewDecide     gin.HandlerFunc
	ManualMaterialAccess   gin.HandlerFunc
	ConfigRead             gin.HandlerFunc
	ConfigUpdate           gin.HandlerFunc
	CredentialRead         gin.HandlerFunc
	CredentialRevoke       gin.HandlerFunc
	SubjectConflictRead    gin.HandlerFunc
	SubjectConflictResolve gin.HandlerFunc
	ConnectorHealthRead    gin.HandlerFunc
	ConnectorManage        gin.HandlerFunc
	StepUpMFA              gin.HandlerFunc
}

type HandlerOption func(*Handler)

func WithAdminAuthorizers(authorizers AdminAuthorizers) HandlerOption {
	return func(handler *Handler) { handler.adminAuthorizers = authorizers }
}

func WithInboundEmailWebhookVerifier(verifier InboundEmailWebhookVerifier) HandlerOption {
	return func(handler *Handler) { handler.inboundWebhookVerifier = verifier }
}

func WithAdminRosterSyncCoordinator(coordinator AdminRosterSyncCoordinator) HandlerOption {
	return func(handler *Handler) { handler.rosterSyncCoordinator = coordinator }
}

func NewHandler(
	service *Service,
	internalUserIDResolver middleware.InternalUserIDResolver,
	serviceVerifier ServiceCredentialVerifier,
	rdb *redis.Client,
	options ...HandlerOption,
) *Handler {
	if service == nil {
		panic("studentverification.NewHandler: service must not be nil")
	}
	if internalUserIDResolver == nil {
		panic("studentverification.NewHandler: internal user id resolver must not be nil")
	}
	handler := &Handler{
		service:                service,
		internalUserIDResolver: internalUserIDResolver,
		serviceVerifier:        serviceVerifier,
		sensitiveLimiter: middleware.NewRedisRateLimiter(
			rdb,
			studentVerificationSensitiveLimit,
			time.Minute,
		),
	}
	for _, option := range options {
		if option != nil {
			option(handler)
		}
	}
	return handler
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, authMW gin.HandlerFunc, stepUpMW gin.HandlerFunc) {
	if stepUpMW == nil {
		stepUpMW = func(c *gin.Context) {
			response.ServiceUnavailable(c, "step-up verification is unavailable", errs.ErrServiceUnavailable)
			c.Abort()
		}
	}
	verification := api.Group("/student-verification")
	verification.Use(authMW)
	verification.GET("/schools", h.handleListSchools)
	verification.POST("/applications", h.handleCreateApplication)
	verification.GET("/applications/:applicationID", h.handleGetApplication)
	verification.DELETE("/applications/:applicationID", h.handleCancelApplication)
	verification.POST(
		"/applications/:applicationID/real-name/verify",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "student-verification-real-name"),
		h.handleVerifyRealName,
	)
	verification.POST(
		"/applications/:applicationID/school-sso/verify",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "student-verification-school-sso"),
		h.handleVerifySchoolSSO,
	)
	verification.POST(
		"/applications/:applicationID/email/outbound/otp",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "student-verification-email-outbound"),
		h.handleRequestEmailOTP,
	)
	verification.POST(
		"/applications/:applicationID/email/outbound/verify",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "student-verification-email-outbound-verify"),
		h.handleVerifyEmailOTP,
	)
	verification.POST(
		"/applications/:applicationID/email/inbound/challenge",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "student-verification-email-inbound"),
		h.handleCreateInboundEmailChallenge,
	)
	verification.GET("/applications/:applicationID/email/inbound/challenge", h.handleGetInboundEmailChallenge)
	verification.GET("/credentials", h.handleListCredentials)
	verification.DELETE("/credentials/:credentialID", h.handleRevokeCredential)
	verification.GET("/eligibility", h.handleGetEligibility)
	verification.PUT(
		"/applications/:applicationID/manual-review",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "student-verification-manual-draft"),
		h.handleUpsertManualReview,
	)
	verification.GET("/applications/:applicationID/manual-review", h.handleGetManualReview)
	verification.POST(
		"/applications/:applicationID/manual-review/camera-captures",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "student-verification-manual-capture"),
		h.handleUploadManualCameraCapture,
	)
	verification.POST(
		"/applications/:applicationID/manual-review/camera-handoffs",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "student-verification-manual-handoff-create"),
		h.handleCreateManualCameraHandoff,
	)
	verification.GET(
		"/applications/:applicationID/manual-review/camera-handoffs/:handoffID",
		h.handleGetManualCameraHandoff,
	)
	verification.POST(
		"/applications/:applicationID/manual-review/submit",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "student-verification-manual-submit"),
		h.handleSubmitManualReview,
	)
	verification.POST(
		"/applications/:applicationID/manual-review/email/otp",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "student-verification-manual-email-otp"),
		h.handleRequestManualReviewEmailOTP,
	)
	verification.POST(
		"/applications/:applicationID/manual-review/email/verify",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "student-verification-manual-email-verify"),
		h.handleVerifyManualReviewEmailOTP,
	)
	verification.POST(
		"/school-suggestions",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "student-verification-school-suggestion"),
		h.handleCreateSchoolVerificationSuggestion,
	)
	verification.POST(
		"/manual-camera-handoffs/:token/resume",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "student-verification-manual-handoff-resume"),
		h.handleResumeManualCameraHandoff,
	)

	internal := api.Group("/internal")
	internal.GET(
		"/student-eligibility/users/:userID/schools/:schoolCode",
		h.requireServiceCredential(internalEligibilityReadScope),
		h.handleGetInternalEligibility,
	)
	internal.GET(
		"/phone-gates/users/:userID",
		h.requireServiceCredential(internalPhoneGateReadScope),
		h.handleGetInternalPhoneGate,
	)

	accountPhone := api.Group("/account/phone")
	accountPhone.Use(authMW)
	accountPhone.GET("", h.handleGetPhoneStatus)
	accountPhone.DELETE("", stepUpMW, h.handleUnbindPhone)
	accountPhone.POST(
		"/operations",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "account-phone-bind"),
		h.handleCreatePhoneBinding,
	)
	accountPhone.POST(
		"/change-operations",
		stepUpMW,
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "account-phone-change"),
		h.handleCreatePhoneChange,
	)
	accountPhone.GET("/operations/:operationID", h.handleGetPhoneOperation)
	accountPhone.POST(
		"/operations/:operationID/sms",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "account-phone-sms"),
		h.handleSendPhoneSMS,
	)
	accountPhone.POST(
		"/operations/:operationID/sms/verify",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "account-phone-sms-verify"),
		h.handleVerifyPhoneSMS,
	)

	api.POST(
		"/webhooks/student-verification/inbound-email",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "student-verification-inbound-webhook"),
		h.handleInboundEmailWebhook,
	)

	publicManual := api.Group("/student-verification/manual-camera-handoffs")
	publicManual.GET(
		"/:token",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "student-verification-manual-handoff-preview"),
		h.handlePreviewManualCameraHandoff,
	)
	publicManual.POST(
		"/:token/camera-capture",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "student-verification-manual-handoff-capture"),
		h.handleUploadManualHandoffCameraCapture,
	)
	publicManual.POST(
		"/:token/continue",
		middleware.EndpointRateLimitMiddleware(h.sensitiveLimiter, "student-verification-manual-handoff-continue"),
		h.handleChooseManualCameraContinuation,
	)
}

func (h *Handler) RegisterAdminRoutes(admin *gin.RouterGroup) {
	h.registerAdminControlPlaneRoutes(admin)
	base := "/student-verification/schools/:schoolCode/roster-snapshots"
	admin.GET(
		base,
		httputil.RouteHandlers(h.handleListAdminRosterSnapshots, h.adminAuthorizers.RosterRead)...,
	)
	admin.GET(
		base+"/:snapshotID",
		httputil.RouteHandlers(h.handleGetAdminRosterSnapshot, h.adminAuthorizers.RosterRead)...,
	)
	admin.POST(
		base+"/:snapshotID/activate",
		httputil.RouteHandlers(
			h.handleActivateAdminRosterSnapshot,
			h.adminAuthorizers.RosterActivate,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
	admin.POST(
		base+"/:snapshotID/rollback",
		httputil.RouteHandlers(
			h.handleRollbackAdminRosterSnapshot,
			h.adminAuthorizers.RosterActivate,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
	admin.GET(
		"/student-verification/schools/:schoolCode/roster-sync-requests",
		httputil.RouteHandlers(
			h.handleListAdminRosterSyncRequests,
			h.adminAuthorizers.ConnectorManage,
		)...,
	)
	admin.POST(
		"/student-verification/schools/:schoolCode/roster-sync-requests",
		httputil.RouteHandlers(
			h.handleCreateAdminRosterSyncRequest,
			h.adminAuthorizers.ConnectorManage,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)

	manualBase := "/student-verification/manual-reviews"
	admin.GET(
		manualBase,
		httputil.RouteHandlers(h.handleListAdminManualReviews, h.adminAuthorizers.ManualReviewRead)...,
	)
	admin.GET(
		manualBase+"/:caseID",
		httputil.RouteHandlers(
			h.handleGetAdminManualReview,
			h.adminAuthorizers.ManualReviewRead,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
	admin.POST(
		manualBase+"/:caseID/materials/:materialID/access",
		httputil.RouteHandlers(
			h.handleGetAdminManualMaterialAccess,
			h.adminAuthorizers.ManualMaterialAccess,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
	admin.POST(
		manualBase+"/:caseID/decision",
		httputil.RouteHandlers(
			h.handleDecideAdminManualReview,
			h.adminAuthorizers.ManualReviewDecide,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
}

type createApplicationHTTPRequest struct {
	SchoolCode        string `json:"schoolCode" binding:"required,len=10,numeric"`
	ContinuationToken string `json:"continuationToken" binding:"omitempty,min=20,max=2048"`
}

type verifyRealNameHTTPRequest struct {
	StudentID            string `json:"studentID" binding:"required,max=64"`
	Name                 string `json:"name" binding:"required,max=100"`
	DocumentNumber       string `json:"documentNumber" binding:"required,len=18"`
	PrivacyNoticeVersion string `json:"privacyNoticeVersion" binding:"required,max=100"`
	SensitiveDataConsent bool   `json:"sensitiveDataConsent" binding:"required"`
}

type studentEmailIdentityHTTPRequest struct {
	StudentID            string `json:"studentID" binding:"required,max=64"`
	Name                 string `json:"name" binding:"required,max=100"`
	PrivacyNoticeVersion string `json:"privacyNoticeVersion" binding:"required,max=100"`
	SensitiveDataConsent bool   `json:"sensitiveDataConsent" binding:"required"`
}

type verifySchoolSSOHTTPRequest struct {
	StudentID            string `json:"studentID" binding:"required,max=64"`
	Password             string `json:"password" binding:"required,max=256"`
	PrivacyNoticeVersion string `json:"privacyNoticeVersion" binding:"required,max=100"`
	SensitiveDataConsent bool   `json:"sensitiveDataConsent" binding:"required"`
}

type verifyStudentEmailOTPHTTPRequest struct {
	Code string `json:"code" binding:"required,min=4,max=10,numeric"`
}

type createPhoneOperationHTTPRequest struct {
	OperationKind string `json:"operationKind" binding:"omitempty,oneof=bind change"`
	Phone         string `json:"phone" binding:"required,max=20"`
	SchoolCode    string `json:"schoolCode" binding:"omitempty,len=10,numeric"`
	StudentID     string `json:"studentID" binding:"omitempty,max=64"`
	Name          string `json:"name" binding:"omitempty,max=100"`
}

type verifyPhoneSMSHTTPRequest struct {
	Code string `json:"code" binding:"required,min=4,max=10,numeric"`
}

type rosterSnapshotSwitchHTTPRequest struct {
	Reason                string `json:"reason" binding:"required,min=4,max=500"`
	AllowSourceRegression bool   `json:"allowSourceRegression"`
}

type inboundEmailWebhookHTTPRequest struct {
	EnvelopeFrom string    `json:"envelopeFrom"`
	HeaderFrom   string    `json:"headerFrom"`
	Subject      string    `json:"subject"`
	TextBody     string    `json:"textBody"`
	SPF          string    `json:"spf"`
	DKIM         string    `json:"dkim"`
	DMARC        string    `json:"dmarc"`
	ReceivedAt   time.Time `json:"receivedAt"`
}

func (h *Handler) handleListSchools(c *gin.Context) {
	schools, err := h.service.ListSchools(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, schools)
}

func (h *Handler) handleCreateApplication(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	var request createApplicationHTTPRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return
	}
	application, err := h.service.CreateApplication(c.Request.Context(), CreateApplicationInput{
		UserID:            userID,
		SchoolCode:        request.SchoolCode,
		ContinuationToken: request.ContinuationToken,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, application)
}

func (h *Handler) handleGetApplication(c *gin.Context) {
	userID, applicationID, ok := h.userAndUUIDParam(c, "applicationID")
	if !ok {
		return
	}
	application, err := h.service.GetApplication(c.Request.Context(), userID, applicationID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, application)
}

func (h *Handler) handleCancelApplication(c *gin.Context) {
	userID, applicationID, ok := h.userAndUUIDParam(c, "applicationID")
	if !ok {
		return
	}
	application, err := h.service.CancelApplication(c.Request.Context(), userID, applicationID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, application)
}

func (h *Handler) handleVerifyRealName(c *gin.Context) {
	userID, applicationID, ok := h.userAndUUIDParam(c, "applicationID")
	if !ok {
		return
	}
	var request verifyRealNameHTTPRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return
	}
	application, err := h.service.VerifyRealName(c.Request.Context(), VerifyRealNameInput{
		UserID:               userID,
		ApplicationID:        applicationID,
		StudentID:            request.StudentID,
		Name:                 request.Name,
		DocumentNumber:       request.DocumentNumber,
		PrivacyNoticeVersion: request.PrivacyNoticeVersion,
		SensitiveDataConsent: request.SensitiveDataConsent,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, application)
}

func (h *Handler) handleRequestEmailOTP(c *gin.Context) {
	userID, applicationID, ok := h.userAndUUIDParam(c, "applicationID")
	if !ok {
		return
	}
	var request studentEmailIdentityHTTPRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return
	}
	challenge, err := h.service.RequestStudentEmailOTP(c.Request.Context(), StudentEmailIdentityInput{
		UserID: userID, ApplicationID: applicationID,
		StudentID: request.StudentID, Name: request.Name,
		PrivacyNoticeVersion: request.PrivacyNoticeVersion,
		SensitiveDataConsent: request.SensitiveDataConsent,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, challenge)
}

func (h *Handler) handleVerifySchoolSSO(c *gin.Context) {
	userID, applicationID, ok := h.userAndUUIDParam(c, "applicationID")
	if !ok {
		return
	}
	var request verifySchoolSSOHTTPRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return
	}
	password := []byte(request.Password)
	request.Password = ""
	defer clear(password)
	application, err := h.service.VerifySchoolSSO(c.Request.Context(), VerifySchoolSSOInput{
		UserID: userID, ApplicationID: applicationID,
		StudentID: request.StudentID, Password: password,
		PrivacyNoticeVersion: request.PrivacyNoticeVersion,
		SensitiveDataConsent: request.SensitiveDataConsent,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, application)
}

func (h *Handler) handleVerifyEmailOTP(c *gin.Context) {
	userID, applicationID, ok := h.userAndUUIDParam(c, "applicationID")
	if !ok {
		return
	}
	var request verifyStudentEmailOTPHTTPRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return
	}
	application, err := h.service.VerifyStudentEmailOTP(c.Request.Context(), VerifyStudentEmailOTPInput{
		UserID: userID, ApplicationID: applicationID, Code: request.Code,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, application)
}

func (h *Handler) handleCreateInboundEmailChallenge(c *gin.Context) {
	userID, applicationID, ok := h.userAndUUIDParam(c, "applicationID")
	if !ok {
		return
	}
	var request studentEmailIdentityHTTPRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return
	}
	challenge, err := h.service.CreateInboundEmailChallenge(c.Request.Context(), StudentEmailIdentityInput{
		UserID: userID, ApplicationID: applicationID,
		StudentID: request.StudentID, Name: request.Name,
		PrivacyNoticeVersion: request.PrivacyNoticeVersion,
		SensitiveDataConsent: request.SensitiveDataConsent,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, challenge)
}

func (h *Handler) handleGetInboundEmailChallenge(c *gin.Context) {
	userID, applicationID, ok := h.userAndUUIDParam(c, "applicationID")
	if !ok {
		return
	}
	challenge, err := h.service.GetInboundEmailChallenge(c.Request.Context(), userID, applicationID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, challenge)
}

func (h *Handler) handleInboundEmailWebhook(c *gin.Context) {
	if h.inboundWebhookVerifier == nil {
		response.ServiceUnavailable(c, "inbound email receiver is unavailable", errs.ErrServiceUnavailable)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64*1024)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil || len(body) == 0 {
		response.BadRequest(c, "invalid webhook request", errs.ErrInvalidParam)
		return
	}
	timestamp := c.GetHeader("X-StuHelper-Webhook-Timestamp")
	eventID := c.GetHeader("X-StuHelper-Webhook-ID")
	signature := c.GetHeader("X-StuHelper-Webhook-Signature")
	if err := h.inboundWebhookVerifier.Verify(timestamp, eventID, signature, body, h.service.now()); err != nil {
		response.Unauthorized(c, "invalid webhook signature")
		return
	}
	var request inboundEmailWebhookHTTPRequest
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	decodeErr := decoder.Decode(&request)
	var trailing any
	trailingErr := decoder.Decode(&trailing)
	if decodeErr != nil || !errors.Is(trailingErr, io.EOF) ||
		request.EnvelopeFrom == "" || request.HeaderFrom == "" || request.Subject == "" ||
		request.TextBody == "" || request.ReceivedAt.IsZero() ||
		!validMailAuthenticationResult(request.SPF) ||
		!validMailAuthenticationResult(request.DKIM) ||
		!validMailAuthenticationResult(request.DMARC) {
		response.BadRequest(c, "invalid webhook request", errs.ErrInvalidParam)
		return
	}
	if err := h.service.ProcessInboundEmailEvent(c.Request.Context(), InboundEmailEvent{
		EventReference: eventID, EnvelopeFrom: request.EnvelopeFrom,
		HeaderFrom: request.HeaderFrom, Subject: request.Subject, TextBody: request.TextBody,
		SPFPass: request.SPF == "pass", DKIMPass: request.DKIM == "pass",
		DMARCPass: request.DMARC == "pass", ReceivedAt: request.ReceivedAt,
	}); err != nil {
		respondError(c, err)
		return
	}
	response.Accepted(c, map[string]string{"status": "accepted"})
}

func validMailAuthenticationResult(value string) bool {
	return value == "pass" || value == "fail" || value == "neutral" || value == "none"
}

func (h *Handler) handleListCredentials(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	credentials, err := h.service.ListCredentials(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, credentials)
}

func (h *Handler) handleRevokeCredential(c *gin.Context) {
	userID, credentialID, ok := h.userAndUUIDParam(c, "credentialID")
	if !ok {
		return
	}
	credential, err := h.service.RevokeCredential(c.Request.Context(), userID, credentialID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, credential)
}

func (h *Handler) handleGetEligibility(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	eligibility, err := h.service.GetEligibility(c.Request.Context(), userID, c.Query("schoolCode"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, eligibility)
}

func (h *Handler) handleGetInternalEligibility(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userID"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return
	}
	eligibility, err := h.service.GetEligibility(c.Request.Context(), userID, c.Param("schoolCode"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, eligibility)
}

func (h *Handler) handleGetPhoneStatus(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	status, err := h.service.GetPhoneStatus(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, status)
}

func (h *Handler) handleCreatePhoneBinding(c *gin.Context) {
	h.handleCreatePhoneOperation(c, PhoneOperationBind)
}

func (h *Handler) handleCreatePhoneChange(c *gin.Context) {
	h.handleCreatePhoneOperation(c, PhoneOperationChange)
}

func (h *Handler) handleCreatePhoneOperation(c *gin.Context, kind PhoneOperationKind) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	var request createPhoneOperationHTTPRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return
	}
	if request.OperationKind != "" && request.OperationKind != string(kind) {
		response.BadRequest(c, "use the dedicated phone operation endpoint", errs.ErrInvalidParam)
		return
	}
	operation, err := h.service.CreatePhoneOperation(c.Request.Context(), CreatePhoneOperationInput{
		UserID: userID, Kind: kind, Phone: request.Phone,
		SchoolCode: request.SchoolCode, StudentID: request.StudentID, Name: request.Name,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, operation)
}

func (h *Handler) handleGetPhoneOperation(c *gin.Context) {
	userID, operationID, ok := h.userAndUUIDParam(c, "operationID")
	if !ok {
		return
	}
	operation, err := h.service.GetPhoneOperation(c.Request.Context(), userID, operationID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, operation)
}

func (h *Handler) handleSendPhoneSMS(c *gin.Context) {
	userID, operationID, ok := h.userAndUUIDParam(c, "operationID")
	if !ok {
		return
	}
	operation, err := h.service.SendPhoneSMS(c.Request.Context(), userID, operationID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, operation)
}

func (h *Handler) handleVerifyPhoneSMS(c *gin.Context) {
	userID, operationID, ok := h.userAndUUIDParam(c, "operationID")
	if !ok {
		return
	}
	var request verifyPhoneSMSHTTPRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return
	}
	operation, err := h.service.VerifyPhoneSMS(c.Request.Context(), VerifyPhoneSMSInput{
		UserID: userID, OperationID: operationID, Code: request.Code,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, operation)
}

func (h *Handler) handleUnbindPhone(c *gin.Context) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	operation, err := h.service.CreatePhoneUnbindOperation(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Accepted(c, operation)
}

func (h *Handler) handleGetInternalPhoneGate(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userID"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return
	}
	eligibility, err := h.service.GetPhoneGateEligibility(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, eligibility)
}

func (h *Handler) handleListAdminRosterSnapshots(c *gin.Context) {
	schoolCode, ok := h.authorizeRosterSchool(c, capability.StudentRosterRead)
	if !ok {
		return
	}
	limit := parseBoundedQueryInteger(c.Query("limit"), 50, 1, 100)
	offset := parseBoundedQueryInteger(c.Query("offset"), 0, 0, 1_000_000)
	snapshots, err := h.service.ListRosterSnapshots(c.Request.Context(), schoolCode, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, snapshots)
}

func (h *Handler) handleGetAdminRosterSnapshot(c *gin.Context) {
	schoolCode, ok := h.authorizeRosterSchool(c, capability.StudentRosterRead)
	if !ok {
		return
	}
	snapshotID, ok := parseUUIDParam(c, "snapshotID")
	if !ok {
		return
	}
	snapshot, err := h.service.GetRosterSnapshotForSchool(c.Request.Context(), schoolCode, snapshotID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, snapshot)
}

func (h *Handler) handleActivateAdminRosterSnapshot(c *gin.Context) {
	h.handleAdminRosterSnapshotSwitch(c, false)
}

func (h *Handler) handleRollbackAdminRosterSnapshot(c *gin.Context) {
	h.handleAdminRosterSnapshotSwitch(c, true)
}

func (h *Handler) handleAdminRosterSnapshotSwitch(c *gin.Context, rollback bool) {
	schoolCode, ok := h.authorizeRosterSchool(c, capability.StudentRosterActivate)
	if !ok {
		return
	}
	snapshotID, ok := parseUUIDParam(c, "snapshotID")
	if !ok {
		return
	}
	actorUserID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	var request rosterSnapshotSwitchHTTPRequest
	if err := c.ShouldBindJSON(&request); err != nil || (rollback && request.AllowSourceRegression) {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return
	}
	input := RosterSnapshotSwitchInput{
		SchoolCode: schoolCode, SnapshotID: snapshotID, ActorUserID: actorUserID,
		Reason: request.Reason, AllowSourceRegression: request.AllowSourceRegression,
	}
	var (
		snapshot *RosterSnapshot
		err      error
	)
	if rollback {
		snapshot, err = h.service.RollbackRosterSnapshot(c.Request.Context(), input)
	} else {
		snapshot, err = h.service.ActivateRosterSnapshot(c.Request.Context(), input)
	}
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, snapshot)
}

func (h *Handler) authorizeRosterSchool(c *gin.Context, capabilityName string) (string, bool) {
	schoolCode := strings.TrimSpace(c.Param("schoolCode"))
	if !schoolCodePattern.MatchString(schoolCode) {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return "", false
	}
	if !middleware.HasCapabilityInSchool(c, capabilityName, schoolCode) {
		response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
		return "", false
	}
	return schoolCode, true
}

func parseUUIDParam(c *gin.Context, name string) (string, bool) {
	value := strings.TrimSpace(c.Param(name))
	if _, err := uuid.Parse(value); err != nil {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return "", false
	}
	return value, true
}

func parseBoundedQueryInteger(raw string, fallback int, minimum int, maximum int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return fallback
	}
	return value
}

func (h *Handler) resolveCurrentUser(c *gin.Context) (int64, bool) {
	return middleware.ResolveRequiredInternalUserID(
		c,
		h.internalUserIDResolver,
		"failed to resolve user",
	)
}

func (h *Handler) userAndUUIDParam(c *gin.Context, name string) (int64, string, bool) {
	userID, ok := h.resolveCurrentUser(c)
	if !ok {
		return 0, "", false
	}
	value := c.Param(name)
	if _, err := uuid.Parse(value); err != nil {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return 0, "", false
	}
	return userID, value, true
}

func (h *Handler) requireServiceCredential(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.serviceVerifier == nil {
			response.ServiceUnavailable(c, "service credential verification is unavailable")
			return
		}
		if c.Request == nil || len(c.Request.Header.Values("Authorization")) != 1 {
			response.Unauthorized(c, "unauthorized")
			return
		}
		raw, ok := parseBearerToken(c.GetHeader("Authorization"))
		if !ok {
			response.Unauthorized(c, "unauthorized")
			return
		}
		if err := h.serviceVerifier.Verify(c.Request.Context(), raw, c.FullPath(), scope); err != nil {
			switch {
			case errors.Is(err, serviceaccount.ErrCredentialForbidden):
				response.Forbidden(c, "forbidden")
			case errors.Is(err, serviceaccount.ErrCredentialStoreUnavailable):
				response.ServiceUnavailable(c, "service credential verification is unavailable")
			default:
				response.Unauthorized(c, "unauthorized")
			}
			return
		}
		c.Next()
	}
}

func parseBearerToken(value string) (string, bool) {
	if !strings.HasPrefix(value, bearerPrefix) {
		return "", false
	}
	raw := strings.TrimSpace(strings.TrimPrefix(value, bearerPrefix))
	return raw, raw != "" && !strings.ContainsAny(raw, " \t\r\n,")
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrSchoolNotFound):
		response.NotFound(c, "verification school not found", errs.ErrStudentVerificationSchoolNotFound)
	case errors.Is(err, ErrSchoolUnavailable):
		response.ServiceUnavailable(c, "school verification is temporarily unavailable", errs.ErrStudentVerificationSchoolUnavailable)
	case errors.Is(err, ErrMethodUnavailable):
		response.ServiceUnavailable(c, "this verification method is temporarily unavailable", errs.ErrStudentVerificationMethodUnavailable)
	case errors.Is(err, ErrApplicationNotFound):
		response.NotFound(c, "verification application not found", errs.ErrStudentVerificationApplicationMissing)
	case errors.Is(err, ErrApplicationExpired):
		response.Error(c, http.StatusGone, errs.ErrStudentVerificationApplicationExpired, "verification application expired")
	case errors.Is(err, ErrApplicationState):
		response.Conflict(c, "verification application state changed; reload and try again", errs.ErrStudentVerificationStateConflict)
	case errors.Is(err, ErrConsentRequired):
		response.BadRequest(c, "accept the current privacy notice before continuing", errs.ErrStudentVerificationConsentRequired)
	case errors.Is(err, ErrInformationMismatch), errors.Is(err, ErrSubjectConflict),
		errors.Is(err, ErrSchoolAccountRejected), errors.Is(err, ErrSchoolAccountLocked),
		errors.Is(err, ErrSchoolAccountNotStudent):
		response.ErrorWithDetails(
			c,
			http.StatusBadRequest,
			errs.ErrStudentVerificationCannotComplete,
			"unable to complete verification with this method",
			map[string]any{"actions": []string{"retry", "choose_another_method", "open_manual_review"}},
		)
	case errors.Is(err, ErrDependencyUnavailable):
		response.ServiceUnavailable(c, "verification service is temporarily unavailable")
	case errors.Is(err, ErrCredentialNotFound):
		response.NotFound(c, "verification credential not found", errs.ErrStudentVerificationCredentialMissing)
	case errors.Is(err, ErrCredentialState):
		response.Conflict(c, "verification credential state changed", errs.ErrStudentVerificationCredentialConflict)
	case errors.Is(err, ErrContinuationInvalid):
		response.BadRequest(c, "verification continuation is invalid", errs.ErrStudentVerificationContinuationInvalid)
	case errors.Is(err, ErrEmailOTPCooldown):
		response.RateLimitExceeded(c, "please wait before requesting another code", errs.ErrStudentVerificationOTPCooldown)
	case errors.Is(err, ErrEmailOTPExpired):
		response.Error(c, http.StatusGone, errs.ErrStudentVerificationOTPExpired, "verification code expired")
	case errors.Is(err, ErrEmailOTPInvalid):
		response.BadRequest(c, "verification code is invalid", errs.ErrStudentVerificationOTPInvalid)
	case errors.Is(err, ErrEmailOTPMaxAttempts):
		response.RateLimitExceeded(c, "too many verification attempts", errs.ErrStudentVerificationOTPMaxAttempts)
	case errors.Is(err, ErrInboundEmailChallengeNotFound):
		response.NotFound(c, "inbound email challenge not found", errs.ErrStudentVerificationInboundChallengeMissing)
	case errors.Is(err, ErrInboundEmailEventInvalid):
		response.BadRequest(c, "invalid inbound email event", errs.ErrInvalidParam)
	case errors.Is(err, ErrPhoneInvalid):
		response.BadRequest(c, "invalid phone number format", errs.ErrAccountPhoneInvalid)
	case errors.Is(err, ErrPhoneOperationNotFound):
		response.NotFound(c, "phone operation not found", errs.ErrAccountPhoneOperationMissing)
	case errors.Is(err, ErrPhoneOperationConflict):
		response.Conflict(c, "phone operation state changed; reload and try again", errs.ErrAccountPhoneOperationConflict)
	case errors.Is(err, ErrPhoneOwnershipConflict):
		response.ErrorWithDetails(
			c,
			http.StatusConflict,
			errs.ErrAccountPhoneBindingConflict,
			"unable to complete this phone operation",
			map[string]any{"actions": []string{"retry", "open_account_recovery"}},
		)
	case errors.Is(err, ErrPhoneAlreadyBound):
		response.Conflict(c, "this account already has a verified phone", errs.ErrAccountPhoneAlreadyBound)
	case errors.Is(err, ErrPhoneNotBound):
		response.Conflict(c, "this account has no verified phone to change", errs.ErrAccountPhoneNotBound)
	case errors.Is(err, ErrPhoneOTPCooldown), errors.Is(err, ErrPhoneOTPRateLimited):
		response.RateLimitExceeded(c, "please wait before requesting another code", errs.ErrAccountPhoneOTPCooldown)
	case errors.Is(err, ErrPhoneOTPExpired):
		response.Error(c, http.StatusGone, errs.ErrAccountPhoneOTPExpired, "verification code expired")
	case errors.Is(err, ErrPhoneOTPInvalid):
		response.BadRequest(c, "verification code is invalid", errs.ErrAccountPhoneOTPInvalid)
	case errors.Is(err, ErrPhoneOTPMaxAttempts):
		response.RateLimitExceeded(c, "too many verification attempts", errs.ErrAccountPhoneOTPMaxAttempts)
	case errors.Is(err, ErrRosterSnapshotNotFound):
		response.NotFound(c, "roster snapshot not found", errs.ErrStudentVerificationRosterSnapshotMissing)
	case errors.Is(err, ErrRosterSnapshotState), errors.Is(err, ErrRosterSourceConflict):
		response.Conflict(c, "roster snapshot state changed; reload and try again", errs.ErrStudentVerificationRosterSnapshotConflict)
	case errors.Is(err, ErrRosterPolicyInvalid), errors.Is(err, ErrRosterQualityFailed),
		errors.Is(err, ErrRosterSourceRegression):
		response.Conflict(c, "roster snapshot cannot be activated", errs.ErrStudentVerificationRosterQualityFailed)
	case errors.Is(err, ErrManualReviewNotFound):
		response.NotFound(c, "manual review case not found", errs.ErrStudentVerificationManualReviewMissing)
	case errors.Is(err, ErrManualReviewState):
		response.Conflict(c, "manual review state changed; reload and try again", errs.ErrStudentVerificationManualReviewConflict)
	case errors.Is(err, ErrManualReviewInvalidForm), errors.Is(err, ErrManualHandoffChoice):
		response.BadRequest(c, "invalid manual review request", errs.ErrStudentVerificationManualReviewInvalid)
	case errors.Is(err, ErrManualReviewSelfDecision):
		response.Forbidden(c, "reviewers cannot decide their own application", errs.ErrStudentVerificationManualReviewSelfDecision)
	case errors.Is(err, ErrManualMaterialStoreUnavailable):
		response.ServiceUnavailable(c, "manual review material service is temporarily unavailable", errs.ErrStudentVerificationManualMaterialUnavailable)
	case errors.Is(err, ErrManualMaterialInvalidType), errors.Is(err, ErrManualMaterialInvalidData),
		errors.Is(err, ErrManualMaterialPixelBounds):
		response.BadRequest(c, "captured material is invalid", errs.ErrStudentVerificationManualMaterialInvalid)
	case errors.Is(err, ErrManualMaterialTooLarge):
		response.Error(c, http.StatusRequestEntityTooLarge, errs.ErrStudentVerificationManualMaterialTooLarge, "captured material is too large")
	case errors.Is(err, ErrManualMaterialLimit):
		response.Conflict(c, "manual review material limit reached", errs.ErrStudentVerificationManualMaterialLimit)
	case errors.Is(err, ErrManualMaterialRequired):
		response.Conflict(c, "capture at least one material before submitting", errs.ErrStudentVerificationManualMaterialMissing)
	case errors.Is(err, ErrManualMaterialNotFound):
		response.NotFound(c, "manual review material not found", errs.ErrStudentVerificationManualMaterialMissing)
	case errors.Is(err, ErrManualHandoffNotFound):
		response.NotFound(c, "camera handoff not found", errs.ErrStudentVerificationManualHandoffMissing)
	case errors.Is(err, ErrManualHandoffExpired):
		response.Error(c, http.StatusGone, errs.ErrStudentVerificationManualHandoffExpired, "camera handoff expired")
	case errors.Is(err, ErrManualHandoffState):
		response.Conflict(c, "camera handoff state changed", errs.ErrStudentVerificationManualHandoffConflict)
	case errors.Is(err, ErrManualEmailVerificationRequired):
		response.Conflict(c, "verify the contact email before submitting", errs.ErrStudentVerificationManualEmailRequired)
	case errors.Is(err, ErrManualEmailOTPCooldown):
		response.RateLimitExceeded(c, "please wait before requesting another code", errs.ErrStudentVerificationManualEmailOTPCooldown)
	case errors.Is(err, ErrManualEmailOTPExpired):
		response.Error(c, http.StatusGone, errs.ErrStudentVerificationManualEmailOTPExpired, "verification code expired")
	case errors.Is(err, ErrManualEmailOTPInvalid):
		response.BadRequest(c, "verification code is invalid", errs.ErrStudentVerificationManualEmailOTPInvalid)
	case errors.Is(err, ErrManualEmailOTPMaxAttempts):
		response.RateLimitExceeded(c, "too many verification attempts", errs.ErrStudentVerificationManualEmailOTPMaxAttempts)
	case errors.Is(err, ErrAdminConfigInvalid):
		response.BadRequest(c, "invalid student verification administration request", errs.ErrStudentVerificationAdminConfigInvalid)
	case errors.Is(err, ErrAdminConfigRevision):
		response.Conflict(c, "student verification configuration changed; reload and try again", errs.ErrStudentVerificationAdminConfigConflict)
	case errors.Is(err, ErrAdminConfigDependency):
		response.Conflict(c, "student verification dependency is not ready for enablement", errs.ErrStudentVerificationAdminDependencyUnavailable)
	case errors.Is(err, ErrAdminConflictNotFound):
		response.NotFound(c, "student subject conflict not found", errs.ErrStudentVerificationSubjectConflictMissing)
	case errors.Is(err, ErrAdminConflictState):
		response.Conflict(c, "student subject conflict state changed; reload and try again", errs.ErrStudentVerificationSubjectConflictState)
	case errors.Is(err, ErrAdminRosterSyncConflict):
		response.Conflict(c, "a manual roster sync is already pending or running", errs.ErrStudentVerificationRosterSyncConflict)
	default:
		response.InternalError(c, "student verification request failed")
	}
}
