package review

import (
	"encoding/csv"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// ListReports 获取举报列表
func (h *Handler) ListReports(c *gin.Context) {
	status := c.DefaultQuery("status", "pending")
	page, pageSize := httputil.ParsePage(c)

	cacheKey := h.cache.BuildVersionedKey(c.Request.Context(), "review:admin:reports",
		"status="+status+":page="+strconv.Itoa(page)+":size="+strconv.Itoa(pageSize))
	if cached, ok := h.cache.Get(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	result, err := h.service.ListReports(c.Request.Context(), ListReportsParams{
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.InternalError(c, "failed to load reports")
		return
	}

	data := gin.H{"list": result.List, "total": result.Total}
	if err := h.cache.Set(c.Request.Context(), cacheKey, data, cache.DefaultTTL); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, data)
}

// ProcessReportRequest 处理举报请求
type ProcessReportRequest struct {
	Action string `json:"action" binding:"required,oneof=reject hide_review delete_review"`
	Note   string `json:"note" binding:"max=500"`
}

// ProcessReport 处理举报
func (h *Handler) ProcessReport(c *gin.Context) {
	reportID, err := httputil.ParseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid report id")
		return
	}

	var req ProcessReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
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
			response.NotFound(c, "report not found")
		case errors.Is(err, ErrInvalidAction):
			response.BadRequest(c, "invalid action")
		default:
			response.InternalError(c, "failed to process report")
		}
		return
	}

	ctx := c.Request.Context()

	// 记录操作日志
	if err := h.service.LogOperation(ctx, LogOperationParams{
		AdminUserID:   userID,
		AdminUsername: username,
		Action:        "process_report_" + req.Action,
		ResourceType:  "report",
		ResourceID:    strconv.FormatInt(reportID, 10),
		OldValue:      map[string]string{"status": "pending"},
		NewValue:      map[string]string{"action": req.Action, "note": req.Note},
		IPAddress:     c.ClientIP(),
		UserAgent:     c.Request.UserAgent(),
	}); err != nil {
		logger.FromGin(c).Warn("failed to log operation", zap.Error(err))
	}

	if err := h.cache.InvalidateByVersion(ctx, "review:admin:reports"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}
	if err := h.cache.InvalidateByVersion(ctx, "review:course"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}
	if err := h.cache.InvalidateByVersion(ctx, "review:latest"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}

	response.Success(c, gin.H{"message": "report processed successfully"})
}

// ListAllReviews 获取所有评论（管理员）
func (h *Handler) ListAllReviews(c *gin.Context) {
	status := c.DefaultQuery("status", "all")
	page, pageSize := httputil.ParsePage(c)

	result, err := h.service.ListAllReviews(c.Request.Context(), ListAllReviewsParams{
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.InternalError(c, "failed to load reviews")
		return
	}

	response.Success(c, gin.H{"list": result.List, "total": result.Total})
}

// AdminUpdateReviewRequest 管理员更新评论请求
type AdminUpdateReviewRequest struct {
	Action string `json:"action" binding:"required,oneof=hide restore delete"`
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
		response.BadRequest(c, err.Error())
		return
	}

	// 获取旧状态用于日志记录
	oldReview, err := h.service.GetReviewByID(c.Request.Context(), reviewID)
	if err != nil {
		logger.FromGin(c).Warn("failed to get review for audit log", zap.Error(err))
	}
	var oldStatus string
	if oldReview != nil {
		oldStatus = oldReview.Status
	}

	err = h.service.AdminUpdateReview(c.Request.Context(), AdminUpdateReviewParams{
		ReviewID: reviewID,
		Action:   req.Action,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrReviewNotFound):
			response.NotFound(c, "review not found")
		case errors.Is(err, ErrInvalidAction):
			response.BadRequest(c, "invalid action")
		default:
			response.InternalError(c, "failed to update review")
		}
		return
	}

	ctx := c.Request.Context()

	// 记录操作日志
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)
	if err := h.service.LogOperation(ctx, LogOperationParams{
		AdminUserID:   userID,
		AdminUsername: username,
		Action:        req.Action,
		ResourceType:  "review",
		ResourceID:    reviewID,
		OldValue:      map[string]string{"status": oldStatus},
		NewValue:      map[string]string{"action": req.Action},
		IPAddress:     c.ClientIP(),
		UserAgent:     c.Request.UserAgent(),
	}); err != nil {
		logger.FromGin(c).Warn("failed to log operation", zap.Error(err))
	}

	if err := h.cache.InvalidateByVersion(ctx, "review:course"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}
	if err := h.cache.InvalidateByVersion(ctx, "review:latest"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}
	if err := h.cache.InvalidateByVersion(ctx, "review:stats"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}

	response.Success(c, gin.H{"message": "review updated successfully"})
}

