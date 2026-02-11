package review

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
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
	ErrDraftNotFound      = errors.New("draft not found")
	ErrInvalidAction      = errors.New("invalid action")
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

	reviewID := uuid.NewString()

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
			// 新投票
			if _, err := s.repo.CreateVote(ctx, tx, params.ReviewID, params.UserHash, params.VoteType); err != nil {
				return err
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
	// 获取评论所有者和状态
	ownerHash, status, err := s.repo.GetReviewOwnerAndStatus(ctx, params.ReviewID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrReviewNotFound
		}
		return err
	}

	// 仅允许编辑已发布的评论
	if status != "published" {
		return ErrReviewNotFound
	}

	// 验证是否为本人
	if ownerHash != params.UserHash {
		return ErrNotReviewOwner
	}

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
	ownerHash, courseID, err := s.repo.GetReviewOwnerAndCourseID(ctx, params.ReviewID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrReviewNotFound
		}
		return err
	}

	if ownerHash != params.UserHash {
		return ErrNotReviewOwner
	}

	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// 在事务内获取当前状态，防止并发重复删除
		status, _, err := s.repo.GetReviewStatusAndCourseIDTx(ctx, tx, params.ReviewID)
		if err != nil {
			return err
		}
		if status == "deleted" {
			return ErrReviewNotFound
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

// ReportReviewParams 举报评论参数
type ReportReviewParams struct {
	ReviewID    string
	UserHash    string
	Reason      string
	Description string
}

// ReportReview 举报评论
func (s *Service) ReportReview(ctx context.Context, params ReportReviewParams) error {
	exists, err := s.repo.ReviewExists(ctx, params.ReviewID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrReviewNotFound
	}

	reported, err := s.repo.ReportExists(ctx, params.ReviewID, params.UserHash)
	if err != nil {
		return err
	}
	if reported {
		return ErrAlreadyReported
	}

	return s.repo.CreateReport(ctx, CreateReportParams{
		ReviewID:     params.ReviewID,
		ReporterHash: params.UserHash,
		Reason:       params.Reason,
		Description:  params.Description,
	})
}

// ListReportsParams 获取举报列表参数
type ListReportsParams struct {
	Status   string
	Page     int
	PageSize int
}

// ListReportsResult 获取举报列表结果
type ListReportsResult struct {
	List  []ReviewReport
	Total int
}

// ListReports 获取举报列表（管理员）
func (s *Service) ListReports(ctx context.Context, params ListReportsParams) (*ListReportsResult, error) {
	offset := (params.Page - 1) * params.PageSize
	list, total, err := s.repo.ListReports(ctx, params.Status, params.PageSize, offset)
	if err != nil {
		return nil, err
	}

	return &ListReportsResult{List: list, Total: total}, nil
}

// ProcessReportParams 处理举报参数
type ProcessReportParams struct {
	ReportID   int64
	Action     string
	Note       string
	ResolvedBy string
}

// ProcessReport 处理举报（管理员）
func (s *Service) ProcessReport(ctx context.Context, params ProcessReportParams) error {
	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// 在事务内获取并加锁，防止并发处理同一举报
		report, err := s.repo.GetReportByIDForUpdate(ctx, tx, params.ReportID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReportNotFound
			}
			return err
		}

		var reportStatus string
		switch params.Action {
		case "reject":
			reportStatus = "rejected"
		case "hide_review":
			reportStatus = "resolved"
			courseID, err := s.repo.GetReviewCourseIDTx(ctx, tx, report.ReviewID)
			if err != nil {
				return err
			}
			if err := s.repo.UpdateReviewStatus(ctx, tx, report.ReviewID, "hidden"); err != nil {
				return err
			}
			if err := s.repo.DecrementCourseReviewCount(ctx, tx, courseID); err != nil {
				return err
			}
		case "delete_review":
			reportStatus = "resolved"
			courseID, err := s.repo.GetReviewCourseIDTx(ctx, tx, report.ReviewID)
			if err != nil {
				return err
			}
			if err := s.repo.SoftDeleteReview(ctx, tx, report.ReviewID); err != nil {
				return err
			}
			if err := s.repo.DecrementCourseReviewCount(ctx, tx, courseID); err != nil {
				return err
			}
		default:
			return ErrInvalidAction
		}

		return s.repo.UpdateReport(ctx, tx, UpdateReportParams{
			ID:         params.ReportID,
			Status:     reportStatus,
			ResolvedBy: params.ResolvedBy,
			Note:       params.Note,
		})
	})
}

