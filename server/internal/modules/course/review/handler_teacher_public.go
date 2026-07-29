package review

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

type teacherListCachePayload struct {
	List  []TeacherSummary
	Total int
}

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
	respondWithCachedData(h,
		c,
		teacherPublicListCacheKey,
		"q="+httputil.SanitizeCacheKey(search)+
			":dept="+strconv.FormatInt(departmentID, 10)+
			":sort="+sort+
			":p="+strconv.Itoa(page)+
			":ps="+strconv.Itoa(pageSize),
		func(ctx context.Context) (*teacherListCachePayload, error) {
			list, total, err := h.service.ListTeachers(ctx, search, deptPtr, sort, page, pageSize)
			if err != nil {
				return nil, err
			}
			if list == nil {
				list = []TeacherSummary{}
			}
			return &teacherListCachePayload{List: list, Total: total}, nil
		},
		func(result *teacherListCachePayload) any {
			return gin.H{"list": result.List, "total": result.Total}
		},
		"failed to list teachers",
		"failed to list teachers",
		nil,
	)
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

	respondWithCachedData(h, c, teacherPublicHotCacheKey, "limit="+strconv.Itoa(limit), func(ctx context.Context) ([]TeacherSummary, error) {
		list, err := h.service.ListHotTeachers(ctx, limit)
		if err != nil {
			return nil, err
		}
		if list == nil {
			return []TeacherSummary{}, nil
		}
		return list, nil
	}, func(list []TeacherSummary) any {
		return gin.H{"list": list}
	}, "failed to list hot teachers", "failed to list hot teachers", nil)
}
