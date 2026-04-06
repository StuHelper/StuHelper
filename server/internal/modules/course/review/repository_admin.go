package review

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// allowedStatuses 允许的 status 参数白名单
var allowedStatuses = map[string]bool{
	"all":       true,
	"published": true,
	"hidden":    true,
	"deleted":   true,
	"pending":   true,
	"resolved":  true,
	"rejected":  true,
}

// validateStatus 校验 status 参数是否在白名单内，不合法时回退为 fallback
func validateStatus(status, fallback string) string {
	if status == "" || !allowedStatuses[status] {
		return fallback
	}
	return status
}

// ListAllReviews 获取所有评论（管理员，含总数）
// 使用 strings.Builder 重构 SQL 构建逻辑，参数绑定更清晰
func (r *Repository) ListAllReviews(ctx context.Context, status string, limit, offset int) ([]Review, int, error) {
	status = validateStatus(status, "all")

	var qb strings.Builder
	qb.WriteString(`
		SELECT r.id, r.course_id, COALESCE(c.name, ''), r.teacher_id, COALESCE(t.name, ''), r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count,
		       r.reply_count,
		       r.status, r.moderation_reason, r.created_at, r.updated_at,
		       COUNT(*) OVER() AS total
		FROM reviews r
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
	`)

	var args []interface{}
	needFilter := status != "" && status != "all"

	if needFilter {
		qb.WriteString(` WHERE r.status = $1`)
		qb.WriteString(` ORDER BY r.created_at DESC LIMIT $2 OFFSET $3`)
		args = []interface{}{status, limit, offset}
	} else {
		qb.WriteString(` ORDER BY r.created_at DESC LIMIT $1 OFFSET $2`)
		args = []interface{}{limit, offset}
	}

	rows, err := r.db.Query(ctx, qb.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListAllReviews: %w", err)
	}
	defer rows.Close()
	return scanReviewsWithTotal(rows)
}

// GetAdminStats 获取管理统计（条件聚合，reviews 单次扫描 + reports 单次扫描）
// 性能优化依赖以下索引（应在 init.sql 中创建）：
//   - reviews: idx_reviews_status (status) — 加速 FILTER 条件聚合
//   - reviews: idx_reviews_created_at (created_at) — 加速 today/week 过滤
//   - review_reports: idx_review_reports_status (status) — 加速 pending 计数
func (r *Repository) GetAdminStats(ctx context.Context) (*AdminStats, error) {
	var stats AdminStats
	// reviews 表：6 个计数合并为单次全表扫描 + FILTER 条件聚合
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total_reviews,
			COUNT(*) FILTER (WHERE status = 'published') AS published_reviews,
			COUNT(*) FILTER (WHERE status = 'hidden') AS hidden_reviews,
			COUNT(*) FILTER (WHERE status = 'deleted') AS deleted_reviews,
			COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE) AS today_reviews,
			COUNT(*) FILTER (WHERE created_at >= date_trunc('week', CURRENT_DATE)) AS week_reviews
		FROM reviews
	`).Scan(
		&stats.TotalReviews,
		&stats.PublishedReviews,
		&stats.HiddenReviews,
		&stats.DeletedReviews,
		&stats.TodayReviews,
		&stats.WeekReviews,
	)
	if err != nil {
		return nil, fmt.Errorf("GetAdminStats reviews: %w", err)
	}
	// reports 表：独立扫描（与 reviews 无关联，无法合并）
	err = r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total_reports,
			COUNT(*) FILTER (WHERE status = 'pending') AS pending_reports
		FROM review_reports
	`).Scan(
		&stats.TotalReports,
		&stats.PendingReports,
	)
	if err != nil {
		return nil, fmt.Errorf("GetAdminStats reports: %w", err)
	}
	return &stats, nil
}

// maxBatchDBSize 数据库层批量操作的最大数量上限（L-36: 纵深防御，防止绕过 handler 直接调用）
const maxBatchDBSize = 1000

// BatchUpdateReviewStatusTx 批量更新评论状态（事务内执行）。
// 调用方需先通过 LockReviewsTx 获取行锁，这里只执行 UPDATE。
// 校验批量上限。
func (r *Repository) BatchUpdateReviewStatusTx(ctx context.Context, tx pgx.Tx, ids []string, status string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	if len(ids) > maxBatchDBSize {
		return 0, fmt.Errorf("batch size %d exceeds db limit of %d", len(ids), maxBatchDBSize)
	}

	result, err := tx.Exec(ctx, `
		UPDATE reviews SET status = $1, updated_at = NOW()
		WHERE id = ANY($2)
	`, status, ids)
	if err != nil {
		return 0, fmt.Errorf("BatchUpdateReviewStatusTx: %w", err)
	}
	return result.RowsAffected(), nil
}

