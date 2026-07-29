package auth

import (
	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/oidc"
)

// GetStepUpURL 生成 MFA step-up OIDC 重认证 URL。
// 该端点只发起真实 provider reauth，不签发本地 mfa proof。
func (h *Handler) GetStepUpURL(c *gin.Context) {
	appKey := h.oidcClient.ApplicationKeyForClientID(middleware.GetAppID(c))
	if appKey == "" {
		appKey = oidc.ApplicationWeb
	}
	h.respondWithFixedApplicationAuthURL(c, appKey, h.oidcClient.GetStepUpAuthURLForApplicationWithRedirectURI)
}
