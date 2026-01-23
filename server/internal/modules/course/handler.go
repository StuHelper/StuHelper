package course

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler 课程评价处理器
type Handler struct {
	// TODO: 添加数据库连接
}

// NewHandler 创建处理器
func NewHandler() *Handler {
	return &Handler{}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	course := r.Group("/course-review")
	{
		// 评分维度配置
		course.GET("/rating-dimensions", h.GetRatingDimensions)

		// 院系和课程
		course.GET("/departments", h.GetDepartments)
		course.GET("/courses", h.GetCourses)
		course.GET("/courses/search", h.SearchCourses)
		course.GET("/courses/:id", h.GetCourse)
		course.GET("/courses/:id/rating-stats", h.GetCourseRatingStats)

		// 测评
		course.GET("/courses/:id/reviews", h.GetCourseReviews)
		course.GET("/reviews/latest", h.GetLatestReviews)
		course.POST("/reviews", h.PostReview)
		course.POST("/reviews/:id/vote", h.VoteReview)

		// 统计
		course.GET("/stats", h.GetStats)
	}
}

// GetRatingDimensions 获取评分维度配置
func (h *Handler) GetRatingDimensions(c *gin.Context) {
	// TODO: 从数据库获取启用的维度配置
	// 目前返回默认维度
	dimensions := []RatingDimension{
		{ID: 1, Key: "overall", Name: "总体评价", Description: "对课程的整体评价", SortOrder: 1, IsActive: true},
		{ID: 2, Key: "content", Name: "内容质量", Description: "课程内容的深度和实用性", SortOrder: 2, IsActive: true},
		{ID: 3, Key: "workload", Name: "工作量", Description: "作业、项目等课业负担", SortOrder: 3, IsActive: true},
		{ID: 4, Key: "grading", Name: "考核/给分", Description: "考核方式和给分情况", SortOrder: 4, IsActive: true},
		{ID: 5, Key: "attendance", Name: "考勤", Description: "点名、签到等考勤要求", SortOrder: 5, IsActive: true},
	}

	c.JSON(http.StatusOK, gin.H{
		"data": dimensions,
	})
}

// GetDepartments 获取院系列表
func (h *Handler) GetDepartments(c *gin.Context) {
	category := c.Query("category")
	_ = category // TODO: 按分类筛选

	// TODO: 从数据库获取
	c.JSON(http.StatusOK, gin.H{
		"data": []Department{},
	})
}

// GetCourses 获取课程列表
func (h *Handler) GetCourses(c *gin.Context) {
	// TODO: 从数据库获取
	c.JSON(http.StatusOK, gin.H{
		"data": []Course{},
	})
}

// SearchCourses 搜索课程
func (h *Handler) SearchCourses(c *gin.Context) {
	q := c.Query("q")
	_ = q // TODO: 搜索

	c.JSON(http.StatusOK, gin.H{
		"data": []Course{},
	})
}

// GetCourse 获取课程详情
func (h *Handler) GetCourse(c *gin.Context) {
	// TODO: 从数据库获取
	c.JSON(http.StatusOK, gin.H{
		"data": nil,
	})
}

// GetCourseRatingStats 获取课程评分统计（雷达图数据）
func (h *Handler) GetCourseRatingStats(c *gin.Context) {
	// TODO: 从数据库获取所有历史维度的评分统计
	// 1. 查询该课程所有出现过的维度
	// 2. 按学期分组统计
	// 3. 生成雷达图数据

	// 示例响应
	response := CourseRatingStatsResponse{
		CourseID: 1,
		AllDimensionKeys: []string{"overall", "content", "workload", "grading", "attendance"},
		Overall: TermRatingStats{
			TermName: "总体",
			Dimensions: []DimensionStats{
				{Key: "overall", Name: "总体评价", AvgRating: 4.2, RatingCount: 100},
				{Key: "content", Name: "内容质量", AvgRating: 4.5, RatingCount: 100},
				{Key: "workload", Name: "工作量", AvgRating: 3.8, RatingCount: 100},
				{Key: "grading", Name: "考核/给分", AvgRating: 4.0, RatingCount: 100},
				{Key: "attendance", Name: "考勤", AvgRating: 3.2, RatingCount: 100},
			},
		},
		ByTerm: []TermRatingStats{},
		RadarChart: RadarChartData{
			Labels: []string{"总体评价", "内容质量", "工作量", "考核/给分", "考勤"},
			Datasets: []RadarChartDataset{
				{
					Label:           "总体",
					Data:            []float64{4.2, 4.5, 3.8, 4.0, 3.2},
					BackgroundColor: "rgba(64, 158, 255, 0.2)",
					BorderColor:     "#409EFF",
				},
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}

// GetCourseReviews 获取课程测评列表
func (h *Handler) GetCourseReviews(c *gin.Context) {
	// TODO: 从数据库获取
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"list":  []Review{},
			"total": 0,
		},
	})
}

// GetLatestReviews 获取最新测评
func (h *Handler) GetLatestReviews(c *gin.Context) {
	// TODO: 从数据库获取
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"list":  []Review{},
			"total": 0,
		},
	})
}

// PostReviewRequest 发布测评请求
type PostReviewRequest struct {
	CourseID  int64         `json:"course_id" binding:"required"`
	TeacherID *int64        `json:"teacher_id"`
	TermID    string        `json:"term_id"`
	Title     string        `json:"title" binding:"max=200"`
	Content   string        `json:"content" binding:"required,min=10,max=5000"`
	Grade     string        `json:"grade"`
	Ratings   ReviewRatings `json:"ratings" binding:"required"`
}

// PostReview 发布测评
func (h *Handler) PostReview(c *gin.Context) {
	var req PostReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证评分维度
	if len(req.Ratings) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少需要一个评分维度"})
		return
	}
	for _, v := range req.Ratings {
		if v < 1 || v > 5 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "评分必须在1-5之间"})
			return
		}
	}

	// TODO: 保存到数据库
	c.JSON(http.StatusOK, gin.H{
		"message": "发布成功",
	})
}

// VoteReview 投票
func (h *Handler) VoteReview(c *gin.Context) {
	var req struct {
		VoteType string `json:"vote_type" binding:"required,oneof=like dislike"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: 保存到数据库
	c.JSON(http.StatusOK, gin.H{
		"message": "投票成功",
	})
}

// GetStats 获取统计数据
func (h *Handler) GetStats(c *gin.Context) {
	// TODO: 从数据库获取
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"courseCount":     0,
			"reviewCount":     0,
			"departmentCount": 0,
		},
	})
}
