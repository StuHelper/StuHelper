package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

func TestResolveRequiredInternalUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newContext := func() (*gin.Context, *httptest.ResponseRecorder) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		return c, w
	}

	t.Run("missing auth returns unauthorized", func(t *testing.T) {
		c, w := newContext()

		userID, ok := ResolveRequiredInternalUserID(c, func(context.Context, string) (int64, error) {
			t.Fatal("resolver should not be called")
			return 0, nil
		}, "failed to resolve user")

		assert.False(t, ok)
		assert.Zero(t, userID)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("resolver error returns internal error", func(t *testing.T) {
		c, w := newContext()
		c.Set(CtxKeyUserID, "external-1")

		userID, ok := ResolveRequiredInternalUserID(c, func(context.Context, string) (int64, error) {
			return 0, errors.New("db down")
		}, "failed to resolve user")

		assert.False(t, ok)
		assert.Zero(t, userID)
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var body struct {
			Error response.APIError `json:"error"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, "failed to resolve user", body.Error.Message)
	})

	t.Run("missing shadow user returns forbidden instead of internal error", func(t *testing.T) {
		c, w := newContext()
		c.Set(CtxKeyUserID, "external-1")

		userID, ok := ResolveRequiredInternalUserID(c, func(context.Context, string) (int64, error) {
			return 0, pgx.ErrNoRows
		}, "failed to resolve user")

		assert.False(t, ok)
		assert.Zero(t, userID)
		assert.Equal(t, http.StatusForbidden, w.Code)

		var body struct {
			Error response.APIError `json:"error"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, "user has not completed provisioning", body.Error.Message)
		assert.Equal(t, string(errs.ErrUserNotFound), body.Error.Code)
	})

	t.Run("success returns internal user id", func(t *testing.T) {
		c, w := newContext()
		c.Set(CtxKeyUserID, "external-1")

		userID, ok := ResolveRequiredInternalUserID(c, func(_ context.Context, externalID string) (int64, error) {
			assert.Equal(t, "external-1", externalID)
			return 42, nil
		}, "failed to resolve user")

		assert.True(t, ok)
		assert.Equal(t, int64(42), userID)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
