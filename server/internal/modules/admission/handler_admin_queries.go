package admission

import (
	"strings"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

func (h *Handler) handleAdminListAdmissionPolicies(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	items, err := h.service.ListAdmissionPolicies(c.Request.Context())
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) handleAdminUpdateAdmissionPolicy(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	var policy AdmissionPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	policy.ID = c.Param("id")
	updated, err := h.service.UpdateAdmissionPolicy(c.Request.Context(), policy)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, updated)
}

func (h *Handler) handleAdminListAdmissionSessions(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	filter, ok := admissionSessionListFilterFromQuery(c)
	if !ok {
		return
	}
	items, total, err := h.service.ListAdmissionSessions(c.Request.Context(), filter)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, gin.H{"list": items, "total": total})
}

func (h *Handler) handleAdminListFreshmanVerifications(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	filter, ok := freshmanApplicationListFilterFromQuery(c)
	if !ok {
		return
	}
	items, total, err := h.service.ListFreshmanApplications(c.Request.Context(), filter)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, gin.H{"list": items, "total": total})
}

func (h *Handler) handleAdminGetFreshmanVerification(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	app, err := h.service.GetFreshmanApplication(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, app)
}

func admissionSessionListFilterFromQuery(c *gin.Context) (AdmissionSessionListFilter, bool) {
	page, pageSize := httputil.ParsePage(c)
	status, ok := parseAdmissionSessionStatus(c.Query("status"))
	if !ok {
		response.BadRequest(c, "invalid admission session status")
		return AdmissionSessionListFilter{}, false
	}
	return AdmissionSessionListFilter{
		Status:   status,
		PageSize: pageSize,
		Offset:   httputil.SafeOffset(page, pageSize),
	}, true
}

func freshmanApplicationListFilterFromQuery(c *gin.Context) (FreshmanApplicationListFilter, bool) {
	page, pageSize := httputil.ParsePage(c)
	status, ok := parseFreshmanApplicationStatus(c.Query("status"))
	if !ok {
		response.BadRequest(c, "invalid freshman application status")
		return FreshmanApplicationListFilter{}, false
	}
	return FreshmanApplicationListFilter{
		Status:   status,
		PageSize: pageSize,
		Offset:   httputil.SafeOffset(page, pageSize),
	}, true
}

func parseAdmissionSessionStatus(raw string) (AdmissionSessionStatus, bool) {
	status := AdmissionSessionStatus(strings.TrimSpace(raw))
	if status == "" {
		return "", true
	}
	switch status {
	case StatusJoinedMuted, StatusLinked, StatusMaterialSubmitted, StatusVerified, StatusExpiredKicked, StatusCancelled:
		return status, true
	default:
		return "", false
	}
}

func parseFreshmanApplicationStatus(raw string) (FreshmanApplicationStatus, bool) {
	status := FreshmanApplicationStatus(strings.TrimSpace(raw))
	if status == "" {
		return "", true
	}
	switch status {
	case FreshmanApplicationPending, FreshmanApplicationApproved, FreshmanApplicationRejected:
		return status, true
	default:
		return "", false
	}
}
