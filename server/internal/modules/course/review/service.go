package review

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/reviewaccess"
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
	ErrCourseNotFound            = errors.New("course not found")
	ErrReviewNotFound            = errors.New("review not found")
	ErrTeacherNotFound           = errors.New("teacher not found")
	ErrTeacherNameInvalid        = errors.New("teacher name is invalid")
	ErrTeacherDepartmentRequired = errors.New("teacher department is required")
	ErrTeacherDepartmentNotFound = errors.New("teacher department not found")
	ErrTeacherHasReviews         = errors.New("teacher has associated reviews")
	ErrAlreadyVoted              = errors.New("already voted")
	ErrAlreadyReviewed           = errors.New("already reviewed this course")
	ErrDangerousContent          = errors.New("content contains dangerous elements")
	ErrSensitiveContent          = errors.New("content contains sensitive words")
	ErrSensitiveWordInvalid      = errors.New("sensitive word input is invalid")
	ErrModerationUnavailable     = errors.New("content moderation unavailable")
	ErrInvalidTermID             = errors.New("invalid term_id format, expected YYYY-S (e.g. 2024-1)")
	ErrTitleEmpty                = errors.New("title cannot be empty after sanitization")
	ErrTitleTooLong              = errors.New("title too long")
	ErrContentEmpty              = errors.New("content cannot be empty after sanitization")
	ErrContentTooShort           = errors.New("content too short")
	ErrReasonTooLong             = errors.New("reason too long")
	ErrInvalidRating             = errors.New("invalid rating value")
	ErrInvalidGrade              = errors.New("invalid grade")
	ErrRatingRequired            = errors.New("at least one rating dimension is required")
	ErrNotReviewOwner            = errors.New("not the review owner")
	ErrAlreadyReported           = errors.New("already reported this review")
	ErrReportNotFound            = errors.New("report not found")
	ErrDraftNotFound             = errors.New("draft not found")
	ErrInvalidAction             = errors.New("invalid action")
	ErrInvalidTransition         = errors.New("invalid status transition")
	ErrUserIdentityRequired      = errors.New("internal user identity required")
	ErrAdminIdentityRequired     = errors.New("admin identity required")
)

// Service 评课服务层
type Service struct {
	db             *db.DB
	repo           *Repository
	filter         *Filter
	dimensionCache atomic.Value // map[string]string
	fgaWriter      reviewFGAWriter
	notifSender    ReviewNotificationSender
	accessReader   ReviewAccessReader
	accessPolicySF singleflight.Group
	asyncCtx       context.Context
	asyncLaunch    func(string, func(context.Context))
}

type ReviewNotification struct {
	UserID       int64
	Type         string
	Title        string
	Body         string
	SourceModule string
	SourceID     string
	CourseID     int64
}

type ReviewNotificationSender interface {
	SendReviewNotification(ctx context.Context, params ReviewNotification) error
}

// ReviewAccessReader 访问控制策略数据源（由 user.Repository 实现）。
type ReviewAccessReader interface {
	ListReviewAccessSchoolConfigs(ctx context.Context) ([]reviewaccess.SchoolConfig, error)
	ListReviewAccessSystemConfigs(ctx context.Context) ([]reviewaccess.SystemConfig, error)
	GetReviewAccessSubject(ctx context.Context, externalSubject string) (*reviewaccess.Subject, error)
}

type ServiceOption func(*serviceOptions)

type serviceOptions struct {
	initialCacheContext context.Context
}

func defaultServiceOptions() serviceOptions {
	return serviceOptions{
		initialCacheContext: context.Background(),
	}
}

// WithInitialCacheContext sets the context used for constructor-time cache warmup.
func WithInitialCacheContext(ctx context.Context) ServiceOption {
	return func(opts *serviceOptions) {
		if ctx != nil {
			opts.initialCacheContext = ctx
		}
	}
}

