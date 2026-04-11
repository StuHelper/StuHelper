package review

import (
	"context"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// fgaWg 跟踪所有 in-flight FGA goroutine，支持优雅关闭时等待完成
var fgaWg sync.WaitGroup

// WaitFGAWrites 等待所有 in-flight FGA 异步写入完成。
// 应在优雅关闭流程中（HTTP server 关闭后、数据库连接关闭前）调用。
func WaitFGAWrites() {
	fgaWg.Wait()
}

// writeFGAReviewTuples 异步写入评课的 OpenFGA 关系 tuple。
// 非阻塞：FGA 未配置或写入失败仅记日志，不影响主流程。
// lookup 和写入均在 goroutine 内执行，不阻塞主请求。
func (h *Handler) writeFGAReviewTuples(c *gin.Context, reviewID, externalUserID string, courseID int64) {
	if h.fga == nil {
		return
	}

	fgaWg.Add(1)
	go func() {
		defer fgaWg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), fga.DefaultWriteTimeout)
		defer cancel()

		schoolID, ok := h.lookupCourseSchoolID(ctx, courseID)
		if !ok {
			return
		}

		if err := h.fga.WriteReviewRelations(ctx, reviewID, externalUserID, strconv.FormatInt(courseID, 10), schoolID); err != nil {
			logger.L().Warn("failed to write FGA review tuples",
				zap.String("review_id", reviewID),
				zap.Error(err),
			)
		}
	}()
}

// writeFGAReportTuplesForReview 异步写入举报 FGA tuple（从 reviewID 查询 courseID）。
func (h *Handler) writeFGAReportTuplesForReview(c *gin.Context, reportID, externalUserID, reviewID string) {
	if h.fga == nil {
		return
	}

	fgaWg.Add(1)
	go func() {
		defer fgaWg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), fga.DefaultWriteTimeout)
		defer cancel()

		courseID, ok := h.lookupReviewCourseID(ctx, reviewID)
		if !ok {
			return
		}

		schoolID, ok := h.lookupCourseSchoolID(ctx, courseID)
		if !ok {
			return
		}

		if err := h.fga.WriteReportRelations(ctx, reportID, externalUserID, reviewID, schoolID); err != nil {
			logger.L().Warn("failed to write FGA report tuples",
				zap.String("report_id", reportID),
				zap.Error(err),
			)
		}
	}()
}

// checkFGA 检查 FGA 权限（FGA 未配置时返回 false）
func (h *Handler) checkFGA(ctx context.Context, user, relation, object string) bool {
	if h.fga == nil {
		return false
	}
	allowed, err := h.fga.Check(ctx, user, relation, object)
	if err != nil {
		logger.L().Warn("FGA check failed, denying",
			zap.String("user", user),
			zap.String("relation", relation),
			zap.String("object", object),
			zap.Error(err),
		)
		return false
	}
	return allowed
}

// lookupCourseSchoolID 查询课程所属学校 ID（用于 FGA tuple）。
// 查询失败时返回 ok=false，调用方应跳过 tuple 写入。
func (h *Handler) lookupCourseSchoolID(ctx context.Context, courseID int64) (string, bool) {
	var schoolID int64
	err := h.db.QueryRow(ctx, "SELECT school_id FROM courses WHERE id = $1", courseID).Scan(&schoolID)
	if err != nil {
		logger.L().Warn("FGA: failed to lookup course school_id, skipping tuple write",
			zap.Int64("course_id", courseID),
			zap.Error(err),
		)
		return "", false
	}
	return strconv.FormatInt(schoolID, 10), true
}

// lookupReviewCourseID 查询评课所属课程 ID（用于 FGA tuple）。
// 查询失败时返回 ok=false，调用方应跳过 tuple 写入。
func (h *Handler) lookupReviewCourseID(ctx context.Context, reviewID string) (int64, bool) {
	var courseID int64
	err := h.db.QueryRow(ctx, "SELECT course_id FROM reviews WHERE id = $1", reviewID).Scan(&courseID)
	if err != nil {
		logger.L().Warn("FGA: failed to lookup review course_id, skipping tuple write",
			zap.String("review_id", reviewID),
			zap.Error(err),
		)
		return 0, false
	}
	return courseID, true
}
