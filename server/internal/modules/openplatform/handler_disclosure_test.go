package openplatform

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

func TestDisclosureRequestFromQueryUsesExplicitClientID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		"GET",
		"/?client_id=third-party-client&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=profile.basic.read+email.read&consent_base_url=%2Fconsent",
		nil,
	)
	c.Request.Header.Set("Authorization", "Bearer user-token")
	c.Set(middleware.CtxKeyAppID, "stuhelper-web")
	c.Set(middleware.CtxKeyRequestID, "disclosure-request")

	req := disclosureRequestFromQuery(c, 42)

	assert.Equal(t, "third-party-client", req.ClientID)
	assert.Equal(t, "stuhelper-web", req.AuthenticatedClientID)
	assert.True(t, req.AuthenticatedByBearer)
	assert.Equal(t, int64(42), req.UserID)
	assert.Equal(t, []string{ScopeProfileBasicRead, ScopeEmailRead}, req.Scopes)
	assert.Equal(t, "https://client.example.com/callback", req.RedirectURI)
	assert.Equal(t, "/consent", req.ConsentBaseURL)
	assert.Equal(t, "disclosure-request", req.RequestID)
}

func TestDisclosureRequestFromQueryFallsBackToAuthenticatedClientID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		"GET",
		"/?redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=profile.basic.read",
		nil,
	)
	c.Set(middleware.CtxKeyAppID, "context-client")

	req := disclosureRequestFromQuery(c, 42)

	assert.Equal(t, "context-client", req.ClientID)
	assert.Equal(t, "context-client", req.AuthenticatedClientID)
	assert.False(t, req.AuthenticatedByBearer)
}
