package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

type authURLProvider func(state string) (string, string)

func (h *Handler) respondWithAuthURLProvider(c *gin.Context, provider authURLProvider) {
	redirect := strings.TrimSpace(c.Query("redirect"))
	isNative := strings.TrimSpace(c.Query("platform")) == "native"

	state, err := generateNonce()
	if err != nil {
		logger.FromGin(c).Error("failed to generate state", zap.Error(err))
		response.InternalError(c, "failed to generate login URL")
		return
	}

	authURL, verifier := provider(state)
	if err := h.storeOIDCState(c.Request.Context(), state, redirect, verifier, isNative); err != nil {
		logger.FromGin(c).Error("failed to persist oidc state", zap.Error(err))
		response.InternalError(c, "failed to generate login URL")
		return
	}

	if !isNative {
		h.setOIDCStateCookie(c, state)
	}
	response.Success(c, gin.H{"url": authURL, "state": state})
}
