package review

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// ReviewWriteAccess 是用户写操作进入 Service 的最终准入事实。
type ReviewWriteAccess struct {
	InternalUserID int64
	CanPostReview  bool
	CanEditOwn     bool
	CanDeleteOwn   bool
}

func (access ReviewWriteAccess) requirePostReview(authorInternalUserID int64) error {
	if !access.CanPostReview || access.InternalUserID <= 0 || authorInternalUserID <= 0 || authorInternalUserID != access.InternalUserID {
		return ErrReviewWriteAccessDenied
	}
	return nil
}

func (access ReviewWriteAccess) requireEditOwn() error {
	if !access.CanEditOwn || access.InternalUserID <= 0 {
		return ErrReviewWriteAccessDenied
	}
	return nil
}

func (access ReviewWriteAccess) requireDeleteOwn() error {
	if !access.CanDeleteOwn || access.InternalUserID <= 0 {
		return ErrReviewWriteAccessDenied
	}
	return nil
}

// PostReviewParams 发布评论参数
type PostReviewParams struct {
	CourseID             int64
	TeacherID            *int64
	TermID               string
	Title                string
	Content              string
	Grade                string
	Ratings              ReviewRatings
	UserHash             string
	AuthorInternalUserID int64
	Access               ReviewWriteAccess
	IPAddress            string
	RequestID            string
}

// PostReviewResult 发布评论结果
type PostReviewResult struct {
	Review Review
}

// VoteReviewParams 投票参数
type VoteReviewParams struct {
	ReviewID string
	UserHash string
	VoteType string
}

// UpdateReviewParams 更新评论参数
type UpdateReviewParams struct {
	ReviewID string
	UserHash string
	Access   ReviewWriteAccess
	Title    *string
	Content  *string
	Grade    *string
	Ratings  *ReviewRatings
}

// DeleteReviewParams 删除评论参数
type DeleteReviewParams struct {
	ReviewID string
	UserHash string
	Access   ReviewWriteAccess
}

// PostReview 发布评论
func (s *Service) PostReview(ctx context.Context, params PostReviewParams) (*PostReviewResult, error) {
	var err error
	var contentFlag *string
	var status string
	if err := params.Access.requirePostReview(params.AuthorInternalUserID); err != nil {
		return nil, err
	}
	params.UserHash, err = normalizeRequiredUserHash(params.UserHash)
	if err != nil {
		return nil, err
	}
	authorUserID, err := formatFGAUserID(params.AuthorInternalUserID)
	if err != nil {
		return nil, err
	}
	params.Grade, err = normalizeReviewGrade(params.Grade)
	if err != nil {
		return nil, err
	}
	params.CourseID, err = normalizeRequiredCourseID(params.CourseID)
	if err != nil {
		return nil, err
	}
	if params.TeacherID != nil {
		teacherID, err := normalizeRequiredTeacherID(*params.TeacherID)
		if err != nil {
			return nil, err
		}
		params.TeacherID = &teacherID
	}
	params.Title, params.Content, status, contentFlag, err = s.validateAndSanitizeReview(ctx, params.Ratings, params.Title, params.Content, params.TermID)
	if err != nil {
		return nil, err
	}

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
		exists, err := s.repo.CourseExistsTx(ctx, tx, params.CourseID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrCourseNotFound
		}
		schoolID, err := s.repo.GetCourseSchoolIDTx(ctx, tx, params.CourseID)
		if err != nil {
			return err
		}

		if params.TeacherID != nil {
			teacherExists, err := s.repo.TeacherBelongsToCourseSchoolTx(ctx, tx, *params.TeacherID, params.CourseID)
			if err != nil {
				return err
			}
			if !teacherExists {
				return ErrTeacherNotFound
			}
		}

		hasReviewed, err := s.repo.UserHasReviewedCourseTx(ctx, tx, params.UserHash, params.CourseID)
		if err != nil {
			return err
		}
		if hasReviewed {
			return ErrAlreadyReviewed
		}

		created, err := s.repo.CreateReturning(ctx, tx, CreateParams{
			ID:          reviewID,
			CourseID:    params.CourseID,
			SchoolID:    schoolID,
			TeacherID:   params.TeacherID,
			TermID:      params.TermID,
			Title:       params.Title,
			Content:     params.Content,
			Grade:       params.Grade,
			Ratings:     ratingsData,
			UserHash:    params.UserHash,
			Status:      status,
			ContentFlag: contentFlag,
		})
		if err != nil {
			if isUniqueConstraintViolation(err, "idx_reviews_user_course") {
				return ErrAlreadyReviewed
			}
			return err
		}
		review = *created
		if !isPublicReviewStatus(review.Status) {
			return s.enqueueReviewFGASyncTx(ctx, tx, reviewID, authorUserID, schoolID)
		}
		if err := s.repo.IncrementCourseReviewCount(ctx, tx, params.CourseID); err != nil {
			return err
		}
		if err := s.refreshReviewTargetTx(ctx, tx, params.CourseID, params.TeacherID); err != nil {
			return err
		}
		return s.enqueueReviewFGASyncTx(ctx, tx, reviewID, authorUserID, schoolID)
	}); err != nil {
		return nil, err
	}

	audit.LogContext(ctx, audit.Event{
		Type:         audit.EventDataCreate,
		UserID:       maskHash(params.UserHash),
		IP:           params.IPAddress,
		RequestID:    params.RequestID,
		Resource:     "review",
		ResourceType: "review",
		ResourceID:   reviewID,
		Action:       "post_review",
		Result:       "success",
		Details: map[string]interface{}{
			"review_id": reviewID,
			"course_id": params.CourseID,
			"status":    review.Status,
		},
	})

	return &PostReviewResult{Review: review}, nil
}

