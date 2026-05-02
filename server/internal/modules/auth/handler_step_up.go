package auth

import "github.com/gin-gonic/gin"

// GetStepUpURL 生成 MFA step-up OIDC 重认证 URL。
// 该端点只发起真实 provider reauth，不签发本地 mfa proof。
func (h *Handler) GetStepUpURL(c *gin.Context) {
	h.respondWithAuthURLProvider(c, h.oidcClient.GetStepUpAuthURL)
}
