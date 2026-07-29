package course

import (
	"context"
	"errors"
	"sort"

	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
)

// 业务错误定义
var (
	ErrCourseNotFound  = errors.New("course not found")
	ErrInvalidCourseID = errors.New("invalid course id")
)

// Service 课程服务层
type Service struct {
	repo *Repository
	log  *zap.Logger
}

// NewService 创建课程服务
func NewService(repo *Repository, log *zap.Logger) *Service {
	if repo == nil {
		panic("course.NewService: repo must not be nil")
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{
		repo: repo,
		log:  log,
	}
}

// FavoriteExists 检查单个课程是否被用户收藏
func (s *Service) FavoriteExists(ctx context.Context, userHash string, courseID int64) (bool, error) {
	if err := validateCourseID(courseID); err != nil {
		return false, err
	}
	return s.repo.FavoriteExists(ctx, userHash, courseID)
}

// BatchFavoritedCourseIDs 批量查询用户已收藏的课程 ID 集合
func (s *Service) BatchFavoritedCourseIDs(ctx context.Context, userHash string, courseIDs []int64) (map[int64]bool, error) {
	if len(courseIDs) == 0 {
		return map[int64]bool{}, nil
	}
	if err := validateCourseIDs(courseIDs); err != nil {
		return nil, err
	}
	return s.repo.BatchFavoritedCourseIDs(ctx, userHash, courseIDs)
}

// GetDepartments 获取院系列表
func (s *Service) GetDepartments(ctx context.Context, category string) ([]Department, error) {
	return s.repo.ListDepartments(ctx, category)
}

func sortTermsForResponse(terms []Term) []Term {
	sorted := append([]Term(nil), terms...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].IsCurrent != sorted[j].IsCurrent {
			return sorted[i].IsCurrent
		}
		return sorted[i].ID > sorted[j].ID
	})
	return sorted
}

func (s *Service) GetTerms(ctx context.Context) ([]Term, error) {
	terms, err := s.repo.ListTerms(ctx)
	if err != nil {
		return nil, err
	}
	return sortTermsForResponse(terms), nil
}

// ListCoursesParams 获取课程列表参数
type ListCoursesParams struct {
	Query        string
	DepartmentID int64
	Category     string
	Sort         string
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
	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	list, total, err := s.repo.ListCourses(ctx, params.Query, params.DepartmentID, params.Category, normalizeCourseSort(params.Sort), pageSize, offset)
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
	query := params.Query

	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	list, total, err := s.repo.SearchCourses(ctx, query, pageSize, offset)
	if err != nil {
		s.log.Error("failed to search courses", zap.String("query", params.Query), zap.Error(err))
		return nil, err
	}

	return &ListCoursesResult{List: list, Total: total}, nil
}

// GetCourse 获取课程详情
func (s *Service) GetCourse(ctx context.Context, id int64) (*Course, error) {
	if err := validateCourseID(id); err != nil {
		return nil, err
	}
	course, err := s.repo.GetCourseByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrCourseNotFound
		}
		s.log.Error("failed to get course", zap.Int64("id", id), zap.Error(err))
		return nil, err
	}
	return course, nil
}

func validateCourseID(id int64) error {
	if id <= 0 {
		return ErrInvalidCourseID
	}
	return nil
}

func validateCourseIDs(ids []int64) error {
	for _, id := range ids {
		if err := validateCourseID(id); err != nil {
			return err
		}
	}
	return nil
}

// GetCourseCategories 获取课程分类列表
func (s *Service) GetCourseCategories(ctx context.Context) ([]CourseCategory, error) {
	return s.repo.ListCourseCategories(ctx)
}

// GetCoursesGrouped 按院系分组返回全部课程（面向课程目录页）
func (s *Service) GetCoursesGrouped(ctx context.Context) ([]DepartmentGroup, error) {
	groups, err := s.repo.ListCoursesGroupedByDepartment(ctx)
	if err != nil {
		s.log.Error("获取按院系分组课程失败", zap.Error(err))
		return nil, err
	}
	return groups, nil
}

// StatsResult 统计结果
type StatsResult struct {
	CourseCount     int
	DepartmentCount int
}

// GetStats 获取统计数据
func (s *Service) GetStats(ctx context.Context) (*StatsResult, error) {
	courseCount, departmentCount, err := s.repo.CountStats(ctx)
	if err != nil {
		s.log.Error("failed to count stats", zap.Error(err))
		return nil, err
	}

	return &StatsResult{
		CourseCount:     courseCount,
		DepartmentCount: departmentCount,
	}, nil
}