// GetAdminStats 获取管理统计
func (h *Handler) GetAdminStats(c *gin.Context) {
	cacheKey := h.cache.BuildVersionedKey(c.Request.Context(), "review:admin:stats", "all")
	if cached, ok := h.cache.Get(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	stats, err := h.service.GetAdminStats(c.Request.Context())
	if err != nil {
		response.InternalError(c, "failed to load stats")
		return
	}

	if err := h.cache.Set(c.Request.Context(), cacheKey, stats, cache.DefaultTTL); err != nil {
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
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.service.BatchUpdateReviews(c.Request.Context(), BatchUpdateReviewsParams{
		IDs:    req.IDs,
		Action: req.Action,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAction):
			response.BadRequest(c, "invalid action")
		default:
			response.InternalError(c, "failed to batch update reviews")
		}
		return
	}

	ctx := c.Request.Context()

	// 记录操作日志
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)
	if err := h.service.LogOperation(ctx, LogOperationParams{
		AdminUserID:   userID,
		AdminUsername: username,
		Action:        "batch_" + req.Action,
		ResourceType:  "review",
		ResourceID:    strings.Join(req.IDs, ","),
		OldValue:      nil,
		NewValue:      map[string]interface{}{"ids": req.IDs, "action": req.Action, "affected": result.Affected},
		IPAddress:     c.ClientIP(),
		UserAgent:     c.Request.UserAgent(),
	}); err != nil {
		logger.FromGin(c).Warn("failed to log operation", zap.Error(err))
	}

	if err := h.cache.InvalidateByVersion(ctx, "review:course"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}
	if err := h.cache.InvalidateByVersion(ctx, "review:latest"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}
	if err := h.cache.InvalidateByVersion(ctx, "review:stats"); err != nil {
		logger.FromGin(c).Warn("failed to invalidate cache", zap.Error(err))
	}

	response.Success(c, gin.H{
		"message":  "batch update completed",
		"affected": result.Affected,
	})
}

// GetOperationLogs 获取操作日志
func (h *Handler) GetOperationLogs(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)

	result, err := h.service.GetOperationLogs(c.Request.Context(), GetOperationLogsParams{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.InternalError(c, "failed to load operation logs")
		return
	}

	response.Success(c, gin.H{"list": result.List, "total": result.Total})
}

// ExportReviews 导出评论数据
func (h *Handler) ExportReviews(c *gin.Context) {
	format := c.DefaultQuery("format", "json")
	status := c.DefaultQuery("status", "all")

	// 验证 format 参数
	if format != "json" && format != "csv" {
		format = "json"
	}
	// 验证 status 参数
	if status != "all" && status != "published" && status != "hidden" && status != "deleted" {
		status = "all"
	}

	// CSV 使用流式导出，避免全量加载到内存
	if format == "csv" {
		h.exportCSVStream(c, status)
		return
	}

	reviews, err := h.service.ExportReviews(c.Request.Context(), ExportReviewsParams{
		Format: format,
		Status: status,
	})
	if err != nil {
		response.InternalError(c, "failed to export reviews")
		return
	}

	response.Success(c, gin.H{"data": reviews, "count": len(reviews)})
}

// exportCSVStream 流式导出 CSV，逐行从数据库读取并写入响应，避免全量加载到内存
func (h *Handler) exportCSVStream(c *gin.Context, status string) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=reviews.csv")

	// 写入BOM以支持Excel中文
	if _, err := c.Writer.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return
	}

	w := csv.NewWriter(c.Writer)
	defer w.Flush()

	// 写入CSV头
	if err := w.Write([]string{
		"ID", "课程ID", "课程名称", "教师ID", "教师名称",
		"学期", "标题", "内容", "成绩", "评分",
		"点赞数", "踩数", "状态", "创建时间",
	}); err != nil {
		return
	}

	// 流式写入数据行（headers 已发送，无法返回 HTTP 错误，仅记录日志）
	if err := h.service.StreamExportReviews(c.Request.Context(), status, func(r Review) error {
		teacherID := ""
		if r.TeacherID != nil {
			teacherID = strconv.FormatInt(*r.TeacherID, 10)
		}
		record := []string{
			r.ID,
			strconv.FormatInt(r.CourseID, 10),
			sanitizeCSVField(r.CourseName),
			teacherID,
			sanitizeCSVField(r.TeacherName),
			r.TermID,
			sanitizeCSVField(r.Title),
			sanitizeCSVField(r.Content),
			r.Grade,
			formatRatingsCSV(r.Ratings),
			strconv.Itoa(r.LikeCount),
			strconv.Itoa(r.DislikeCount),
			r.Status,
			r.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		return w.Write(record)
	}); err != nil {
		logger.L().Warn("failed to stream export reviews", zap.Error(err))
	}
}

// sanitizeCSVField 防止 CSV 公式注入（RFC 4180 转义由 csv.Writer 处理）
func sanitizeCSVField(s string) string {
	dangerousPrefix := func(c byte) bool {
		return c == '=' || c == '+' || c == '-' || c == '@' || c == '\t' || c == '\r'
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if len(trimmed) > 0 && dangerousPrefix(trimmed[0]) {
			lines[i] = "'" + line
		}
	}
	return strings.Join(lines, "\n")
}

// formatRatingsCSV 将评分 map 序列化为 CSV 友好的字符串
func formatRatingsCSV(ratings ReviewRatings) string {
	if len(ratings) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ratings))
	for k, v := range ratings {
		parts = append(parts, k+":"+strconv.Itoa(v))
	}
	return strings.Join(parts, ";")
}
