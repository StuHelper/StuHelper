package admission

import (
	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

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
