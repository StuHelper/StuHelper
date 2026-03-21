package auth

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// GetCurrentUser 获取当前用户信息
func (h *Handler) GetCurrentUser(c *gin.Context) {
	userInfo, err := h.buildUserInfo(
		c.Request.Context(),
		middleware.GetUserID(c),
		middleware.GetUsername(c),
		middleware.GetDisplayName(c),
		middleware.GetEmail(c),
		nil,
		middleware.GetIsAdmin(c),
	)
	if err != nil {
		logger.FromGin(c).Error("failed to build current user info", zap.Error(err))
		response.ServiceUnavailable(c, "identity service temporarily unavailable")
		return
	}
	response.Success(c, userInfo)
}

func (h *Handler) buildUserInfo(
	ctx context.Context,
	userID string,
	fallbackName string,
	fallbackDisplayName string,
	fallbackEmail string,
	fallbackAvatar *string,
	fallbackIsPlatformAdmin bool,
) (gin.H, error) {
	log := logger.FromContext(ctx)

	// 1. Casdoor 缓存：失败用 JWT fallback 值继续，不中断
	cachedUser, err := h.ssoClient.GetCachedUserByID(ctx, userID)
	if err != nil {
		log.Warn("casdoor cache unavailable, using token fallback",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		cachedUser = nil
	}

	name := fallbackName
	displayName := fallbackDisplayName
	email := fallbackEmail
	isPlatformAdmin := fallbackIsPlatformAdmin

	if cachedUser != nil {
		if cachedUser.Name != "" {
			name = cachedUser.Name
		}
		if cachedUser.DisplayName != "" {
			displayName = cachedUser.DisplayName
		}
		if cachedUser.Email != "" {
			email = cachedUser.Email
		}
		isPlatformAdmin = cachedUser.IsAdmin
	}

	if displayName == "" {
		displayName = name
	}

	// 2. 用户同步：失败仅 warn，不影响用户体验
	if h.userSyncRepo != nil {
		if err := h.userSyncRepo.UpsertUser(ctx, UserSyncInput{
			ExternalID: userID,
			Username:   name,
			Email:      email,
			AvatarURL:  fallbackAvatar,
		}); err != nil {
			log.Warn("user sync failed, skipping",
				zap.String("user_id", userID),
				zap.Error(err),
			)
		}
	}

	// 3. 能力查询：失败返回空集（最小权限原则）
	capabilitySnapshot := capability.UserAccessSnapshot{
		Capabilities:       []string{},
		GlobalCapabilities: []string{},
		CapabilityGrants:   []capability.Grant{},
	}
	if h.capabilityReader != nil {
		resolved, err := h.capabilityReader.GetUserCapabilitySnapshot(ctx, userID)
		if err != nil {
			log.Warn("capability query failed, returning empty permission set",
				zap.String("user_id", userID),
				zap.Error(err),
			)
		} else {
			capabilitySnapshot = resolved
		}
	}

	return gin.H{
		"id":                 userID,
		"name":               name,
		"displayName":        displayName,
		"email":              email,
		"avatar":             fallbackAvatar,
		"isPlatformAdmin":    isPlatformAdmin,
		"capabilities":       capabilitySnapshot.Capabilities,
		"globalCapabilities": capabilitySnapshot.GlobalCapabilities,
		"capabilityGrants":   capabilitySnapshot.CapabilityGrants,
		"canAccessAdmin":     capability.CanAccessAdmin(capabilitySnapshot.GlobalCapabilities),
	}, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
