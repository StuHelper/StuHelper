package review

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"

	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/sanitizer"
)

// maskHash 返回哈希值的前 12 个字符，用于日志脱敏，防止跨日志条目追踪用户
func maskHash(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12] + "..."
}

// 业务错误定义
var (
	ErrCourseNotFound       = errors.New("course not found")
	ErrReviewNotFound       = errors.New("review not found")
	ErrTeacherNotFound      = errors.New("teacher not found")
	ErrAlreadyVoted         = errors.New("already voted")
	ErrAlreadyReviewed      = errors.New("already reviewed this course")
	ErrDangerousContent     = errors.New("content contains dangerous elements")
	ErrSensitiveContent     = errors.New("content contains sensitive words")
	ErrTitleEmpty           = errors.New("title cannot be empty after sanitization")
	ErrContentEmpty         = errors.New("content cannot be empty after sanitization")
	ErrInvalidRating        = errors.New("invalid rating value")
	ErrRatingRequired       = errors.New("at least one rating dimension is required")
	ErrNotReviewOwner       = errors.New("not the review owner")
	ErrAlreadyReported      = errors.New("already reported this review")
	ErrReportNotFound       = errors.New("report not found")
	ErrDraftNotFound        = errors.New("draft not found")
	ErrNotificationNotFound = errors.New("notification not found")
	ErrInvalidAction        = errors.New("invalid action")
	ErrInvalidTransition    = errors.New("invalid status transition")
)

// Service 评课服务层
type Service struct {
	db             *db.DB
	repo           *Repository
	filter         *Filter
	dimensionCache atomic.Value // map[string]string
}

// NewService 创建评课服务
func NewService(database *db.DB, repo *Repository) *Service {
	filter := NewFilter(repo)
	s := &Service{
		db:     database,
		repo:   repo,
		filter: filter,
	}
	// 初始化时加载维度缓存
	if err := s.refreshDimensionCache(context.Background()); err != nil {
		logger.L().Warn("failed to initialize dimension cache", zap.Error(err))
	}
	return s
}

// refreshDimensionCache 刷新评分维度缓存
func (s *Service) refreshDimensionCache(ctx context.Context) error {
	dims, err := s.repo.GetDimensionNames(ctx)
	if err != nil {
		return err
	}
	s.dimensionCache.Store(dims)
	return nil
}

// getDimensionNames 获取缓存的评分维度列表
func (s *Service) getDimensionNames() map[string]string {
	if v := s.dimensionCache.Load(); v != nil {
		return v.(map[string]string) //nolint:errcheck // typed by atomic.Store
	}
	return nil
}

// validTermID 学期 ID 格式校验：如 "2024-1"（春季）或 "2024-2"（秋季）
var validTermIDFormat = regexp.MustCompile(`^\d{4}-[12]$`)

// validateAndSanitizeReview 校验评分、清洗标题/内容、检测危险内容和敏感词
// 返回清洗后的 title、content，调用方应使用返回值覆盖原始参数
// validRatingKey 评分维度 key 仅允许小写字母、数字、下划线（对齐 DB VARCHAR(50)）
var validRatingKey = regexp.MustCompile(`^[a-z][a-z0-9_]{0,49}$`)

