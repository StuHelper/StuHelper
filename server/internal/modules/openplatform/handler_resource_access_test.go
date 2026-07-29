package openplatform

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

type openPlatformHandlerEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestCheckResourceAccessHandlerAcceptsBearerClientCredentialsToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	resourceFGA := newFakeResourceFGA()
	service, err := NewService(repo, redis.Client, WithResourceFGAClient(resourceFGA))
	require.NoError(t, err)

	adminID := seedOpenPlatformUser(t, postgres, "handler-resource-token-admin")
	ownerID := seedOpenPlatformUser(t, postgres, "handler-resource-token-owner")
	app := seedApprovedOpenPlatformApp(t, ctx, repo, ownerID, []string{
		ScopeResourceRead,
		ScopeResourceWrite,
	})
	_, err = service.GrantResourceAccess(ctx, ResourceGrantInput{
		AppID:          app.ID,
		ReviewerUserID: adminID,
		ResourceType:   ResourceTypeResourceItem,
		ResourceID:     "handler-token-42",
		Actions:        []string{ResourceAccessActionRead, ResourceAccessActionWrite},
		Reason:         "handler token access",
		RequestID:      "handler-grant-token-resource-access",
	})
	require.NoError(t, err)

	verifier := &fakeResourceAccessTokenVerifier{
		token: ResourceAccessToken{
			ClientID: app.ClientID,
			Scopes:   []string{ScopeResourceRead},
		},
	}
	router := newResourceAccessHandlerRouter(t, service, verifier)

	resp := performResourceAccessCheck(t, router, "Bearer app-only-token", map[string]any{
		"resourceType": ResourceTypeResourceItem,
		"resourceID":   "handler-token-42",
		"action":       ResourceAccessActionRead,
	})
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	decision := decodeResourceAccessDecision(t, resp)
	assert.True(t, decision.Allowed)
	assert.Equal(t, app.ClientID, decision.ClientID)
	assert.Equal(t, ResourceAccessActionRead, decision.Action)
	assert.Equal(t, ResourceRelationReadByApp, decision.Relation)
	assert.Equal(t, "allowed", decision.Reason)
	assert.Equal(t, []string{"app-only-token"}, verifier.rawTokens)

	resp = performResourceAccessCheck(t, router, "Bearer app-only-token", map[string]any{
		"resourceType": ResourceTypeResourceItem,
		"resourceID":   "handler-token-42",
		"action":       ResourceAccessActionWrite,
	})
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	decision = decodeResourceAccessDecision(t, resp)
	assert.False(t, decision.Allowed)
	assert.Equal(t, app.ClientID, decision.ClientID)
	assert.Equal(t, ResourceAccessActionWrite, decision.Action)
	assert.Equal(t, ResourceRelationWriteByApp, decision.Relation)
	assert.Equal(t, "token_scope_missing", decision.Reason)
	assert.Equal(t, []string{"app-only-token", "app-only-token"}, verifier.rawTokens)
}

func TestCheckResourceAccessHandlerRejectsBearerTokenWithoutVerifier(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newResourceAccessHandlerRouter(t, nil, nil)

	resp := performResourceAccessCheck(t, router, "Bearer app-only-token", map[string]any{
		"resourceType": ResourceTypeResourceItem,
		"resourceID":   "handler-token-42",
		"action":       ResourceAccessActionRead,
	})

	require.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())
	envelope := decodeOpenPlatformHandlerEnvelope(t, resp)
	require.False(t, envelope.Success)
	require.NotNil(t, envelope.Error)
	assert.Equal(t, string(errs.ErrTokenInvalid), envelope.Error.Code)
	assert.Equal(t, "open platform resource access token is invalid", envelope.Error.Message)
}

func TestCheckResourceAccessHandlerRejectsInvalidBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	verifier := &fakeResourceAccessTokenVerifier{err: errors.New("token expired")}
	router := newResourceAccessHandlerRouter(t, nil, verifier)

	resp := performResourceAccessCheck(t, router, "Bearer expired-token", map[string]any{
		"resourceType": ResourceTypeResourceItem,
		"resourceID":   "handler-token-42",
		"action":       ResourceAccessActionRead,
	})

	require.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())
	envelope := decodeOpenPlatformHandlerEnvelope(t, resp)
	require.False(t, envelope.Success)
	require.NotNil(t, envelope.Error)
	assert.Equal(t, string(errs.ErrTokenInvalid), envelope.Error.Code)
	assert.Equal(t, []string{"expired-token"}, verifier.rawTokens)
}

