package review

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/sanitizer"
)

// 业务错误定义
var (
	ErrCourseNotFound     = errors.New("course not found")
	ErrReviewNotFound     = errors.New("review not found")
	ErrAlreadyVoted       = errors.New("already voted")
	ErrDangerousContent   = errors.New("content contains dangerous elements")
	ErrInvalidRating      = errors.New("invalid rating value")
	ErrRatingRequired     = errors.New("at least one rating dimension is required")
)

// Service 评课服务层
type Service struct {
	db   *db.DB
	repo *Repository
	log  *zap.Logger
}

// NewService 创建评课服务
func NewService(database *db.DB, repo *Repository) *Service {
	return &Service{
		db:   database,
		repo: repo,
		log:  logger.L(),
	}
}

// GetCourseReviewsParams 获取课程评论参数
type GetCourseReviewsParams struct {
	CourseID int64
	Page     int
	PageSize int
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
}

// PostReviewResult 发布评论结果
type PostReviewResult struct {
	ID string
}

// VoteReviewParams 投票参数
type VoteReviewParams struct {
	ReviewID string
	UserHash string
	VoteType string // "like" or "dislike"
}

// StatsResult 统计结果
type StatsResult struct {
	ReviewCount int
}

// GetCourseReviews 获取课程评论列表
func (s *Service) GetCourseReviews(ctx context.Context, params GetCourseReviewsParams) (*GetCourseReviewsResult, error) {
	total, err := s.repo.CountByCourse(ctx, params.CourseID)
	if err != nil {
		return nil, err
	}

	offset := (params.Page - 1) * params.PageSize
	list, err := s.repo.ListByCourse(ctx, params.CourseID, params.PageSize, offset)
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
}

// GetLatestReviews 获取最新评论列表
func (s *Service) GetLatestReviews(ctx context.Context, params GetLatestReviewsParams) (*GetCourseReviewsResult, error) {
	total, err := s.repo.CountAll(ctx)
	if err != nil {
		return nil, err
	}

	offset := (params.Page - 1) * params.PageSize
	list, err := s.repo.ListLatest(ctx, params.PageSize, offset)
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

	// 检查课程是否存在
	exists, err := s.repo.CourseExists(ctx, params.CourseID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCourseNotFound
	}

	// 序列化评分数据
	ratingsData, err := json.Marshal(params.Ratings)
	if err != nil {
		return nil, err
	}

	reviewID := uuid.NewString()

	// 开启事务
	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.log.Error("failed to begin transaction", zap.Error(err))
		return nil, err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			s.log.Warn("failed to rollback transaction", zap.Error(err))
		}
	}()

	// 插入评论
	err = s.repo.Create(ctx, tx, CreateParams{
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
		s.log.Error("failed to insert review",
			zap.String("review_id", reviewID),
			zap.Int64("course_id", params.CourseID),
			zap.Error(err),
		)
		return nil, err
	}

	// 更新课程评论计数
	err = s.repo.IncrementCourseReviewCount(ctx, tx, params.CourseID)
	if err != nil {
		s.log.Error("failed to update review count",
			zap.Int64("course_id", params.CourseID),
			zap.Error(err),
		)
		return nil, err
	}

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		s.log.Error("failed to commit transaction",
			zap.String("review_id", reviewID),
			zap.Error(err),
		)
		return nil, err
	}

	return &PostReviewResult{ID: reviewID}, nil
}

// VoteReview 投票
func (s *Service) VoteReview(ctx context.Context, params VoteReviewParams) error {
	// 检查评论是否存在
	exists, err := s.repo.ReviewExists(ctx, params.ReviewID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrReviewNotFound
	}

	// 开启事务
	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.log.Error("failed to begin transaction", zap.Error(err))
		return err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			s.log.Warn("failed to rollback transaction", zap.Error(err))
		}
	}()

	// 创建投票记录
	created, err := s.repo.CreateVote(ctx, tx, params.ReviewID, params.UserHash, params.VoteType)
	if err != nil {
		s.log.Error("failed to insert vote",
			zap.String("review_id", params.ReviewID),
			zap.Error(err),
		)
		return err
	}
	if !created {
		return ErrAlreadyVoted
	}

	// 更新投票计数
	if params.VoteType == "like" {
		err = s.repo.IncrementLikeCount(ctx, tx, params.ReviewID)
	} else {
		err = s.repo.IncrementDislikeCount(ctx, tx, params.ReviewID)
	}
	if err != nil {
		s.log.Error("failed to update vote count",
			zap.String("review_id", params.ReviewID),
			zap.String("vote_type", params.VoteType),
			zap.Error(err),
		)
		return err
	}

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		s.log.Error("failed to commit vote transaction",
			zap.String("review_id", params.ReviewID),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// GetStats 获取统计数据
func (s *Service) GetStats(ctx context.Context) (*StatsResult, error) {
	count, err := s.repo.CountAll(ctx)
	if err != nil {
		return nil, err
	}
	return &StatsResult{ReviewCount: count}, nil
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
