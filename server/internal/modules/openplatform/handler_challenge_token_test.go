package openplatform

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
)

func TestHandlerRejectsInvalidChallengeTokenQuery(t *testing.T) {
	router := newOpenPlatformChallengeTokenRouter(t)

	tests := []struct {
		name          string
		target        string
		expectedError string
	}{
		{
			name:          "consent missing token",
			target:        "/api/v1/open-platform/consent",
			expectedError: string(errs.ErrOpenPlatformConsentInvalid),
		},
		{
			name:          "consent repeated token",
			target:        "/api/v1/open-platform/consent?token=consent-token&token=other-token",
			expectedError: string(errs.ErrOpenPlatformConsentInvalid),
		},
		{
			name:          "profile completion blank token",
			target:        "/api/v1/open-platform/profile-completion?token=%20%20%20",
			expectedError: string(errs.ErrOpenPlatformConsentInvalid),
		},
		{
			name:          "profile completion repeated token",
			target:        "/api/v1/open-platform/profile-completion?token=completion-token&token=other-token",
			expectedError: string(errs.ErrOpenPlatformConsentInvalid),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			require.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
			envelope := decodeOpenPlatformHandlerEnvelope(t, resp)
			require.False(t, envelope.Success)
			require.NotNil(t, envelope.Error)
			assert.Equal(t, tt.expectedError, envelope.Error.Code)
		})
	}
}

func newOpenPlatformChallengeTokenRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	handler := NewHandler(&Service{})
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api, func(c *gin.Context) {})
	return router
}
