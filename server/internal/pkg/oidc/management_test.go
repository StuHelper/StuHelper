package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
)

func TestNewManagementClient(t *testing.T) {
	t.Run("nil when PAT missing", func(t *testing.T) {
		assert.Nil(t, NewManagementClient(config.CasdoorConfig{}))
	})

	t.Run("builds client when PAT configured", func(t *testing.T) {
		cfg := config.CasdoorConfig{
			Issuer:        "https://issuer.example.com",
			ManagementPAT: "test-pat",
			ProjectID:     "project-1",
			OrgID:         "org-1",
		}
		client := NewManagementClient(cfg)
		require.NotNil(t, client)
		assert.Equal(t, cfg.Issuer, client.baseURL)
		assert.Equal(t, cfg.ManagementPAT, client.pat)
		assert.Equal(t, cfg.ProjectID, client.projectID)
		assert.Equal(t, cfg.OrgID, client.orgID)
		require.NotNil(t, client.client)
	})
}

func TestManagementClientDoRequest(t *testing.T) {
	var gotAuth string
	var gotOrg string
	var gotBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotOrg = r.Header.Get("x-zitadel-orgid")
		payload, _ := io.ReadAll(r.Body)
		gotBody = string(payload)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	m := &ManagementClient{
		baseURL: server.URL,
		pat:     "secret-pat",
		orgID:   "org-123",
		client:  server.Client(),
	}

	err := m.doRequest(context.Background(), http.MethodPost, server.URL+"/grant", map[string]any{"role": "school_admin"})
	require.NoError(t, err)
	assert.Equal(t, "Bearer secret-pat", gotAuth)
	assert.Equal(t, "org-123", gotOrg)
	assert.JSONEq(t, `{"role":"school_admin"}`, gotBody)
}

func TestManagementClientDoRequest_StatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer server.Close()

	m := &ManagementClient{pat: "secret", client: server.Client()}
	err := m.doRequest(context.Background(), http.MethodDelete, server.URL, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 502")
	assert.Contains(t, err.Error(), "boom")
}

func TestFindUserGrant(t *testing.T) {
	t.Run("returns matching grant id", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.True(t, strings.HasSuffix(r.URL.Path, "/management/v1/users/user-1/grants/_search"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"result":[{"id":"grant-1","roleKeys":["viewer","school_admin"]}]}`)
		}))
		defer server.Close()

		m := &ManagementClient{baseURL: server.URL, projectID: "proj-1", pat: "pat", client: server.Client()}
		grantID, err := m.findUserGrant(context.Background(), "user-1", "school_admin")
		require.NoError(t, err)
		assert.Equal(t, "grant-1", grantID)
	})

	t.Run("returns empty string when role missing", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"result":[{"id":"grant-1","roleKeys":["viewer"]}]}`)
		}))
		defer server.Close()

		m := &ManagementClient{baseURL: server.URL, projectID: "proj-1", pat: "pat", client: server.Client()}
		grantID, err := m.findUserGrant(context.Background(), "user-1", "school_admin")
		require.NoError(t, err)
		assert.Empty(t, grantID)
	})

	t.Run("returns status error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "search failed", http.StatusInternalServerError)
		}))
		defer server.Close()

		m := &ManagementClient{baseURL: server.URL, projectID: "proj-1", pat: "pat", client: server.Client()}
		_, err := m.findUserGrant(context.Background(), "user-1", "school_admin")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "search grants")
	})
}

func TestBuildRoleSyncFunc(t *testing.T) {
	t.Run("nil management returns explicit configuration error", func(t *testing.T) {
		var called atomic.Bool
		syncFn := BuildRoleSyncFunc(nil, func(ctx context.Context, userID int64) (string, error) {
			called.Store(true)
			return "", nil
		})

		err := syncFn(context.Background(), 42, "school_admin", true)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrManagementPATNotConfigured)
		assert.False(t, called.Load())
	})

	t.Run("grant and revoke delegate to management api", func(t *testing.T) {
		var grantBody map[string]any
		var grantAuth string
		var deletePath string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/grants"):
				grantAuth = r.Header.Get("Authorization")
				defer r.Body.Close()
				require.NoError(t, json.NewDecoder(r.Body).Decode(&grantBody))
				w.WriteHeader(http.StatusCreated)
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/grants/_search"):
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"result":[{"id":"grant-xyz","roleKeys":["school_admin"]}]}`)
			case r.Method == http.MethodDelete:
				deletePath = r.URL.Path
				w.WriteHeader(http.StatusNoContent)
			default:
				http.Error(w, fmt.Sprintf("unexpected %s %s", r.Method, r.URL.Path), http.StatusBadRequest)
			}
		}))
		defer server.Close()

		mgmt := &ManagementClient{
			baseURL:   server.URL,
			pat:       "secret",
			projectID: "project-1",
			client:    server.Client(),
		}
		syncFn := BuildRoleSyncFunc(mgmt, func(ctx context.Context, userID int64) (string, error) {
			assert.Equal(t, int64(7), userID)
			return "oidc-user-7", nil
		})

		require.NoError(t, syncFn(context.Background(), 7, "school_admin", true))
		assert.Equal(t, "Bearer secret", grantAuth)
		assert.Equal(t, "project-1", grantBody["projectId"])
		roles, ok := grantBody["roleKeys"].([]any)
		require.True(t, ok)
		assert.Equal(t, []any{"school_admin"}, roles)

		require.NoError(t, syncFn(context.Background(), 7, "school_admin", false))
		assert.Equal(t, "/management/v1/users/oidc-user-7/grants/grant-xyz", deletePath)
	})
}
