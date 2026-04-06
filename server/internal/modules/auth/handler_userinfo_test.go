package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

func TestBuildUserPayload_OmitsNilAvatarAndEmptyEmail(t *testing.T) {
	h := &Handler{}

	payload := h.buildUserPayload(
		"user-1",
		"alice",
		"Alice",
		"",
		nil,
		[]string{"user"},
		[]string{"review:list:brief"},
	)

	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.NotContains(t, decoded, "avatar")
	assert.NotContains(t, decoded, "email")
}

func TestBuildUserPayload_IncludesAvatarWhenPresent(t *testing.T) {
	h := &Handler{}
	avatar := "https://cdn.example.com/avatar.png"

	payload := h.buildUserPayload(
		"user-1",
		"alice",
		"Alice",
		"alice@example.com",
		&avatar,
		[]string{"super_admin"},
		[]string{"admin:dashboard:view"},
	)

	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, avatar, decoded["avatar"])
	assert.Equal(t, "alice@example.com", decoded["email"])
}

func TestGetCurrentUser_HTTPContract_OmitsNilAvatar(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handler{}
	r := gin.New()
	r.GET("/api/v1/auth/me", func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "user-1")
		c.Set(middleware.CtxKeyUsername, "alice")
		c.Set(middleware.CtxKeyDisplayName, "Alice")
		c.Set(middleware.CtxKeyEmail, "")
		c.Set(middleware.CtxKeyAvatar, "")
		c.Set(middleware.CtxKeyRoles, []string{"user"})
		c.Set(middleware.CtxKeyCapabilities, []string{"review:list:brief"})
		h.GetCurrentUser(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.NotContains(t, resp.Data, "avatar")
	assert.NotContains(t, resp.Data, "email")
}
