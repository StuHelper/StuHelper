package admission

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/botcredential"
	"git.stuhelper.com/StuHelper/StuHelper/internal/platform/serviceaccount"
)

type fakeAdmissionBotCredentialVerifier struct {
	seenToken string
}

func (f *fakeAdmissionBotCredentialVerifier) Verify(_ context.Context, rawToken, _, _ string) error {
	f.seenToken = rawToken
	return nil
}

func TestAdmissionBotCredentialRejectsRepeatedAuthorizationHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	verifier := &fakeAdmissionBotCredentialVerifier{}
	h := &Handler{botCredentialVerifier: verifier}
	router := gin.New()
	router.GET(
		"/bot/admission/protected",
		h.requireBotCredential(botcredential.ScopeBotAdmissionSession),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)

	req := httptest.NewRequest(http.MethodGet, "/bot/admission/protected", nil)
	req.Header.Add("Authorization", "Bearer service-token")
	req.Header.Add("Authorization", "Bearer other-service-token")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())
	assert.Empty(t, verifier.seenToken)
}

// F023 回归：typed-nil *serviceaccount.Verifier 装入 BotCredentialVerifier 接口后，
// handler 的 nil 守卫不会命中；带 Bearer 头的请求必须 fail-closed 返回 503，而非 panic。
func TestAdmissionBotCredentialTypedNilVerifierFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var typedNil *serviceaccount.Verifier
	h := &Handler{botCredentialVerifier: typedNil}
	router := gin.New()
	router.GET(
		"/bot/admission/protected",
		h.requireBotCredential(botcredential.ScopeBotAdmissionSession),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)

	req := httptest.NewRequest(http.MethodGet, "/bot/admission/protected", nil)
	req.Header.Set("Authorization", "Bearer service-token")
	resp := httptest.NewRecorder()

	require.NotPanics(t, func() { router.ServeHTTP(resp, req) })
	assert.Equal(t, http.StatusServiceUnavailable, resp.Code, resp.Body.String())
}
