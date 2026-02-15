package review

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

const (
	// maxBatchSize 批量操作的最大数量上限
	maxBatchSize = 100
	// maxUserAgentLen UserAgent 字段最大长度，超出截断
	maxUserAgentLen = 256
)

// truncateUserAgent 从请求中获取 UserAgent 并截断到安全长度（按 rune 截断，避免破坏 UTF-8）
func truncateUserAgent(c *gin.Context) string {
	ua := c.GetHeader("User-Agent")
	runes := []rune(ua)
	if len(runes) > maxUserAgentLen {
		return string(runes[:maxUserAgentLen])
	}
	return ua
}

// ListReports 获取举报列表
func (h *Handler) ListReports(c *gin.Context) {
	status := c.DefaultQuery("status", "pending")
	// 白名单校验 status 参数
	if status != "pending" && status != "resolved" && status != "rejected" && status != "all" {
		status = "pending"
	}
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
	Action string `json:"action" binding:"required,oneof=reject hide_review delete_review"`
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
			logger.FromGin(c).Error("failed to process report", zap.Error(err))
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
		ResourceID:    reportID,
		OldValue:      map[string]string{"status": "pending"},
		NewValue:      map[string]string{"action": req.Action, "note": req.Note},
		IPAddress:     c.ClientIP(),
		UserAgent:     truncateUserAgent(c),
	}); err != nil {
		logger.FromGin(c).Warn("failed to log operation", zap.Error(err))
	}

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
	if status != "published" && status != "hidden" && status != "deleted" && status != "all" {
		status = "all"
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

	result, err := h.service.AdminUpdateReview(c.Request.Context(), AdminUpdateReviewParams{
		ReviewID: reviewID,
		Action:   req.Action,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrReviewNotFound):
			response.NotFound(c, "review not found")
		case errors.Is(err, ErrInvalidAction):
			response.BadRequest(c, "invalid action")
		case errors.Is(err, ErrInvalidTransition):
			response.BadRequest(c, err.Error())
		default:
			logger.FromGin(c).Error("failed to update review", zap.Error(err))
			response.InternalError(c, "failed to update review")
		}
		return
	}

	ctx := c.Request.Context()

	// 记录操作日志（oldStatus 来自事务内读取，无 TOCTOU 竞态）
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)
	if err := h.service.LogOperation(ctx, LogOperationParams{
		AdminUserID:   userID,
		AdminUsername: username,
		Action:        req.Action,
		ResourceType:  "review",
		ResourceID:    reviewID,
		OldValue:      map[string]string{"status": result.OldStatus},
		NewValue:      map[string]string{"action": req.Action},
		IPAddress:     c.ClientIP(),
		UserAgent:     truncateUserAgent(c),
	}); err != nil {
		logger.FromGin(c).Warn("failed to log operation", zap.Error(err))
	}

	h.invalidateReviewCaches(c, 0, "review:stats")

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
		response.BadRequest(c, err.Error())
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

	result, err := h.service.BatchUpdateReviews(c.Request.Context(), BatchUpdateReviewsParams{
		IDs:    req.IDs,
		Action: req.Action,
	})
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

	ctx := c.Request.Context()

	// 记录操作日志
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)
	if err := h.service.LogOperation(ctx, LogOperationParams{
		AdminUserID:   userID,
		AdminUsername: username,
		Action:        "batch_" + req.Action,
		ResourceType:  "review",
		ResourceID:    fmt.Sprintf("batch:%d_items", len(req.IDs)),
		OldValue:      nil,
		NewValue:      map[string]interface{}{"ids": req.IDs, "action": req.Action, "affected": result.Affected},
		IPAddress:     c.ClientIP(),
		UserAgent:     truncateUserAgent(c),
	}); err != nil {
		logger.FromGin(c).Warn("failed to log operation", zap.Error(err))
	}

	h.invalidateReviewCaches(c, 0, "review:stats")

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
		logger.FromGin(c).Error("failed to load operation logs", zap.Error(err))
		response.InternalError(c, "failed to load operation logs")
		return
	}

	response.Success(c, gin.H{"list": result.List, "total": result.Total})
}

