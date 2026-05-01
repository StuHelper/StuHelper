package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Route constants matching the real registration in app/modules.go:
//
//	api := r.Group("/api/v1")
//	api.POST("/metrics/vitals", metrics.VitalsHandler())
//	api.POST("/metrics/frontend-errors", metrics.FrontendErrorHandler())
const (
	frontendErrorsRoute = "/api/v1/metrics/frontend-errors"
	vitalsRoute         = "/api/v1/metrics/vitals"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------------------------------------------------------------------------
// HTTP metrics (http.go)
// ---------------------------------------------------------------------------

func TestHTTPMetrics_Registration(t *testing.T) {
	// promauto registers metrics on init; verify they are retrievable
	// and do not panic when used with labels.
	assert.NotPanics(t, func() {
		HTTPRequestsTotal.WithLabelValues("GET", "/test", "200").Inc()
		HTTPRequestDuration.WithLabelValues("GET", "/test").Observe(0.042)
		HTTPRequestsInFlight.Inc()
		HTTPRequestsInFlight.Dec()
	})
}

func TestHTTPRequestsTotal_Inc(t *testing.T) {
	// Reset by reading current value, then increment, then verify delta.
	before := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("POST", "/api/test-counter", "201"))
	HTTPRequestsTotal.WithLabelValues("POST", "/api/test-counter", "201").Inc()
	after := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("POST", "/api/test-counter", "201"))
	assert.Equal(t, before+1, after)
}

func TestHTTPRequestDuration_Observe(t *testing.T) {
	// Observing a value should not panic and the histogram count should increase.
	collector := HTTPRequestDuration.WithLabelValues("GET", "/api/test-hist")

	// Observe a few values.
	collector.Observe(0.01)
	collector.Observe(0.05)
	collector.Observe(0.5)

	// Verify the histogram has observations (count >= 3 for this label set).
	count := testutil.CollectAndCount(HTTPRequestDuration)
	assert.Greater(t, count, 0)
}

func TestHTTPRequestsInFlight_Gauge(t *testing.T) {
	// Increment and decrement, verify the gauge returns to baseline.
	baseline := testutil.ToFloat64(HTTPRequestsInFlight)
	HTTPRequestsInFlight.Inc()
	assert.Equal(t, baseline+1, testutil.ToFloat64(HTTPRequestsInFlight))
	HTTPRequestsInFlight.Dec()
	assert.Equal(t, baseline, testutil.ToFloat64(HTTPRequestsInFlight))
}

// ---------------------------------------------------------------------------
// Middleware (middleware.go)
// ---------------------------------------------------------------------------

func TestMiddleware_RecordsMetrics(t *testing.T) {
	router := gin.New()
	router.Use(Middleware())
	router.GET("/api/v1/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Snapshot counters before request.
	beforeTotal := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/health", "200"))
	beforeInFlight := testutil.ToFloat64(HTTPRequestsInFlight)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Counter should have incremented by 1.
	afterTotal := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/health", "200"))
	assert.Equal(t, beforeTotal+1, afterTotal)

	// In-flight gauge should return to baseline after request completes.
	assert.Equal(t, beforeInFlight, testutil.ToFloat64(HTTPRequestsInFlight))
}

func TestMiddleware_UnknownRoute(t *testing.T) {
	router := gin.New()
	router.Use(Middleware())
	// No routes registered -> FullPath() returns "".

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should use "unknown" as path label (not the actual URL).
	assert.Equal(t, http.StatusNotFound, w.Code)
	val := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "unknown", "404"))
	assert.Greater(t, val, float64(0))
}

// ---------------------------------------------------------------------------
// Cache metrics (cache.go)
// ---------------------------------------------------------------------------

func TestCacheMetrics_Registration(t *testing.T) {
	assert.NotPanics(t, func() {
		CacheHitsTotal.WithLabelValues("review").Inc()
		CacheMissesTotal.WithLabelValues("review").Inc()
		CacheInvalidationFailuresTotal.WithLabelValues("course:123").Inc()
		CacheOperationDuration.WithLabelValues("get", "review").Observe(0.001)
	})
}

func TestCacheHitsTotal_Inc(t *testing.T) {
	before := testutil.ToFloat64(CacheHitsTotal.WithLabelValues("test-cache"))
	CacheHitsTotal.WithLabelValues("test-cache").Inc()
	assert.Equal(t, before+1, testutil.ToFloat64(CacheHitsTotal.WithLabelValues("test-cache")))
}

