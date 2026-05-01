package auth

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

func TestUnlockAuthAccountClearsHardLock(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-auth-admin-unlock-secret-32!", false))
	gin.SetMode(gin.TestMode)
	fixture := redisfixture.Start(t)
	handler := &Handler{authFailureGuard: NewAuthFailureGuard(fixture.Client)}
	phone := "13800139100"
	hardLockAuthAccount(t, fixture, handler.authFailureGuard, phone)

	router := gin.New()
	router.Use(authAdminTestContext())
	admin := router.Group("/api/v1/admin")
	handler.RegisterAdminRoutes(admin)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/account-locks/unlock", bytes.NewBufferString(`{"phone":"`+phone+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), phone)
	require.NoError(t, handler.authFailureGuard.EnsureAccountAllowed(t.Context(), phone))
}

func TestAuthAccountUnlockAuditEventMasksPhone(t *testing.T) {
	event := authAccountUnlockAuditEvent("13800139101", errors.New("redis down"))

	assert.Equal(t, "iam.auth.account_unlocked", string(event.Type))
	assert.Equal(t, "audit", event.Category)
	assert.Equal(t, "admin", event.ActorType)
	assert.Equal(t, "auth.account", event.ResourceType)
	assert.Equal(t, "phone:138****9101", event.ResourceID)
	assert.Equal(t, "unlock", event.Action)
	assert.Equal(t, "failure", event.Result)
	assert.NotContains(t, event.ResourceID, "13800139101")
	assert.NotContains(t, event.Details["account"], "13800139101")
}

func hardLockAuthAccount(t *testing.T, fixture *redisfixture.Fixture, guard *AuthFailureGuard, phone string) {
	t.Helper()
	for attempts := 1; attempts <= authFailureAccountHardLimit; attempts++ {
		err := guard.RecordAccountFailure(t.Context(), phone)
		if attempts == authFailureAccountHardLimit {
			require.ErrorIs(t, err, ErrAuthAccountHardLocked)
			return
		}
		if errors.Is(err, ErrAuthAccountSoftLocked) {
			fixture.Server.FastForward(authFailureAccountSoftLock + time.Second)
			continue
		}
		require.NoError(t, err)
	}
}

func authAdminTestContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "admin-user")
		c.Set(middleware.CtxKeyUsername, "admin-root")
		c.Set(middleware.CtxKeyCapabilityGrants, []capability.Grant{{
			Name:   capability.UserSystemUpdate,
			Global: true,
		}})
		c.Next()
	}
}
