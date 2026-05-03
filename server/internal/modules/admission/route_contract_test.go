package admission

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/routeassert"
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
}

func TestAdmissionErrorCodes(t *testing.T) {
	raw, err := os.ReadFile("../../../api/components/schemas/admission.yaml")
	if err != nil {
		t.Fatalf("read admission schema: %v", err)
	}

	schema := string(raw)
	assertSchemaContains(t, schema, "admission.qq_mismatch")
	assertSchemaContains(t, schema, "admission.token_consumed")
	assertSchemaContains(t, schema, "admission.token_expired")
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
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admission/school-email/request-otp")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admission/school-email/verify-otp")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admission/school-sso/:schoolID/login")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admission/school-sso/:schoolID/callback")
}

func assertAdmissionBotRoutes(t *testing.T, routes gin.RoutesInfo) {
	t.Helper()

	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/sessions")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/join-requests/events")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/bot/admission/qq-users/:qqID/access")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/bot/admission/sessions/pending")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/sessions/:id/events")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/bot/admission/freshman/applications/pending-forward")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/freshman/applications/:id/forwarded")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/admission/freshman/applications/:id/review")
}

func assertAdmissionAdminRoutes(t *testing.T, routes gin.RoutesInfo) {
	t.Helper()

	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/admission/policies")
	routeassert.Exists(t, routes, http.MethodPut, "/api/v1/admin/admission/policies/:id")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/admission/sessions")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/freshman-verifications")
	routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/freshman-verifications/:id")
	routeassert.Exists(t, routes, http.MethodPut, "/api/v1/admin/freshman-verifications/:id")
	routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admin/admission/blacklist/:qqID/release")
}
