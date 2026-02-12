package review

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/sanitizer"
)

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
	ErrInvalidAction          = errors.New("invalid action")
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
	offset := (params.Page - 1) * params.PageSize
	list, total, err := s.repo.ListByCourseWithSort(ctx, ListByCourseWithSortParams{
		CourseID:  params.CourseID,
		Sort:      params.Sort,
		TermID:    params.TermID,
		TeacherID: params.TeacherID,
		Limit:     params.PageSize,
		Offset:    offset,
	})
	if err != nil {
		return nil, err
	}

	return &GetCourseReviewsResult{List: list, Total: total}, nil
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
	offset := (params.Page - 1) * params.PageSize
	list, total, err := s.repo.ListLatest(ctx, params.PageSize, offset, params.Sort)
	if err != nil {
		return nil, err
	}

	return &GetCourseReviewsResult{List: list, Total: total}, nil
}

// PostReview 发布评论
func (s *Service) PostReview(ctx context.Context, params PostReviewParams) (*PostReviewResult, error) {
	// 验证评分
	if len(params.Ratings) == 0 {
		return nil, ErrRatingRequired
	}
	for _, v := range params.Ratings {
		if v < 1 || v > 5 {
			return nil, ErrInvalidRating
		}
	}

	// XSS 防护
	if sanitizer.ContainsDangerousContent(params.Title) || sanitizer.ContainsDangerousContent(params.Content) {
		return nil, ErrDangerousContent
	}
	params.Title = sanitizer.SanitizeTitle(params.Title)
	params.Content = sanitizer.SanitizeText(params.Content)

	// 清洗后验证内容非空
	if strings.TrimSpace(params.Content) == "" {
		return nil, ErrContentEmpty
	}

	// 敏感词检查
	checkResult := s.filter.CheckContent(ctx, params.Title+" "+params.Content)
	if !checkResult.IsValid {
		return nil, ErrSensitiveContent
	}

	// 检查课程是否存在
	exists, err := s.repo.CourseExists(ctx, params.CourseID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCourseNotFound
	}

	// 检查用户是否已对该课程发布评论
	hasReviewed, err := s.repo.UserHasReviewedCourse(ctx, params.UserHash, params.CourseID)
	if err != nil {
		return nil, err
	}
	if hasReviewed {
		return nil, ErrAlreadyReviewed
	}

	// 序列化评分数据
	ratingsData, err := json.Marshal(params.Ratings)
	if err != nil {
		return nil, err
	}

	reviewID, err := id.New()
	if err != nil {
		return nil, err
	}

	if err := s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.Create(ctx, tx, CreateParams{
			ID:        reviewID,
			CourseID:  params.CourseID,
			TeacherID: params.TeacherID,
			TermID:    params.TermID,
			Title:     params.Title,
			Content:   params.Content,
			Grade:     params.Grade,
			Ratings:   ratingsData,
			UserHash:  params.UserHash,
		}); err != nil {
			return err
		}
		return s.repo.IncrementCourseReviewCount(ctx, tx, params.CourseID)
	}); err != nil {
		return nil, err
	}

	// 记录用户操作日志
	audit.Log(audit.Event{
		Type:     audit.EventDataCreate,
		UserID:   params.UserHash,
		IP:       params.IPAddress,
		Resource: "review",
		Action:   "post_review",
		Result:   "success",
		Details: map[string]interface{}{
			"review_id": reviewID,
			"course_id": params.CourseID,
		},
	})

	// 获取完整的 Review 对象返回给调用方
	review, err := s.repo.GetReviewByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}

	return &PostReviewResult{Review: *review}, nil
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
	// 验证评分
	if len(params.Ratings) == 0 {
		return ErrRatingRequired
	}
	for _, v := range params.Ratings {
		if v < 1 || v > 5 {
			return ErrInvalidRating
		}
	}

	// XSS 防护
	if sanitizer.ContainsDangerousContent(params.Title) || sanitizer.ContainsDangerousContent(params.Content) {
		return ErrDangerousContent
	}
	params.Title = sanitizer.SanitizeTitle(params.Title)
	params.Content = sanitizer.SanitizeText(params.Content)

	// 清洗后验证内容非空
	if strings.TrimSpace(params.Content) == "" {
		return ErrContentEmpty
	}

	// 敏感词检查
	checkResult := s.filter.CheckContent(ctx, params.Title+" "+params.Content)
	if !checkResult.IsValid {
		return ErrSensitiveContent
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
		// 仅从 published 状态删除时递减计数
		if status == "published" {
			return s.repo.DecrementCourseReviewCount(ctx, tx, courseID)
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
