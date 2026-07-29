package review

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

// GetRatingDimensions 获取评分维度配置
func (h *Handler) GetRatingDimensions(c *gin.Context) {
	respondWithCachedData(h, c, "review:rating_dimensions", "all", h.service.GetRatingDimensions, nil, "failed to load rating dimensions", "failed to load rating dimensions", nil)
}

// GetCourseRatingStats 获取课程评分统计（雷达图数据）
func (h *Handler) GetCourseRatingStats(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "courseID")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	respondWithCachedData(h, c, courseRatingStatsCacheKey, strconv.FormatInt(courseID, 10), func(ctx context.Context) (CourseRatingStatsResponse, error) {
		dimensionNames, err := h.service.GetDimensionNames(ctx)
		if err != nil {
			return CourseRatingStatsResponse{}, err
		}

		stats, err := h.service.GetCourseRatingStats(ctx, courseID)
		if err != nil {
			return CourseRatingStatsResponse{}, err
		}

		byTerm := make(map[string]*TermRatingStats)
		overall := TermRatingStats{Dimensions: []DimensionStats{}}
		allKeysSet := make(map[string]bool)

		for _, s := range stats {
			allKeysSet[s.DimensionKey] = true
			dist := map[int]int{}
			if err := json.Unmarshal(s.RatingDist, &dist); err != nil {
				logger.L().Warn("failed to unmarshal rating distribution",
					zap.String("dimension_key", s.DimensionKey),
					zap.Error(err),
				)
			}

			ds := DimensionStats{
				Key:          s.DimensionKey,
				Name:         dimensionNames[s.DimensionKey],
				AvgRating:    s.AvgRating,
				RatingCount:  s.RatingCount,
				Distribution: dist,
			}

			if s.TermID == nil {
				overall.Dimensions = append(overall.Dimensions, ds)
				continue
			}

			term, ok := byTerm[*s.TermID]
			if !ok {
				term = &TermRatingStats{TermID: *s.TermID, TermName: *s.TermID}
				byTerm[*s.TermID] = term
			}
			term.Dimensions = append(term.Dimensions, ds)
		}

		allKeys := make([]string, 0, len(allKeysSet))
		for k := range allKeysSet {
			allKeys = append(allKeys, k)
		}
		sort.Strings(allKeys)

		byTermList := make([]TermRatingStats, 0, len(byTerm))
		for _, v := range byTerm {
			byTermList = append(byTermList, *v)
		}

		return CourseRatingStatsResponse{
			CourseID:         courseID,
			Overall:          overall,
			ByTerm:           byTermList,
			AllDimensionKeys: allKeys,
		}, nil
	}, nil, "failed to load rating stats", "failed to load rating stats", nil)
}
