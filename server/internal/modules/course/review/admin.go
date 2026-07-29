package review

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

const (
	// maxBatchSize 批量操作的最大数量上限
	maxBatchSize = 100

	batchStepUpThreshold = 5
)

// ListReports 获取举报列表
func (h *Handler) ListReports(c *gin.Context) {
	status := c.DefaultQuery("status", "pending")
	// 白名单校验 status 参数
	if !isValidReportStatus(status) {
		status = ReportStatusPending
	}
	page, pageSize := httputil.ParsePage(c)
	scope := resolveModerationScope(c)
	schoolIDs := scope.schoolIDs()

	respondWithCachedData(h, c, "review:admin:reports", adminReportsCacheKey(status, page, pageSize, schoolIDs), func(ctx context.Context) (*ListReportsResult, error) {
		return h.service.ListReports(ctx, ListReportsParams{
			Status:    status,
			Page:      page,
			PageSize:  pageSize,
			SchoolIDs: schoolIDs,
		})
	}, func(result *ListReportsResult) any {
		return gin.H{"list": result.List, "total": result.Total}
	}, "failed to load reports", "failed to load reports", nil)
}

// ProcessReportRequest 处理举报请求
type ProcessReportRequest struct {
	Action string `json:"action" binding:"required,oneof=reject hide delete"`
	Note   string `json:"note" binding:"max=500"`
}

// ProcessReport 处理举报
func (h *Handler) ProcessReport(c *gin.Context) {
	reportID, err := httputil.ParseUUIDParam(c, "reportID")
	if err != nil {
		response.BadRequest(c, "invalid report id")
		return
	}

	var req ProcessReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	if !h.authorizeReportModeration(c, reportID) {
		return
	}

	err = h.service.ProcessReport(c.Request.Context(), ProcessReportParams{
		ReportID:   reportID,
		Action:     req.Action,
		Note:       req.Note,
		ResolvedBy: middleware.GetUserID(c),
	})
	if err != nil {
		if respondProcessReportError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to process report", zap.Error(err))
		response.InternalError(c, "failed to process report")
		return
	}

	ctx := c.Request.Context()

	// 记录操作日志
	h.logAdminOp(c, "process_report_"+req.Action, "report", reportID,
		map[string]string{"status": "pending"},
		map[string]string{"action": req.Action, "note": req.Note})

	if err := h.cache.InvalidateByVersion(ctx, "review:admin:reports"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}
	h.invalidateReviewAggregateCaches(c)
	if req.Action == "hide" || req.Action == "delete" {
		h.invalidateCourseReviewCountCaches(c)
		h.refreshTeacherPublicStatsAndInvalidateCaches(c)
	}

	response.Success(c, gin.H{"message": "report processed successfully"})
}

// ListAllReviews 获取所有评论（管理员）
func (h *Handler) ListAllReviews(c *gin.Context) {
	status := c.DefaultQuery("status", "all")
	// 白名单校验 status 参数
	if !isValidReviewStatus(status) {
		status = StatusAll
	}
	page, pageSize := httputil.ParsePage(c)
	scope := resolveModerationScope(c)

	result, err := h.service.ListAllReviews(c.Request.Context(), ListAllReviewsParams{
		Status:    status,
		Page:      page,
		PageSize:  pageSize,
		SchoolIDs: scope.schoolIDs(),
	})
	if err != nil {
		logger.FromGin(c).Error("failed to load reviews", zap.Error(err))
		response.InternalError(c, "failed to load reviews")
		return
	}

	response.Success(c, gin.H{"list": result.List, "total": result.Total})
}

// AdminUpdateReviewRequest 管理员更新评论请求
type AdminUpdateReviewRequest struct {
	Action string `json:"action" binding:"required,oneof=hide restore delete"`
	Reason string `json:"reason" binding:"max=500"`
}