// NewService 创建评课服务
func NewService(
	database *db.DB,
	repo *Repository,
	notifSender ReviewNotificationSender,
	fgaWriter reviewFGAWriter,
	accessReader ReviewAccessReader,
	options ...ServiceOption,
) *Service {
	if database == nil {
		panic("review.NewService: database must not be nil")
	}
	if repo == nil {
		panic("review.NewService: repo must not be nil")
	}
	if fgaWriter == nil {
		panic("review.NewService: fgaWriter must not be nil")
	}
	if notifSender == nil {
		panic("review.NewService: notifSender must not be nil")
	}
	if accessReader == nil {
		panic("review.NewService: accessReader must not be nil")
	}
	opts := defaultServiceOptions()
	for _, option := range options {
		if option != nil {
			option(&opts)
		}
	}

	filter := NewFilter(repo)
	s := &Service{
		db:           database,
		repo:         repo,
		filter:       filter,
		fgaWriter:    fgaWriter,
		notifSender:  notifSender,
		accessReader: accessReader,
	}
	// 初始化时加载维度缓存
	if err := s.refreshDimensionCache(opts.initialCacheContext); err != nil {
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

// RefreshTeacherPublicStats 刷新公开教师统计物化视图。
func (s *Service) RefreshTeacherPublicStats(ctx context.Context) error {
	return s.repo.RefreshTeacherPublicStats(ctx)
}

// validTermID 学期 ID 格式校验：如 "2024-1"（春季）或 "2024-2"（秋季）
var validTermIDFormat = regexp.MustCompile(`^\d{4}-[12]$`)

// validateAndSanitizeReview 校验评分、清洗标题/内容，并根据敏感词命中结果返回最终审核决策。
// review 级别内容进入 pending_review；warn 级别内容保持 published 并打 content_flag；
// block 级别内容直接拒绝。
// validRatingKey 评分维度 key 仅允许小写字母、数字、下划线（对齐 DB VARCHAR(50)）
var validRatingKey = regexp.MustCompile(`^[a-z][a-z0-9_]{0,49}$`)

const (
	minReviewContentRunes    = 10
	minAdminEditContentRunes = 1
	maxReviewTitleRunes      = 200
	maxReviewContentRunes    = 5000
	maxAdminEditReasonRunes  = 500
	maxAdminTeacherNameRunes = 100
)

func (s *Service) validateAndSanitizeReview(ctx context.Context, ratings ReviewRatings, title, content, termID string) (string, string, string, *string, error) {
	// 校验 term_id 格式
	if termID != "" && !validTermIDFormat.MatchString(termID) {
		return "", "", "", nil, ErrInvalidTermID
	}

	if err := s.validateRatingValues(ctx, ratings, true); err != nil {
		return "", "", "", nil, err
	}

	if err := ensureSafeReviewText(title, content); err != nil {
		return "", "", "", nil, err
	}

	title = sanitizer.SanitizeTitle(title)
	content = sanitizer.SanitizeText(content)

	if strings.TrimSpace(title) == "" {
		return "", "", "", nil, ErrTitleEmpty
	}
	if err := ensureSafeReviewText(title, content); err != nil {
		return "", "", "", nil, err
	}
	if strings.TrimSpace(content) == "" {
		return "", "", "", nil, ErrContentEmpty
	}
	if err := validateReviewTextLengths(title, content, minReviewContentRunes); err != nil {
		return "", "", "", nil, err
	}

	checkResult, err := s.filter.CheckContent(ctx, title+" "+content)
	if err != nil {
		return "", "", "", nil, err
	}

	decision, err := buildReviewModerationDecision(checkResult)
	if err != nil {
		return "", "", "", nil, err
	}

	return title, content, decision.Status, decision.ContentFlag, nil
}

func ensureSafeReviewText(title, content string) error {
	if sanitizer.ContainsDangerousContent(title) || sanitizer.ContainsDangerousContent(content) {
		return ErrDangerousContent
	}
	return nil
}

func normalizeRequiredUserHash(userHash string) (string, error) {
	userHash = strings.TrimSpace(userHash)
	if userHash == "" {
		return "", ErrUserIdentityRequired
	}
	return userHash, nil
}

func normalizeRequiredAdminID(adminID string) (string, error) {
	adminID = strings.TrimSpace(adminID)
	if adminID == "" {
		return "", ErrAdminIdentityRequired
	}
	return adminID, nil
}

func normalizeRequiredCourseID(courseID int64) (int64, error) {
	if courseID <= 0 {
		return 0, ErrCourseNotFound
	}
	return courseID, nil
}

func normalizeRequiredTeacherID(teacherID int64) (int64, error) {
	if teacherID <= 0 {
		return 0, ErrTeacherNotFound
	}
	return teacherID, nil
}

func validateReviewTextLengths(title, content string, minContentRunes int) error {
	if utf8.RuneCountInString(title) > maxReviewTitleRunes {
		return ErrTitleTooLong
	}
	contentRunes := utf8.RuneCountInString(content)
	if contentRunes < minContentRunes {
		return ErrContentTooShort
	}
	if contentRunes > maxReviewContentRunes {
		return ErrContentTooLong
	}
	return nil
}

func validateAdminEditReasonLength(reason string) error {
	if utf8.RuneCountInString(strings.TrimSpace(reason)) > maxAdminEditReasonRunes {
		return ErrReasonTooLong
	}
	return nil
}

func normalizeReviewGrade(grade string) (string, error) {
	grade = strings.TrimSpace(grade)
	if !isValidGrade(grade) {
		return "", ErrInvalidGrade
	}
	return grade, nil
}

func (s *Service) validateRatingValues(ctx context.Context, ratings ReviewRatings, requireAny bool) error {
	if len(ratings) == 0 {
		if requireAny {
			return ErrRatingRequired
		}
		return nil
	}

	validKeys := s.getDimensionNames()
	if validKeys == nil {
		var err error
		validKeys, err = s.repo.GetDimensionNames(ctx)
		if err != nil {
			return fmt.Errorf("failed to load rating dimensions: %w", err)
		}
	}

	for k, v := range ratings {
		if !validRatingKey.MatchString(k) {
			return ErrInvalidRating
		}
		if _, ok := validKeys[k]; !ok {
			return fmt.Errorf("%w: unknown dimension key %q", ErrInvalidRating, k)
		}
		if v < 1 || v > 5 {
			return ErrInvalidRating
		}
	}
	return nil
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
	UserCount       int
}

// GetCourseReviews 获取课程评论列表
func (s *Service) GetCourseReviews(ctx context.Context, params GetCourseReviewsParams) (*GetCourseReviewsResult, error) {
	courseID, err := normalizeRequiredCourseID(params.CourseID)
	if err != nil {
		return nil, err
	}
	params.CourseID = courseID
	if params.TeacherID != nil {
		teacherID, err := normalizeRequiredTeacherID(*params.TeacherID)
		if err != nil {
			return nil, err
		}
		params.TeacherID = &teacherID
	}

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
	courseIDs := make([]int64, 0, len(params.CourseIDs))
	for _, courseID := range params.CourseIDs {
		normalized, err := normalizeRequiredCourseID(courseID)
		if err != nil {
			return nil, err
		}
		courseIDs = append(courseIDs, normalized)
	}
	if len(courseIDs) == 0 {
		return &BatchCourseReviewsResult{
			Reviews: map[int64][]Review{},
			Totals:  map[int64]int{},
		}, nil
	}

	pageSize := httputil.ClampPageSize(params.PageSize)
	reviews, totals, err := s.repo.ListByMultipleCourses(ctx, courseIDs, params.Sort, pageSize)
	if err != nil {
		return nil, err
	}
	return &BatchCourseReviewsResult{Reviews: reviews, Totals: totals}, nil
}

// CheckCourseExists 检查课程是否存在
func (s *Service) CheckCourseExists(ctx context.Context, courseID int64) (bool, error) {
	courseID, err := normalizeRequiredCourseID(courseID)
	if err != nil {
		return false, err
	}
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

// SearchReviewsParams 搜索测评参数
type SearchReviewsParams struct {
	Query        string
	DepartmentID int64
	TeacherName  string
	TermID       string
	Page         int
	PageSize     int
	Sort         string
}

// SearchReviews 搜索测评列表
func (s *Service) SearchReviews(ctx context.Context, params SearchReviewsParams) (*GetCourseReviewsResult, error) {
	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	list, total, err := s.repo.SearchReviews(ctx, SearchReviewsQueryParams{
		Query:        params.Query,
		DepartmentID: params.DepartmentID,
		TeacherName:  params.TeacherName,
		TermID:       params.TermID,
		Sort:         params.Sort,
		Limit:        pageSize,
		Offset:       offset,
	})
	if err != nil {
		return nil, err
	}

	return &GetCourseReviewsResult{List: list, Total: total}, nil
}

// GetStats 获取统计数据
func (s *Service) GetStats(ctx context.Context) (*StatsResult, error) {
	courseCount, reviewCount, departmentCount, userCount, err := s.repo.GetPortalStats(ctx)
	if err != nil {
		return nil, err
	}
	return &StatsResult{
		CourseCount:     courseCount,
		ReviewCount:     reviewCount,
		DepartmentCount: departmentCount,
		UserCount:       userCount,
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
	courseID, err := normalizeRequiredCourseID(courseID)
	if err != nil {
		return nil, err
	}

	stats, err := s.repo.ListCourseRatingStats(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if len(stats) > 0 {
		return stats, nil
	}

	count, err := s.repo.CountByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return stats, nil
	}

	if err := s.repo.RefreshCourseRatingStats(ctx, courseID); err != nil {
		return nil, err
	}
	return s.repo.ListCourseRatingStats(ctx, courseID)
}

// GetCourseTeachers 获取课程的授课教师列表
func (s *Service) GetCourseTeachers(ctx context.Context, courseID int64) ([]CourseTeacherStats, error) {
	courseID, err := normalizeRequiredCourseID(courseID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListCourseTeachers(ctx, courseID)
}

// GetReviewByID 根据 ID 获取评论
func (s *Service) GetReviewByID(ctx context.Context, reviewID string) (*Review, error) {
	return s.repo.GetReviewByID(ctx, reviewID)
}

// CheckContent 检查内容是否包含敏感词
func (s *Service) CheckContent(ctx context.Context, content string) (*ContentCheckResult, error) {
	return s.filter.CheckContent(ctx, content)
}

// --- 内容标记管理 pass-through ---

// ListFlaggedReviews 获取待人工复核评课列表（content_flag in warn/review）
func (s *Service) ListFlaggedReviews(ctx context.Context, limit, offset int, schoolIDs ...int64) ([]Review, int, error) {
	reviews, total, err := s.repo.ListFlaggedReviews(ctx, limit, offset, schoolIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("list flagged reviews: %w", err)
	}
	return reviews, total, nil
}

// ClearContentFlag 管理员复核通过，必要时发布 pending_review 并清除 content_flag。
func (s *Service) ClearContentFlag(ctx context.Context, reviewID, adminUserID string) error {
	adminUserID, err := normalizeRequiredAdminID(adminUserID)
	if err != nil {
		return err
	}

	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		status, contentFlag, courseID, teacherID, err := s.repo.GetReviewContentFlagStateTx(ctx, tx, reviewID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReviewNotFound
			}
			return err
		}
		if contentFlag == nil || (*contentFlag != ContentFlagWarn && *contentFlag != ContentFlagReview) {
			return ErrReviewNotFound
		}
		if err := s.repo.ClearContentFlagTx(ctx, tx, reviewID, adminUserID); err != nil {
			return err
		}
		if status == StatusPendingReview && *contentFlag == ContentFlagReview {
			if err := s.repo.UpdateReviewStatus(ctx, tx, reviewID, StatusPublished); err != nil {
				return err
			}
			if err := s.repo.IncrementCourseReviewCount(ctx, tx, courseID); err != nil {
				return err
			}
			return s.refreshReviewTargetTx(ctx, tx, courseID, teacherID)
		}
		return nil
	})
}

func (s *Service) GetReviewSchoolID(ctx context.Context, reviewID string) (int64, error) {
	schoolID, err := s.repo.GetReviewSchoolID(ctx, reviewID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrReviewNotFound
	}
	return schoolID, err
}

func (s *Service) GetReportSchoolID(ctx context.Context, reportID string) (int64, error) {
	schoolID, err := s.repo.GetReportSchoolID(ctx, reportID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrReportNotFound
	}
	return schoolID, err
}

func (s *Service) ListReviewSchoolIDs(ctx context.Context, reviewIDs []string) (map[string]int64, error) {
	return s.repo.ListReviewSchoolIDs(ctx, reviewIDs)
}

// --- 敏感词管理 pass-through ---

func (s *Service) ListSensitiveWords(ctx context.Context, category, level string, limit, offset int) ([]SensitiveWord, int, error) {
	words, total, err := s.repo.ListSensitiveWords(ctx, category, level, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list sensitive words: %w", err)
	}
	return words, total, nil
}

func (s *Service) CreateSensitiveWord(ctx context.Context, word, category, level string) (SensitiveWord, error) {
	normalizedWord, err := validateSensitiveWordText(word)
	if err != nil {
		return SensitiveWord{}, err
	}
	category = strings.TrimSpace(category)
	if err := validateSensitiveWordCategory(category); err != nil {
		return SensitiveWord{}, err
	}
	level = strings.TrimSpace(level)
	if err := validateSensitiveWordLevel(level); err != nil {
		return SensitiveWord{}, err
	}

	sw, err := s.repo.CreateSensitiveWord(ctx, normalizedWord, category, level)
	if err != nil {
		return SensitiveWord{}, fmt.Errorf("create sensitive word: %w", err)
	}
	return sw, nil
}

func (s *Service) UpdateSensitiveWord(ctx context.Context, wordID string, word, category, level *string, isActive *bool) error {
	if word != nil {
		normalizedWord, err := validateSensitiveWordText(*word)
		if err != nil {
			return err
		}
		word = &normalizedWord
	}
	if category != nil {
		normalizedCategory := strings.TrimSpace(*category)
		if err := validateSensitiveWordCategory(normalizedCategory); err != nil {
			return err
		}
		category = &normalizedCategory
	}
	if level != nil {
		normalizedLevel := strings.TrimSpace(*level)
		if err := validateSensitiveWordLevel(normalizedLevel); err != nil {
			return err
		}
		level = &normalizedLevel
	}

	if err := s.repo.UpdateSensitiveWord(ctx, wordID, word, category, level, isActive); err != nil {
		return fmt.Errorf("update sensitive word %s: %w", wordID, err)
	}
	return nil
}

func (s *Service) DeleteSensitiveWord(ctx context.Context, wordID string) error {
	if err := s.repo.DeleteSensitiveWord(ctx, wordID); err != nil {
		return fmt.Errorf("delete sensitive word %s: %w", wordID, err)
	}
	return nil
}

// --- 公开教师列表 ---

// ListTeachers 获取公开教师列表（分页、搜索、排序）
func (s *Service) ListTeachers(ctx context.Context, search string, departmentID *int64, sort string, page, pageSize int) ([]TeacherSummary, int, error) {
	pageSize = httputil.ClampPageSize(pageSize)
	offset := httputil.SafeOffset(page, pageSize)
	teachers, total, err := s.repo.ListPublicTeachers(ctx, search, departmentID, sort, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list teachers: %w", err)
	}
	return teachers, total, nil
}

// ListHotTeachers 获取热门教师列表
func (s *Service) ListHotTeachers(ctx context.Context, limit int) ([]TeacherSummary, error) {
	teachers, err := s.repo.ListHotTeachers(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list hot teachers: %w", err)
	}
	return teachers, nil
}

// --- 教师管理 pass-through ---

func (s *Service) ListAdminTeachers(ctx context.Context, search string, departmentID int64, limit, offset int) ([]AdminTeacher, int, error) {
	teachers, total, err := s.repo.ListAdminTeachers(ctx, search, departmentID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list teachers: %w", err)
	}
	return teachers, total, nil
}

func (s *Service) CreateTeacher(ctx context.Context, name string, departmentID *int64) (*AdminTeacher, error) {
	normalizedName, err := s.validateAdminTeacherInput(ctx, name, departmentID, true)
	if err != nil {
		return nil, err
	}
	t, err := s.repo.CreateTeacher(ctx, normalizedName, departmentID)
	if err != nil {
		return nil, fmt.Errorf("create teacher: %w", err)
	}
	t.Name = normalizedName
	return t, nil
}

func (s *Service) UpdateTeacher(ctx context.Context, id int64, name string, departmentID *int64) error {
	id, err := normalizeRequiredTeacherID(id)
	if err != nil {
		return err
	}
	normalizedName, err := s.validateAdminTeacherInput(ctx, name, departmentID, false)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateTeacher(ctx, id, normalizedName, departmentID); err != nil {
		return fmt.Errorf("update teacher %d: %w", id, err)
	}
	return nil
}

func (s *Service) DeleteTeacher(ctx context.Context, id int64) error {
	id, err := normalizeRequiredTeacherID(id)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteTeacher(ctx, id); err != nil {
		return fmt.Errorf("delete teacher %d: %w", id, err)
	}
	return nil
}

func (s *Service) validateAdminTeacherInput(ctx context.Context, name string, departmentID *int64, requireDepartment bool) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxAdminTeacherNameRunes {
		return "", ErrTeacherNameInvalid
	}
	if departmentID == nil {
		if requireDepartment {
			return "", ErrTeacherDepartmentRequired
		}
		return name, nil
	}
	if *departmentID <= 0 {
		return "", ErrTeacherDepartmentNotFound
	}
	exists, err := s.repo.DepartmentExists(ctx, *departmentID)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", ErrTeacherDepartmentNotFound
	}
	return name, nil
}
