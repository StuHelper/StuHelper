package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/sanitizer"
)

// maskHash 返回哈希值的前 8 个字符，用于日志脱敏，防止跨日志条目追踪用户
func maskHash(hash string) string {
	if len(hash) <= 8 {
		return hash
	}
	return hash[:8] + "..."
}

// 业务错误定义
var (
	ErrCourseNotFound     = errors.New("course not found")
	ErrReviewNotFound     = errors.New("review not found")
	ErrTeacherNotFound    = errors.New("teacher not found")
	ErrAlreadyVoted       = errors.New("already voted")
	ErrAlreadyReviewed    = errors.New("already reviewed this course")
	ErrDangerousContent   = errors.New("content contains dangerous elements")
	ErrSensitiveContent   = errors.New("content contains sensitive words")
	ErrContentEmpty       = errors.New("content cannot be empty after sanitization")
	ErrInvalidRating      = errors.New("invalid rating value")
	ErrRatingRequired     = errors.New("at least one rating dimension is required")
	ErrNotReviewOwner     = errors.New("not the review owner")
	ErrAlreadyReported    = errors.New("already reported this review")
	ErrReportNotFound     = errors.New("report not found")
	ErrDraftNotFound          = errors.New("draft not found")
	ErrNotificationNotFound   = errors.New("notification not found")
	ErrInvalidAction     = errors.New("invalid action")
	ErrInvalidTransition = errors.New("invalid status transition")
)

// Service 评课服务层
type Service struct {
	db     *db.DB
	repo   *Repository
	filter *Filter
	log    *zap.Logger
}

// NewService 创建评课服务
func NewService(database *db.DB, repo *Repository) *Service {
	filter := NewFilter(repo)
	return &Service{
		db:     database,
		repo:   repo,
		filter: filter,
		log:    logger.L(),
	}
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

	// M-114: 从 DB 获取有效的评分维度 key 白名单，校验提交的 key 是否合法
	validKeys, err := s.repo.GetDimensionNames(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to load rating dimensions: %w", err)
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

// PostReviewParams 发布评论参数
type PostReviewParams struct {
	CourseID  int64
	TeacherID *int64
	TermID    string
	Title     string
	Content   string
	Grade     string
	Ratings   ReviewRatings
	UserHash  string
	IPAddress string
	RequestID string // L-35: 由 handler 层从 gin context 提取后传入
}

// PostReviewResult 发布评论结果
type PostReviewResult struct {
	Review Review
}

// VoteReviewParams 投票参数
type VoteReviewParams struct {
	ReviewID string
	UserHash string
	VoteType string // "like" or "dislike"
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

// PostReview 发布评论
func (s *Service) PostReview(ctx context.Context, params PostReviewParams) (*PostReviewResult, error) {
	var err error
	params.Title, params.Content, err = s.validateAndSanitizeReview(ctx, params.Ratings, params.Title, params.Content, params.TermID)
	if err != nil {
		return nil, err
	}

	// 序列化评分数据
	ratingsData, err := json.Marshal(params.Ratings)
	if err != nil {
		logger.L().Error("failed to marshal ratings", zap.Any("ratings", params.Ratings), zap.Error(err))
		return nil, err
	}

	reviewID, err := id.New()
	if err != nil {
		return nil, err
	}

	var review Review
	if err := s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// 在事务内检查课程是否存在，防止 TOCTOU 竞态
		exists, err := s.repo.CourseExistsTx(ctx, tx, params.CourseID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrCourseNotFound
		}

		// H-23: 在事务内校验 teacher_id 是否有效
		if params.TeacherID != nil {
			teacherExists, err := s.repo.TeacherExistsTx(ctx, tx, *params.TeacherID)
			if err != nil {
				return err
			}
			if !teacherExists {
				return ErrTeacherNotFound
			}
		}

		// 在事务内检查用户是否已对该课程发布评论
		hasReviewed, err := s.repo.UserHasReviewedCourseTx(ctx, tx, params.UserHash, params.CourseID)
		if err != nil {
			return err
		}
		if hasReviewed {
			return ErrAlreadyReviewed
		}

		// H-16: 使用 RETURNING 在事务内直接获取完整 Review，避免孤儿记录问题
		created, err := s.repo.CreateReturning(ctx, tx, CreateParams{
			ID:        reviewID,
			CourseID:  params.CourseID,
			TeacherID: params.TeacherID,
			TermID:    params.TermID,
			Title:     params.Title,
			Content:   params.Content,
			Grade:     params.Grade,
			Ratings:   ratingsData,
			UserHash:  params.UserHash,
		})
		if err != nil {
			return err
		}
		review = *created
		return s.repo.IncrementCourseReviewCount(ctx, tx, params.CourseID)
	}); err != nil {
		return nil, err
	}

	// 记录用户操作日志
	audit.Log(audit.Event{
		Type:      audit.EventDataCreate,
		UserID:    maskHash(params.UserHash),
		IP:        params.IPAddress,
		RequestID: params.RequestID,
		Resource:  "review",
		Action:    "post_review",
		Result:    "success",
		Details: map[string]interface{}{
			"review_id": reviewID,
			"course_id": params.CourseID,
		},
	})

	return &PostReviewResult{Review: review}, nil
}

// VoteReview 投票（支持新建、取消、切换）
func (s *Service) VoteReview(ctx context.Context, params VoteReviewParams) error {
	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// 在事务内检查评论是否存在，防止 TOCTOU 竞态
		exists, err := s.repo.ReviewExistsTx(ctx, tx, params.ReviewID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrReviewNotFound
		}

		existing, err := s.repo.GetVoteType(ctx, tx, params.ReviewID, params.UserHash)
		if err != nil {
			return err
		}

		switch {
		case existing == "":
			// 新投票：检查是否实际插入，防止并发重复计数
			inserted, err := s.repo.CreateVote(ctx, tx, params.ReviewID, params.UserHash, params.VoteType)
			if err != nil {
				return err
			}
			if !inserted {
				return nil // 并发冲突，ON CONFLICT DO NOTHING 跳过了插入
			}
			if params.VoteType == "like" {
				return s.repo.IncrementLikeCount(ctx, tx, params.ReviewID)
			}
			return s.repo.IncrementDislikeCount(ctx, tx, params.ReviewID)

		case existing == params.VoteType:
			// 相同类型 → 取消投票
			if err := s.repo.DeleteVote(ctx, tx, params.ReviewID, params.UserHash); err != nil {
				return err
			}
			if existing == "like" {
				return s.repo.DecrementLikeCount(ctx, tx, params.ReviewID)
			}
			return s.repo.DecrementDislikeCount(ctx, tx, params.ReviewID)

		default:
			// 不同类型 → 切换投票
			if err := s.repo.UpdateVoteType(ctx, tx, params.ReviewID, params.UserHash, params.VoteType); err != nil {
				return err
			}
			if params.VoteType == "like" {
				if err := s.repo.DecrementDislikeCount(ctx, tx, params.ReviewID); err != nil {
					return err
				}
				return s.repo.IncrementLikeCount(ctx, tx, params.ReviewID)
			}
			if err := s.repo.DecrementLikeCount(ctx, tx, params.ReviewID); err != nil {
				return err
			}
			return s.repo.IncrementDislikeCount(ctx, tx, params.ReviewID)
		}
	})
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

