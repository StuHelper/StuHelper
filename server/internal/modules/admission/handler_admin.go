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
