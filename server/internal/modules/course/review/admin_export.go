package review

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

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

	if format != "json" && format != "ndjson" && format != "csv" {
		format = "json"
	}
	if !isValidReviewStatus(status) {
		status = StatusAll
	}

	var (
		rowCount  int
		streamErr error
	)
	if format == "csv" {
		rowCount, streamErr = h.exportCSVStream(c, status)
	} else {
		rowCount, streamErr = h.exportNDJSONStream(c, status)
	}

	result := "success"
	reason := ""
	if streamErr != nil {
		result = "failure"
		reason = streamErr.Error()
	}
	audit.LogFromGin(c, audit.Event{
		Type:         audit.EventDataExport,
		Category:     "admin_operation",
		ActorType:    "admin",
		Resource:     "review",
		ResourceType: "review",
		ResourceID:   "bulk",
		Action:       "export",
		Result:       result,
		Reason:       reason,
		Details: map[string]any{
			"format":    format,
			"status":    status,
			"row_count": rowCount,
			"row_limit": maxExportLimit,
		},
	})
}

// exportNDJSONStream 流式导出 NDJSON，逐行从数据库读取并写入响应，避免全量加载到内存
func (h *Handler) exportNDJSONStream(c *gin.Context, status string) (int, error) {
	c.Header("Content-Type", "application/x-ndjson; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=reviews.ndjson")

	rowCount := 0
	streamErr := h.service.StreamExportReviews(c.Request.Context(), status, func(r Review) error {
		line, err := json.Marshal(r)
		if err != nil {
			return fmt.Errorf("marshal review %s: %w", r.ID, err)
		}
		line = append(line, '\n')
		if err := writeExportChunk(c.Writer, line); err != nil {
			return err
		}
		rowCount++
		return nil
	})
	if streamErr != nil {
		logger.FromGin(c).Warn("failed to stream export reviews (ndjson)", zap.Error(streamErr))
		return rowCount, streamErr
	}

	if err := writeExportChunk(c.Writer, []byte("# EXPORT_COMPLETE\n")); err != nil {
		logger.FromGin(c).Warn("failed to write export completion marker", zap.Error(err))
		return rowCount, fmt.Errorf("write NDJSON completion marker: %w", err)
	}
	return rowCount, nil
}

// exportCSVStream 流式导出 CSV，逐行从数据库读取并写入响应，避免全量加载到内存
func (h *Handler) exportCSVStream(c *gin.Context, status string) (int, error) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=reviews.csv")

	if err := writeExportChunk(c.Writer, []byte{0xEF, 0xBB, 0xBF}); err != nil {
		return 0, fmt.Errorf("write CSV BOM: %w", err)
	}

	w := csv.NewWriter(c.Writer)

	if err := w.Write([]string{
		"ID", "课程ID", "课程名称", "教师ID", "教师名称",
		"学期", "标题", "内容", "成绩", "评分",
		"点赞数", "踩数", "状态", "创建时间",
	}); err != nil {
		return 0, fmt.Errorf("write CSV header: %w", err)
	}

	rowCount := 0
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
		if err := w.Write(record); err != nil {
			return err
		}
		rowCount++
		return nil
	})
	w.Flush()
	writeErr := w.Error()
	if streamErr != nil {
		logger.FromGin(c).Warn("failed to stream export reviews", zap.Error(streamErr))
		if writeErr != nil {
			return rowCount, fmt.Errorf("stream CSV export: %w; flush CSV export: %v", streamErr, writeErr)
		}
		return rowCount, streamErr
	}
	if writeErr != nil {
		logger.FromGin(c).Warn("failed to flush export reviews", zap.Error(writeErr))
		return rowCount, fmt.Errorf("flush CSV export: %w", writeErr)
	}

	if err := writeExportChunk(c.Writer, []byte("# EXPORT_COMPLETE\n")); err != nil {
		logger.FromGin(c).Warn("failed to write export completion marker", zap.Error(err))
		return rowCount, fmt.Errorf("write CSV completion marker: %w", err)
	}
	return rowCount, nil
}

func writeExportChunk(w io.Writer, data []byte) error {
	n, err := w.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

// sanitizeCSVField 防止 CSV 公式注入
func sanitizeCSVField(s string) string {
	if s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		runes := []rune(strings.TrimLeft(line, " "))
		if len(runes) > 0 && isDangerousCSVPrefix(runes[0]) {
			lines[i] = "'" + line
		}
	}
	return strings.Join(lines, "\n")
}

// isDangerousCSVPrefix 检查字符是否为 CSV 公式注入危险前缀
func isDangerousCSVPrefix(c rune) bool {
	switch c {
	case '=', '+', '-', '@', '\t', '\r', '＝', '＋', '－', '＠':
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
