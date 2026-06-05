package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
)

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

		_, _, _, _, err := h.consumeOIDCState(c, state)
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

		_, _, _, _, err := h.consumeOIDCState(c, state)
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

		redirect, verifier, appKey, isNative, retryErr := h.consumeOIDCState(retryCtx, state)
		require.NoError(t, retryErr)
		assert.Equal(t, "/courses/1", redirect)
		assert.Equal(t, "verifier-2", verifier)
		assert.Equal(t, oidc.ApplicationWeb, appKey)
		assert.False(t, isNative)
	})

	t.Run("legacy redirect payload", func(t *testing.T) {
		const state = "state-legacy-payload"
		require.NoError(t, fixture.Client.Set(context.Background(), oidcStateRedisPrefix+state, "/legacy/path", stateMaxAge).Err())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodGet, "/callback?state="+state, nil)
		req.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: state})
		c.Request = req

		redirect, verifier, appKey, isNative, err := h.consumeOIDCState(c, state)
		require.NoError(t, err)
		assert.Equal(t, "/legacy/path", redirect)
		assert.Empty(t, verifier)
		assert.Equal(t, oidc.ApplicationWeb, appKey)
		assert.False(t, isNative)
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

		redirect, verifier, appKey, isNative, err := h.consumeOIDCState(c, state)
		require.NoError(t, err)
		assert.Equal(t, "stuhelper://auth/callback", redirect)
		assert.Equal(t, "verifier-native", verifier)
		assert.Equal(t, oidc.ApplicationUniapp, appKey)
		assert.True(t, isNative)

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