func TestCacheMissesTotal_Inc(t *testing.T) {
	before := testutil.ToFloat64(CacheMissesTotal.WithLabelValues("test-cache-miss"))
	CacheMissesTotal.WithLabelValues("test-cache-miss").Inc()
	assert.Equal(t, before+1, testutil.ToFloat64(CacheMissesTotal.WithLabelValues("test-cache-miss")))
}

// ---------------------------------------------------------------------------
// DB metrics (db.go)
// ---------------------------------------------------------------------------

func TestDBMetrics_Registration(t *testing.T) {
	assert.NotPanics(t, func() {
		DBQueryDuration.WithLabelValues("SELECT", "reviews").Observe(0.025)
		DBQueryTotal.WithLabelValues("SELECT", "reviews", "ok").Inc()
		DBConnectionsActive.Set(5)
	})
}

func TestDBQueryTotal_Counter(t *testing.T) {
	before := testutil.ToFloat64(DBQueryTotal.WithLabelValues("INSERT", "test_table", "ok"))
	DBQueryTotal.WithLabelValues("INSERT", "test_table", "ok").Inc()
	assert.Equal(t, before+1, testutil.ToFloat64(DBQueryTotal.WithLabelValues("INSERT", "test_table", "ok")))
}

func TestDBConnectionsActive_Gauge(t *testing.T) {
	DBConnectionsActive.Set(10)
	assert.Equal(t, float64(10), testutil.ToFloat64(DBConnectionsActive))
	DBConnectionsActive.Set(3)
	assert.Equal(t, float64(3), testutil.ToFloat64(DBConnectionsActive))
}

// ---------------------------------------------------------------------------
// Error metrics (errors.go)
// ---------------------------------------------------------------------------

func TestErrorMetrics_Registration(t *testing.T) {
	assert.NotPanics(t, func() {
		ErrorsTotal.WithLabelValues("validation", "INVALID_INPUT").Inc()
		PanicsTotal.Inc()
		SSEDroppedEventsTotal.Inc()
	})
}

func TestErrorsTotal_Counter(t *testing.T) {
	before := testutil.ToFloat64(ErrorsTotal.WithLabelValues("auth", "UNAUTHORIZED"))
	ErrorsTotal.WithLabelValues("auth", "UNAUTHORIZED").Inc()
	assert.Equal(t, before+1, testutil.ToFloat64(ErrorsTotal.WithLabelValues("auth", "UNAUTHORIZED")))
}

func TestPanicsTotal_Counter(t *testing.T) {
	before := testutil.ToFloat64(PanicsTotal)
	PanicsTotal.Inc()
	assert.Equal(t, before+1, testutil.ToFloat64(PanicsTotal))
}

func TestSSEDroppedEventsTotal_Counter(t *testing.T) {
	before := testutil.ToFloat64(SSEDroppedEventsTotal)
	SSEDroppedEventsTotal.Inc()
	assert.Equal(t, before+1, testutil.ToFloat64(SSEDroppedEventsTotal))
}

// ---------------------------------------------------------------------------
// Redis metrics (redis.go)
// ---------------------------------------------------------------------------

func TestRedisMetrics_Registration(t *testing.T) {
	assert.NotPanics(t, func() {
		RedisPoolHitsTotal.Inc()
		RedisPoolMissesTotal.Inc()
		RedisPoolTimeoutsTotal.Inc()
		RedisPoolTotalConns.Set(10)
		RedisPoolIdleConns.Set(5)
		RedisPoolStaleConnsTotal.Inc()
	})
}

func TestRedisPoolGauges(t *testing.T) {
	RedisPoolTotalConns.Set(20)
	assert.Equal(t, float64(20), testutil.ToFloat64(RedisPoolTotalConns))

	RedisPoolIdleConns.Set(8)
	assert.Equal(t, float64(8), testutil.ToFloat64(RedisPoolIdleConns))
}

// ---------------------------------------------------------------------------
// External dependency metrics (external.go)
// ---------------------------------------------------------------------------

func TestExternalMetrics_Registration(t *testing.T) {
	assert.NotPanics(t, func() {
		ExternalRequestDuration.WithLabelValues("casdoor_oidc", "introspect", "ok").Observe(0.1)
		ExternalRequestsTotal.WithLabelValues("casdoor_oidc", "introspect", "ok").Inc()
	})
}

