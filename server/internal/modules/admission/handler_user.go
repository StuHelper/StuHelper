package admission

import (
	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

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

func (h *Handler) ready(c *gin.Context) bool {
	if h.service != nil {
		return true
	}
	notImplemented(c)
	return false
}
