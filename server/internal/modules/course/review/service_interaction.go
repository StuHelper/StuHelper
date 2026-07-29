package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/sanitizer"
)

const maxReplyContentRunes = 1000

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
	var err error
	params.UserHash, err = normalizeRequiredUserHash(params.UserHash)
	if err != nil {
		return err
	}
	params.CourseID, err = normalizeRequiredCourseID(params.CourseID)
	if err != nil {
		return err
	}
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
	userHash, err := normalizeRequiredUserHash(userHash)
	if err != nil {
		return err
	}
	courseID, err = normalizeRequiredCourseID(courseID)
	if err != nil {
		return err
	}
	return s.repo.DeleteFavorite(ctx, userHash, courseID)
}

// GetFavoriteStatus 获取当前用户对课程的收藏状态
func (s *Service) GetFavoriteStatus(ctx context.Context, userHash string, courseID int64) (bool, error) {
	userHash, err := normalizeRequiredUserHash(userHash)
	if err != nil {
		return false, err
	}
	courseID, err = normalizeRequiredCourseID(courseID)
	if err != nil {
		return false, err
	}
	return s.repo.FavoriteExists(ctx, userHash, courseID)
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
	var err error
	params.UserHash, err = normalizeRequiredUserHash(params.UserHash)
	if err != nil {
		return nil, err
	}
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
	var err error
	params.UserHash, err = normalizeRequiredUserHash(params.UserHash)
	if err != nil {
		return nil, err
	}
	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	list, total, err := s.repo.ListByUserHash(ctx, params.UserHash, pageSize, offset)
	if err != nil {
		return nil, err
	}
	if err := s.populateUserVotes(ctx, params.UserHash, list); err != nil {
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
	var err error
	params.UserHash, err = normalizeRequiredUserHash(params.UserHash)
	if err != nil {
		return nil, err
	}
	if !isValidVoteType(params.VoteType) {
		return nil, ErrInvalidAction
	}

	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	list, total, err := s.repo.ListVotedReviews(ctx, params.UserHash, params.VoteType, pageSize, offset)
	if err != nil {
		return nil, err
	}
	setKnownUserVote(list, params.VoteType)

	return &GetCourseReviewsResult{List: list, Total: total}, nil
}

// SaveDraftParams 保存草稿参数
type SaveDraftParams struct {
	UserHash  string
	CourseID  *int64
	TeacherID *int64
	TermID    string
	Title     string
	Content   string
	Grade     string
	Ratings   ReviewRatings
}

// SaveDraft 保存草稿（UPSERT）
func (s *Service) SaveDraft(ctx context.Context, params SaveDraftParams) (*ReviewDraft, error) {
	var err error
	params.UserHash, err = normalizeRequiredUserHash(params.UserHash)
	if err != nil {
		return nil, err
	}
	if params.CourseID != nil {
		if _, err := normalizeRequiredCourseID(*params.CourseID); err != nil {
			return nil, err
		}
	}
	if params.TeacherID != nil {
		if params.CourseID == nil {
			return nil, ErrTeacherNotFound
		}
		if _, err := normalizeRequiredTeacherID(*params.TeacherID); err != nil {
			return nil, err
		}
	}
	if params.CourseID != nil {
		exists, err := s.repo.CourseExists(ctx, *params.CourseID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrCourseNotFound
		}
	}
	if params.TeacherID != nil {
		teacherExists, err := s.repo.TeacherBelongsToCourseSchool(ctx, *params.TeacherID, *params.CourseID)
		if err != nil {
			return nil, err
		}
		if !teacherExists {
			return nil, ErrTeacherNotFound
		}
	}

	// termID 格式校验（与发布链路一致）
	if params.TermID != "" && !validTermIDFormat.MatchString(params.TermID) {
		return nil, ErrInvalidTermID
	}

	// XSS 防护
	if sanitizer.ContainsDangerousContent(params.Title) || sanitizer.ContainsDangerousContent(params.Content) {
		return nil, ErrDangerousContent
	}
	rawContent := params.Content
	params.Title = sanitizer.SanitizeTitle(params.Title)
	params.Content = sanitizer.SanitizeText(params.Content)
	if err := validateDraftTextLengths(params.Title, rawContent, params.Content); err != nil {
		return nil, err
	}
	grade, err := normalizeReviewGrade(params.Grade)
	if err != nil {
		return nil, err
	}
	params.Grade = grade

	// 序列化评分数据
	ratings := params.Ratings
	if ratings == nil {
		ratings = ReviewRatings{}
	}
	if err := s.validateRatingValues(ctx, ratings, false); err != nil {
		return nil, err
	}
	ratingsData, err := json.Marshal(ratings)
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

func validateDraftTextLengths(title, rawContent, content string) error {
	if utf8.RuneCountInString(title) > maxReviewTitleRunes {
		return ErrTitleTooLong
	}
	if rawContent == "" {
		return nil
	}
	if strings.TrimSpace(content) == "" {
		return ErrContentEmpty
	}
	if utf8.RuneCountInString(content) > maxReviewContentRunes {
		return ErrContentTooLong
	}
	return nil
}

// GetDraft 获取草稿
func (s *Service) GetDraft(ctx context.Context, userHash string) (*ReviewDraft, error) {
	userHash, err := normalizeRequiredUserHash(userHash)
	if err != nil {
		return nil, err
	}
	return s.repo.GetDraft(ctx, userHash)
}

// DeleteDraft 删除草稿
func (s *Service) DeleteDraft(ctx context.Context, userHash string) error {
	userHash, err := normalizeRequiredUserHash(userHash)
	if err != nil {
		return err
	}
	return s.repo.DeleteDraft(ctx, userHash)
}

// CreateReplyResult 创建回复结果
type CreateReplyResult struct {
	Reply Reply
}

// CreateReply 创建回复
func (s *Service) CreateReply(ctx context.Context, params CreateReplyParams) (*CreateReplyResult, error) {
	var err error
	params.UserHash, err = normalizeRequiredUserHash(params.UserHash)
	if err != nil {
		return nil, err
	}
	content, err := validateAndSanitizeReplyContent(params.Content)
	if err != nil {
		return nil, err
	}
	params.Content = content

	// 敏感词检查
	checkResult, err := s.filter.CheckContent(ctx, params.Content)
	if err != nil {
		return nil, err
	}

	replyStatus, err := buildReplyModerationStatus(checkResult)
	if err != nil {
		return nil, err
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
		if params.ParentID != nil {
			belongs, err := s.repo.ReplyBelongsToReviewTx(ctx, tx, *params.ParentID, params.ReviewID)
			if err != nil {
				return err
			}
			if !belongs {
				return ErrReplyNotFound
			}
		}

		replyID, replyTS, err = s.repo.CreateReply(ctx, tx, CreateReplyParams{
			ReviewID: params.ReviewID,
			ParentID: params.ParentID,
			UserHash: params.UserHash,
			Content:  params.Content,
			Status:   replyStatus,
		})
		if err != nil {
			return err
		}
		if !isPublicReviewStatus(replyStatus) {
			return nil
		}
		return s.repo.IncrementReplyCount(ctx, tx, params.ReviewID)
	}); err != nil {
		return nil, err
	}

	// 防御性 nil 检查，RETURNING 正常情况下不会返回 nil
	if replyTS == nil {
		return nil, fmt.Errorf("CreateReply: unexpected nil timestamps from RETURNING clause")
	}

	// 发送回复通知给评价作者
	if isPublicReviewStatus(replyStatus) {
		s.dispatchNotification(ctx, func(notifCtx context.Context) {
			s.sendReplyNotification(notifCtx, params.ReviewID, params.UserHash)
		})
	}

	return &CreateReplyResult{
		Reply: Reply{
			ID:        replyID,
			ReviewID:  params.ReviewID,
			ParentID:  params.ParentID,
			Content:   params.Content,
			LikeCount: 0,
			Status:    replyStatus,
			IsOwner:   true,
			CreatedAt: replyTS.CreatedAt,
			UpdatedAt: replyTS.UpdatedAt,
		},
	}, nil
}

func validateAndSanitizeReplyContent(content string) (string, error) {
	if sanitizer.ContainsDangerousContent(content) {
		return "", ErrDangerousContent
	}
	content = sanitizer.SanitizeText(content)
	if sanitizer.ContainsDangerousContent(content) {
		return "", ErrDangerousContent
	}
	if strings.TrimSpace(content) == "" {
		return "", ErrContentEmpty
	}
	if utf8.RuneCountInString(content) > maxReplyContentRunes {
		return "", ErrContentTooLong
	}
	return content, nil
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
	var err error
	params.UserHash, err = normalizeRequiredUserHash(params.UserHash)
	if err != nil {
		return err
	}
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
		if isPublicReviewStatus(status) {
			return s.repo.DecrementReplyCount(ctx, tx, reviewID)
		}
		return nil
	})
}

// sendReplyNotification 异步发送回复通知给评价作者
func (s *Service) sendReplyNotification(ctx context.Context, reviewID, replierHash string) {
	review, err := s.repo.GetReviewByID(ctx, reviewID)
	if err != nil || review == nil {
		return
	}
	// 自己回复自己时不发送通知
	if review.UserHash == replierHash {
		return
	}
	authorID, err := s.repo.GetUserIDByUserHash(ctx, review.UserHash)
	if err != nil || authorID == 0 {
		return
	}
	if err := s.notifSender.SendReviewNotification(ctx, ReviewNotification{
		UserID:       authorID,
		Type:         "reply",
		Title:        "你的评价收到了新回复",
		Body:         "有人回复了你对课程的评价",
		SourceModule: "review",
		SourceID:     reviewID,
		CourseID:     review.CourseID,
	}); err != nil {
		logger.L().Warn("failed to send reply notification",
			zap.String("review_id", reviewID),
			zap.Int64("author_id", authorID),
			zap.Error(err),
		)
	}
}
