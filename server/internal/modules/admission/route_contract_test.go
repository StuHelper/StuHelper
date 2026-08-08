package admission

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/testutil/routeassert"
)

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
	for _, path := range []string{
		"/api/v1/admission/freshman/applications",
		"/api/v1/admission/freshman/applications/:id/camera-captures",
		"/api/v1/admission/freshman/camera-handoffs/:id/events",
		"/api/v1/admission/school-email/academic-match",
		"/api/v1/admission/school-email/request-otp",
		"/api/v1/admission/school-email/verify-otp",
	} {
		routeassert.NotExists(t, routes, http.MethodPost, path)
	}
	routeassert.NotExists(t, routes, http.MethodGet, "/api/v1/admission/school-sso/:schoolCode/login")
	routeassert.NotExists(t, routes, http.MethodGet, "/api/v1/admission/school-sso/:schoolCode/callback")
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
	routeassert.NotExists(t, routes, http.MethodGet, "/api/v1/bot/admission/freshman/applications/pending-forward")
	routeassert.NotExists(t, routes, http.MethodPost, "/api/v1/bot/admission/freshman/applications/:id/forwarded")
	routeassert.NotExists(t, routes, http.MethodPost, "/api/v1/bot/admission/freshman/applications/:id/view")
	routeassert.NotExists(t, routes, http.MethodPost, "/api/v1/bot/admission/freshman/applications/:id/review")
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
	routeassert.NotExists(t, routes, http.MethodGet, "/api/v1/admin/freshman-verifications")
	routeassert.NotExists(t, routes, http.MethodGet, "/api/v1/admin/freshman-verifications/:id")
	routeassert.NotExists(t, routes, http.MethodPut, "/api/v1/admin/freshman-verifications/:id")
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