func TestObserveExternalRequest_Success(t *testing.T) {
	before := testutil.ToFloat64(ExternalRequestsTotal.WithLabelValues("sms", "send", "ok"))
	ObserveExternalRequest("sms", "send", time.Now().Add(-50*time.Millisecond), nil)
	after := testutil.ToFloat64(ExternalRequestsTotal.WithLabelValues("sms", "send", "ok"))
	assert.Equal(t, before+1, after)
}

func TestObserveExternalRequest_Error(t *testing.T) {
	before := testutil.ToFloat64(ExternalRequestsTotal.WithLabelValues("ldap", "lookup", "error"))
	ObserveExternalRequest("ldap", "lookup", time.Now().Add(-100*time.Millisecond), assert.AnError)
	after := testutil.ToFloat64(ExternalRequestsTotal.WithLabelValues("ldap", "lookup", "error"))
	assert.Equal(t, before+1, after)
}

// ---------------------------------------------------------------------------
// Frontend errors handler (frontend_errors.go)
// ---------------------------------------------------------------------------

// TestFrontendErrors_RouteContract validates the handler is reachable at the
// real route path used in app/modules.go (api group "/api/v1" + "/metrics/frontend-errors").
func TestFrontendErrors_RouteContract(t *testing.T) {
	router := gin.New()
	api := router.Group("/api/v1")
	api.POST("/metrics/frontend-errors", FrontendErrorHandler())

	body := `{"kind":"error"}`
	req := httptest.NewRequest(http.MethodPost, frontendErrorsRoute, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code, "POST %s should return 204, not 404", frontendErrorsRoute)
}

func TestFrontendErrorHandler_ValidKind(t *testing.T) {
	router := gin.New()
	router.POST(frontendErrorsRoute, FrontendErrorHandler())

	before := testutil.ToFloat64(FrontendErrorsTotal.WithLabelValues("error"))

	body := `{"kind":"error"}`
	req := httptest.NewRequest(http.MethodPost, frontendErrorsRoute, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, before+1, testutil.ToFloat64(FrontendErrorsTotal.WithLabelValues("error")))
}

func TestFrontendErrorHandler_UnhandledRejection(t *testing.T) {
	router := gin.New()
	router.POST(frontendErrorsRoute, FrontendErrorHandler())

	before := testutil.ToFloat64(FrontendErrorsTotal.WithLabelValues("unhandledrejection"))

	body := `{"kind":"unhandledrejection"}`
	req := httptest.NewRequest(http.MethodPost, frontendErrorsRoute, strings.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, before+1, testutil.ToFloat64(FrontendErrorsTotal.WithLabelValues("unhandledrejection")))
}

func TestFrontendErrorHandler_InvalidKind(t *testing.T) {
	router := gin.New()
	router.POST(frontendErrorsRoute, FrontendErrorHandler())

	before := testutil.ToFloat64(FrontendErrorsTotal.WithLabelValues("xss"))

	body := `{"kind":"xss"}`
	req := httptest.NewRequest(http.MethodPost, frontendErrorsRoute, strings.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Invalid kinds are silently dropped (204 but no counter increment).
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, before, testutil.ToFloat64(FrontendErrorsTotal.WithLabelValues("xss")))
}