func (s *Service) validateAndSanitizeReview(ctx context.Context, ratings ReviewRatings, title, content, termID string) (string, string, error) {
	// L-20: 校验 term_id 格式
	if termID != "" && !validTermIDFormat.MatchString(termID) {
		return "", "", fmt.Errorf("invalid term_id format, expected YYYY-S (e.g. 2024-1)")
	}

	if len(ratings) == 0 {
		return "", "", ErrRatingRequired
	}

	// 从缓存获取有效的评分维度 key 白名单
	validKeys := s.getDimensionNames()
	if validKeys == nil {
		// 缓存未初始化，回退到数据库查询
		var err error
		validKeys, err = s.repo.GetDimensionNames(ctx)
		if err != nil {
			return "", "", fmt.Errorf("failed to load rating dimensions: %w", err)
		}
	}

	for k, v := range ratings {
		if !validRatingKey.MatchString(k) {
			return "", "", ErrInvalidRating
		}
		// L-25: 校验 key 是否在 rating_dimensions 表中存在
		if _, ok := validKeys[k]; !ok {
			return "", "", fmt.Errorf("%w: unknown dimension key %q", ErrInvalidRating, k)
		}
		if v < 1 || v > 5 {
			return "", "", ErrInvalidRating
		}
	}

	title = sanitizer.SanitizeTitle(title)
	content = sanitizer.SanitizeText(content)

	if strings.TrimSpace(title) == "" {
		return "", "", ErrTitleEmpty
	}
	if sanitizer.ContainsDangerousContent(title) || sanitizer.ContainsDangerousContent(content) {
		return "", "", ErrDangerousContent
	}
	if strings.TrimSpace(content) == "" {
		return "", "", ErrContentEmpty
	}

	checkResult := s.filter.CheckContent(ctx, title+" "+content)
	if !checkResult.IsValid {
		return "", "", ErrSensitiveContent
	}

	return title, content, nil
}

// GetCourseReviewsParams 获取课程评论参数
type GetCourseReviewsParams struct {
	CourseID  int64
	Page      int
	PageSize  int
	Sort      string // time, likes, rating
	TermID    string
	TeacherID *int64
}

// GetCourseReviewsResult 获取课程评论结果
type GetCourseReviewsResult struct {
	List  []Review
	Total int
}

// StatsResult 统计结果
type StatsResult struct {
	CourseCount     int
	ReviewCount     int
	DepartmentCount int
}

// GetCourseReviews 获取课程评论列表
func (s *Service) GetCourseReviews(ctx context.Context, params GetCourseReviewsParams) (*GetCourseReviewsResult, error) {
	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	list, total, err := s.repo.ListByCourseWithSort(ctx, ListByCourseWithSortParams{
		CourseID:  params.CourseID,
		Sort:      params.Sort,
		TermID:    params.TermID,
		TeacherID: params.TeacherID,
		Limit:     pageSize,
		Offset:    offset,
	})
	if err != nil {
		return nil, err
	}

	return &GetCourseReviewsResult{List: list, Total: total}, nil
}

// GetBatchCourseReviewsParams 批量获取课程测评参数
type GetBatchCourseReviewsParams struct {
	CourseIDs []int64
	PageSize  int
	Sort      string
}

// BatchCourseReviewsResult 批量课程测评结果（按课程ID分组）
type BatchCourseReviewsResult struct {
	Reviews map[int64][]Review
	Totals  map[int64]int
}

// GetBatchCourseReviews 批量获取多个课程的测评列表
func (s *Service) GetBatchCourseReviews(ctx context.Context, params GetBatchCourseReviewsParams) (*BatchCourseReviewsResult, error) {
	pageSize := httputil.ClampPageSize(params.PageSize)
	reviews, totals, err := s.repo.ListByMultipleCourses(ctx, params.CourseIDs, params.Sort, pageSize)
	if err != nil {
		return nil, err
	}
	return &BatchCourseReviewsResult{Reviews: reviews, Totals: totals}, nil
}

// CheckCourseExists 检查课程是否存在
func (s *Service) CheckCourseExists(ctx context.Context, courseID int64) (bool, error) {
	return s.repo.CourseExists(ctx, courseID)
}

// GetLatestReviewsParams 获取最新评论参数
type GetLatestReviewsParams struct {
	Page     int
	PageSize int
	Sort     string
}

// GetLatestReviews 获取最新评论列表
func (s *Service) GetLatestReviews(ctx context.Context, params GetLatestReviewsParams) (*GetCourseReviewsResult, error) {
	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	list, total, err := s.repo.ListLatest(ctx, pageSize, offset, params.Sort)
	if err != nil {
		return nil, err
	}

	return &GetCourseReviewsResult{List: list, Total: total}, nil
}

