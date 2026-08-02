package admission

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/StuHelper/StuHelper/server/internal/testutil/routeassert"
)

type admissionOpenAPIOperation struct {
	Security  *[]map[string][]string `yaml:"security"`
	Responses map[string]any         `yaml:"responses"`
}

type admissionOpenAPIPath struct {
	Get  *admissionOpenAPIOperation `yaml:"get"`
	Post *admissionOpenAPIOperation `yaml:"post"`
}

func TestAdmissionRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	api := r.Group("/api/v1")
	noOp := func(c *gin.Context) { c.Next() }
	h := &Handler{}

	h.RegisterRoutes(api, noOp)
	h.RegisterBotRoutes(api)
	h.RegisterAdminRoutes(api.Group("/admin"))

	routes := r.Routes()
	assertAdmissionUserRoutes(t, routes)
	assertAdmissionBotRoutes(t, routes)
	assertAdmissionAdminRoutes(t, routes)
	assertAdmissionRoutesImplemented(t, routes)
}

func TestAdmissionErrorCodes(t *testing.T) {
	raw, err := os.ReadFile("../../../api/components/schemas/admission.yaml")
	if err != nil {
		t.Fatalf("read admission schema: %v", err)
	}

	schema := string(raw)
	assertSchemaContains(t, schema, "admission.member_blacklisted")
	assertSchemaContains(t, schema, "admission.qq_mismatch")
	assertSchemaContains(t, schema, "admission.token_consumed")
	assertSchemaContains(t, schema, "admission.token_expired")
	assertSchemaContains(t, schema, "admission.token_not_found")
	assertSchemaContains(t, schema, "admission.session_not_found")
}

func TestAdmissionOpenAPISecurityContract(t *testing.T) {
	raw, err := os.ReadFile("../../../api/paths/admission.yaml")
	require.NoError(t, err)

	paths := map[string]admissionOpenAPIPath{}
	require.NoError(t, yaml.Unmarshal(raw, &paths))

	anonymousOperations := []*admissionOpenAPIOperation{
		paths["/api/v1/admission/freshman/mobile-camera-handoffs/{token}"].Get,
		paths["/api/v1/admission/freshman/mobile-camera-handoffs/{token}/camera-capture"].Post,
		paths["/api/v1/admission/freshman/mobile-camera-handoffs/{token}/continue"].Post,
	}
	for _, operation := range anonymousOperations {
		require.NotNil(t, operation)
		require.NotNil(t, operation.Security, "token handoff operations must override global authentication")
		require.Empty(t, *operation.Security)
	}

	authenticatedOperations := []*admissionOpenAPIOperation{
		paths["/api/v1/admission/school-sso/{schoolCode}/login"].Get,
		paths["/api/v1/admission/school-sso/{schoolCode}/callback"].Get,
	}
	for _, operation := range authenticatedOperations {
		require.NotNil(t, operation)
		require.Nil(t, operation.Security, "school SSO operations must inherit global authentication")
		for _, status := range []string{"400", "401", "403", "404", "409", "503"} {
			require.Contains(t, operation.Responses, status)
		}
	}
}

func assertSchemaContains(t *testing.T, schema string, code string) {
	t.Helper()

	if !strings.Contains(schema, code) {
		t.Fatalf("expected admission schema to include error code %q", code)
	}
}

func assertAdmissionUserRoutes(t *testing.T, routes gin.RoutesInfo) {
	t.Helper()

	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admission/sessions/:token")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admission/sessions/:token/link")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admission/me")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admission/freshman/applications")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admission/freshman/applications/:id/camera-captures")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admission/freshman/camera-handoffs/:id/events")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admission/school-email/academic-match")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admission/school-email/request-otp")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admission/school-email/verify-otp")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admission/school-sso/:schoolCode/login")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admission/school-sso/:schoolCode/callback")
}

func assertAdmissionBotRoutes(t *testing.T, routes gin.RoutesInfo) {
	t.Helper()

	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/sessions")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/bot/admission/sessions/member")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/sessions/member/resend")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/sessions/member/regenerate")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/sessions/member/skip")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/failures/reset")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/join-requests/events")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/join-requests/decision")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/bot/admission/policies/targets")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/bot/member-blacklist/access")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/bot/member-blacklist")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/member-blacklist")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/member-blacklist/:id/release")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/member-blacklist/release-by-subject")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/bot/admission/sessions/pending")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/bot/admission/actions/stream")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/actions/claim")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/actions/:id/events")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/sessions/:id/events")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/bot/admission/freshman/applications/pending-forward")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/freshman/applications/:id/forwarded")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/freshman/applications/:id/view")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/freshman/applications/:id/review")
}

func assertAdmissionAdminRoutes(t *testing.T, routes gin.RoutesInfo) {
	t.Helper()

	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/admission/policies")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admin/admission/policies")
	routeassert.Exists(t, routes, http.MethodPut, "/api/v1/admin/admission/policies/:id")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/admission/sessions")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admin/admission/sessions/:id/resend")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admin/admission/sessions/:id/regenerate")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admin/admission/sessions/:id/cancel")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/freshman-verifications")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/freshman-verifications/:id")
	routeassert.Exists(t, routes, http.MethodPut, "/api/v1/admin/freshman-verifications/:id")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/member-blacklist")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admin/member-blacklist")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admin/member-blacklist/:id/release")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admin/member-blacklist/release-by-subject")
}

func assertAdmissionRoutesImplemented(t *testing.T, routes gin.RoutesInfo) {
	t.Helper()

	for _, route := range routes {
		if !strings.Contains(route.Path, "/admission") &&
			!strings.Contains(route.Path, "/freshman-verifications") &&
			!strings.Contains(route.Path, "/member-blacklist") {
			continue
		}
		if strings.Contains(route.Handler, ".notImplemented") {
			t.Fatalf("%s %s is still registered to notImplemented", route.Method, route.Path)
		}
	}
}