// VoteReview 投票
func (s *Service) VoteReview(ctx context.Context, params VoteReviewParams) (int64, error) {
	var err error
	params.UserHash, err = normalizeRequiredUserHash(params.UserHash)
	if err != nil {
		return 0, err
	}
	if !isValidVoteType(params.VoteType) {
		return 0, ErrInvalidAction
	}

	var courseID int64
	var shouldNotifyLike bool
	err = s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		exists, err := s.repo.ReviewExistsTx(ctx, tx, params.ReviewID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrReviewNotFound
		}

		courseID, err = s.repo.GetReviewCourseIDTx(ctx, tx, params.ReviewID)
		if err != nil {
			return err
		}

		existing, err := s.repo.GetVoteType(ctx, tx, params.ReviewID, params.UserHash)
		if err != nil {
			return err
		}

		switch existing {
		case "":
			inserted, err := s.repo.CreateVote(ctx, tx, params.ReviewID, params.UserHash, params.VoteType)
			if err != nil {
				return err
			}
			if !inserted {
				return nil
			}
			if params.VoteType == voteTypeLike {
				shouldNotifyLike = true
				return s.repo.IncrementLikeCount(ctx, tx, params.ReviewID)
			}
			return s.repo.IncrementDislikeCount(ctx, tx, params.ReviewID)

		case params.VoteType:
			if err := s.repo.DeleteVote(ctx, tx, params.ReviewID, params.UserHash); err != nil {
				return err
			}
			if existing == voteTypeLike {
				return s.repo.DecrementLikeCount(ctx, tx, params.ReviewID)
			}
			return s.repo.DecrementDislikeCount(ctx, tx, params.ReviewID)

		default:
			if err := s.repo.UpdateVoteType(ctx, tx, params.ReviewID, params.UserHash, params.VoteType); err != nil {
				return err
			}
			if params.VoteType == voteTypeLike {
				shouldNotifyLike = true
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
	if err != nil {
		return 0, err
	}
	// 仅在新增 upvote 时通知评价作者
	if shouldNotifyLike {
		s.dispatchNotification(ctx, func(notifCtx context.Context) {
			s.sendVoteNotification(notifCtx, params.ReviewID, params.UserHash)
		})
	}
	return courseID, nil
}

// UpdateReview 更新评论
func (s *Service) UpdateReview(ctx context.Context, params UpdateReviewParams) error {
	var err error
	if err := params.Access.requireEditOwn(); err != nil {
		return err
	}
	params.UserHash, err = normalizeRequiredUserHash(params.UserHash)
	if err != nil {
		return err
	}
	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		state, err := s.repo.GetReviewUpdateStateTx(ctx, tx, params.ReviewID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReviewNotFound
			}
			return err
		}
		if state.status != StatusPublished && state.status != StatusPendingReview {
			return ErrReviewNotFound
		}
		if state.userHash != params.UserHash {
			return ErrNotReviewOwner
		}

		title, content, grade, ratings := mergeReviewUpdate(state, params)
		grade, err = normalizeReviewGrade(grade)
		if err != nil {
			return err
		}
		sanitizedTitle, sanitizedContent, nextStatus, contentFlag, err := s.validateAndSanitizeReview(ctx, ratings, title, content, "")
		if err != nil {
			return err
		}
		ratingsData, err := json.Marshal(ratings)
		if err != nil {
			return err
		}

		if err := s.repo.Update(ctx, tx, UpdateParams{
			ID:          params.ReviewID,
			Title:       sanitizedTitle,
			Content:     sanitizedContent,
			Grade:       grade,
			Ratings:     ratingsData,
			Status:      nextStatus,
			ContentFlag: contentFlag,
		}); err != nil {
			return err
		}

		wasPublic := isPublicReviewStatus(state.status)
		isPublic := isPublicReviewStatus(nextStatus)
		switch {
		case wasPublic && !isPublic:
			if err := s.repo.DecrementCourseReviewCount(ctx, tx, state.courseID); err != nil {
				return err
			}
		case !wasPublic && isPublic:
			if err := s.repo.IncrementCourseReviewCount(ctx, tx, state.courseID); err != nil {
				return err
			}
		}

		if wasPublic || isPublic {
			return s.refreshReviewTargetTx(ctx, tx, state.courseID, state.teacherID)
		}
		return nil
	})
}