// AdminUpdateReviewParams 管理员更新评论参数
type AdminUpdateReviewParams struct {
	ReviewID string
	Action   string
}

// AdminUpdateReview 管理员更新评论
func (s *Service) AdminUpdateReview(ctx context.Context, params AdminUpdateReviewParams) error {
	exists, err := s.repo.ReviewExistsAny(ctx, params.ReviewID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrReviewNotFound
	}

	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// 获取当前状态和课程ID，用于判断是否需要调整计数
		currentStatus, courseID, err := s.repo.GetReviewStatusAndCourseIDTx(ctx, tx, params.ReviewID)
		if err != nil {
			return err
		}

		switch params.Action {
		case "hide":
			if err := s.repo.UpdateReviewStatus(ctx, tx, params.ReviewID, "hidden"); err != nil {
				return err
			}
			// 从 published 隐藏时递减计数
			if currentStatus == "published" {
				return s.repo.DecrementCourseReviewCount(ctx, tx, courseID)
			}
			return nil
		case "restore":
			if err := s.repo.UpdateReviewStatus(ctx, tx, params.ReviewID, "published"); err != nil {
				return err
			}
			// 从 hidden/deleted 恢复到 published 时递增计数
			if currentStatus != "published" {
				return s.repo.IncrementCourseReviewCount(ctx, tx, courseID)
			}
			return nil
		case "delete":
			if err := s.repo.SoftDeleteReview(ctx, tx, params.ReviewID); err != nil {
				return err
			}
			// 从 published 删除时递减计数（hidden 状态已不计数）
			if currentStatus == "published" {
				return s.repo.DecrementCourseReviewCount(ctx, tx, courseID)
			}
			return nil
		default:
			return ErrInvalidAction
		}
	})
}

// ListAllReviewsParams 获取所有评论参数
type ListAllReviewsParams struct {
	Status   string
	Page     int
	PageSize int
}

// ListAllReviews 获取所有评论（管理员）
func (s *Service) ListAllReviews(ctx context.Context, params ListAllReviewsParams) (*GetCourseReviewsResult, error) {
	offset := (params.Page - 1) * params.PageSize
	list, total, err := s.repo.ListAllReviews(ctx, params.Status, params.PageSize, offset)
	if err != nil {
		return nil, err
	}

	return &GetCourseReviewsResult{List: list, Total: total}, nil
}

// GetAdminStats 获取管理统计
func (s *Service) GetAdminStats(ctx context.Context) (*AdminStats, error) {
	return s.repo.GetAdminStats(ctx)
}

// AddFavoriteParams 添加收藏参数
type AddFavoriteParams struct {
	UserHash string
	CourseID int64
}

// AddFavorite 添加收藏
func (s *Service) AddFavorite(ctx context.Context, params AddFavoriteParams) error {
	exists, err := s.repo.CourseExists(ctx, params.CourseID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCourseNotFound
	}
	return s.repo.CreateFavorite(ctx, params.UserHash, params.CourseID)
}

// RemoveFavorite 移除收藏
func (s *Service) RemoveFavorite(ctx context.Context, userHash string, courseID int64) error {
	return s.repo.DeleteFavorite(ctx, userHash, courseID)
}

// GetUserFavoritesParams 获取用户收藏参数
type GetUserFavoritesParams struct {
	UserHash string
	Page     int
	PageSize int
}

// GetUserFavoritesResult 获取用户收藏结果
type GetUserFavoritesResult struct {
	List  []FavoriteCourse
	Total int
}

// GetUserFavorites 获取用户收藏列表
func (s *Service) GetUserFavorites(ctx context.Context, params GetUserFavoritesParams) (*GetUserFavoritesResult, error) {
	offset := (params.Page - 1) * params.PageSize
	list, total, err := s.repo.ListFavorites(ctx, params.UserHash, params.PageSize, offset)
	if err != nil {
		return nil, err
	}

	return &GetUserFavoritesResult{List: list, Total: total}, nil
}

// GetUserReviewsParams 获取用户评论参数
type GetUserReviewsParams struct {
	UserHash string
	Page     int
	PageSize int
}

