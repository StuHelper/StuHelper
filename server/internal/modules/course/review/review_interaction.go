package review

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// AddFavorite 添加收藏
func (h *Handler) AddFavorite(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "courseID")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	_, userHash, ok := h.resolveRequiredUserHash(c)
	if !ok {
		return
	}

	err = h.service.AddFavorite(c.Request.Context(), AddFavoriteParams{
		UserHash: userHash,
		CourseID: courseID,
	})
	if err != nil {
		if respondAddFavoriteError(c, err) {
			return
		}
		logger.FromGin(c).Error("failed to add favorite", zap.Error(err))
		response.InternalError(c, "failed to add favorite")
		return
	}

	response.Success(c, gin.H{"message": "course favorited successfully"})
}

// GetFavoriteStatus 获取当前用户对课程的收藏状态
func (h *Handler) GetFavoriteStatus(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "courseID")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	_, userHash, ok := h.resolveRequiredUserHash(c)
	if !ok {
		return
	}

	favorited, err := h.service.GetFavoriteStatus(c.Request.Context(), userHash, courseID)
	if err != nil {
		logger.FromGin(c).Error("failed to get favorite status", zap.Error(err))
		response.InternalError(c, "failed to get favorite status")
		return
	}

	response.Success(c, gin.H{"favorited": favorited})
}

// RemoveFavorite 取消收藏
func (h *Handler) RemoveFavorite(c *gin.Context) {
	courseID, err := httputil.ParseIDParam(c, "courseID")
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	_, userHash, ok := h.resolveRequiredUserHash(c)
	if !ok {
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
	_, userHash, ok := h.resolveRequiredUserHash(c)
	if !ok {
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
	_, userHash, ok := h.resolveRequiredUserHash(c)
	if !ok {
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
	_, userHash, ok := h.resolveRequiredUserHash(c)
	if !ok {
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
