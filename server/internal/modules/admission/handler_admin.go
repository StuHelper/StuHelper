package admission

import (
	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

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