// ExportReviews 导出评论数据（JSON/NDJSON 和 CSV 均使用流式导出，避免 OOM）
func (h *Handler) ExportReviews(c *gin.Context) {
	format := c.DefaultQuery("format", "json")
	status := c.DefaultQuery("status", "all")

	// 验证 format 参数：json 和 ndjson 均映射到 NDJSON 流式导出
	if format != "json" && format != "ndjson" && format != "csv" {
		format = "json"
	}
	// 验证 status 参数
	if status != "all" && status != "published" && status != "hidden" && status != "deleted" {
		status = "all"
	}

	if format == "csv" {
		h.exportCSVStream(c, status)
		return
	}

	// JSON/NDJSON 均走 NDJSON 流式导出
	h.exportNDJSONStream(c, status)
}

// exportNDJSONStream 流式导出 NDJSON，逐行从数据库读取并写入响应，避免全量加载到内存
func (h *Handler) exportNDJSONStream(c *gin.Context, status string) {
	c.Header("Content-Type", "application/x-ndjson; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=reviews.ndjson")

	// 流式写入数据行（headers 已发送，无法返回 HTTP 错误，仅记录日志）
	// 写入成功后追加 "# EXPORT_COMPLETE" 标记行，客户端可据此判断导出是否完整
	streamErr := h.service.StreamExportReviews(c.Request.Context(), status, func(r Review) error {
		line, err := json.Marshal(r)
		if err != nil {
			return fmt.Errorf("marshal review %s: %w", r.ID, err)
		}
		line = append(line, '\n')
		_, err = c.Writer.Write(line)
		return err
	})
	if streamErr != nil {
		logger.L().Warn("failed to stream export reviews (ndjson)", zap.Error(streamErr))
		return // 不写入完成标记，客户端可据此检测到导出不完整
	}

	// 所有行写入成功，追加完成标记
	if _, err := c.Writer.Write([]byte("# EXPORT_COMPLETE\n")); err != nil {
		logger.L().Warn("failed to write export completion marker", zap.Error(err))
	}
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
	// 写入成功后追加 "# EXPORT_COMPLETE" 标记行，客户端可据此判断导出是否完整
	streamErr := h.service.StreamExportReviews(c.Request.Context(), status, func(r Review) error {
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
			sanitizeCSVField(r.Grade),
			formatRatingsCSV(r.Ratings),
			strconv.Itoa(r.LikeCount),
			strconv.Itoa(r.DislikeCount),
			r.Status,
			r.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		return w.Write(record)
	})
	if streamErr != nil {
		logger.L().Warn("failed to stream export reviews", zap.Error(streamErr))
		return // 不写入完成标记，客户端可据此检测到导出不完整
	}

	// 所有行写入成功，追加完成标记
	w.Flush()
	if _, err := c.Writer.Write([]byte("# EXPORT_COMPLETE\n")); err != nil {
		logger.L().Warn("failed to write export completion marker", zap.Error(err))
	}
}

// sanitizeCSVField 防止 CSV 公式注入（RFC 4180 转义由 csv.Writer 处理）
// 同时检测 Unicode 全角变体（＝＋－＠），防止旧版 Excel 解释为公式
func sanitizeCSVField(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		runes := []rune(strings.TrimLeft(line, " "))
		if len(runes) > 0 && isDangerousCSVPrefix(runes[0]) {
			lines[i] = "'" + line
		}
	}
	return strings.Join(lines, "\n")
}

// isDangerousCSVPrefix 检查字符是否为 CSV 公式注入危险前缀（含 Unicode 全角变体）
func isDangerousCSVPrefix(c rune) bool {
	switch c {
	case '=', '+', '-', '@', '\t', '\r',
		'＝', '＋', '－', '＠': // Unicode fullwidth variants
		return true
	}
	return false
}

// formatRatingsCSV 将评分 map 序列化为 CSV 友好的字符串
func formatRatingsCSV(ratings ReviewRatings) string {
	if len(ratings) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ratings))
	for k, v := range ratings {
		parts = append(parts, sanitizeCSVField(k)+":"+strconv.Itoa(v))
	}
	return strings.Join(parts, ";")
}