// LockReviewsTx 在事务内对指定评论行加排他锁，防止并发操作导致计数不一致
func (r *Repository) LockReviewsTx(ctx context.Context, tx pgx.Tx, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `SELECT id FROM reviews WHERE id = ANY($1) FOR UPDATE`, ids)
	if err != nil {
		return fmt.Errorf("LockReviewsTx: %w", err)
	}
	return nil
}

// AdjustCourseCountsForBatchDelete 批量删除时减少相关课程的评论计数
// 仅对当前状态非 deleted 的评论所属课程进行计数调整
// 此子查询依赖索引 idx_reviews_id_status (id, status) 或主键索引 + status 过滤，
// 确保 WHERE id = ANY($1) 走索引扫描而非全表扫描
func (r *Repository) AdjustCourseCountsForBatchDelete(ctx context.Context, tx pgx.Tx, ids []string) error {
	_, err := tx.Exec(ctx, `
		UPDATE courses SET review_count = GREATEST(review_count - sub.cnt, 0)
		FROM (
			SELECT course_id, COUNT(*) AS cnt
			FROM reviews
			WHERE id = ANY($1) AND status != 'deleted'
			GROUP BY course_id
		) sub
		WHERE courses.id = sub.course_id
	`, ids)
	if err != nil {
		return fmt.Errorf("AdjustCourseCountsForBatchDelete: %w", err)
	}
	return nil
}

// AdjustCourseCountsForBatchHide 批量隐藏时减少相关课程的评论计数
// 仅对当前状态为 published 的评论所属课程进行计数调整
func (r *Repository) AdjustCourseCountsForBatchHide(ctx context.Context, tx pgx.Tx, ids []string) error {
	_, err := tx.Exec(ctx, `
		UPDATE courses SET review_count = GREATEST(review_count - sub.cnt, 0)
		FROM (
			SELECT course_id, COUNT(*) AS cnt
			FROM reviews
			WHERE id = ANY($1) AND status = 'published'
			GROUP BY course_id
		) sub
		WHERE courses.id = sub.course_id
	`, ids)
	if err != nil {
		return fmt.Errorf("AdjustCourseCountsForBatchHide: %w", err)
	}
	return nil
}

// AdjustCourseCountsForBatchRestore 批量恢复时增加相关课程的评论计数
// 仅对当前状态为 deleted 的评论所属课程进行计数调整
func (r *Repository) AdjustCourseCountsForBatchRestore(ctx context.Context, tx pgx.Tx, ids []string) error {
	_, err := tx.Exec(ctx, `
		UPDATE courses SET review_count = review_count + sub.cnt
		FROM (
			SELECT course_id, COUNT(*) AS cnt
			FROM reviews
			WHERE id = ANY($1) AND status != 'published'
			GROUP BY course_id
		) sub
		WHERE courses.id = sub.course_id
	`, ids)
	if err != nil {
		return fmt.Errorf("AdjustCourseCountsForBatchRestore: %w", err)
	}
	return nil
}

// ModerateReviewTx 设置屏蔽原因 + 状态（事务内执行）
func (r *Repository) ModerateReviewTx(ctx context.Context, tx pgx.Tx, reviewID, reason, moderatedBy string) error {
	_, err := tx.Exec(ctx, `
		UPDATE reviews SET
			moderation_reason = $2, moderated_by = $3, moderated_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`, reviewID, reason, moderatedBy)
	if err != nil {
		return fmt.Errorf("ModerateReviewTx: %w", err)
	}
	return nil
}

// ClearModerationTx 恢复时清除屏蔽信息（事务内执行）
func (r *Repository) ClearModerationTx(ctx context.Context, tx pgx.Tx, reviewID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE reviews SET
			moderation_reason = NULL, moderated_by = NULL, moderated_at = NULL,
			updated_at = NOW()
		WHERE id = $1
	`, reviewID)
	if err != nil {
		return fmt.Errorf("ClearModerationTx: %w", err)
	}
	return nil
}

// AdminEditReviewContentTx 管理员编辑内容（首次编辑保存原始内容，事务内执行）
func (r *Repository) AdminEditReviewContentTx(ctx context.Context, tx pgx.Tx, reviewID, title, content, reason, moderatedBy string, contentFlag *string) error {
	_, err := tx.Exec(ctx, `
		UPDATE reviews SET
			original_title = CASE WHEN original_title IS NULL THEN title ELSE original_title END,
			original_content = CASE WHEN original_content IS NULL THEN content ELSE original_content END,
			title = $2, content = $3, content_flag = $6,
			moderation_reason = $4, moderated_by = $5, moderated_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`, reviewID, title, content, reason, moderatedBy, contentFlag)
	if err != nil {
		return fmt.Errorf("AdminEditReviewContentTx: %w", err)
	}
	return nil
}