// UpdateReviewParams 更新评论参数
type UpdateReviewParams struct {
	ReviewID string
	UserHash string
	Title    string
	Content  string
	Grade    string
	Ratings  ReviewRatings
}

// UpdateReview 更新评论
func (s *Service) UpdateReview(ctx context.Context, params UpdateReviewParams) error {
	var err error
	// UpdateReview 不涉及 termID 变更，传空字符串跳过校验
	params.Title, params.Content, err = s.validateAndSanitizeReview(ctx, params.Ratings, params.Title, params.Content, "")
	if err != nil {
		return err
	}

	ratingsData, err := json.Marshal(params.Ratings)
	if err != nil {
		return err
	}

	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// 在事务内获取所有者和状态，消除 TOCTOU 竞态
		ownerHash, status, err := s.repo.GetReviewOwnerAndStatusTx(ctx, tx, params.ReviewID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReviewNotFound
			}
			return err
		}
		if status != "published" {
			return ErrReviewNotFound
		}
		if ownerHash != params.UserHash {
			return ErrNotReviewOwner
		}

		return s.repo.Update(ctx, tx, UpdateParams{
			ID:      params.ReviewID,
			Title:   params.Title,
			Content: params.Content,
			Grade:   params.Grade,
			Ratings: ratingsData,
		})
	})
}

// DeleteReviewParams 删除评论参数
type DeleteReviewParams struct {
	ReviewID string
	UserHash string
}

// DeleteReview 删除评论
func (s *Service) DeleteReview(ctx context.Context, params DeleteReviewParams) error {
	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// 在事务内获取所有者、课程ID和状态（带行锁），消除 TOCTOU 竞态
		ownerHash, courseID, status, err := s.repo.GetReviewOwnerCourseIDAndStatusTx(ctx, tx, params.ReviewID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReviewNotFound
			}
			return err
		}
		if status == "deleted" {
			return ErrReviewNotFound
		}
		if ownerHash != params.UserHash {
			return ErrNotReviewOwner
		}

		if err := s.repo.SoftDeleteReview(ctx, tx, params.ReviewID); err != nil {
			return err
		}
		// 仅从 published 状态删除时递减计数并刷新评分统计
		if status == "published" {
			if err := s.repo.DecrementCourseReviewCount(ctx, tx, courseID); err != nil {
				return err
			}
			return s.repo.RefreshCourseRatingStatsTx(ctx, tx, courseID)
		}
		return nil
	})
}

// GetReviewByID 根据 ID 获取评论
func (s *Service) GetReviewByID(ctx context.Context, reviewID string) (*Review, error) {
	return s.repo.GetReviewByID(ctx, reviewID)
}

// CheckContent 检查内容是否包含敏感词
func (s *Service) CheckContent(ctx context.Context, content string) *ContentCheckResult {
	return s.filter.CheckContent(ctx, content)
}

// CheckQuality 检查内容质量
func (s *Service) CheckQuality(content string) *QualityCheckResult {
	return s.filter.CheckQuality(content)
}
