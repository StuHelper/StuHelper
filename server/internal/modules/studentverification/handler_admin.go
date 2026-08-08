package studentverification

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

const adminVerificationConfigBodyLimit = 128 << 10

type updateAdminVerificationSchoolHTTPRequest struct {
	AdapterID                   string         `json:"adapterID"`
	AdapterVersion              string         `json:"adapterVersion"`
	EmailDomains                []string       `json:"emailDomains"`
	StudentIDPolicy             map[string]any `json:"studentIDPolicy"`
	NameMatchPolicy             map[string]any `json:"nameMatchPolicy"`
	EnrollmentPolicy            map[string]any `json:"enrollmentPolicy"`
	ManualFormSchema            map[string]any `json:"manualFormSchema"`
	SnapshotSyncIntervalSeconds int            `json:"snapshotSyncIntervalSeconds"`
	SnapshotWarningAfterSeconds int            `json:"snapshotWarningAfterSeconds"`
	SnapshotHardExpirySeconds   int            `json:"snapshotHardExpirySeconds"`
	SnapshotGraceSeconds        int            `json:"snapshotGraceSeconds"`
	SnapshotAutoActivate        bool           `json:"snapshotAutoActivate"`
	ExpectedRevision            int64          `json:"expectedRevision"`
	Reason                      string         `json:"reason"`
}

type createAdminVerificationSchoolHTTPRequest struct {
	SchoolCode                  string         `json:"schoolCode"`
	AdapterID                   string         `json:"adapterID"`
	AdapterVersion              string         `json:"adapterVersion"`
	EmailDomains                []string       `json:"emailDomains"`
	StudentIDPolicy             map[string]any `json:"studentIDPolicy"`
	NameMatchPolicy             map[string]any `json:"nameMatchPolicy"`
	EnrollmentPolicy            map[string]any `json:"enrollmentPolicy"`
	ManualFormSchema            map[string]any `json:"manualFormSchema"`
	SnapshotSyncIntervalSeconds int            `json:"snapshotSyncIntervalSeconds"`
	SnapshotWarningAfterSeconds int            `json:"snapshotWarningAfterSeconds"`
	SnapshotHardExpirySeconds   int            `json:"snapshotHardExpirySeconds"`
	SnapshotGraceSeconds        int            `json:"snapshotGraceSeconds"`
	SnapshotAutoActivate        bool           `json:"snapshotAutoActivate"`
	Reason                      string         `json:"reason"`
}

type updateAdminVerificationMethodHTTPRequest struct {
	DisplayName           string         `json:"displayName"`
	Description           string         `json:"description"`
	AdapterID             string         `json:"adapterID"`
	AdapterVersion        string         `json:"adapterVersion"`
	RosterDependency      string         `json:"rosterDependency"`
	ConditionalPolicy     map[string]any `json:"conditionalPolicy"`
	PublicFormSchema      map[string]any `json:"publicFormSchema"`
	RiskPolicy            map[string]any `json:"riskPolicy"`
	CredentialTTLSeconds  *int           `json:"credentialTTLSeconds"`
	ConnectorOperationKey *string        `json:"connectorOperationKey"`
	PrivacyNoticeVersion  *string        `json:"privacyNoticeVersion"`
	PrivacyNotice         map[string]any `json:"privacyNotice"`
	ExpectedRevision      int64          `json:"expectedRevision"`
	Reason                string         `json:"reason"`
}

type validateAdminVerificationConfigHTTPRequest struct {
	Enable           bool   `json:"enable"`
	ExpectedRevision int64  `json:"expectedRevision"`
	Reason           string `json:"reason"`
}

type adminCredentialRevokeHTTPRequest struct {
	Reason           string `json:"reason"`
	ExpectedRevision int64  `json:"expectedRevision"`
}

type adminSubjectConflictDecisionHTTPRequest struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

type createAdminRosterSyncHTTPRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) registerAdminControlPlaneRoutes(admin *gin.RouterGroup) {
	base := "/student-verification"
	admin.GET(
		base+"/schools",
		httputil.RouteHandlers(h.handleListAdminVerificationSchools, h.adminAuthorizers.ConfigRead)...,
	)
	admin.POST(
		base+"/schools",
		httputil.RouteHandlers(
			h.handleCreateAdminVerificationSchool,
			h.adminAuthorizers.ConfigUpdate,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
	admin.GET(
		base+"/schools/:schoolCode",
		httputil.RouteHandlers(h.handleGetAdminVerificationSchool, h.adminAuthorizers.ConfigRead)...,
	)
	admin.PUT(
		base+"/schools/:schoolCode",
		httputil.RouteHandlers(
			h.handleUpdateAdminVerificationSchool,
			h.adminAuthorizers.ConfigUpdate,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
	admin.POST(
		base+"/schools/:schoolCode/validate",
		httputil.RouteHandlers(
			h.handleValidateAdminVerificationSchool,
			h.adminAuthorizers.ConfigUpdate,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
	admin.PUT(
		base+"/schools/:schoolCode/methods/:method",
		httputil.RouteHandlers(
			h.handleUpdateAdminVerificationMethod,
			h.adminAuthorizers.ConfigUpdate,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
	admin.POST(
		base+"/schools/:schoolCode/methods/:method/validate",
		httputil.RouteHandlers(
			h.handleValidateAdminVerificationMethod,
			h.adminAuthorizers.ConfigUpdate,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
	admin.GET(
		base+"/credentials",
		httputil.RouteHandlers(h.handleListAdminStudentCredentials, h.adminAuthorizers.CredentialRead)...,
	)
	admin.POST(
		base+"/credentials/:credentialID/revoke",
		httputil.RouteHandlers(
			h.handleRevokeAdminStudentCredential,
			h.adminAuthorizers.CredentialRevoke,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
	admin.GET(
		base+"/subject-conflicts",
		httputil.RouteHandlers(h.handleListAdminStudentSubjectConflicts, h.adminAuthorizers.SubjectConflictRead)...,
	)
	admin.POST(
		base+"/subject-conflicts/:conflictID/decision",
		httputil.RouteHandlers(
			h.handleDecideAdminStudentSubjectConflict,
			h.adminAuthorizers.SubjectConflictResolve,
			h.adminAuthorizers.StepUpMFA,
		)...,
	)
	admin.GET(
		base+"/connectors",
		httputil.RouteHandlers(h.handleListAdminCampusConnectorHealth, h.adminAuthorizers.ConnectorHealthRead)...,
	)
}

func (h *Handler) handleListAdminVerificationSchools(c *gin.Context) {
	profiles, err := h.service.ListAdminVerificationSchools(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	filtered := make([]AdminVerificationSchoolConfig, 0, len(profiles))
	for _, profile := range profiles {
		if middleware.HasCapabilityInSchool(c, capability.StudentVerificationConfigRead, profile.SchoolCode) {
			filtered = append(filtered, profile)
		}
	}
	response.Success(c, filtered)
}

func (h *Handler) handleCreateAdminVerificationSchool(c *gin.Context) {
	actorUserID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	var request createAdminVerificationSchoolHTTPRequest
	if !bindManualJSON(c, &request, adminVerificationConfigBodyLimit) {
		return
	}
	schoolCode := strings.TrimSpace(request.SchoolCode)
	if !schoolCodePattern.MatchString(schoolCode) {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return
	}
	if !middleware.HasCapabilityInSchool(c, capability.StudentVerificationConfigUpdate, schoolCode) {
		response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
		return
	}
	profile, err := h.service.CreateAdminVerificationSchool(c.Request.Context(), CreateAdminVerificationSchoolConfigInput{
		SchoolCode: schoolCode, ActorUserID: actorUserID,
		AdapterID: request.AdapterID, AdapterVersion: request.AdapterVersion,
		EmailDomains: request.EmailDomains, StudentIDPolicy: request.StudentIDPolicy,
		NameMatchPolicy: request.NameMatchPolicy, EnrollmentPolicy: request.EnrollmentPolicy,
		ManualFormSchema:            request.ManualFormSchema,
		SnapshotSyncIntervalSeconds: request.SnapshotSyncIntervalSeconds,
		SnapshotWarningAfterSeconds: request.SnapshotWarningAfterSeconds,
		SnapshotHardExpirySeconds:   request.SnapshotHardExpirySeconds,
		SnapshotGraceSeconds:        request.SnapshotGraceSeconds,
		SnapshotAutoActivate:        request.SnapshotAutoActivate, Reason: request.Reason,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, profile)
}

func (h *Handler) handleGetAdminVerificationSchool(c *gin.Context) {
	schoolCode, ok := h.authorizeAdminSchool(c, capability.StudentVerificationConfigRead)
	if !ok {
		return
	}
	profile, err := h.service.GetAdminVerificationSchool(c.Request.Context(), schoolCode)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, profile)
}

func (h *Handler) handleUpdateAdminVerificationSchool(c *gin.Context) {
	schoolCode, ok := h.authorizeAdminSchool(c, capability.StudentVerificationConfigUpdate)
	if !ok {
		return
	}
	actorUserID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	var request updateAdminVerificationSchoolHTTPRequest
	if !bindManualJSON(c, &request, adminVerificationConfigBodyLimit) {
		return
	}
	profile, err := h.service.UpdateAdminVerificationSchool(c.Request.Context(), UpdateAdminVerificationSchoolConfigInput{
		SchoolCode: schoolCode, ActorUserID: actorUserID,
		AdapterID: request.AdapterID, AdapterVersion: request.AdapterVersion,
		EmailDomains: request.EmailDomains, StudentIDPolicy: request.StudentIDPolicy,
		NameMatchPolicy: request.NameMatchPolicy, EnrollmentPolicy: request.EnrollmentPolicy,
		ManualFormSchema:            request.ManualFormSchema,
		SnapshotSyncIntervalSeconds: request.SnapshotSyncIntervalSeconds,
		SnapshotWarningAfterSeconds: request.SnapshotWarningAfterSeconds,
		SnapshotHardExpirySeconds:   request.SnapshotHardExpirySeconds,
		SnapshotGraceSeconds:        request.SnapshotGraceSeconds,
		SnapshotAutoActivate:        request.SnapshotAutoActivate,
		ExpectedRevision:            request.ExpectedRevision, Reason: request.Reason,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, profile)
}

func (h *Handler) handleValidateAdminVerificationSchool(c *gin.Context) {
	schoolCode, ok := h.authorizeAdminSchool(c, capability.StudentVerificationConfigUpdate)
	if !ok {
		return
	}
	actorUserID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	var request validateAdminVerificationConfigHTTPRequest
	if !bindManualJSON(c, &request, manualJSONBodyLimit) {
		return
	}
	profile, err := h.service.ValidateAdminVerificationSchool(c.Request.Context(), ValidateAdminVerificationConfigInput{
		SchoolCode: schoolCode, ActorUserID: actorUserID, Enable: request.Enable,
		ExpectedRevision: request.ExpectedRevision, Reason: request.Reason,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, profile)
}

func (h *Handler) handleUpdateAdminVerificationMethod(c *gin.Context) {
	schoolCode, ok := h.authorizeAdminSchool(c, capability.StudentVerificationConfigUpdate)
	if !ok {
		return
	}
	method, ok := parseAdminVerificationMethod(c)
	if !ok {
		return
	}
	actorUserID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	var request updateAdminVerificationMethodHTTPRequest
	if !bindManualJSON(c, &request, adminVerificationConfigBodyLimit) {
		return
	}
	result, err := h.service.UpdateAdminVerificationMethod(c.Request.Context(), UpdateAdminVerificationMethodConfigInput{
		SchoolCode: schoolCode, Method: method, ActorUserID: actorUserID,
		DisplayName: request.DisplayName, Description: request.Description,
		AdapterID: request.AdapterID, AdapterVersion: request.AdapterVersion,
		RosterDependency:  request.RosterDependency,
		ConditionalPolicy: request.ConditionalPolicy, PublicFormSchema: request.PublicFormSchema,
		RiskPolicy: request.RiskPolicy, CredentialTTLSeconds: request.CredentialTTLSeconds,
		ConnectorOperationKey: request.ConnectorOperationKey,
		PrivacyNoticeVersion:  request.PrivacyNoticeVersion, PrivacyNotice: request.PrivacyNotice,
		ExpectedRevision: request.ExpectedRevision, Reason: request.Reason,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) handleValidateAdminVerificationMethod(c *gin.Context) {
	schoolCode, ok := h.authorizeAdminSchool(c, capability.StudentVerificationConfigUpdate)
	if !ok {
		return
	}
	method, ok := parseAdminVerificationMethod(c)
	if !ok {
		return
	}
	actorUserID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	var request validateAdminVerificationConfigHTTPRequest
	if !bindManualJSON(c, &request, manualJSONBodyLimit) {
		return
	}
	result, err := h.service.ValidateAdminVerificationMethod(c.Request.Context(), ValidateAdminVerificationConfigInput{
		SchoolCode: schoolCode, Method: &method, ActorUserID: actorUserID,
		Enable: request.Enable, ExpectedRevision: request.ExpectedRevision, Reason: request.Reason,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) handleListAdminStudentCredentials(c *gin.Context) {
	schoolCode, ok := h.authorizeAdminQuerySchool(c, capability.StudentCredentialRead)
	if !ok {
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
	credentials, err := h.service.ListAdminStudentCredentials(
		c.Request.Context(), schoolCode, CredentialStatus(strings.TrimSpace(c.Query("status"))), limit, offset,
	)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, credentials)
}

func (h *Handler) handleRevokeAdminStudentCredential(c *gin.Context) {
	credentialID := strings.TrimSpace(c.Param("credentialID"))
	if credentialID == "" || len(credentialID) > 64 {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return
	}
	credential, err := h.service.GetAdminStudentCredential(c.Request.Context(), credentialID)
	if err != nil {
		respondError(c, err)
		return
	}
	if !middleware.HasCapabilityInSchool(c, capability.StudentCredentialRevoke, credential.SchoolCode) {
		response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
		return
	}
	actorUserID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	var request adminCredentialRevokeHTTPRequest
	if !bindManualJSON(c, &request, manualJSONBodyLimit) {
		return
	}
	credential, err = h.service.RevokeAdminStudentCredential(c.Request.Context(), AdminCredentialRevokeInput{
		CredentialID: credentialID, ActorUserID: actorUserID,
		ExpectedRevision: request.ExpectedRevision, Reason: request.Reason,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, credential)
}

func (h *Handler) handleListAdminStudentSubjectConflicts(c *gin.Context) {
	schoolCode, ok := h.authorizeAdminQuerySchool(c, capability.StudentSubjectConflictRead)
	if !ok {
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
	conflicts, err := h.service.ListAdminStudentSubjectConflicts(
		c.Request.Context(), schoolCode, c.Query("status"), limit, offset,
	)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, conflicts)
}

func (h *Handler) handleDecideAdminStudentSubjectConflict(c *gin.Context) {
	conflictID, ok := parseUUIDParam(c, "conflictID")
	if !ok {
		return
	}
	conflict, err := h.service.GetAdminStudentSubjectConflict(c.Request.Context(), conflictID)
	if err != nil {
		respondError(c, err)
		return
	}
	if !middleware.HasCapabilityInSchool(c, capability.StudentSubjectConflictResolve, conflict.SchoolCode) {
		response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
		return
	}
	actorUserID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	var request adminSubjectConflictDecisionHTTPRequest
	if !bindManualJSON(c, &request, manualJSONBodyLimit) {
		return
	}
	conflict, err = h.service.DecideAdminStudentSubjectConflict(c.Request.Context(), AdminSubjectConflictDecisionInput{
		ConflictID: conflictID, ActorUserID: actorUserID, Action: request.Action, Reason: request.Reason,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, conflict)
}

func (h *Handler) handleListAdminCampusConnectorHealth(c *gin.Context) {
	schoolCode := strings.TrimSpace(c.Query("schoolCode"))
	if schoolCode != "" && !schoolCodePattern.MatchString(schoolCode) {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return
	}
	if schoolCode != "" && !middleware.HasCapabilityInSchool(c, capability.CampusConnectorHealthRead, schoolCode) {
		response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
		return
	}
	nodes, err := h.service.ListAdminCampusConnectorHealth(c.Request.Context(), schoolCode)
	if err != nil {
		respondError(c, err)
		return
	}
	if schoolCode == "" && !middleware.HasGlobalCapability(c, capability.CampusConnectorHealthRead) {
		filtered := make([]AdminCampusConnectorHealth, 0, len(nodes))
		for _, node := range nodes {
			operations := make([]AdminCampusConnectorOperationHealth, 0, len(node.Operations))
			for _, operation := range node.Operations {
				if middleware.HasCapabilityInSchool(c, capability.CampusConnectorHealthRead, operation.SchoolCode) {
					operations = append(operations, operation)
				}
			}
			if len(operations) > 0 {
				node.Operations = operations
				filtered = append(filtered, node)
			}
		}
		nodes = filtered
	}
	response.Success(c, nodes)
}

func (h *Handler) handleListAdminRosterSyncRequests(c *gin.Context) {
	schoolCode, ok := h.authorizeAdminSchool(c, capability.CampusConnectorManage)
	if !ok {
		return
	}
	if h.rosterSyncCoordinator == nil {
		respondError(c, ErrDependencyUnavailable)
		return
	}
	limit, ok := parseStrictBoundedQueryInteger(c, "limit", 20, 1, 50)
	if !ok {
		return
	}
	requests, err := h.rosterSyncCoordinator.ListRosterSyncRequests(
		c.Request.Context(), schoolCode, limit,
	)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, requests)
}

func (h *Handler) handleCreateAdminRosterSyncRequest(c *gin.Context) {
	schoolCode, ok := h.authorizeAdminSchool(c, capability.CampusConnectorManage)
	if !ok {
		return
	}
	actorUserID, ok := h.resolveCurrentUser(c)
	if !ok {
		return
	}
	if h.rosterSyncCoordinator == nil {
		respondError(c, ErrDependencyUnavailable)
		return
	}
	var request createAdminRosterSyncHTTPRequest
	if !bindManualJSON(c, &request, manualJSONBodyLimit) {
		return
	}
	result, err := h.rosterSyncCoordinator.RequestRosterSync(
		c.Request.Context(),
		AdminRosterSyncInput{
			SchoolCode: schoolCode, ActorUserID: actorUserID, Reason: request.Reason,
		},
	)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Accepted(c, result)
}

func (h *Handler) authorizeAdminSchool(c *gin.Context, capabilityName string) (string, bool) {
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

func (h *Handler) authorizeAdminQuerySchool(c *gin.Context, capabilityName string) (string, bool) {
	schoolCode := strings.TrimSpace(c.Query("schoolCode"))
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

func parseAdminVerificationMethod(c *gin.Context) (Method, bool) {
	method := Method(strings.TrimSpace(c.Param("method")))
	if !validVerificationMethod(method) {
		response.BadRequest(c, "invalid request parameters", errs.ErrInvalidParam)
		return "", false
	}
	return method, true
}
