package review

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// GetRatingDimensions 获取评分维度配置
func (h *Handler) GetRatingDimensions(c *gin.Context) {
	cacheKey := "review:rating_dimensions"
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		c.JSON(http.StatusOK, gin.H{"data": cached})
		return
	}

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT id, school_id, key, name, description, sort_order, is_active, created_at, updated_at
		FROM rating_dimensions
		WHERE is_active = true
		ORDER BY sort_order ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load rating dimensions"})
		return
	}
	defer rows.Close()

	dimensions := make([]RatingDimension, 0)
	for rows.Next() {
		var d RatingDimension
		if err := rows.Scan(
			&d.ID, &d.SchoolID, &d.Key, &d.Name, &d.Description,
			&d.SortOrder, &d.IsActive, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse rating dimensions"})
			return
		}
		dimensions = append(dimensions, d)
	}

	_ = h.setCache(c.Request.Context(), cacheKey, dimensions, cacheTTL)
	c.JSON(http.StatusOK, gin.H{"data": dimensions})
}

// GetCourseRatingStats 获取课程评分统计（雷达图数据）
func (h *Handler) GetCourseRatingStats(c *gin.Context) {
	courseID, err := parseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	cacheKey := "review:rating_stats:" + strconv.FormatInt(courseID, 10)
	if cached, ok := h.getCache(c.Request.Context(), cacheKey); ok {
		c.JSON(http.StatusOK, gin.H{"data": cached})
		return
	}

	ctx := c.Request.Context()
	dimensionNames, err := h.loadDimensionNames(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load dimension names"})
		return
	}

	rows, err := h.db.Query(ctx, `
		SELECT term_id, dimension_key, avg_rating, rating_count, rating_dist
		FROM course_rating_stats
		WHERE course_id = $1
		ORDER BY term_id NULLS FIRST
	`, courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load rating stats"})
		return
	}
	defer rows.Close()

	byTerm := make(map[string]*TermRatingStats)
	var overall TermRatingStats
	allKeysSet := make(map[string]bool)

	for rows.Next() {
		var termID *string
		var key string
		var avg float64
		var count int
		var distJSON []byte
		if err := rows.Scan(&termID, &key, &avg, &count, &distJSON); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse rating stats"})
			return
		}

		allKeysSet[key] = true
		dist := map[int]int{}
		if err := json.Unmarshal(distJSON, &dist); err != nil {
			logger.L().Warn("failed to unmarshal rating distribution",
				zap.String("dimension_key", key),
				zap.Error(err),
			)
		}

		ds := DimensionStats{
			Key:          key,
			Name:         dimensionNames[key],
			AvgRating:    avg,
			RatingCount:  count,
			Distribution: dist,
		}

		if termID == nil {
			overall.Dimensions = append(overall.Dimensions, ds)
			continue
		}

		term, ok := byTerm[*termID]
		if !ok {
			term = &TermRatingStats{TermID: *termID, TermName: *termID}
			byTerm[*termID] = term
		}
		term.Dimensions = append(term.Dimensions, ds)
	}

	allKeys := make([]string, 0, len(allKeysSet))
	for k := range allKeysSet {
		allKeys = append(allKeys, k)
	}

	byTermList := make([]TermRatingStats, 0, len(byTerm))
	for _, v := range byTerm {
		byTermList = append(byTermList, *v)
	}

	response := CourseRatingStatsResponse{
		CourseID:         courseID,
		Overall:          overall,
		ByTerm:           byTermList,
		AllDimensionKeys: allKeys,
		RadarChart:       buildRadarChart(allKeys, dimensionNames, overall),
	}

	_ = h.setCache(ctx, cacheKey, response, cacheTTL)
	c.JSON(http.StatusOK, gin.H{"data": response})
}

func (h *Handler) loadDimensionNames(ctx context.Context) (map[string]string, error) {
	cacheKey := "review:dimension_names"
	if cached, ok := h.getCache(ctx, cacheKey); ok {
		if m, ok := cached.(map[string]interface{}); ok {
			result := make(map[string]string, len(m))
			for k, v := range m {
				if s, ok := v.(string); ok {
					result[k] = s
				}
			}
			return result, nil
		}
	}

	rows, err := h.db.Query(ctx, `SELECT key, name FROM rating_dimensions WHERE is_active = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]string{}
	for rows.Next() {
		var key, name string
		if err := rows.Scan(&key, &name); err != nil {
			return nil, err
		}
		result[key] = name
	}

	_ = h.setCache(ctx, cacheKey, result, 30*time.Minute)
	return result, nil
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
				Label:           "总体",
				Data:            data,
				BackgroundColor: "rgba(64, 158, 255, 0.2)",
				BorderColor:     "#409EFF",
			},
		},
	}
}
