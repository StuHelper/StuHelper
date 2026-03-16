package review

import (
	"context"
	"errors"
	"sort"

	"github.com/jackc/pgx/v5"
)

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
	teacherName, departmentName, err := s.repo.GetTeacherInfo(ctx, teacherID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}

	dimensions, err := s.repo.ListRatingDimensions(ctx)
	if err != nil {
		return nil, err
	}

	stats, err := s.repo.GetTeacherRatingStats(ctx, teacherID)
	if err != nil {
		return nil, err
	}

	courses, err := s.repo.ListTeacherCourses(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	if courses == nil {
		courses = []TeacherCourse{}
	}

	reviewCount, err := s.repo.GetTeacherReviewCount(ctx, teacherID)
	if err != nil {
		return nil, err
	}

	resp := &TeacherRatingStatsResponse{
		TeacherID:      teacherID,
		TeacherName:    teacherName,
		DepartmentName: departmentName,
		CourseCount:    len(courses),
		ReviewCount:    reviewCount,
		Courses:        courses,
	}

	overallStats := s.groupRatingStats(stats, dimensions, resp)
	s.buildRatingCharts(resp, dimensions, overallStats, teacherName)
	return resp, nil
}

// groupRatingStats 按学期分组统计评分数据
func (s *Service) groupRatingStats(stats []TeacherRatingStats, dimensions []RatingDimension, resp *TeacherRatingStatsResponse) map[string]*DimensionStats {
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
			continue
		}

		if _, ok := termStats[termID]; !ok {
			termStats[termID] = &TermRatingStats{
				TermID:     termID,
				TermName:   termID,
				Dimensions: []DimensionStats{},
			}
		}
		termStats[termID].Dimensions = append(termStats[termID].Dimensions, *ds)
	}

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
		Datasets: []RadarChartDataset{{
			Label:           teacherName,
			Data:            data,
			BackgroundColor: radarBgColor,
			BorderColor:     radarBorderColor,
		}},
	}

	if len(resp.Overall.Dimensions) > 0 {
		var sum float64
		for _, d := range resp.Overall.Dimensions {
			sum += d.AvgRating
		}
		avg := sum / float64(len(resp.Overall.Dimensions))
		resp.AvgRating = &avg
	}

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
