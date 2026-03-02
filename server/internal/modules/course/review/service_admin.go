package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
)

// AdminUpdateReviewParams 管理员更新评论参数
type AdminUpdateReviewParams struct {
	ReviewID string
	Action   string
	Reason   string // 可选，hide 时记录屏蔽原因
	AdminID  string // 可选，hide/restore 时记录操作人
}

// validTransitions 定义合法的状态转移白名单
// key: action, value: 允许的源状态集合
var validTransitions = map[string]map[string]bool{
	"hide":    {StatusPublished: true},
	"restore": {StatusHidden: true},
	"delete":  {StatusPublished: true, StatusHidden: true},
}

// AdminUpdateReviewResult 管理员更新评论结果
type AdminUpdateReviewResult struct {
	OldStatus string // 事务内读取的旧状态，用于审计日志
}

// AdminUpdateReview 管理员更新评论，返回事务内读取的旧状态
// hide 时如果提供了 Reason/AdminID，同时记录屏蔽原因；restore 时自动清除屏蔽信息
func (s *Service) AdminUpdateReview(ctx context.Context, params AdminUpdateReviewParams) (*AdminUpdateReviewResult, error) {
	var oldStatus string
	err := s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		currentStatus, courseID, err := s.repo.GetReviewStatusAndCourseIDTx(ctx, tx, params.ReviewID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReviewNotFound
			}
			return err
		}
		oldStatus = currentStatus

		allowed, ok := validTransitions[params.Action]
		if !ok {
			return ErrInvalidAction
		}
		if !allowed[currentStatus] {
			return fmt.Errorf("%w: cannot %s from %s", ErrInvalidTransition, params.Action, currentStatus)
		}

		switch params.Action {
		case "hide":
			if err := s.repo.UpdateReviewStatus(ctx, tx, params.ReviewID, StatusHidden); err != nil {
				return err
			}
			if params.AdminID != "" {
				if err := s.repo.ModerateReviewTx(ctx, tx, params.ReviewID, params.Reason, params.AdminID); err != nil {
					return err
				}
			}
			return s.repo.DecrementCourseReviewCount(ctx, tx, courseID)
		case "restore":
			if err := s.repo.UpdateReviewStatus(ctx, tx, params.ReviewID, StatusPublished); err != nil {
				return err
			}
			if err := s.repo.ClearModerationTx(ctx, tx, params.ReviewID); err != nil {
				return err
			}
			return s.repo.IncrementCourseReviewCount(ctx, tx, courseID)
		case "delete":
			if err := s.repo.SoftDeleteReview(ctx, tx, params.ReviewID); err != nil {
				return err
			}
			if currentStatus == StatusPublished {
				return s.repo.DecrementCourseReviewCount(ctx, tx, courseID)
			}
			return nil
		default:
			return ErrInvalidAction
		}
	})
	if err != nil {
		return nil, err
	}
	return &AdminUpdateReviewResult{OldStatus: oldStatus}, nil
}

// AdminEditReviewParams 管理员编辑评论内容参数
type AdminEditReviewParams struct {
	ReviewID string
	Title    string
	Content  string
	Reason   string
	AdminID  string
}

// AdminEditReview 管理员编辑评论内容
func (s *Service) AdminEditReview(ctx context.Context, params AdminEditReviewParams) error {
	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// 确认评论存在
		_, _, err := s.repo.GetReviewStatusAndCourseIDTx(ctx, tx, params.ReviewID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReviewNotFound
			}
			return err
		}
		return s.repo.AdminEditReviewContentTx(ctx, tx, params.ReviewID, params.Title, params.Content, params.Reason, params.AdminID)
	})
}

// ListAllReviewsParams 获取所有评论参数
type ListAllReviewsParams struct {
	Status   string
	Page     int
	PageSize int
}

