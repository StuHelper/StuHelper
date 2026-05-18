package app

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	apigen "git.stuhelper.com/StuHelper/StuHelper/internal/api/gen"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/academics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/admission"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/auth"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/course"
	reviewmodule "git.stuhelper.com/StuHelper/StuHelper/internal/modules/course/review"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/notification"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/openplatform"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/resource"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/storage"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/health"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

var openAPIParamPattern = regexp.MustCompile(`\{([^}]+)\}`)

func TestOpenAPIRoutes_AreFullyRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	api := router.Group("/api/v1")
	noOp := func(c *gin.Context) { c.Next() }

	authHandler := &auth.Handler{}
	authHandler.RegisterPublicRoutes(api)
	authHandler.RegisterRoutes(api, nil, nil)

	health.NewHandler(nil, nil, health.BuildInfo{}, false, 0).RegisterRoutes(router)

	notificationHandler := &notification.Handler{}
	notificationHandler.RegisterRoutes(api, noOp)

	academicsHandler := &academics.Handler{}
	academicsHandler.RegisterRoutes(api, noOp)

	resourceHandler := &resource.Handler{}
	resourceHandler.RegisterRoutes(api, noOp, noOp)

	storageHandler := &storage.Handler{}
	storageHandler.RegisterAdminRoutes(api, noOp)

	courseHandler := course.NewHandler(&cache.Helper{}, &course.Service{}, &reviewmodule.Handler{})
	courseHandler.RegisterRoutes(api, noOp, noOp)

	userHandler := &user.Handler{}
	userHandler.RegisterRoutes(api, noOp)
	botHandler := &user.BotHandler{}
	botHandler.RegisterRoutes(api)
	admissionHandler := &admission.Handler{}
	admissionHandler.RegisterRoutes(api, noOp)
	admissionHandler.RegisterBotRoutes(api)
	openPlatformHandler := openplatform.NewHandler(nil)
	openPlatformHandler.RegisterRoutes(api, noOp)
	adminGroup := api.Group("/admin")
	adminGroup.Use(noOp)
	authHandler.RegisterAdminRoutes(adminGroup)
	userHandler.RegisterAdminRoutes(adminGroup)
	admissionHandler.RegisterAdminRoutes(adminGroup)
	openPlatformHandler.RegisterAdminRoutes(adminGroup)

	metricsGroup := api.Group("/metrics")
	metricsGroup.Use(metrics.OriginValidationMiddleware([]string{"http://localhost:3000"}))
	metricsGroup.POST("/vitals", metrics.VitalsHandler())
	metricsGroup.POST("/frontend-errors", metrics.FrontendErrorHandler())

	swagger, err := apigen.GetSwagger()
	require.NoError(t, err)

	expected := collectOpenAPIRoutes(swagger)
	actual := collectGinRoutes(router.Routes())

	missing := diffRouteSets(expected, actual)
	extra := diffRouteSets(actual, expected)

	require.Emptyf(t, missing, "OpenAPI routes missing from gin router:\n%s", strings.Join(missing, "\n"))
	require.Emptyf(t, extra, "Gin routes missing from OpenAPI:\n%s", strings.Join(extra, "\n"))
}

func collectOpenAPIRoutes(swagger *openapi3.T) []string {
	if swagger == nil || swagger.Paths == nil {
		return nil
	}

	routes := make([]string, 0, swagger.Paths.Len()*2)
	for path, item := range swagger.Paths.Map() {
		if item == nil {
			continue
		}
		ginPath := openAPIParamPattern.ReplaceAllString(path, ":$1")
		fullPath := ginPath
		if !strings.HasPrefix(fullPath, "/api/") && !strings.HasPrefix(fullPath, "/health/") && !strings.HasPrefix(fullPath, "/metrics") {
			fullPath = "/api/v1" + fullPath
		}
		for method := range item.Operations() {
			routes = append(routes, strings.ToUpper(method)+" "+fullPath)
		}
	}
	sort.Strings(routes)
	return routes
}

func collectGinRoutes(routes gin.RoutesInfo) []string {
	result := make([]string, 0, len(routes))
	for _, route := range routes {
		if !strings.HasPrefix(route.Path, "/api/v1/") && !strings.HasPrefix(route.Path, "/health/") {
			continue
		}
		result = append(result, route.Method+" "+route.Path)
	}
	sort.Strings(result)
	return result
}

func diffRouteSets(left, right []string) []string {
	rightSet := make(map[string]struct{}, len(right))
	for _, route := range right {
		rightSet[route] = struct{}{}
	}

	var diff []string
	for _, route := range left {
		if _, ok := rightSet[route]; ok {
			continue
		}
		diff = append(diff, route)
	}
	sort.Strings(diff)
	return diff
}