// GetUserReviews 获取用户评论列表
func (s *Service) GetUserReviews(ctx context.Context, params GetUserReviewsParams) (*GetCourseReviewsResult, error) {
	offset := (params.Page - 1) * params.PageSize
	list, total, err := s.repo.ListByUserHash(ctx, params.UserHash, params.PageSize, offset)
	if err != nil {
		return nil, err
	}

	return &GetCourseReviewsResult{List: list, Total: total}, nil
}

// GetUserVotesParams 获取用户点赞参数
type GetUserVotesParams struct {
	UserHash string
	VoteType string // like, dislike
	Page     int
	PageSize int
}

// GetUserVotes 获取用户点赞列表
func (s *Service) GetUserVotes(ctx context.Context, params GetUserVotesParams) (*GetCourseReviewsResult, error) {
	offset := (params.Page - 1) * params.PageSize
	list, total, err := s.repo.ListVotedReviews(ctx, params.UserHash, params.VoteType, params.PageSize, offset)
	if err != nil {
		return nil, err
	}

	return &GetCourseReviewsResult{List: list, Total: total}, nil
}

// SaveDraftParams 保存草稿参数
type SaveDraftParams struct {
	UserHash  string
	CourseID  int64
	TeacherID *int64
	TermID    string
	Title     string
	Content   string
	Grade     string
	Ratings   ReviewRatings
}

// SaveDraft 保存草稿（UPSERT）
func (s *Service) SaveDraft(ctx context.Context, params SaveDraftParams) (*ReviewDraft, error) {
	// 检查课程是否存在
	exists, err := s.repo.CourseExists(ctx, params.CourseID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCourseNotFound
	}

	// XSS 防护
	if sanitizer.ContainsDangerousContent(params.Title) || sanitizer.ContainsDangerousContent(params.Content) {
		return nil, ErrDangerousContent
	}
	params.Title = sanitizer.SanitizeTitle(params.Title)
	params.Content = sanitizer.SanitizeText(params.Content)

	// 序列化评分数据
	ratingsData, err := json.Marshal(params.Ratings)
	if err != nil {
		return nil, err
	}

	return s.repo.UpsertDraft(ctx, UpsertDraftParams{
		UserHash:  params.UserHash,
		CourseID:  params.CourseID,
		TeacherID: params.TeacherID,
		TermID:    params.TermID,
		Title:     params.Title,
		Content:   params.Content,
		Grade:     params.Grade,
		Ratings:   ratingsData,
	})
}

// GetReviewByID 根据 ID 获取评论
func (s *Service) GetReviewByID(ctx context.Context, reviewID string) (*Review, error) {
	return s.repo.GetReviewByID(ctx, reviewID)
}

// GetDraft 获取草稿
func (s *Service) GetDraft(ctx context.Context, userHash string, courseID int64) (*ReviewDraft, error) {
	draft, err := s.repo.GetDraft(ctx, userHash, courseID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDraftNotFound
		}
		return nil, err
	}
	return draft, nil
}

// DeleteDraft 删除草稿
func (s *Service) DeleteDraft(ctx context.Context, userHash string, courseID int64) error {
	return s.repo.DeleteDraft(ctx, userHash, courseID)
}

// 回复相关错误
var (
	ErrReplyNotFound  = errors.New("reply not found")
	ErrNotReplyOwner  = errors.New("not the reply owner")
	ErrContentTooLong = errors.New("content too long")
)

// CreateReplyResult 创建回复结果
type CreateReplyResult struct {
	Reply Reply
}