// ListAllReviews 获取所有评论（管理员）
func (s *Service) ListAllReviews(ctx context.Context, params ListAllReviewsParams) (*GetCourseReviewsResult, error) {
	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	list, total, err := s.repo.ListAllReviews(ctx, params.Status, pageSize, offset)
	if err != nil {
		return nil, err
	}

	return &GetCourseReviewsResult{List: list, Total: total}, nil
}

// GetAdminStats 获取管理统计
func (s *Service) GetAdminStats(ctx context.Context) (*AdminStats, error) {
	return s.repo.GetAdminStats(ctx)
}

// GetRatingTrend 获取课程评分趋势
func (s *Service) GetRatingTrend(ctx context.Context, courseID int64) ([]RatingTrendItem, error) {
	return s.repo.GetRatingTrend(ctx, courseID)
}

// GetHotCourses 获取热门课程排行
func (s *Service) GetHotCourses(ctx context.Context, period string, limit int) ([]HotCourse, error) {
	return s.repo.ListHotCourses(ctx, period, limit)
}

// GetTeacherRatingStats 获取教师评分统计
func (s *Service) GetTeacherRatingStats(ctx context.Context, teacherID int64) (*TeacherRatingStatsResponse, error) {
	// 获取教师基本信息（名称 + 院系）
	teacherName, departmentName, err := s.repo.GetTeacherInfo(ctx, teacherID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}

	// 获取评分维度配置
	dimensions, err := s.repo.ListRatingDimensions(ctx)
	if err != nil {
		return nil, err
	}

	// 获取统计数据
	stats, err := s.repo.GetTeacherRatingStats(ctx, teacherID)
	if err != nil {
		return nil, err
	}

	// 获取教师授课课程列表
	courses, err := s.repo.ListTeacherCourses(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	if courses == nil {
		courses = []TeacherCourse{}
	}

	// 获取教师评论总数
	reviewCount, err := s.repo.GetTeacherReviewCount(ctx, teacherID)
	if err != nil {
		return nil, err
	}

	// 构建响应
	resp := &TeacherRatingStatsResponse{
		TeacherID:      teacherID,
		TeacherName:    teacherName,
		DepartmentName: departmentName,
		CourseCount:    len(courses),
		ReviewCount:    reviewCount,
		Courses:        courses,
	}

	// 分组统计并填充响应
	overallStats := s.groupRatingStats(stats, dimensions, resp)
	s.buildRatingCharts(resp, dimensions, overallStats, teacherName)

	return resp, nil
}

// groupRatingStats 按学期分组统计评分数据，填充 resp.Overall 和 resp.ByTerm
// 返回总体维度统计 map（用于雷达图和平均分计算）
func (s *Service) groupRatingStats(stats []TeacherRatingStats, dimensions []RatingDimension, resp *TeacherRatingStatsResponse) map[string]*DimensionStats {
	// 构建维度名称查找表
	dimNameMap := make(map[string]string, len(dimensions))
	for _, d := range dimensions {
		dimNameMap[d.Key] = d.Name
	}

	termStats := make(map[string]*TermRatingStats)
	overallStats := make(map[string]*DimensionStats)

	for _, stat := range stats {
		termID := ""
		if stat.TermID != nil {
			termID = *stat.TermID
		}

		dimName := stat.DimensionKey
		if name, ok := dimNameMap[stat.DimensionKey]; ok {
			dimName = name
		}

		ds := &DimensionStats{
			Key:         stat.DimensionKey,
			Name:        dimName,
			AvgRating:   stat.AvgRating,
			RatingCount: stat.RatingCount,
		}

		if termID == "" {
			overallStats[stat.DimensionKey] = ds
		} else {
			if _, ok := termStats[termID]; !ok {
				termStats[termID] = &TermRatingStats{
					TermID:     termID,
					TermName:   termID,
					Dimensions: []DimensionStats{},
				}
			}
			termStats[termID].Dimensions = append(termStats[termID].Dimensions, *ds)
		}
	}

	// 构建总体统计（按维度配置顺序）
	var overallDims []DimensionStats
	for _, d := range dimensions {
		if ds, ok := overallStats[d.Key]; ok {
			overallDims = append(overallDims, *ds)
		}
	}
	resp.Overall = TermRatingStats{
		TermID:     "",
		TermName:   "overall",
		Dimensions: overallDims,
	}

	// 构建学期统计列表（按 TermID 排序，确保确定性输出）
	for _, ts := range termStats {
		resp.ByTerm = append(resp.ByTerm, *ts)
	}
	sort.Slice(resp.ByTerm, func(i, j int) bool {
		return resp.ByTerm[i].TermID < resp.ByTerm[j].TermID
	})

	return overallStats
}

// buildRatingCharts 构建雷达图、平均分和评分趋势
func (s *Service) buildRatingCharts(resp *TeacherRatingStatsResponse, dimensions []RatingDimension, overallStats map[string]*DimensionStats, teacherName string) {
	// 雷达图数据
	labels := make([]string, 0, len(dimensions))
	data := make([]float64, 0, len(dimensions))
	for _, d := range dimensions {
		labels = append(labels, d.Name)
		if ds, ok := overallStats[d.Key]; ok {
			data = append(data, ds.AvgRating)
		} else {
			data = append(data, 0)
		}
	}
	resp.RadarChart = RadarChartData{
		Labels: labels,
		Datasets: []RadarChartDataset{
			{
				Label:           teacherName,
				Data:            data,
				BackgroundColor: radarBgColor,
				BorderColor:     radarBorderColor,
			},
		},
	}

	// 总体平均分
	if len(resp.Overall.Dimensions) > 0 {
		var sum float64
		for _, d := range resp.Overall.Dimensions {
			sum += d.AvgRating
		}
		avg := sum / float64(len(resp.Overall.Dimensions))
		resp.AvgRating = &avg
	}

	// 评分趋势（按学期的平均分）
	ratingTrend := make([]RatingTrendItem, 0, len(resp.ByTerm))
	for _, ts := range resp.ByTerm {
		if len(ts.Dimensions) == 0 {
			continue
		}
		var sum float64
		for _, d := range ts.Dimensions {
			sum += d.AvgRating
		}
		ratingTrend = append(ratingTrend, RatingTrendItem{
			TermID:    ts.TermID,
			TermName:  ts.TermName,
			AvgRating: sum / float64(len(ts.Dimensions)),
		})
	}
	resp.RatingTrend = ratingTrend
}

// BatchUpdateReviewsParams 批量更新评论参数
type BatchUpdateReviewsParams struct {
	IDs    []string
	Action string // hide, restore, delete
}

// BatchUpdateReviewsResult 批量更新评论结果
type BatchUpdateReviewsResult struct {
	Affected int64 `json:"affected"`
}

// BatchUpdateReviews 批量更新评论状态（管理员）
func (s *Service) BatchUpdateReviews(ctx context.Context, params BatchUpdateReviewsParams) (*BatchUpdateReviewsResult, error) {
	// 纵深防御：service 层也校验批量上限，防止绕过 handler 直接调用
	if len(params.IDs) > maxBatchSize {
		return nil, fmt.Errorf("batch size %d exceeds limit of %d", len(params.IDs), maxBatchSize)
	}

	var status string
	switch params.Action {
	case "hide":
		status = StatusHidden
	case "restore":
		status = StatusPublished
	case "delete":
		status = StatusDeleted
	default:
		return nil, ErrInvalidAction
	}

	var affected int64
	if err := s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// 先锁定涉及的评论行，防止并发批量操作导致计数不一致
		if err := s.repo.LockReviewsTx(ctx, tx, params.IDs); err != nil {
			return err
		}

		// 先调整课程评论计数（在状态变更前，基于当前状态判断）
		switch params.Action {
		case "hide":
			if err := s.repo.AdjustCourseCountsForBatchHide(ctx, tx, params.IDs); err != nil {
				return err
			}
		case "delete":
			if err := s.repo.AdjustCourseCountsForBatchDelete(ctx, tx, params.IDs); err != nil {
				return err
			}
		case "restore":
			if err := s.repo.AdjustCourseCountsForBatchRestore(ctx, tx, params.IDs); err != nil {
				return err
			}
		}

		var err error
		affected, err = s.repo.BatchUpdateReviewStatusTx(ctx, tx, params.IDs, status)
		return err
	}); err != nil {
		return nil, err
	}

	return &BatchUpdateReviewsResult{Affected: affected}, nil
}

