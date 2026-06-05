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