func mergeReviewUpdate(state reviewUpdateState, params UpdateReviewParams) (string, string, string, ReviewRatings) {
	title := state.title
	if params.Title != nil {
		title = *params.Title
	}

	content := state.content
	if params.Content != nil {
		content = *params.Content
	}

	grade := state.grade
	if params.Grade != nil {
		grade = *params.Grade
	}

	ratings := cloneReviewRatings(state.ratings)
	if params.Ratings != nil {
		ratings = cloneReviewRatings(*params.Ratings)
	}
	return title, content, grade, ratings
}

func cloneReviewRatings(source ReviewRatings) ReviewRatings {
	if len(source) == 0 {
		return ReviewRatings{}
	}
	cloned := make(ReviewRatings, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

// DeleteReview 删除评论
func (s *Service) DeleteReview(ctx context.Context, params DeleteReviewParams) error {
	var err error
	if err := params.Access.requireDeleteOwn(); err != nil {
		return err
	}
	params.UserHash, err = normalizeRequiredUserHash(params.UserHash)
	if err != nil {
		return err
	}
	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		ownerHash, courseID, teacherID, status, err := s.repo.GetReviewOwnerCourseTeacherStatusTx(ctx, tx, params.ReviewID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReviewNotFound
			}
			return err
		}
		if status == StatusDeleted {
			return ErrReviewNotFound
		}
		if ownerHash != params.UserHash {
			return ErrNotReviewOwner
		}

		if err := s.repo.SoftDeleteReview(ctx, tx, params.ReviewID); err != nil {
			return err
		}
		if status == StatusPublished {
			if err := s.repo.DecrementCourseReviewCount(ctx, tx, courseID); err != nil {
				return err
			}
			return s.refreshReviewTargetTx(ctx, tx, courseID, teacherID)
		}
		return nil
	})
}

// sendVoteNotification 异步发送点赞通知给评价作者
func (s *Service) sendVoteNotification(ctx context.Context, reviewID, voterHash string) {
	review, err := s.repo.GetReviewByID(ctx, reviewID)
	if err != nil || review == nil {
		return
	}
	if review.UserHash == voterHash {
		return
	}
	authorID, err := s.repo.GetUserIDByUserHash(ctx, review.UserHash)
	if err != nil || authorID == 0 {
		return
	}
	if err := s.notifSender.SendReviewNotification(ctx, ReviewNotification{
		UserID:       authorID,
		Type:         voteTypeLike,
		Title:        "你的评价获得了一个赞",
		Body:         "有人赞了你的评价",
		SourceModule: "review",
		SourceID:     reviewID,
		CourseID:     review.CourseID,
	}); err != nil {
		logger.L().Warn("failed to send vote notification",
			zap.String("review_id", reviewID),
			zap.Int64("author_id", authorID),
			zap.Error(err),
		)
	}
}
