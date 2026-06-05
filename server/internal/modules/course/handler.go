package course

import (
	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

// Handler 学习中心处理器
type Handler struct {
	cache   *cache.Helper
	service *Service
}

// NewHandler 创建处理器
func NewHandler(cacheHelper *cache.Helper, service *Service) *Handler {
	if cacheHelper == nil {
		panic("course.NewHandler: cacheHelper must not be nil")
	}
	if service == nil {
		panic("course.NewHandler: service must not be nil")
	}
	return &Handler{
		cache:   cacheHelper,
		service: service,
	}
}

// RegisterRoutes 注册学习中心路由
func (h *Handler) RegisterRoutes(
	r *gin.RouterGroup,
	authMiddleware gin.HandlerFunc,
	optionalAuthMiddleware gin.HandlerFunc,
	adminMiddlewares ...gin.HandlerFunc,
) {
	course := r.Group("/course")
	{
		// 通用实体接口（课程、院系等）
		course.GET("/departments", h.GetDepartments)
		course.GET("/terms", h.GetTerms)
		course.GET("/categories", h.GetCourseCategories)
		course.GET("/courses", optionalAuthMiddleware, middleware.RequireHealthyOptionalAuth(), h.GetCourses)
		course.GET("/courses/grouped", h.GetCoursesGrouped)
		course.GET("/courses/search", optionalAuthMiddleware, middleware.RequireHealthyOptionalAuth(), h.SearchCourses)
		course.GET("/courses/:courseID", optionalAuthMiddleware, middleware.RequireHealthyOptionalAuth(), h.GetCourse)
		course.GET("/stats", h.GetStats)
	}
}
