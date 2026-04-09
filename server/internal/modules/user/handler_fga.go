package user

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// reconcileFGAUserProfileTuples 同步 user_profile 的 FGA 关系 tuple。
// - 始终确保 owner 关系存在，供用户查看自己的资料
// - approved=true：再写回当前 school 关系，供学校管理员复核/查看
// - approved=false：仅保留 owner，显式清理 school 关系，避免旧授权残留
func (h *Handler) reconcileFGAUserProfileTuples(c *gin.Context, userID int64, approved bool) {
	if h.fga == nil {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), fga.DefaultWriteTimeout)
	defer cancel()

	// 查询 external_id 和 school_id
	externalID, err := h.service.repo.GetExternalID(ctx, userID)
	if err != nil {
		logger.L().Warn("FGA user_profile: failed to get external ID",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return
	}

	profile, err := h.service.repo.GetProfileByUserID(ctx, userID)
	if err != nil || profile == nil || profile.SchoolID == nil {
		logger.L().Warn("FGA user_profile: failed to get profile or school",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return
	}

	profileObj := "user_profile:" + strconv.FormatInt(userID, 10)
	schoolTuple := fga.Tuple{
		User:     "school:" + strconv.FormatInt(*profile.SchoolID, 10),
		Relation: "school",
		Object:   profileObj,
	}

	if err := h.fga.DeleteTuples(ctx, []fga.Tuple{schoolTuple}); err != nil {
		logger.L().Warn("FGA user_profile: failed to delete tuples",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return
	}

	tuples := []fga.Tuple{{User: "user:" + externalID, Relation: "owner", Object: profileObj}}
	if approved {
		tuples = append(tuples, schoolTuple)
	}

	if err := h.fga.WriteTuples(ctx, tuples); err != nil {
		logger.L().Warn("FGA user_profile: failed to write tuples",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return
	}

	logger.L().Info("FGA user_profile tuples reconciled",
		zap.Int64("user_id", userID),
		zap.Int64("school_id", *profile.SchoolID),
		zap.Bool("approved", approved),
	)
}
