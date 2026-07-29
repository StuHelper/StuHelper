package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/oidc"
)

func TestStoreOIDCStateNormalizesInput(t *testing.T) {
	h, fixture := newTestHandler(t)

	require.NoError(t, h.storeOIDCState(context.Background(), oidcStateInput{
		state:        " \tstate-normalized\n ",
		redirect:     " \t/courses/1\n ",
		codeVerifier: " \tverifier-1\n ",
		application:  " \t" + oidc.ApplicationUniapp + "\n ",
		callbackURI:  " \tHTTP://JOIN.localhost:3000/api/v1/auth/callback\n ",
		native:       true,
	}))

	raw, err := fixture.Client.Get(context.Background(), oidcStateRedisPrefix+"state-normalized").Result()
	require.NoError(t, err)
	var payload oidcStatePayload
	require.NoError(t, json.Unmarshal([]byte(raw), &payload))
	assert.Equal(t, "/courses/1", payload.RedirectURL)
	assert.Equal(t, "verifier-1", payload.CodeVerifier)
	assert.Equal(t, oidc.ApplicationUniapp, payload.Application)
	assert.Equal(t, "http://join.localhost:3000/api/v1/auth/callback", payload.CallbackRedirectURI)
	assert.True(t, payload.Native)
}

func TestStoreOIDCStateRejectsInvalidInput(t *testing.T) {
	h, _ := newTestHandler(t)

	require.ErrorContains(t, h.storeOIDCState(context.Background(), oidcStateInput{
		state:        " \t\n ",
		codeVerifier: "verifier-1",
		application:  oidc.ApplicationWeb,
	}), "empty state")
	require.ErrorContains(t, h.storeOIDCState(context.Background(), oidcStateInput{
		state:        "state-missing-verifier",
		codeVerifier: " \t\n ",
		application:  oidc.ApplicationWeb,
	}), "code_verifier missing")
	require.ErrorContains(t, h.storeOIDCState(context.Background(), oidcStateInput{
		state:        "state-native-web",
		codeVerifier: "verifier-1",
		application:  oidc.ApplicationWeb,
		native:       true,
	}), "native oidc state application mismatch")
	require.ErrorContains(t, h.storeOIDCState(context.Background(), oidcStateInput{
		state:        "state-unknown-app",
		codeVerifier: "verifier-1",
		application:  "unknown",
	}), "unknown oidc state application")
	require.ErrorContains(t, h.storeOIDCState(context.Background(), oidcStateInput{
		state:        "state-bad-callback",
		codeVerifier: "verifier-1",
		application:  oidc.ApplicationWeb,
		callbackURI:  "https://web.example.com/other",
	}), "invalid oidc callback redirect uri")
}