// AdminUpdateReview 管理员更新评论
func (h *Handler) AdminUpdateReview(c *gin.Context) {
	reviewID, err := httputil.ParseUUIDParam(c, "reviewID")
	if err != nil {
		response.BadRequest(c, "invalid review ID")
		return
	}

	var req AdminUpdateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	if !h.authorizeReviewModeration(c, reviewID, req.Action) {
		return
	}

	userID := middleware.GetUserID(c)
	result, err := h.service.AdminUpdateReview(c.Request.Context(), AdminUpdateReviewParams{
		ReviewID: reviewID,
		Action:   req.Action,
		Reason:   req.Reason,
		AdminID:  userID,
	})
	if err != nil {
		if respondAdminUpdateReviewError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to update review", zap.Error(err))
		response.InternalError(c, "failed to update review")
		return
	}

	// 记录操作日志
	newValue := map[string]string{"action": req.Action}
	if req.Reason != "" {
		newValue["reason"] = req.Reason
	}
	h.logAdminOp(c, req.Action, "review", reviewID,
		map[string]string{"status": result.OldStatus},
		newValue)

	// 记录审计事件（restore 操作）
	if req.Action == "restore" {
		audit.LogContext(c.Request.Context(), audit.Event{
			Type:         audit.EventAdminReviewRestore,
			ActorType:    "admin",
			UserID:       userID,
			Username:     middleware.GetUsername(c),
			IP:           c.ClientIP(),
			Resource:     "review",
			ResourceType: "review",
			ResourceID:   reviewID,
			Action:       "restore",
			Result:       "success",
			Details: map[string]any{
				"review_id":  reviewID,
				"old_status": result.OldStatus,
			},
		})
	}

	h.invalidateReviewAggregateCaches(c)
	h.invalidateCourseReviewCountCaches(c)
	h.refreshTeacherPublicStatsAndInvalidateCaches(c)

	response.Success(c, gin.H{"message": "review updated successfully"})
}

// GetAdminStats 获取管理统计
func (h *Handler) GetAdminStats(c *gin.Context) {
	respondWithCachedData(h, c, "review:admin:stats", "all", h.service.GetAdminStats, nil, "failed to load stats", "failed to load stats", nil)
}

// BatchUpdateReviewsRequest 批量更新评论请求
type BatchUpdateReviewsRequest struct {
	IDs    []string `json:"ids" binding:"required,min=1,max=100"`
	Action string   `json:"action" binding:"required,oneof=hide restore delete"`
}

// BatchUpdateReviews 批量更新评论状态
func (h *Handler) BatchUpdateReviews(c *gin.Context) {
	var req BatchUpdateReviewsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	// 校验所有 ID 为合法 UUID 格式
	for _, id := range req.IDs {
		if _, err := uuid.Parse(id); err != nil {
			response.BadRequest(c, fmt.Sprintf("invalid UUID: %s", id))
			return
		}
	}

	userID := middleware.GetUserID(c)
	if batchReviewStepUpRequired(req) && !h.ensureBatchStepUp(c) {
		return
	}
	if !h.authorizeBatchReviewModeration(c, req.IDs, req.Action) {
		return
	}

	result, err := h.service.BatchUpdateReviews(c.Request.Context(), BatchUpdateReviewsParams{
		IDs:     req.IDs,
		Action:  req.Action,
		AdminID: userID,
	})
	if err != nil {
		if response.RespondMappedErrorGroups(c, err, reviewAdminIdentityErrorMappings, reviewAdminActionErrorMappings) {
			return
		}
		logger.FromGin(c).Error("failed to batch update reviews", zap.Error(err))
		response.InternalError(c, "failed to batch update reviews")
		return
	}

	// 记录操作日志
	h.logAdminOp(c, "batch_"+req.Action, "review", fmt.Sprintf("batch:%d_items", len(req.IDs)),
		nil,
		map[string]interface{}{"ids": req.IDs, "action": req.Action, "affected": result.Affected})

	h.invalidateReviewAggregateCaches(c)
	h.invalidateCourseReviewCountCaches(c)
	h.refreshTeacherPublicStatsAndInvalidateCaches(c)

	response.Success(c, gin.H{
		"message":  "batch update completed",
		"affected": result.Affected,
	})
}

func (h *Handler) ensureBatchStepUp(c *gin.Context) bool {
	if h.adminAuthorizers.StepUpVerified == nil {
		return true
	}
	return h.adminAuthorizers.StepUpVerified(c)
}

func batchReviewStepUpRequired(req BatchUpdateReviewsRequest) bool {
	return len(req.IDs) >= batchStepUpThreshold && (req.Action == "hide" || req.Action == "delete")
}
