package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
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
		[]capability.Grant{{Name: "review:list:brief", Global: true}},
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
		[]capability.Grant{{Name: "admin:dashboard:view", Global: true}},
	)

	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, avatar, decoded["avatar"])
	assert.Equal(t, "alice@example.com", decoded["email"])
}

func TestBuildUserPayload_UsesCasdoorAccountSettingsURL(t *testing.T) {
	h := &Handler{oidcIssuer: " http://localhost:8085/ "}

	payload := h.buildUserPayload(
		"user-1",
		"alice",
		"Alice",
		"",
		nil,
		[]string{"user"},
		[]capability.Grant{},
	)

	assert.Equal(t, "http://localhost:8085/account", payload.AccountSettingsURL)
	assert.NotContains(t, payload.AccountSettingsURL, "/ui/v2/login/password/change")
}

func TestBuildUserPayload_OmitsAccountSettingsURLWithoutIssuer(t *testing.T) {
	h := &Handler{}

	payload := h.buildUserPayload(
		"user-1",
		"alice",
		"Alice",
		"",
		nil,
		[]string{"user"},
		[]capability.Grant{},
	)

	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.NotContains(t, decoded, "accountSettingsUrl")
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
		c.Set(middleware.CtxKeyCapabilityGrants, []capability.Grant{{Name: "review:list:brief", Global: true}})
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

func TestBuildUserPayload_SchoolAdminScopesStayNonGlobal(t *testing.T) {
	h := &Handler{}

	payload := h.buildUserPayload(
		"user-1",
		"alice",
		"Alice",
		"",
		nil,
		[]string{"school_admin"},
		capability.ExpandRoleGrants([]string{"school_admin"}, map[string][]string{
			"school_admin": {"1002", "1001"},
		}),
	)

	assert.Empty(t, payload.GlobalCapabilities)
	assert.True(t, payload.CanAccessAdmin)
	require.NotEmpty(t, payload.CapabilityGrants)
	for _, grant := range payload.CapabilityGrants {
		assert.False(t, grant.Global)
		assert.Equal(t, []string{"1001", "1002"}, grant.ScopeSchoolIDs)
	}
}