func TestConsumeOIDCState_EdgeBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, fixture := newTestHandler(t)

	t.Run("missing cookie", func(t *testing.T) {
		const state = "state-missing-cookie"
		require.NoError(t, h.storeOIDCState(context.Background(), oidcStateInput{
			state:        state,
			redirect:     "/courses/1",
			codeVerifier: "verifier-1",
			application:  oidc.ApplicationWeb,
		}))
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/callback?state="+state, nil)

		_, err := h.consumeOIDCState(c, state)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "state cookie missing")

		remaining, getErr := fixture.Client.Get(context.Background(), oidcStateRedisPrefix+state).Result()
		require.NoError(t, getErr)
		assert.NotEmpty(t, remaining)
	})

	t.Run("cookie mismatch", func(t *testing.T) {
		const state = "state-cookie-mismatch"
		require.NoError(t, h.storeOIDCState(context.Background(), oidcStateInput{
			state:        state,
			redirect:     "/courses/1",
			codeVerifier: "verifier-2",
			application:  oidc.ApplicationWeb,
		}))
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodGet, "/callback?state="+state, nil)
		req.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: "other-state"})
		c.Request = req

		_, err := h.consumeOIDCState(c, state)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "state cookie mismatch")

		remaining, getErr := fixture.Client.Get(context.Background(), oidcStateRedisPrefix+state).Result()
		require.NoError(t, getErr)
		assert.NotEmpty(t, remaining)

		retryW := httptest.NewRecorder()
		retryCtx, _ := gin.CreateTestContext(retryW)
		retryReq := httptest.NewRequest(http.MethodGet, "/callback?state="+state, nil)
		retryReq.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: state})
		retryCtx.Request = retryReq

		result, retryErr := h.consumeOIDCState(retryCtx, state)
		require.NoError(t, retryErr)
		assert.Equal(t, "/courses/1", result.redirectURL)
		assert.Equal(t, "verifier-2", result.codeVerifier)
		assert.Equal(t, oidc.ApplicationWeb, result.appKey)
		assert.Empty(t, result.callbackRedirectURI)
		assert.False(t, result.native)
	})

	t.Run("invalid payload is rejected", func(t *testing.T) {
		const state = "state-invalid-payload"
		require.NoError(t, fixture.Client.Set(context.Background(), oidcStateRedisPrefix+state, "/legacy/path", stateMaxAge).Err())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodGet, "/callback?state="+state, nil)
		req.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: state})
		c.Request = req

		_, err := h.consumeOIDCState(c, state)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid oidc state payload")
	})

	t.Run("missing code verifier is rejected", func(t *testing.T) {
		const state = "state-missing-verifier"
		require.NoError(t, fixture.Client.Set(
			context.Background(),
			oidcStateRedisPrefix+state,
			`{"redirectURL":"/courses/1","codeVerifier":" \t\n ","application":"web"}`,
			stateMaxAge,
		).Err())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodGet, "/callback?state="+state, nil)
		req.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: state})
		c.Request = req

		_, err := h.consumeOIDCState(c, state)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "code_verifier missing")
	})

	t.Run("native path stores verifier for exchange", func(t *testing.T) {
		const state = "state-native-verifier"
		require.NoError(t, h.storeOIDCState(context.Background(), oidcStateInput{
			state:        state,
			redirect:     "stuhelper://auth/callback",
			codeVerifier: "verifier-native",
			application:  oidc.ApplicationUniapp,
			native:       true,
		}))
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/callback?state="+state, nil)

		result, err := h.consumeOIDCState(c, state)
		require.NoError(t, err)
		assert.Equal(t, "stuhelper://auth/callback", result.redirectURL)
		assert.Equal(t, "verifier-native", result.codeVerifier)
		assert.Equal(t, oidc.ApplicationUniapp, result.appKey)
		assert.Empty(t, result.callbackRedirectURI)
		assert.True(t, result.native)

		consumed, consumedApp, err := h.consumeNativeCodeVerifier(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, "verifier-native", consumed)
		assert.Equal(t, oidc.ApplicationUniapp, consumedApp)
	})

	t.Run("native promotion preserves verifier on repeated transition", func(t *testing.T) {
		const state = "state-native-repeat"
		require.NoError(t, h.storeOIDCState(context.Background(), oidcStateInput{
			state:        state,
			redirect:     "stuhelper://auth/callback",
			codeVerifier: "verifier-repeat",
			application:  oidc.ApplicationUniapp,
			native:       true,
		}))
		raw, err := fixture.Client.Get(context.Background(), oidcStateRedisPrefix+state).Result()
		require.NoError(t, err)
		payload := nativeCodeVerifierPayload{
			CodeVerifier: "verifier-repeat",
			Application:  oidc.ApplicationUniapp,
		}

		require.NoError(t, h.promoteOIDCStateToNativeVerifier(context.Background(), state, raw, payload))
		assert.Equal(t, int64(0), fixture.Client.Exists(context.Background(), oidcStateRedisPrefix+state).Val())
		stored, err := fixture.Client.Get(context.Background(), nativeCodeVerifierPrefix+state).Result()
		require.NoError(t, err)

		err = h.promoteOIDCStateToNativeVerifier(context.Background(), state, raw, payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "state expired or already used")
		remaining, getErr := fixture.Client.Get(context.Background(), nativeCodeVerifierPrefix+state).Result()
		require.NoError(t, getErr)
		assert.Equal(t, stored, remaining)
	})
}