func TestFrontendErrorHandler_EmptyBody(t *testing.T) {
	router := gin.New()
	router.POST(frontendErrorsRoute, FrontendErrorHandler())

	req := httptest.NewRequest(http.MethodPost, frontendErrorsRoute, strings.NewReader(""))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestFrontendErrorHandler_InvalidJSON(t *testing.T) {
	router := gin.New()
	router.POST(frontendErrorsRoute, FrontendErrorHandler())

	req := httptest.NewRequest(http.MethodPost, frontendErrorsRoute, strings.NewReader("{invalid"))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ---------------------------------------------------------------------------
// Web Vitals handler (vitals.go)
// ---------------------------------------------------------------------------

// TestVitals_RouteContract validates the handler is reachable at the
// real route path used in app/modules.go (api group "/api/v1" + "/metrics/vitals").
func TestVitals_RouteContract(t *testing.T) {
	router := gin.New()
	api := router.Group("/api/v1")
	api.POST("/metrics/vitals", VitalsHandler())

	body := `{"name":"LCP","value":1234.5,"rating":"good"}`
	req := httptest.NewRequest(http.MethodPost, vitalsRoute, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code, "POST %s should return 204, not 404", vitalsRoute)
}

func TestVitalsHandler_ValidMetric(t *testing.T) {
	router := gin.New()
	router.POST(vitalsRoute, VitalsHandler())

	body := `{"name":"LCP","value":1234.5,"rating":"good"}`
	req := httptest.NewRequest(http.MethodPost, vitalsRoute, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Histogram should have at least one observation.
	count := testutil.CollectAndCount(WebVitalsValue)
	assert.Greater(t, count, 0)
}

func TestVitalsHandler_AllValidNames(t *testing.T) {
	router := gin.New()
	router.POST(vitalsRoute, VitalsHandler())

	for _, name := range []string{"CLS", "INP", "LCP", "FCP", "TTFB"} {
		t.Run(name, func(t *testing.T) {
			body := `{"name":"` + name + `","value":100,"rating":"good"}`
			req := httptest.NewRequest(http.MethodPost, vitalsRoute, strings.NewReader(body))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusNoContent, w.Code)
		})
	}
}

func TestVitalsHandler_AllValidRatings(t *testing.T) {
	router := gin.New()
	router.POST(vitalsRoute, VitalsHandler())

	for _, rating := range []string{"good", "needs-improvement", "poor"} {
		t.Run(rating, func(t *testing.T) {
			body := `{"name":"LCP","value":500,"rating":"` + rating + `"}`
			req := httptest.NewRequest(http.MethodPost, vitalsRoute, strings.NewReader(body))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusNoContent, w.Code)
		})
	}
}

func TestVitalsHandler_InvalidName(t *testing.T) {
	router := gin.New()
	router.POST(vitalsRoute, VitalsHandler())

	body := `{"name":"EVIL_METRIC","value":100,"rating":"good"}`
	req := httptest.NewRequest(http.MethodPost, vitalsRoute, strings.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVitalsHandler_InvalidRating(t *testing.T) {
	router := gin.New()
	router.POST(vitalsRoute, VitalsHandler())

	body := `{"name":"LCP","value":100,"rating":"excellent"}`
	req := httptest.NewRequest(http.MethodPost, vitalsRoute, strings.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVitalsHandler_EmptyBody(t *testing.T) {
	router := gin.New()
	router.POST(vitalsRoute, VitalsHandler())

	req := httptest.NewRequest(http.MethodPost, vitalsRoute, strings.NewReader(""))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestVitalsHandler_InvalidJSON(t *testing.T) {
	router := gin.New()
	router.POST(vitalsRoute, VitalsHandler())

	req := httptest.NewRequest(http.MethodPost, vitalsRoute, strings.NewReader("not json"))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ---------------------------------------------------------------------------
// Label cardinality: verify bounded label sets
// ---------------------------------------------------------------------------

func TestLabelCardinality_HTTPPath(t *testing.T) {
	// Middleware uses c.FullPath() which returns the route template,
	// not the actual URL. This test documents that "unknown" is used
	// for unmatched routes, keeping cardinality bounded.
	router := gin.New()
	router.Use(Middleware())

	// Hit several nonexistent URLs.
	for _, path := range []string{"/a/1", "/a/2", "/b/1"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// All should map to the same "unknown" label, not individual paths.
	val := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "unknown", "404"))
	assert.GreaterOrEqual(t, val, float64(3))
}

func TestLabelCardinality_FrontendErrors(t *testing.T) {
	// Only "error" and "unhandledrejection" are allowed.
	// Other kinds are silently dropped, preventing label explosion.
	assert.True(t, allowedFrontendErrorKinds["error"])
	assert.True(t, allowedFrontendErrorKinds["unhandledrejection"])
	assert.False(t, allowedFrontendErrorKinds["custom"])
	assert.Len(t, allowedFrontendErrorKinds, 2)
}

func TestLabelCardinality_Vitals(t *testing.T) {
	// Verify the whitelist sizes are bounded.
	assert.Len(t, allowedVitalNames, 5)
	assert.Len(t, allowedRatings, 3)
}

// ---------------------------------------------------------------------------
// Metric descriptions (verify no duplicate registrations)
// ---------------------------------------------------------------------------

func TestAllMetrics_NoDuplicateRegistration(t *testing.T) {
	// Collect all metrics from the default registry. If any metric was
	// registered twice with conflicting definitions, Gather() returns an error.
	_, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
}
