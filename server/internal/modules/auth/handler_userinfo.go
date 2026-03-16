package auth

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
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
	cachedUser, err := h.ssoClient.GetCachedUserByID(ctx, userID)
	if err != nil {
		return nil, err
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

	if h.userSyncRepo != nil {
		if err := h.userSyncRepo.UpsertUser(ctx, UserSyncInput{
			ExternalID: userID,
			Username:   name,
			Email:      email,
			AvatarURL:  fallbackAvatar,
		}); err != nil {
			return nil, err
		}
	}

	capabilities := []string{}
	if h.capabilityReader != nil {
		resolvedCapabilities, err := h.capabilityReader.GetUserCapabilities(ctx, userID)
		if err != nil {
			return nil, err
		}
		capabilities = capability.Normalize(resolvedCapabilities)
	}

	return gin.H{
		"id":              userID,
		"name":            name,
		"displayName":     displayName,
		"email":           email,
		"avatar":          fallbackAvatar,
		"isPlatformAdmin": isPlatformAdmin,
		"capabilities":    capabilities,
		"canAccessAdmin":  capability.CanAccessAdmin(capabilities),
	}, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
