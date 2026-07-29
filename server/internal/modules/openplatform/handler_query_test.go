package openplatform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
)

func TestAuthorizeRejectsRepeatedSingleValueQueryParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(&Service{})

	tests := []struct {
		name   string
		target string
	}{
		{
			name:   "client id",
			target: "/api/v1/open-platform/authorize?client_id=client-a&client_id=client-b&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=profile.basic.read&state=opaque-state",
		},
		{
			name:   "redirect uri",
			target: "/api/v1/open-platform/authorize?client_id=client-a&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&redirect_uri=https%3A%2F%2Fevil.example.com%2Fcallback&scope=profile.basic.read&state=opaque-state",
		},
		{
			name:   "scope",
			target: "/api/v1/open-platform/authorize?client_id=client-a&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=profile.basic.read&scope=email.read&state=opaque-state",
		},
		{
			name:   "state",
			target: "/api/v1/open-platform/authorize?client_id=client-a&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=profile.basic.read&state=opaque-state&state=other-state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, resp := newOpenPlatformQueryTestContext(http.MethodGet, tt.target)

			handler.authorize(c)

			require.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
			envelope := decodeOpenPlatformHandlerEnvelope(t, resp)
			require.False(t, envelope.Success)
			require.NotNil(t, envelope.Error)
			assert.Equal(t, string(errs.ErrInvalidParam), envelope.Error.Code)
		})
	}
}

func TestDisclosureRejectsRepeatedSingleValueQueryParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(&Service{})

	tests := []struct {
		name   string
		target string
	}{
		{
			name:   "client id",
			target: "/api/v1/open-platform/userinfo?client_id=client-a&client_id=client-b&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=profile.basic.read&consent_base_url=%2Fconsent",
		},
		{
			name:   "redirect uri",
			target: "/api/v1/open-platform/userinfo?client_id=client-a&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&redirect_uri=https%3A%2F%2Fevil.example.com%2Fcallback&scope=profile.basic.read&consent_base_url=%2Fconsent",
		},
		{
			name:   "scope",
			target: "/api/v1/open-platform/userinfo?client_id=client-a&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=profile.basic.read&scope=email.read&consent_base_url=%2Fconsent",
		},
		{
			name:   "consent base url",
			target: "/api/v1/open-platform/userinfo?client_id=client-a&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&scope=profile.basic.read&consent_base_url=%2Fconsent&consent_base_url=https%3A%2F%2Fevil.example.com%2Fconsent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, resp := newOpenPlatformQueryTestContext(http.MethodGet, tt.target)
			called := false

			handler.disclose(c, func(context.Context, DisclosureRequest) (map[string]any, error) {
				called = true
				return map[string]any{"ok": true}, nil
			})

			require.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
			envelope := decodeOpenPlatformHandlerEnvelope(t, resp)
			require.False(t, envelope.Success)
			require.NotNil(t, envelope.Error)
			assert.Equal(t, string(errs.ErrInvalidParam), envelope.Error.Code)
			assert.False(t, called)
		})
	}
}

func TestOpenPlatformListHandlersRejectAmbiguousSingleValueQueryParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(&Service{})

	tests := []struct {
		name   string
		target string
		call   func(*gin.Context)
	}{
		{
			name:   "owned apps repeated status",
			target: "/api/v1/open-platform/apps?status=pending&status=approved",
			call:   handler.listApps,
		},
		{
			name:   "admin apps ambiguous page size aliases",
			target: "/api/v1/admin/open-platform/apps?page_size=10&pageSize=20",
			call:   handler.listAdminApps,
		},
		{
			name:   "admin audit repeated app id",
			target: "/api/v1/admin/open-platform/audit-events?appID=1&appID=2",
			call:   handler.listAdminAuditEvents,
		},
		{
			name:   "admin consents repeated user id",
			target: "/api/v1/admin/open-platform/consents?userID=1&userID=2",
			call:   handler.listAdminConsents,
		},
		{
			name:   "owned app audit repeated scope",
			target: "/api/v1/open-platform/apps/1/audit-events?scope=email.read&scope=phone.read",
			call:   handler.listOwnedAppAuditEvents,
		},
		{
			name:   "token probe evidence repeated client id",
			target: "/api/v1/admin/open-platform/token-probe-evidence?clientID=client-a&clientID=client-b",
			call:   handler.listAdminTokenProbeEvidence,
		},
		{
			name:   "resource grants repeated resource type",
			target: "/api/v1/admin/open-platform/apps/1/resource-grants?resourceType=user_profile&resourceType=resource_item",
			call:   handler.listAdminResourceGrants,
		},
		{
			name:   "disclosure report repeated window hours",
			target: "/api/v1/admin/open-platform/disclosure-report?windowHours=24&windowHours=48",
			call:   handler.getAdminDisclosureReport,
		},
		{
			name:   "consent audit repeated event type",
			target: "/api/v1/open-platform/consents/audit-events?eventType=open_platform.consent.granted&eventType=open_platform.consent.revoked",
			call:   handler.listConsentAuditEvents,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, resp := newOpenPlatformQueryTestContext(http.MethodGet, tt.target)

			tt.call(c)

			require.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
			envelope := decodeOpenPlatformHandlerEnvelope(t, resp)
			require.False(t, envelope.Success)
			require.NotNil(t, envelope.Error)
			assert.Equal(t, string(errs.ErrInvalidParam), envelope.Error.Code)
		})
	}
}

func newOpenPlatformQueryTestContext(method, target string) (*gin.Context, *httptest.ResponseRecorder) {
	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Request = httptest.NewRequest(method, target, nil)
	return c, resp
}
