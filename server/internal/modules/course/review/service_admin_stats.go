package review

import (
	"context"
	"errors"
	"sort"

	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"
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
	g, gctx := errgroup.WithContext(ctx)

	var (
		teacherName    string
		departmentName string
		dimensions     []RatingDimension
		stats          []TeacherRatingStats
		courses        []TeacherCourse
		reviewCount    int
	)

	g.Go(func() error {
		var err error
		teacherName, departmentName, err = s.repo.GetTeacherInfo(gctx, teacherID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrTeacherNotFound
			}
			return err
		}
		return nil
	})
	g.Go(func() error {
		var err error
		dimensions, err = s.repo.ListRatingDimensions(gctx)
		return err
	})
	g.Go(func() error {
		var err error
		stats, err = s.repo.GetTeacherRatingStats(gctx, teacherID)
		return err
	})
	g.Go(func() error {
		var err error
		courses, err = s.repo.ListTeacherCourses(gctx, teacherID)
		return err
	})
	g.Go(func() error {
		var err error
		reviewCount, err = s.repo.GetTeacherReviewCount(gctx, teacherID)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	if len(stats) == 0 && reviewCount > 0 {
		if err := s.repo.RefreshTeacherRatingStats(ctx, teacherID); err != nil {
			return nil, err
		}
		var err error
		stats, err = s.repo.GetTeacherRatingStats(ctx, teacherID)
		if err != nil {
			return nil, err
		}
	}

	if courses == nil {
		courses = []TeacherCourse{}
	}

	resp := &TeacherRatingStatsResponse{
		TeacherID:      teacherID,
		TeacherName:    teacherName,
		DepartmentName: departmentName,
		CourseCount:    len(courses),
		ReviewCount:    reviewCount,
		Courses:        courses,
	}

	s.groupRatingStats(stats, dimensions, resp)
	s.buildRatingStats(resp)
	return resp, nil
}

// groupRatingStats 按学期分组统计评分数据
func (s *Service) groupRatingStats(stats []TeacherRatingStats, dimensions []RatingDimension, resp *TeacherRatingStatsResponse) {
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
}

// buildRatingStats 构建平均分和评分趋势
func (s *Service) buildRatingStats(resp *TeacherRatingStatsResponse) {
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
