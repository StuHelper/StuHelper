package review

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

const (
	// maxBatchSize 批量操作的最大数量上限
	maxBatchSize = 100
)

// ListReports 获取举报列表
func (h *Handler) ListReports(c *gin.Context) {
	status := c.DefaultQuery("status", "pending")
	// 白名单校验 status 参数
	if !isValidReportStatus(status) {
		status = ReportStatusPending
	}
	page, pageSize := httputil.ParsePage(c)

	cacheKey := h.cache.BuildVersionedKey(c.Request.Context(), "review:admin:reports",
		"status="+status+":page="+strconv.Itoa(page)+":size="+strconv.Itoa(pageSize))
	if cached, ok := h.cache.GetRaw(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	result, err := h.service.ListReports(c.Request.Context(), ListReportsParams{
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		logger.FromGin(c).Error("failed to load reports", zap.Error(err))
		response.InternalError(c, "failed to load reports")
		return
	}

	data := gin.H{"list": result.List, "total": result.Total}
	if err := h.cache.Set(c.Request.Context(), cacheKey, data, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, data)
}

// ProcessReportRequest 处理举报请求
type ProcessReportRequest struct {
	Action string `json:"action" binding:"required,oneof=reject hide delete"`
	Note   string `json:"note" binding:"max=500"`
}

// ProcessReport 处理举报
func (h *Handler) ProcessReport(c *gin.Context) {
	reportID, err := httputil.ParseUUIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid report id")
		return
	}

	var req ProcessReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	username := middleware.GetUsername(c)

	err = h.service.ProcessReport(c.Request.Context(), ProcessReportParams{
		ReportID:   reportID,
		Action:     req.Action,
		Note:       req.Note,
		ResolvedBy: username,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrReportNotFound):
			response.NotFound(c, "report not found", errs.ErrReportNotFound)
		case errors.Is(err, ErrInvalidAction):
			response.BadRequest(c, "invalid action")
		default:
			logger.FromGin(c).Error("failed to process report", zap.Error(err))
			response.InternalError(c, "failed to process report")
		}
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
	h.invalidateReviewCaches(c, 0)

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

	result, err := h.service.ListAllReviews(c.Request.Context(), ListAllReviewsParams{
		Status:   status,
		Page:     page,
		PageSize: pageSize,
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
	reviewID, err := httputil.ParseUUIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid review ID")
		return
	}

	var req AdminUpdateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
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
		switch {
		case errors.Is(err, ErrReviewNotFound):
			response.NotFound(c, "review not found", errs.ErrReviewNotFound)
		case errors.Is(err, ErrInvalidAction):
			response.BadRequest(c, "invalid action")
		case errors.Is(err, ErrInvalidTransition):
			response.BadRequest(c, "invalid status transition for this action", errs.ErrInvalidTransition)
		default:
			logger.FromGin(c).Error("failed to update review", zap.Error(err))
			response.InternalError(c, "failed to update review")
		}
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

	h.invalidateReviewCaches(c, 0, "review:stats")

	response.Success(c, gin.H{"message": "review updated successfully"})
}

// GetAdminStats 获取管理统计
func (h *Handler) GetAdminStats(c *gin.Context) {
	cacheKey := h.cache.BuildVersionedKey(c.Request.Context(), "review:admin:stats", "all")
	if cached, ok := h.cache.GetRaw(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	stats, err := h.service.GetAdminStats(c.Request.Context())
	if err != nil {
		logger.FromGin(c).Error("failed to load stats", zap.Error(err))
		response.InternalError(c, "failed to load stats")
		return
	}

	if err := h.cache.Set(c.Request.Context(), cacheKey, stats, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, stats)
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

	// 纵深防御：显式校验批量上限（binding tag 已有 max=100，此处为双重保障）
	if len(req.IDs) > maxBatchSize {
		response.BadRequest(c, fmt.Sprintf("batch size %d exceeds limit of %d", len(req.IDs), maxBatchSize))
		return
	}

	// 校验所有 ID 为合法 UUID 格式
	for _, id := range req.IDs {
		if _, err := uuid.Parse(id); err != nil {
			response.BadRequest(c, fmt.Sprintf("invalid UUID: %s", id))
			return
		}
	}

	result, err := h.service.BatchUpdateReviews(c.Request.Context(), BatchUpdateReviewsParams(req))
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAction):
			response.BadRequest(c, "invalid action")
		default:
			logger.FromGin(c).Error("failed to batch update reviews", zap.Error(err))
			response.InternalError(c, "failed to batch update reviews")
		}
		return
	}

	// 记录操作日志
	h.logAdminOp(c, "batch_"+req.Action, "review", fmt.Sprintf("batch:%d_items", len(req.IDs)),
		nil,
		map[string]interface{}{"ids": req.IDs, "action": req.Action, "affected": result.Affected})

	h.invalidateReviewCaches(c, 0, "review:stats")

	response.Success(c, gin.H{
		"message":  "batch update completed",
		"affected": result.Affected,
	})
}
