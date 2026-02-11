package course

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// 业务错误定义
var (
	ErrCourseNotFound = errors.New("course not found")
)

// Service 课程服务层
type Service struct {
	db   *db.DB
	repo *Repository
	log  *zap.Logger
}

// NewService 创建课程服务
func NewService(database *db.DB, repo *Repository) *Service {
	return &Service{
		db:   database,
		repo: repo,
		log:  logger.L(),
	}
}

// GetDepartments 获取院系列表
func (s *Service) GetDepartments(ctx context.Context, category string) ([]Department, error) {
	return s.repo.ListDepartments(ctx, category)
}

// ListCoursesParams 获取课程列表参数
type ListCoursesParams struct {
	DepartmentID int64
	Page         int
	PageSize     int
}

// ListCoursesResult 获取课程列表结果
type ListCoursesResult struct {
	List  []Course
	Total int
}

// GetCourses 获取课程列表
func (s *Service) GetCourses(ctx context.Context, params ListCoursesParams) (*ListCoursesResult, error) {
	offset := (params.Page - 1) * params.PageSize
	list, total, err := s.repo.ListCourses(ctx, params.DepartmentID, params.PageSize, offset)
	if err != nil {
		s.log.Error("failed to list courses", zap.Error(err))
		return nil, err
	}

	return &ListCoursesResult{List: list, Total: total}, nil
}

// SearchCoursesParams 搜索课程参数
type SearchCoursesParams struct {
	Query    string
	Page     int
	PageSize int
}

// SearchCourses 搜索课程
func (s *Service) SearchCourses(ctx context.Context, params SearchCoursesParams) (*ListCoursesResult, error) {
	pattern := "%" + escapeLikePattern(params.Query) + "%"

	offset := (params.Page - 1) * params.PageSize
	list, total, err := s.repo.SearchCourses(ctx, pattern, params.PageSize, offset)
	if err != nil {
		s.log.Error("failed to search courses", zap.String("query", params.Query), zap.Error(err))
		return nil, err
	}

	return &ListCoursesResult{List: list, Total: total}, nil
}

// GetCourse 获取课程详情
func (s *Service) GetCourse(ctx context.Context, id int64) (*Course, error) {
	course, err := s.repo.GetCourseByID(ctx, id)
	if err != nil {
		s.log.Error("failed to get course", zap.Int64("id", id), zap.Error(err))
		return nil, err
	}
	if course == nil {
		return nil, ErrCourseNotFound
	}
	return course, nil
}

// StatsResult 统计结果
type StatsResult struct {
	CourseCount     int
	DepartmentCount int
}

// GetStats 获取统计数据
func (s *Service) GetStats(ctx context.Context) (*StatsResult, error) {
	courseCount, err := s.repo.CountCourses(ctx, 0)
	if err != nil {
		s.log.Error("failed to count courses", zap.Error(err))
		return nil, err
	}

	departmentCount, err := s.repo.CountDepartments(ctx)
	if err != nil {
		s.log.Error("failed to count departments", zap.Error(err))
		return nil, err
	}

	return &StatsResult{
		CourseCount:     courseCount,
		DepartmentCount: departmentCount,
	}, nil
}