// CreateReply 创建回复
func (s *Service) CreateReply(ctx context.Context, params CreateReplyParams) (*CreateReplyResult, error) {
	// 检查评论是否存在
	exists, err := s.repo.ReviewExists(ctx, params.ReviewID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrReviewNotFound
	}

	// XSS 防护
	if sanitizer.ContainsDangerousContent(params.Content) {
		return nil, ErrDangerousContent
	}
	params.Content = sanitizer.SanitizeText(params.Content)

	// 敏感词检查
	checkResult := s.filter.CheckContent(ctx, params.Content)
	if !checkResult.IsValid {
		return nil, ErrSensitiveContent
	}

	var id int64
	now := time.Now()
	if err := s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		id, err = s.repo.CreateReply(ctx, tx, CreateReplyParams{
			ReviewID: params.ReviewID,
			ParentID: params.ParentID,
			UserHash: params.UserHash,
			Content:  params.Content,
		})
		if err != nil {
			return err
		}
		return s.repo.IncrementReplyCount(ctx, tx, params.ReviewID)
	}); err != nil {
		return nil, err
	}

	return &CreateReplyResult{
		Reply: Reply{
			ID:        id,
			ReviewID:  params.ReviewID,
			ParentID:  params.ParentID,
			Content:   params.Content,
			LikeCount: 0,
			Status:    "published",
			IsOwner:   true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}

// GetRepliesParams 获取回复列表参数
type GetRepliesParams struct {
	ReviewID string
	UserHash string
	Page     int
	PageSize int
}

// GetRepliesResult 获取回复列表结果
type GetRepliesResult struct {
	List  []Reply
	Total int
}

// GetReplies 获取回复列表
func (s *Service) GetReplies(ctx context.Context, params GetRepliesParams) (*GetRepliesResult, error) {
	offset := (params.Page - 1) * params.PageSize
	list, total, err := s.repo.ListReplies(ctx, params.ReviewID, params.PageSize, offset)
	if err != nil {
		return nil, err
	}

	if params.UserHash != "" {
		for i := range list {
			list[i].IsOwner = list[i].UserHash == params.UserHash
		}
	}

	return &GetRepliesResult{List: list, Total: total}, nil
}

// DeleteReplyParams 删除回复参数
type DeleteReplyParams struct {
	ReplyID  int64
	UserHash string
}

// DeleteReply 删除回复
func (s *Service) DeleteReply(ctx context.Context, params DeleteReplyParams) error {
	// 获取回复所有者和评论ID（单次查询）
	ownerHash, reviewID, err := s.repo.GetReplyOwnerAndReviewID(ctx, params.ReplyID)
	if err != nil {
		return ErrReplyNotFound
	}

	// 验证是否为本人
	if ownerHash != params.UserHash {
		return ErrNotReplyOwner
	}

	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.SoftDeleteReply(ctx, tx, params.ReplyID); err != nil {
			return err
		}
		return s.repo.DecrementReplyCount(ctx, tx, reviewID)
	})
}

// GetNotificationsParams 获取通知列表参数
type GetNotificationsParams struct {
	UserHash string
	Page     int
	PageSize int
}

// GetNotificationsResult 获取通知列表结果
type GetNotificationsResult struct {
	List   []Notification
	Total  int
	Unread int
}

// GetNotifications 获取通知列表
func (s *Service) GetNotifications(ctx context.Context, params GetNotificationsParams) (*GetNotificationsResult, error) {
	unread, err := s.repo.CountUnreadNotifications(ctx, params.UserHash)
	if err != nil {
		return nil, err
	}

	offset := (params.Page - 1) * params.PageSize
	list, total, err := s.repo.ListNotifications(ctx, params.UserHash, params.PageSize, offset)
	if err != nil {
		return nil, err
	}

	return &GetNotificationsResult{List: list, Total: total, Unread: unread}, nil
}

// GetUnreadNotificationCount 获取未读通知数量
func (s *Service) GetUnreadNotificationCount(ctx context.Context, userHash string) (int, error) {
	return s.repo.CountUnreadNotifications(ctx, userHash)
}

// MarkNotificationRead 标记通知已读
func (s *Service) MarkNotificationRead(ctx context.Context, id int64, userHash string) error {
	return s.repo.MarkNotificationRead(ctx, id, userHash)
}

// MarkAllNotificationsRead 标记所有通知已读
func (s *Service) MarkAllNotificationsRead(ctx context.Context, userHash string) error {
	return s.repo.MarkAllNotificationsRead(ctx, userHash)
}

// GetRatingTrend 获取课程评分趋势
func (s *Service) GetRatingTrend(ctx context.Context, courseID int64) ([]RatingTrendItem, error) {
	return s.repo.GetRatingTrend(ctx, courseID)
}

// GetHotCourses 获取热门课程排行
func (s *Service) GetHotCourses(ctx context.Context, period string, limit int) ([]HotCourse, error) {
	return s.repo.ListHotCourses(ctx, period, limit)
}

