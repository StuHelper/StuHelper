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
	RequestID string
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
	Title    string
	Content  string
	Grade    string
	Ratings  ReviewRatings
}

// DeleteReviewParams 删除评论参数
type DeleteReviewParams struct {
	ReviewID string
	UserHash string
}

// PostReview 发布评论
func (s *Service) PostReview(ctx context.Context, params PostReviewParams) (*PostReviewResult, error) {
	var err error
	params.Title, params.Content, err = s.validateAndSanitizeReview(ctx, params.Ratings, params.Title, params.Content, params.TermID)
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

		if params.TeacherID != nil {
			teacherExists, err := s.repo.TeacherExistsTx(ctx, tx, *params.TeacherID)
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

// VoteReview 投票
func (s *Service) VoteReview(ctx context.Context, params VoteReviewParams) error {
	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
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

		switch existing {
		case "":
			inserted, err := s.repo.CreateVote(ctx, tx, params.ReviewID, params.UserHash, params.VoteType)
			if err != nil {
				return err
			}
			if !inserted {
				return nil
			}
			if params.VoteType == "like" {
				return s.repo.IncrementLikeCount(ctx, tx, params.ReviewID)
			}
			return s.repo.IncrementDislikeCount(ctx, tx, params.ReviewID)

		case params.VoteType:
			if err := s.repo.DeleteVote(ctx, tx, params.ReviewID, params.UserHash); err != nil {
				return err
			}
			if existing == "like" {
				return s.repo.DecrementLikeCount(ctx, tx, params.ReviewID)
			}
			return s.repo.DecrementDislikeCount(ctx, tx, params.ReviewID)

		default:
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

// UpdateReview 更新评论
func (s *Service) UpdateReview(ctx context.Context, params UpdateReviewParams) error {
	var err error
	params.Title, params.Content, err = s.validateAndSanitizeReview(ctx, params.Ratings, params.Title, params.Content, "")
	if err != nil {
		return err
	}

	ratingsData, err := json.Marshal(params.Ratings)
	if err != nil {
		return err
	}

	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		ownerHash, status, err := s.repo.GetReviewOwnerAndStatusTx(ctx, tx, params.ReviewID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReviewNotFound
			}
			return err
		}
		if status != StatusPublished {
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

// DeleteReview 删除评论
func (s *Service) DeleteReview(ctx context.Context, params DeleteReviewParams) error {
	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		ownerHash, courseID, status, err := s.repo.GetReviewOwnerCourseIDAndStatusTx(ctx, tx, params.ReviewID)
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
			return s.repo.RefreshCourseRatingStatsTx(ctx, tx, courseID)
		}
		return nil
	})
}
