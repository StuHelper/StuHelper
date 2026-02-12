package review

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/sanitizer"
)

// 回复相关错误
var (
	ErrReplyNotFound  = errors.New("reply not found")
	ErrNotReplyOwner  = errors.New("not the reply owner")
	ErrContentTooLong = errors.New("content too long")
)

// ReportReviewParams 举报评论参数
type ReportReviewParams struct {
	ReviewID    string
	UserHash    string
	Reason      string
	Description string
}

// ReportReview 举报评论
func (s *Service) ReportReview(ctx context.Context, params ReportReviewParams) error {
	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// 在事务内检查评论是否存在，消除 TOCTOU 竞态
		exists, err := s.repo.ReviewExistsTx(ctx, tx, params.ReviewID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrReviewNotFound
		}

		reported, err := s.repo.ReportExistsTx(ctx, tx, params.ReviewID, params.UserHash)
		if err != nil {
			return err
		}
		if reported {
			return ErrAlreadyReported
		}

		return s.repo.CreateReportTx(ctx, tx, CreateReportParams{
			ReviewID:     params.ReviewID,
			ReporterHash: params.UserHash,
			Reason:       params.Reason,
			Description:  params.Description,
		})
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
			currentStatus, courseID, err := s.repo.GetReviewStatusAndCourseIDTx(ctx, tx, report.ReviewID)
			if err != nil {
				return err
			}
			if err := s.repo.UpdateReviewStatus(ctx, tx, report.ReviewID, "hidden"); err != nil {
				return err
			}
			// 仅从 published 状态隐藏时递减计数
			if currentStatus == "published" {
				if err := s.repo.DecrementCourseReviewCount(ctx, tx, courseID); err != nil {
					return err
				}
			}
		case "delete_review":
			reportStatus = "resolved"
			currentStatus, courseID, err := s.repo.GetReviewStatusAndCourseIDTx(ctx, tx, report.ReviewID)
			if err != nil {
				return err
			}
			if err := s.repo.SoftDeleteReview(ctx, tx, report.ReviewID); err != nil {
				return err
			}
			// 仅从 published 状态删除时递减计数
			if currentStatus == "published" {
				if err := s.repo.DecrementCourseReviewCount(ctx, tx, courseID); err != nil {
					return err
				}
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

// CreateReplyResult 创建回复结果
type CreateReplyResult struct {
	Reply Reply
}

// CreateReply 创建回复
func (s *Service) CreateReply(ctx context.Context, params CreateReplyParams) (*CreateReplyResult, error) {
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

	var replyID string
	now := time.Now()
	if err := s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// 在事务内检查评论是否存在，消除 TOCTOU 竞态
		exists, err := s.repo.ReviewExistsTx(ctx, tx, params.ReviewID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrReviewNotFound
		}

		replyID, err = s.repo.CreateReply(ctx, tx, CreateReplyParams{
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
			ID:        replyID,
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

	for i := range list {
		if params.UserHash != "" {
			list[i].IsOwner = list[i].UserHash == params.UserHash
		}
		list[i].UserHash = ""
	}

	return &GetRepliesResult{List: list, Total: total}, nil
}

// DeleteReplyParams 删除回复参数
type DeleteReplyParams struct {
	ReplyID  string
	UserHash string
}

// DeleteReply 删除回复
func (s *Service) DeleteReply(ctx context.Context, params DeleteReplyParams) error {
	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// 在事务内获取回复所有者、评论ID和状态，消除 TOCTOU 竞态
		ownerHash, reviewID, status, err := s.repo.GetReplyOwnerAndReviewIDTx(ctx, tx, params.ReplyID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReplyNotFound
			}
			return err
		}

		// 已删除的回复不可重复删除（防止并发双击导致计数双减）
		if status == "deleted" {
			return ErrReplyNotFound
		}

		// 验证是否为本人
		if ownerHash != params.UserHash {
			return ErrNotReplyOwner
		}

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
func (s *Service) MarkNotificationRead(ctx context.Context, id string, userHash string) error {
	return s.repo.MarkNotificationRead(ctx, id, userHash)
}

// MarkAllNotificationsRead 标记所有通知已读
func (s *Service) MarkAllNotificationsRead(ctx context.Context, userHash string) error {
	return s.repo.MarkAllNotificationsRead(ctx, userHash)
}
