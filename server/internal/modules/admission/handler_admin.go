package admission

import (
	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

type adminFreshmanReviewHTTPRequest struct {
	Action        FreshmanReviewAction `json:"action" binding:"required"`
	Reason        *string              `json:"reason"`
	ExpiresInDays *int                 `json:"expiresInDays"`
}

func (h *Handler) handleAdminReviewFreshmanVerification(c *gin.Context) {
	userID, ok := middleware.ResolveRequiredInternalUserID(
		c,
		h.internalUserIDResolver,
		"failed to resolve admission reviewer",
	)
	if !ok {
		return
	}
	var req adminFreshmanReviewHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	app, err := h.service.ReviewFreshmanApplicationFromAdmin(
		c.Request.Context(),
		adminFreshmanReviewInput(c.Param("id"), userID, req),
	)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, app)
}

func (h *Handler) handleAdminResendAdmissionSession(c *gin.Context) {
	input, ok := h.adminAdmissionSessionActionInput(c)
	if !ok {
		return
	}
	session, err := h.service.ResendAdminAdmissionSession(c.Request.Context(), input)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, session)
}

func (h *Handler) handleAdminRegenerateAdmissionSession(c *gin.Context) {
	input, ok := h.adminAdmissionSessionActionInput(c)
	if !ok {
		return
	}
	created, err := h.service.RegenerateAdminAdmissionSession(c.Request.Context(), input)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Created(c, created)
}

func (h *Handler) handleAdminCancelAdmissionSession(c *gin.Context) {
	input, ok := h.adminAdmissionSessionActionInput(c)
	if !ok {
		return
	}
	session, err := h.service.CancelAdminAdmissionSession(c.Request.Context(), input)
	if err != nil {
		respondAdmissionError(c, err)
		return
	}
	response.Success(c, session)
}

func (h *Handler) adminAdmissionSessionActionInput(c *gin.Context) (AdminAdmissionSessionActionInput, bool) {
	userID, ok := middleware.ResolveRequiredInternalUserID(
		c,
		h.internalUserIDResolver,
		"failed to resolve admission session operator",
	)
	if !ok {
		return AdminAdmissionSessionActionInput{}, false
	}
	return AdminAdmissionSessionActionInput{
		SessionID:      c.Param("id"),
		OperatorUserID: userID,
	}, true
}

func adminFreshmanReviewInput(
	applicationID string,
	userID int64,
	req adminFreshmanReviewHTTPRequest,
) AdminFreshmanReviewInput {
	return AdminFreshmanReviewInput{
		ApplicationID:  applicationID,
		Action:         req.Action,
		Reason:         req.Reason,
		ExpiresInDays:  req.ExpiresInDays,
		OperatorUserID: userID,
	}
}