// GetStats 获取统计数据
func (s *Service) GetStats(ctx context.Context) (*StatsResult, error) {
	courseCount, reviewCount, departmentCount, err := s.repo.GetPortalStats(ctx)
	if err != nil {
		return nil, err
	}
	return &StatsResult{
		CourseCount:     courseCount,
		ReviewCount:     reviewCount,
		DepartmentCount: departmentCount,
	}, nil
}

// GetRatingDimensions 获取评分维度列表
func (s *Service) GetRatingDimensions(ctx context.Context) ([]RatingDimension, error) {
	return s.repo.ListRatingDimensions(ctx)
}

// GetDimensionNames 获取维度名称映射
func (s *Service) GetDimensionNames(ctx context.Context) (map[string]string, error) {
	return s.repo.GetDimensionNames(ctx)
}

// GetCourseRatingStats 获取课程评分统计
func (s *Service) GetCourseRatingStats(ctx context.Context, courseID int64) ([]RatingStatRow, error) {
	return s.repo.ListCourseRatingStats(ctx, courseID)
}

// GetCourseTeachers 获取课程的授课教师列表
func (s *Service) GetCourseTeachers(ctx context.Context, courseID int64) ([]CourseTeacherStats, error) {
	return s.repo.ListCourseTeachers(ctx, courseID)
}

// GetReviewByID 根据 ID 获取评论
func (s *Service) GetReviewByID(ctx context.Context, reviewID string) (*Review, error) {
	return s.repo.GetReviewByID(ctx, reviewID)
}

// CheckContent 检查内容是否包含敏感词
func (s *Service) CheckContent(ctx context.Context, content string) *ContentCheckResult {
	return s.filter.CheckContent(ctx, content)
}

// --- 敏感词管理 pass-through ---

// ListSensitiveWords returns a paginated list of sensitive words (pass-through to Repository).
func (s *Service) ListSensitiveWords(ctx context.Context, category, level string, limit, offset int) ([]SensitiveWord, int, error) {
	return s.repo.ListSensitiveWords(ctx, category, level, limit, offset)
}

// CreateSensitiveWord creates a new sensitive word entry (pass-through to Repository).
func (s *Service) CreateSensitiveWord(ctx context.Context, word, category, level string) (SensitiveWord, error) {
	return s.repo.CreateSensitiveWord(ctx, word, category, level)
}

// UpdateSensitiveWord updates an existing sensitive word (pass-through to Repository).
func (s *Service) UpdateSensitiveWord(ctx context.Context, wordID string, word, category, level *string, isActive *bool) error {
	return s.repo.UpdateSensitiveWord(ctx, wordID, word, category, level, isActive)
}

// DeleteSensitiveWord deletes a sensitive word by ID (pass-through to Repository).
func (s *Service) DeleteSensitiveWord(ctx context.Context, wordID string) error {
	return s.repo.DeleteSensitiveWord(ctx, wordID)
}

// --- 教师管理 pass-through ---

// ListAdminTeachers returns a paginated list of teachers for admin management (pass-through to Repository).
func (s *Service) ListAdminTeachers(ctx context.Context, search string, departmentID int64, limit, offset int) ([]AdminTeacher, int, error) {
	return s.repo.ListAdminTeachers(ctx, search, departmentID, limit, offset)
}

// CreateTeacher creates a new teacher record (pass-through to Repository).
func (s *Service) CreateTeacher(ctx context.Context, name string, departmentID *int64) (*AdminTeacher, error) {
	return s.repo.CreateTeacher(ctx, name, departmentID)
}

// UpdateTeacher updates an existing teacher's information (pass-through to Repository).
func (s *Service) UpdateTeacher(ctx context.Context, id int64, name string, departmentID *int64) error {
	return s.repo.UpdateTeacher(ctx, id, name, departmentID)
}

// DeleteTeacher deletes a teacher by ID (pass-through to Repository).
func (s *Service) DeleteTeacher(ctx context.Context, id int64) error {
	return s.repo.DeleteTeacher(ctx, id)
}
