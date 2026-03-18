package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/sanitizer"
)

// 回复相关错误
var (
	ErrReplyNotFound  = errors.New("reply not found")
	ErrNotReplyOwner  = errors.New("not the reply owner")
	ErrContentTooLong = errors.New("content too long")
)

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
	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	list, total, err := s.repo.ListFavorites(ctx, params.UserHash, pageSize, offset)
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
	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	list, total, err := s.repo.ListByUserHash(ctx, params.UserHash, pageSize, offset)
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
	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	list, total, err := s.repo.ListVotedReviews(ctx, params.UserHash, params.VoteType, pageSize, offset)
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

	// termID 格式校验（与发布链路一致）
	if params.TermID != "" && !validTermIDFormat.MatchString(params.TermID) {
		return nil, fmt.Errorf("invalid term_id format, expected YYYY-S (e.g. 2024-1)")
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
	return s.repo.GetDraft(ctx, userHash, courseID)
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
	var replyTS *ReplyTimestamps
	if err := s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// 在事务内检查评论是否存在，消除 TOCTOU 竞态
		exists, err := s.repo.ReviewExistsTx(ctx, tx, params.ReviewID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrReviewNotFound
		}

		replyID, replyTS, err = s.repo.CreateReply(ctx, tx, CreateReplyParams{
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

	// M-116: 防御性 nil 检查，RETURNING 正常情况下不会返回 nil
	if replyTS == nil {
		return nil, fmt.Errorf("CreateReply: unexpected nil timestamps from RETURNING clause")
	}

	return &CreateReplyResult{
		Reply: Reply{
			ID:        replyID,
			ReviewID:  params.ReviewID,
			ParentID:  params.ParentID,
			Content:   params.Content,
			LikeCount: 0,
			Status:    StatusPublished,
			IsOwner:   true,
			CreatedAt: replyTS.CreatedAt,
			UpdatedAt: replyTS.UpdatedAt,
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
	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	list, total, err := s.repo.ListReplies(ctx, params.ReviewID, pageSize, offset)
	if err != nil {
		return nil, err
	}

	for i := range list {
		if params.UserHash != "" {
			list[i].IsOwner = list[i].UserHash == params.UserHash
		}
		// 纵深防御：json:"-" 阻止序列化，手动清空防止 tag 被误删后泄漏
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
		if status == StatusDeleted {
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
	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	result, err := s.repo.ListNotifications(ctx, params.UserHash, pageSize, offset)
	if err != nil {
		return nil, err
	}

	return &GetNotificationsResult{List: result.List, Total: result.Total, Unread: result.Unread}, nil
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