// GetTeacherRatingStats 获取教师评分统计
func (s *Service) GetTeacherRatingStats(ctx context.Context, teacherID int64) (*TeacherRatingStatsResponse, error) {
	// 获取教师基本信息（名称 + 院系）
	teacherName, departmentName, err := s.repo.GetTeacherInfo(ctx, teacherID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}

	// 获取评分维度配置
	dimensions, err := s.repo.ListRatingDimensions(ctx)
	if err != nil {
		return nil, err
	}

	// 获取统计数据
	stats, err := s.repo.GetTeacherRatingStats(ctx, teacherID)
	if err != nil {
		return nil, err
	}

	// 获取教师授课课程列表
	courses, err := s.repo.ListTeacherCourses(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	if courses == nil {
		courses = []TeacherCourse{}
	}

	// 获取教师评论总数
	reviewCount, err := s.repo.GetTeacherReviewCount(ctx, teacherID)
	if err != nil {
		return nil, err
	}

	// 构建响应
	resp := &TeacherRatingStatsResponse{
		TeacherID:      teacherID,
		TeacherName:    teacherName,
		DepartmentName: departmentName,
		CourseCount:    len(courses),
		ReviewCount:    reviewCount,
		Courses:        courses,
	}

	// 按学期分组统计
	termStats := make(map[string]*TermRatingStats)
	overallStats := make(map[string]*DimensionStats)

	for _, stat := range stats {
		termID := ""
		if stat.TermID != nil {
			termID = *stat.TermID
		}

		// 查找维度名称
		dimName := stat.DimensionKey
		for _, d := range dimensions {
			if d.Key == stat.DimensionKey {
				dimName = d.Name
				break
			}
		}

		ds := &DimensionStats{
			Key:         stat.DimensionKey,
			Name:        dimName,
			AvgRating:   stat.AvgRating,
			RatingCount: stat.RatingCount,
		}

		if termID == "" {
			// 总体统计
			overallStats[stat.DimensionKey] = ds
		} else {
			// 按学期统计
			if _, ok := termStats[termID]; !ok {
				termStats[termID] = &TermRatingStats{
					TermID:     termID,
					TermName:   termID,
					Dimensions: []DimensionStats{},
				}
			}
			termStats[termID].Dimensions = append(termStats[termID].Dimensions, *ds)
		}
	}

	// 构建总体统计
	var overallDims []DimensionStats
	for _, d := range dimensions {
		if ds, ok := overallStats[d.Key]; ok {
			overallDims = append(overallDims, *ds)
		}
	}
	resp.Overall = TermRatingStats{
		TermID:     "",
		TermName:   "overall",
		Dimensions: overallDims,
	}

	// 构建学期统计列表
	for _, ts := range termStats {
		resp.ByTerm = append(resp.ByTerm, *ts)
	}

	// 构建雷达图数据
	var labels []string
	var data []float64
	for _, d := range dimensions {
		labels = append(labels, d.Name)
		if ds, ok := overallStats[d.Key]; ok {
			data = append(data, ds.AvgRating)
		} else {
			data = append(data, 0)
		}
	}
	resp.RadarChart = RadarChartData{
		Labels: labels,
		Datasets: []RadarChartDataset{
			{
				Label:           teacherName,
				Data:            data,
				BackgroundColor: "rgba(54, 162, 235, 0.2)",
				BorderColor:     "rgba(54, 162, 235, 1)",
			},
		},
	}

	// 计算总体平均分（所有维度的均值）
	if len(overallDims) > 0 {
		var sum float64
		for _, d := range overallDims {
			sum += d.AvgRating
		}
		avg := sum / float64(len(overallDims))
		resp.AvgRating = &avg
	}

	// 构建评分趋势（按学期的平均分）
	ratingTrend := make([]RatingTrendItem, 0, len(resp.ByTerm))
	for _, ts := range resp.ByTerm {
		if len(ts.Dimensions) == 0 {
			continue
		}
		var sum float64
		for _, d := range ts.Dimensions {
			sum += d.AvgRating
		}
		ratingTrend = append(ratingTrend, RatingTrendItem{
			TermID:    ts.TermID,
			TermName:  ts.TermName,
			AvgRating: sum / float64(len(ts.Dimensions)),
		})
	}
	resp.RatingTrend = ratingTrend

	return resp, nil
}

// CheckContent 检查内容是否包含敏感词
func (s *Service) CheckContent(ctx context.Context, content string) *ContentCheckResult {
	return s.filter.CheckContent(ctx, content)
}

// CheckQuality 检查内容质量
func (s *Service) CheckQuality(content string) *QualityCheckResult {
	return s.filter.CheckQuality(content)
}

// BatchUpdateReviewsParams 批量更新评论参数
type BatchUpdateReviewsParams struct {
	IDs    []string
	Action string // hide, restore, delete
}

// BatchUpdateReviewsResult 批量更新评论结果
type BatchUpdateReviewsResult struct {
	Affected int64 `json:"affected"`
}

// BatchUpdateReviews 批量更新评论状态（管理员）
func (s *Service) BatchUpdateReviews(ctx context.Context, params BatchUpdateReviewsParams) (*BatchUpdateReviewsResult, error) {
	var status string
	switch params.Action {
	case "hide":
		status = "hidden"
	case "restore":
		status = "published"
	case "delete":
		status = "deleted"
	default:
		return nil, ErrInvalidAction
	}

	var affected int64
	if err := s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// 先调整课程评论计数（在状态变更前，基于当前状态判断）
		switch params.Action {
		case "delete":
			if err := s.repo.AdjustCourseCountsForBatchDelete(ctx, tx, params.IDs); err != nil {
				return err
			}
		case "restore":
			if err := s.repo.AdjustCourseCountsForBatchRestore(ctx, tx, params.IDs); err != nil {
				return err
			}
		}

		var err error
		affected, err = s.repo.BatchUpdateReviewStatusTx(ctx, tx, params.IDs, status)
		return err
	}); err != nil {
		return nil, err
	}

	return &BatchUpdateReviewsResult{Affected: affected}, nil
}

