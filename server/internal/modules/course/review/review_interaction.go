package review

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// AddFavorite 添加收藏
func (h *Handler) AddFavorite(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	userID := middleware.GetUserID(c)
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return
	}

	err = h.service.AddFavorite(c.Request.Context(), AddFavoriteParams{
		UserHash: userHash,
		CourseID: courseID,
	})
	if err != nil {
		if errors.Is(err, ErrCourseNotFound) {
			response.NotFound(c, "course not found", errs.ErrCourseNotFound)
			return
		}
		logger.FromGin(c).Error("failed to add favorite", zap.Error(err))
		response.InternalError(c, "failed to add favorite")
		return
	}

	response.Success(c, gin.H{"message": "course favorited successfully"})
}

// RemoveFavorite 取消收藏
func (h *Handler) RemoveFavorite(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	userID := middleware.GetUserID(c)
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return
	}

	err = h.service.RemoveFavorite(c.Request.Context(), userHash, courseID)
	if err != nil {
		logger.FromGin(c).Error("failed to remove favorite", zap.Error(err))
		response.InternalError(c, "failed to remove favorite")
		return
	}

	response.Success(c, gin.H{"message": "favorite removed successfully"})
}

// GetUserFavorites 获取用户收藏列表
func (h *Handler) GetUserFavorites(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	userID := middleware.GetUserID(c)
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return
	}

	result, err := h.service.GetUserFavorites(c.Request.Context(), GetUserFavoritesParams{
		UserHash: userHash,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		logger.FromGin(c).Error("failed to load favorites", zap.Error(err))
		response.InternalError(c, "failed to load favorites")
		return
	}

	response.Success(c, gin.H{"list": result.List, "total": result.Total})
}

// GetUserReviews 获取用户评论列表
func (h *Handler) GetUserReviews(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	userID := middleware.GetUserID(c)
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return
	}

	result, err := h.service.GetUserReviews(c.Request.Context(), GetUserReviewsParams{
		UserHash: userHash,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		logger.FromGin(c).Error("failed to load reviews", zap.Error(err))
		response.InternalError(c, "failed to load reviews")
		return
	}

	response.Success(c, gin.H{"list": result.List, "total": result.Total})
}

// GetUserVotes 获取用户点赞列表
func (h *Handler) GetUserVotes(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	if c.Query("vote_type") != "" {
		response.BadRequest(c, "voteType must be 'like' or 'dislike'")
		return
	}
	voteType := c.DefaultQuery("voteType", "like")
	if voteType != "like" && voteType != "dislike" {
		response.BadRequest(c, "voteType must be 'like' or 'dislike'")
		return
	}
	userID := middleware.GetUserID(c)
	userHash, err := httputil.HashUserID(userID)
	if err != nil {
		response.InternalError(c, "failed to hash user identity")
		return
	}

	result, err := h.service.GetUserVotes(c.Request.Context(), GetUserVotesParams{
		UserHash: userHash,
		VoteType: voteType,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		logger.FromGin(c).Error("failed to load votes", zap.Error(err))
		response.InternalError(c, "failed to load votes")
		return
	}

	response.Success(c, gin.H{"list": result.List, "total": result.Total})
}