// BatchUpdateReviewsWithAudit 批量更新评论状态并记录审计日志（管理员）
// M-49: 封装审计日志记录，供 handler 层调用
func (s *Service) BatchUpdateReviewsWithAudit(ctx context.Context, params BatchUpdateReviewsParams, adminUserID, adminUsername string) (*BatchUpdateReviewsResult, error) {
	result, err := s.BatchUpdateReviews(ctx, params)
	if err != nil {
		return nil, err
	}

	// M-49: 记录批量操作审计日志
	audit.Log(audit.Event{
		Type:     audit.EventAdminBatchOp,
		UserID:   adminUserID,
		Username: adminUsername,
		Resource: "review",
		Action:   "batch_" + params.Action,
		Result:   "success",
		Details: map[string]any{
			"ids":      params.IDs,
			"action":   params.Action,
			"affected": result.Affected,
		},
	})

	return result, nil
}

// LogOperationParams 记录操作日志参数
type LogOperationParams struct {
	AdminUserID   string
	AdminUsername string
	Action        string
	ResourceType  string
	ResourceID    string
	OldValue      any
	NewValue      any
	IPAddress     string
	UserAgent     string
}

// LogOperation 记录操作日志
func (s *Service) LogOperation(ctx context.Context, params LogOperationParams) error {
	var oldValue, newValue []byte
	var err error

	if params.OldValue != nil {
		oldValue, err = json.Marshal(params.OldValue)
		if err != nil {
			return err
		}
	}
	if params.NewValue != nil {
		newValue, err = json.Marshal(params.NewValue)
		if err != nil {
			return err
		}
	}

	return s.repo.CreateOperationLog(ctx, CreateOperationLogParams{
		AdminUserID:   params.AdminUserID,
		AdminUsername: params.AdminUsername,
		Action:        params.Action,
		ResourceType:  params.ResourceType,
		ResourceID:    params.ResourceID,
		OldValue:      oldValue,
		NewValue:      newValue,
		IPAddress:     params.IPAddress,
		UserAgent:     params.UserAgent,
	})
}

// GetOperationLogsParams 获取操作日志参数
type GetOperationLogsParams struct {
	Page     int
	PageSize int
}

// GetOperationLogsResult 获取操作日志结果
type GetOperationLogsResult struct {
	List  []AdminOperationLog
	Total int
}

// GetOperationLogs 获取操作日志列表
func (s *Service) GetOperationLogs(ctx context.Context, params GetOperationLogsParams) (*GetOperationLogsResult, error) {
	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	list, total, err := s.repo.ListOperationLogs(ctx, pageSize, offset)
	if err != nil {
		return nil, err
	}

	return &GetOperationLogsResult{List: list, Total: total}, nil
}

// StreamExportReviews 流式导出评论，逐行回调
func (s *Service) StreamExportReviews(ctx context.Context, status string, fn func(Review) error) error {
	return s.repo.ForEachReviewForExport(ctx, status, fn)
}

// CleanupOldOperationLogs removes operation logs older than the retention period.
// Returns the number of deleted rows.
func (s *Service) CleanupOldOperationLogs(ctx context.Context, retentionDays int) (int64, error) {
	return s.repo.CleanupOldOperationLogs(ctx, retentionDays)
}