// LogOperationParams 记录操作日志参数
type LogOperationParams struct {
	AdminUserID   string
	AdminUsername string
	Action        string
	ResourceType  string
	ResourceID    string
	OldValue      interface{}
	NewValue      interface{}
	IPAddress     string
	UserAgent     string
}

// LogOperation 记录操作日志
func (s *Service) LogOperation(ctx context.Context, params LogOperationParams) error {
	var oldValue, newValue []byte
	var err error

	if params.OldValue != nil {
		oldValue, err = json.Marshal(params.OldValue)
		if err != nil {
			return err
		}
	}
	if params.NewValue != nil {
		newValue, err = json.Marshal(params.NewValue)
		if err != nil {
			return err
		}
	}

	return s.repo.CreateOperationLog(ctx, CreateOperationLogParams{
		AdminUserID:   params.AdminUserID,
		AdminUsername: params.AdminUsername,
		Action:        params.Action,
		ResourceType:  params.ResourceType,
		ResourceID:    params.ResourceID,
		OldValue:      oldValue,
		NewValue:      newValue,
		IPAddress:     params.IPAddress,
		UserAgent:     params.UserAgent,
	})
}

// GetOperationLogsParams 获取操作日志参数
type GetOperationLogsParams struct {
	Page     int
	PageSize int
}

// GetOperationLogsResult 获取操作日志结果
type GetOperationLogsResult struct {
	List  []AdminOperationLog
	Total int
}

// GetOperationLogs 获取操作日志列表
func (s *Service) GetOperationLogs(ctx context.Context, params GetOperationLogsParams) (*GetOperationLogsResult, error) {
	offset := (params.Page - 1) * params.PageSize
	list, total, err := s.repo.ListOperationLogs(ctx, params.PageSize, offset)
	if err != nil {
		return nil, err
	}

	return &GetOperationLogsResult{List: list, Total: total}, nil
}

// ExportReviewsParams 导出评论参数
type ExportReviewsParams struct {
	Format string // csv, json
	Status string // all, published, hidden, deleted
}

// ExportReviews 导出评论数据
func (s *Service) ExportReviews(ctx context.Context, params ExportReviewsParams) ([]Review, error) {
	return s.repo.ListAllReviewsForExport(ctx, params.Status)
}

// StreamExportReviews 流式导出评论，逐行回调
func (s *Service) StreamExportReviews(ctx context.Context, status string, fn func(Review) error) error {
	return s.repo.ForEachReviewForExport(ctx, status, fn)
}
