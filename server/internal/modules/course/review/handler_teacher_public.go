package review

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// ListTeachers 获取公开教师列表（分页、搜索、排序）
func (h *Handler) ListTeachers(c *gin.Context) {
	search := c.Query("q")
	departmentID, ok := httputil.ParseOptionalInt64Query(c, "departmentID")
	if !ok {
		response.BadRequest(c, "invalid departmentID")
		return
	}
	var deptPtr *int64
	if departmentID > 0 {
		deptPtr = &departmentID
	}

	sort := c.DefaultQuery("sort", "reviews")
	if sort != "reviews" && sort != "rating" && sort != "name" {
		sort = "reviews"
	}

	page, pageSize := httputil.ParsePage(c)

	ctx := c.Request.Context()
	cacheKey := h.cache.BuildVersionedKey(ctx, "review:teachers",
		"q="+httputil.SanitizeCacheKey(search)+
			":dept="+strconv.FormatInt(departmentID, 10)+
			":sort="+sort+
			":p="+strconv.Itoa(page)+
			":ps="+strconv.Itoa(pageSize))
	if cached, ok := h.cache.GetRaw(ctx, cacheKey); ok {
		response.Success(c, cached)
		return
	}

	list, total, err := h.service.ListTeachers(ctx, search, deptPtr, sort, page, pageSize)
	if err != nil {
		logger.FromGin(c).Error("failed to list teachers", zap.Error(err))
		response.InternalError(c, "failed to list teachers")
		return
	}
	if list == nil {
		list = []TeacherSummary{}
	}

	data := gin.H{"list": list, "total": total}
	if err := h.cache.Set(ctx, cacheKey, data, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, data)
}

// ListHotTeachers 获取热门教师列表
func (h *Handler) ListHotTeachers(c *gin.Context) {
	limit, ok := httputil.ParseOptionalIntQuery(c, "limit")
	if !ok {
		response.BadRequest(c, "invalid limit parameter")
		return
	}
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	ctx := c.Request.Context()
	cacheKey := h.cache.BuildVersionedKey(ctx, "review:hot_teachers", "limit="+strconv.Itoa(limit))
	if cached, ok := h.cache.GetRaw(ctx, cacheKey); ok {
		response.Success(c, cached)
		return
	}

	list, err := h.service.ListHotTeachers(ctx, limit)
	if err != nil {
		logger.FromGin(c).Error("failed to list hot teachers", zap.Error(err))
		response.InternalError(c, "failed to list hot teachers")
		return
	}
	if list == nil {
		list = []TeacherSummary{}
	}

	data := gin.H{"list": list}
	if err := h.cache.Set(ctx, cacheKey, data, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, data)
}
