package review

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

const (
	reviewRelationCanHide        = "can_hide"
	reviewRelationCanRestore     = "can_restore"
	reviewRelationCanAdminDelete = "can_admin_delete"
	reviewRelationCanAdminEdit   = "can_admin_edit"
	reportRelationCanProcess     = "can_process"
)

func reviewAdminRelationForAction(action string) (string, bool) {
	switch action {
	case "hide":
		return reviewRelationCanHide, true
	case "restore":
		return reviewRelationCanRestore, true
	case "delete":
		return reviewRelationCanAdminDelete, true
	default:
		return "", false
	}
}

func (h *Handler) authorizeReviewModeration(c *gin.Context, reviewID, action string) bool {
	relation, ok := h.reviewModerationRelationForAction(c, reviewID, action)
	if !ok {
		return false
	}
	return h.authorizeReviewRelation(c, reviewID, relation, "insufficient permission for this review")
}

func (h *Handler) authorizeReviewContentEdit(c *gin.Context, reviewID string) bool {
	return h.authorizeReviewRelation(c, reviewID, reviewRelationCanAdminEdit, "insufficient permission for this review")
}

func (h *Handler) authorizeReviewContentFlagClear(c *gin.Context, reviewID string) bool {
	return h.authorizeReviewRelation(c, reviewID, reviewRelationCanRestore, "insufficient permission for this review")
}

func (h *Handler) authorizeBatchReviewModeration(c *gin.Context, reviewIDs []string, action string) bool {
	if action == "restore" {
		return h.authorizeBatchRestoreModeration(c, reviewIDs)
	}
	relation, ok := reviewAdminRelationForAction(action)
	if !ok {
		response.BadRequest(c, "invalid review action")
		return false
	}
	if !h.ensureBatchReviewsExist(c, reviewIDs) {
		return false
	}
	return h.authorizeObjectsRelation(c, "review", reviewIDs, relation, "insufficient permission for one or more reviews")
}

func (h *Handler) reviewModerationRelationForAction(c *gin.Context, reviewID, action string) (string, bool) {
	if action != "restore" {
		relation, ok := reviewAdminRelationForAction(action)
		if !ok {
			response.BadRequest(c, "invalid review action")
		}
		return relation, ok
	}
	status, ok := h.resolveReviewStatus(c, reviewID, "review not found", "failed to resolve review status")
	if !ok {
		return "", false
	}
	return reviewRestoreRelationForStatus(status), true
}

func reviewRestoreRelationForStatus(status string) string {
	if status == StatusPendingReview {
		return reviewRelationCanHide
	}
	return reviewRelationCanRestore
}

func (h *Handler) authorizeBatchRestoreModeration(c *gin.Context, reviewIDs []string) bool {
	statuses, ok := h.resolveBatchReviewStatuses(c, reviewIDs)
	if !ok {
		return false
	}
	user, ok := h.resolveAuthorizationUser(c)
	if !ok {
		return false
	}
	for _, reviewID := range reviewIDs {
		relation := reviewRestoreRelationForStatus(statuses[reviewID])
		object := "review:" + reviewID
		if !h.checkAuthorizationRelation(c, user, relation, object, "insufficient permission for one or more reviews") {
			return false
		}
	}
	return true
}

func (h *Handler) authorizeReportModeration(c *gin.Context, reportID string) bool {
	if !h.ensureReportExists(c, reportID) {
		return false
	}
	return h.authorizeObjectsRelation(c, "report", []string{reportID}, reportRelationCanProcess, "insufficient permission for this report")
}

func (h *Handler) authorizeReviewRelation(c *gin.Context, reviewID, relation, forbiddenMessage string) bool {
	if !h.ensureReviewExists(c, reviewID) {
		return false
	}
	return h.authorizeObjectsRelation(c, "review", []string{reviewID}, relation, forbiddenMessage)
}

func (h *Handler) ensureReviewExists(c *gin.Context, reviewID string) bool {
	if h.service == nil {
		response.Forbidden(c, "insufficient permission for this review", errs.ErrAccessDenied)
		return false
	}
	_, err := h.service.GetReviewSchoolID(c.Request.Context(), reviewID)
	return respondModerationLookupError(c, err, "review not found", "failed to resolve review scope")
}