func TestCheckResourceAccessHandlerMapsBearerVerifierOutageToServiceUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	verifier := &fakeResourceAccessTokenVerifier{err: ErrResourceAccessUnavailable}
	router := newResourceAccessHandlerRouter(t, nil, verifier)

	resp := performResourceAccessCheck(t, router, "Bearer app-only-token", map[string]any{
		"resourceType": ResourceTypeResourceItem,
		"resourceID":   "handler-token-42",
		"action":       ResourceAccessActionRead,
	})

	require.Equal(t, http.StatusServiceUnavailable, resp.Code, resp.Body.String())
	envelope := decodeOpenPlatformHandlerEnvelope(t, resp)
	require.False(t, envelope.Success)
	require.NotNil(t, envelope.Error)
	assert.Equal(t, string(errs.ErrServiceUnavailable), envelope.Error.Code)
	assert.Equal(t, "open platform resource authorization unavailable", envelope.Error.Message)
	assert.Equal(t, []string{"app-only-token"}, verifier.rawTokens)
}

func TestCheckResourceAccessHandlerRejectsUnsupportedAuthorizationScheme(t *testing.T) {
	gin.SetMode(gin.TestMode)
	verifier := &fakeResourceAccessTokenVerifier{
		token: ResourceAccessToken{
			ClientID: "token-client",
			Scopes:   []string{ScopeResourceRead},
		},
	}
	router := newResourceAccessHandlerRouter(t, nil, verifier)

	resp := performResourceAccessCheck(t, router, "Basic Y2xpZW50OnNlY3JldA==", map[string]any{
		"clientID":     "body-client",
		"clientSecret": "body-secret",
		"resourceType": ResourceTypeResourceItem,
		"resourceID":   "handler-token-42",
		"action":       ResourceAccessActionRead,
	})

	require.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())
	envelope := decodeOpenPlatformHandlerEnvelope(t, resp)
	require.False(t, envelope.Success)
	require.NotNil(t, envelope.Error)
	assert.Equal(t, string(errs.ErrTokenInvalid), envelope.Error.Code)
	assert.Empty(t, verifier.rawTokens)
}

func TestCheckResourceAccessHandlerRejectsRepeatedAuthorizationHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	verifier := &fakeResourceAccessTokenVerifier{
		token: ResourceAccessToken{
			ClientID: "token-client",
			Scopes:   []string{ScopeResourceRead},
		},
	}
	router := newResourceAccessHandlerRouter(t, nil, verifier)
	payload, err := json.Marshal(map[string]any{
		"resourceType": ResourceTypeResourceItem,
		"resourceID":   "handler-token-42",
		"action":       ResourceAccessActionRead,
	})
	require.NoError(t, err)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/open-platform/resources/access/check",
		bytes.NewReader(payload),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer app-only-token")
	req.Header.Add("Authorization", "Bearer other-app-only-token")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())
	envelope := decodeOpenPlatformHandlerEnvelope(t, resp)
	require.False(t, envelope.Success)
	require.NotNil(t, envelope.Error)
	assert.Equal(t, string(errs.ErrTokenInvalid), envelope.Error.Code)
	assert.Empty(t, verifier.rawTokens)
}

func TestCheckResourceAccessHandlerRejectsBearerClientIDMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	postgres := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	repo := NewRepository(postgres.DB)
	service, err := NewService(repo, redis.Client, WithResourceFGAClient(newFakeResourceFGA()))
	require.NoError(t, err)

	verifier := &fakeResourceAccessTokenVerifier{
		token: ResourceAccessToken{
			ClientID: "token-client",
			Scopes:   []string{ScopeResourceRead},
		},
	}
	router := newResourceAccessHandlerRouter(t, service, verifier)

	resp := performResourceAccessCheck(t, router, "Bearer app-only-token", map[string]any{
		"clientID":     "body-client",
		"resourceType": ResourceTypeResourceItem,
		"resourceID":   "handler-token-42",
		"action":       ResourceAccessActionRead,
	})

	require.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
	envelope := decodeOpenPlatformHandlerEnvelope(t, resp)
	require.False(t, envelope.Success)
	require.NotNil(t, envelope.Error)
	assert.Equal(t, string(errs.ErrInvalidParam), envelope.Error.Code)
	assert.Empty(t, verifier.rawTokens)
}

