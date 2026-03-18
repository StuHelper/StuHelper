package review

import (
	"encoding/json"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// GetRatingDimensions 获取评分维度配置
func (h *Handler) GetRatingDimensions(c *gin.Context) {
	// 检查缓存
	cacheKey := h.cache.BuildVersionedKey(c.Request.Context(), "review:rating_dimensions", "all")
	if cached, ok := h.cache.GetRaw(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	// 调用 Service 层
	dimensions, err := h.service.GetRatingDimensions(c.Request.Context())
	if err != nil {
		logger.FromGin(c).Error("failed to load rating dimensions", zap.Error(err))
		response.InternalError(c, "failed to load rating dimensions")
		return
	}

	if err := h.cache.Set(c.Request.Context(), cacheKey, dimensions, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, dimensions)
}

// GetCourseRatingStats 获取课程评分统计（雷达图数据）
func (h *Handler) GetCourseRatingStats(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	// 检查缓存
	cacheKey := h.cache.BuildVersionedKey(c.Request.Context(), "review:rating_stats", strconv.FormatInt(courseID, 10))
	if cached, ok := h.cache.GetRaw(c.Request.Context(), cacheKey); ok {
		response.Success(c, cached)
		return
	}

	ctx := c.Request.Context()

	// 调用 Service 层获取维度名称
	dimensionNames, err := h.service.GetDimensionNames(ctx)
	if err != nil {
		logger.FromGin(c).Error("failed to load dimension names", zap.Error(err))
		response.InternalError(c, "failed to load dimension names")
		return
	}

	// 调用 Service 层获取评分统计
	stats, err := h.service.GetCourseRatingStats(ctx, courseID)
	if err != nil {
		logger.FromGin(c).Error("failed to load rating stats", zap.Error(err))
		response.InternalError(c, "failed to load rating stats")
		return
	}

	// 处理统计数据
	byTerm := make(map[string]*TermRatingStats)
	var overall TermRatingStats
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

	resp := CourseRatingStatsResponse{
		CourseID:         courseID,
		Overall:          overall,
		ByTerm:           byTermList,
		AllDimensionKeys: allKeys,
		RadarChart:       buildRadarChart(allKeys, dimensionNames, overall),
	}

	if err := h.cache.Set(ctx, cacheKey, resp, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, resp)
}

func buildRadarChart(keys []string, names map[string]string, overall TermRatingStats) RadarChartData {
	labels := make([]string, 0, len(keys))
	data := make([]float64, 0, len(keys))
	statMap := make(map[string]DimensionStats)
	for _, d := range overall.Dimensions {
		statMap[d.Key] = d
	}
	for _, k := range keys {
		labels = append(labels, names[k])
		data = append(data, statMap[k].AvgRating)
	}

	return RadarChartData{
		Labels: labels,
		Datasets: []RadarChartDataset{
			{
				Label:           "overall",
				Data:            data,
				BackgroundColor: radarBgColor,
				BorderColor:     radarBorderColor,
			},
		},
	}
}