func (h *Handler) ensureBatchReviewsExist(c *gin.Context, reviewIDs []string) bool {
	if h.service == nil {
		response.Forbidden(c, "insufficient permission for one or more reviews", errs.ErrAccessDenied)
		return false
	}
	schoolIDs, err := h.service.ListReviewSchoolIDs(c.Request.Context(), reviewIDs)
	if !respondModerationLookupError(c, err, "review not found", "failed to resolve review scope") {
		return false
	}
	for _, reviewID := range reviewIDs {
		if _, ok := schoolIDs[reviewID]; !ok {
			response.NotFound(c, "review not found")
			return false
		}
	}
	return true
}

func (h *Handler) resolveReviewStatus(c *gin.Context, reviewID, notFoundMessage, internalMessage string) (string, bool) {
	if h.service == nil {
		response.Forbidden(c, "insufficient permission for this review", errs.ErrAccessDenied)
		return "", false
	}
	status, err := h.service.GetReviewStatus(c.Request.Context(), reviewID)
	return status, respondModerationLookupError(c, err, notFoundMessage, internalMessage)
}

func (h *Handler) resolveBatchReviewStatuses(c *gin.Context, reviewIDs []string) (map[string]string, bool) {
	if h.service == nil {
		response.Forbidden(c, "insufficient permission for one or more reviews", errs.ErrAccessDenied)
		return nil, false
	}
	statuses, err := h.service.ListReviewStatuses(c.Request.Context(), reviewIDs)
	if !respondModerationLookupError(c, err, "review not found", "failed to resolve review status") {
		return nil, false
	}
	for _, reviewID := range reviewIDs {
		if _, ok := statuses[reviewID]; !ok {
			response.NotFound(c, "review not found")
			return nil, false
		}
	}
	return statuses, true
}

func (h *Handler) ensureReportExists(c *gin.Context, reportID string) bool {
	if h.service == nil {
		response.Forbidden(c, "insufficient permission for this report", errs.ErrAccessDenied)
		return false
	}
	_, err := h.service.GetReportSchoolID(c.Request.Context(), reportID)
	return respondModerationLookupError(c, err, "report not found", "failed to resolve report scope")
}

func (h *Handler) authorizeObjectsRelation(c *gin.Context, objectType string, objectIDs []string, relation, forbiddenMessage string) bool {
	user, ok := h.resolveAuthorizationUser(c)
	if !ok {
		return false
	}
	for _, objectID := range objectIDs {
		if !h.checkAuthorizationRelation(c, user, relation, objectType+":"+objectID, forbiddenMessage) {
			return false
		}
	}
	return true
}

func (h *Handler) resolveAuthorizationUser(c *gin.Context) (string, bool) {
	if h.internalUserIDResolver == nil {
		logger.FromGin(c).Error("review admin authorization user resolver is not configured")
		response.Forbidden(c, "insufficient permissions", errs.ErrAccessDenied)
		return "", false
	}
	userID, ok := middleware.ResolveRequiredInternalUserID(c, h.internalUserIDResolver, "failed to resolve admin identity")
	if !ok {
		return "", false
	}
	return "user:" + strconv.FormatInt(userID, 10), true
}

func (h *Handler) checkAuthorizationRelation(c *gin.Context, user, relation, object, forbiddenMessage string) bool {
	if h.fga == nil {
		response.Forbidden(c, forbiddenMessage, errs.ErrAccessDenied)
		return false
	}
	allowed, err := h.fga.Check(c.Request.Context(), user, relation, object)
	if err != nil {
		return h.respondAuthorizationCheckError(c, err, relation, object)
	}
	if !allowed {
		response.Forbidden(c, forbiddenMessage, errs.ErrAccessDenied)
		return false
	}
	return true
}

func (h *Handler) respondAuthorizationCheckError(c *gin.Context, err error, relation, object string) bool {
	if errors.Is(err, errAuthorizationProviderNotConfigured) {
		response.Forbidden(c, "insufficient permissions", errs.ErrAccessDenied)
		return false
	}
	logger.FromGin(c).Error("review admin authorization check failed",
		zap.String("relation", relation),
		zap.String("object", object),
		zap.Error(err),
	)
	response.ServiceUnavailable(c, "authorization service temporarily unavailable", errs.ErrServiceUnavailable)
	return false
}

func respondModerationLookupError(c *gin.Context, err error, notFoundMessage, internalMessage string) bool {
	switch {
	case err == nil:
		return true
	case errors.Is(err, ErrReviewNotFound), errors.Is(err, ErrReportNotFound), errors.Is(err, pgx.ErrNoRows):
		response.NotFound(c, notFoundMessage)
	default:
		response.InternalError(c, internalMessage)
	}
	return false
}