func TestCheckResourceAccessHandlerRejectsMixedBearerAndBodyClientCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	verifier := &fakeResourceAccessTokenVerifier{
		token: ResourceAccessToken{
			ClientID: "token-client",
			Scopes:   []string{ScopeResourceRead},
		},
	}
	router := newResourceAccessHandlerRouter(t, nil, verifier)

	tests := []struct {
		name string
		body map[string]any
	}{
		{
			name: "body client id",
			body: map[string]any{
				"clientID":     "token-client",
				"resourceType": ResourceTypeResourceItem,
				"resourceID":   "handler-token-42",
				"action":       ResourceAccessActionRead,
			},
		},
		{
			name: "body client secret",
			body: map[string]any{
				"clientSecret": "body-secret-must-not-mix-with-bearer",
				"resourceType": ResourceTypeResourceItem,
				"resourceID":   "handler-token-42",
				"action":       ResourceAccessActionRead,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := performResourceAccessCheck(t, router, "Bearer app-only-token", tt.body)

			require.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
			envelope := decodeOpenPlatformHandlerEnvelope(t, resp)
			require.False(t, envelope.Success)
			require.NotNil(t, envelope.Error)
			assert.Equal(t, string(errs.ErrInvalidParam), envelope.Error.Code)
			assert.Empty(t, verifier.rawTokens)
		})
	}
}

func TestResourceAccessBearerTokenParsesCaseInsensitiveScheme(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Run("bearer token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
		c.Request.Header.Set("Authorization", "bEaReR  token-value  ")

		token, ok := resourceAccessBearerToken(c)

		require.True(t, ok)
		assert.Equal(t, "token-value", token)
	})

	t.Run("non bearer header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
		c.Request.Header.Set("Authorization", "Basic token-value")

		token, ok := resourceAccessBearerToken(c)

		require.False(t, ok)
		assert.Empty(t, token)
	})

	t.Run("repeated authorization header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
		c.Request.Header.Add("Authorization", "Bearer token-value")
		c.Request.Header.Add("Authorization", "Bearer other-token")

		token, ok := resourceAccessBearerToken(c)

		require.False(t, ok)
		assert.Empty(t, token)
	})
}

func newResourceAccessHandlerRouter(
	t *testing.T,
	service *Service,
	verifier ResourceAccessTokenVerifier,
) *gin.Engine {
	t.Helper()
	if service == nil {
		service = &Service{}
	}
	handler := NewHandler(service)
	handler.SetResourceAccessTokenVerifier(verifier)
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api, func(c *gin.Context) {})
	return router
}

func performResourceAccessCheck(
	t *testing.T,
	router *gin.Engine,
	authorization string,
	body map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/open-platform/resources/access/check",
		bytes.NewReader(payload),
	)
	req.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func decodeOpenPlatformHandlerEnvelope(t *testing.T, resp *httptest.ResponseRecorder) openPlatformHandlerEnvelope {
	t.Helper()
	var envelope openPlatformHandlerEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &envelope), resp.Body.String())
	return envelope
}

func decodeResourceAccessDecision(t *testing.T, resp *httptest.ResponseRecorder) resourceAccessDecisionResponse {
	t.Helper()
	envelope := decodeOpenPlatformHandlerEnvelope(t, resp)
	require.True(t, envelope.Success, resp.Body.String())
	var decision resourceAccessDecisionResponse
	require.NoError(t, json.Unmarshal(envelope.Data, &decision))
	return decision
}

type fakeResourceAccessTokenVerifier struct {
	token     ResourceAccessToken
	err       error
	rawTokens []string
}

func (f *fakeResourceAccessTokenVerifier) VerifyOpenPlatformResourceAccessToken(
	_ context.Context,
	rawToken string,
) (ResourceAccessToken, error) {
	f.rawTokens = append(f.rawTokens, rawToken)
	if f.err != nil {
		return ResourceAccessToken{}, f.err
	}
	return f.token, nil
}
