package auth

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

var (
	errNativeOIDCApplicationMismatch = errors.New("native OIDC login must use uniapp application")
	errUnknownOIDCApplication        = errors.New("unknown OIDC application")
)

type authURLProvider func(appKey, state string) (string, string, error)

type authURLRequest struct {
	provider authURLProvider
	appKey   string
	redirect string
	native   bool
}

func (h *Handler) respondWithAuthURLProvider(c *gin.Context, provider authURLProvider) {
	redirect := strings.TrimSpace(c.Query("redirect"))
	isNative := strings.TrimSpace(c.Query("platform")) == "native"
	appKey, err := requestedOIDCApplication(c.Query("app"), isNative)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.respondWithResolvedAuthURL(c, authURLRequest{
		provider: provider,
		appKey:   appKey,
		redirect: redirect,
		native:   isNative,
	})
}

func (h *Handler) respondWithFixedApplicationAuthURL(c *gin.Context, appKey string, provider authURLProvider) {
	redirect := strings.TrimSpace(c.Query("redirect"))
	h.respondWithResolvedAuthURL(c, authURLRequest{
		provider: provider,
		appKey:   appKey,
		redirect: redirect,
	})
}

func (h *Handler) respondWithResolvedAuthURL(c *gin.Context, req authURLRequest) {
	state, err := generateNonce()
	if err != nil {
		logger.FromGin(c).Error("failed to generate state", zap.Error(err))
		response.InternalError(c, "failed to generate login URL")
		return
	}

	authURL, verifier, err := req.provider(req.appKey, state)
	if err != nil {
		logger.FromGin(c).Error("failed to generate oidc authorization URL", zap.String("app", req.appKey), zap.Error(err))
		response.InternalError(c, "failed to generate login URL")
		return
	}
	if err := h.storeOIDCState(c.Request.Context(), oidcStateInput{
		state:        state,
		redirect:     req.redirect,
		codeVerifier: verifier,
		application:  req.appKey,
		native:       req.native,
	}); err != nil {
		logger.FromGin(c).Error("failed to persist oidc state", zap.Error(err))
		response.InternalError(c, "failed to generate login URL")
		return
	}

	if !req.native {
		h.setOIDCStateCookie(c, state)
	}
	response.Success(c, gin.H{"url": authURL, "state": state})
}

func requestedOIDCApplication(rawApp string, native bool) (string, error) {
	appKey := strings.TrimSpace(rawApp)
	if appKey == "" && native {
		return oidc.ApplicationUniapp, nil
	}
	if appKey == "" {
		return oidc.ApplicationWeb, nil
	}
	if native && appKey != oidc.ApplicationUniapp {
		return "", errNativeOIDCApplicationMismatch
	}
	switch appKey {
	case oidc.ApplicationWeb, oidc.ApplicationAdmin, oidc.ApplicationUniapp:
		return appKey, nil
	default:
		return "", errUnknownOIDCApplication
	}
}
