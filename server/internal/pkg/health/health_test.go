package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupMiniredis 启动一个 miniredis 实例并返回共享 fixture。
func setupMiniredis(t *testing.T) *redisfixture.Fixture {
	t.Helper()
	return redisfixture.Start(t)
}

// responseEnvelope 匹配 response.Response 的 JSON 结构。
type responseEnvelope struct {
	Success bool             `json:"success"`
	Data    *json.RawMessage `json:"data,omitempty"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ---- NewHandler ----

func TestNewHandler_DefaultTimeout(t *testing.T) {
	h := NewHandler(nil, nil, BuildInfo{}, false, 0)
	if h.checkTimeout != defaultCheckTimeout {
		t.Errorf("expected default timeout %v, got %v", defaultCheckTimeout, h.checkTimeout)
	}
}

func TestNewHandler_CustomTimeout(t *testing.T) {
	custom := 5 * time.Second
	h := NewHandler(nil, nil, BuildInfo{}, false, custom)
	if h.checkTimeout != custom {
		t.Errorf("expected custom timeout %v, got %v", custom, h.checkTimeout)
	}
}

func TestNewHandler_NegativeTimeout_FallsBackToDefault(t *testing.T) {
	h := NewHandler(nil, nil, BuildInfo{}, false, -1)
	if h.checkTimeout != defaultCheckTimeout {
		t.Errorf("expected default timeout %v for negative input, got %v", defaultCheckTimeout, h.checkTimeout)
	}
}

func TestNewHandler_BuildInfo(t *testing.T) {
	info := BuildInfo{Version: "v1.0.0", GitCommit: "abc123", BuildTime: "2026-01-01"}
	h := NewHandler(nil, nil, info, true, 0)
	if h.buildInfo != info {
		t.Errorf("buildInfo not stored correctly")
	}
	if !h.isProduction {
		t.Error("expected isProduction = true")
	}
}

// ---- Liveness ----

func TestLiveness_ReturnsOK(t *testing.T) {
	h := NewHandler(nil, nil, BuildInfo{}, false, 0)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/health/live", h.Liveness)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var env responseEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !env.Success {
		t.Errorf("expected success = true")
	}
	if env.Error != nil {
		t.Errorf("expected no error field in liveness response")
	}
}

func TestLiveness_ResponseContainsStatusAndTimestamp(t *testing.T) {
	h := NewHandler(nil, nil, BuildInfo{}, false, 0)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/health/live", h.Liveness)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	r.ServeHTTP(w, req)

	var env responseEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if env.Data == nil {
		t.Fatal("expected data field in liveness response")
	}

	var data map[string]string
	if err := json.Unmarshal(*env.Data, &data); err != nil {
		t.Fatalf("failed to parse data: %v", err)
	}

	if data["status"] != "ok" {
		t.Errorf("data.status = %q, want %q", data["status"], "ok")
	}
	if data["timestamp"] == "" {
		t.Error("expected non-empty timestamp")
	}
	// 验证 timestamp 符合 RFC3339 格式
	if _, err := time.Parse(time.RFC3339, data["timestamp"]); err != nil {
		t.Errorf("timestamp %q is not valid RFC3339: %v", data["timestamp"], err)
	}
}

// ---- Readiness: failing dependencies ----

func TestReadiness_NilPgPool_Returns503(t *testing.T) {
	// nil pgPool 将导致 checkPostgres goroutine panic，被 recover 捕获后标记为 unhealthy
	redisFixture := setupMiniredis(t)
	h := NewHandler(nil, redisFixture.Client, BuildInfo{}, false, time.Second)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/health/ready", h.Readiness)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var env responseEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if env.Success {
		t.Error("expected success = false for unhealthy readiness")
	}
	if env.Error == nil {
		t.Fatal("expected error field for unhealthy readiness")
	}
	if env.Error.Code != string(errs.ErrServiceUnavailable) {
		t.Errorf("error.code = %q, want %q (ErrServiceUnavailable)", env.Error.Code, errs.ErrServiceUnavailable)
	}
}

func TestReadiness_RedisDown_Returns503(t *testing.T) {
	// 使用 SetError 让 miniredis 对所有命令返回错误
	redisFixture := setupMiniredis(t)
	redisFixture.Server.SetError("forced-failure")

	// pgPool 为 nil 也会导致 postgres unhealthy，但重点是 redis 也失败
	h := NewHandler(nil, redisFixture.Client, BuildInfo{}, false, time.Second)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/health/ready", h.Readiness)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestReadiness_BothDepsDown_Returns503(t *testing.T) {
	// Redis 指向一个不可达地址
	deadClient := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1", // 几乎一定被拒绝
		DialTimeout: 100 * time.Millisecond,
	})
	t.Cleanup(func() { _ = deadClient.Close() })

	h := NewHandler(nil, deadClient, BuildInfo{}, false, time.Second)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/health/ready", h.Readiness)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestCheckPostgres_NilPool_ReturnsExplicitUnhealthy(t *testing.T) {
	h := NewHandler(nil, nil, BuildInfo{}, false, time.Second)

	result := h.checkPostgres(t.Context())

	if result.Status != "unhealthy" {
		t.Fatalf("status = %q, want %q", result.Status, "unhealthy")
	}
	if result.Error != "not configured" {
		t.Fatalf("error = %q, want %q", result.Error, "not configured")
	}
}

func TestCheckRedis_NilClient_ReturnsExplicitUnhealthy(t *testing.T) {
	h := NewHandler(nil, nil, BuildInfo{}, false, time.Second)

	result := h.checkRedis(t.Context())

	if result.Status != "unhealthy" {
		t.Fatalf("status = %q, want %q", result.Status, "unhealthy")
	}
	if result.Error != "not configured" {
		t.Fatalf("error = %q, want %q", result.Error, "not configured")
	}
}

// ---- Readiness: response format ----

func TestReadiness_ErrorResponse_JSONStructure(t *testing.T) {
	redisFixture := setupMiniredis(t)
	h := NewHandler(nil, redisFixture.Client, BuildInfo{}, false, time.Second)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/health/ready", h.Readiness)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	r.ServeHTTP(w, req)

	// 验证 Content-Type
	ct := w.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json; charset=utf-8")
	}

	// 验证可以反序列化为统一 response 结构
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	// 必须包含 success 和 error 字段（失败路径）
	if _, ok := raw["success"]; !ok {
		t.Error("missing 'success' field in response")
	}
	if _, ok := raw["error"]; !ok {
		t.Error("missing 'error' field in unhealthy response")
	}
}

// ---- RegisterRoutes ----

func TestRegisterRoutes_BindsPaths(t *testing.T) {
	h := NewHandler(nil, nil, BuildInfo{}, false, 0)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	h.RegisterRoutes(r)

	// /health/live 应该被注册
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("/health/live status = %d, want %d", w.Code, http.StatusOK)
	}

	// /health/ready 也应该被注册（返回 503 因为依赖不健康，但不应该是 404）
	req = httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusNotFound {
		t.Error("/health/ready returned 404, route not registered")
	}
}

// ---- CheckResult struct ----

func TestCheckResult_JSONSerialization(t *testing.T) {
	cr := CheckResult{
		Status:  "healthy",
		Latency: "1.23ms",
	}
	data, err := json.Marshal(cr)
	if err != nil {
		t.Fatalf("failed to marshal CheckResult: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal CheckResult: %v", err)
	}
	if parsed["status"] != "healthy" {
		t.Errorf("status = %q, want %q", parsed["status"], "healthy")
	}
	if parsed["latency"] != "1.23ms" {
		t.Errorf("latency = %q, want %q", parsed["latency"], "1.23ms")
	}
	// omitempty: error 和 details 应该不出现
	if _, ok := parsed["error"]; ok {
		t.Error("expected 'error' field to be omitted when empty")
	}
	if _, ok := parsed["details"]; ok {
		t.Error("expected 'details' field to be omitted when nil")
	}
}

func TestCheckResult_JSONWithError(t *testing.T) {
	cr := CheckResult{
		Status:  "unhealthy",
		Latency: "500ms",
		Error:   "connection refused",
	}
	data, err := json.Marshal(cr)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if parsed["error"] != "connection refused" {
		t.Errorf("error = %q, want %q", parsed["error"], "connection refused")
	}
}

// ---- Production vs development response ----

func TestLiveness_ProductionMode_StillReturns200(t *testing.T) {
	h := NewHandler(nil, nil, BuildInfo{Version: "v1.0"}, true, 0)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/health/live", h.Liveness)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ---- getSystemInfo ----

func TestGetSystemInfo_ContainsExpectedFields(t *testing.T) {
	info := BuildInfo{Version: "v2.0.0", GitCommit: "deadbeef", BuildTime: "2026-04-05T00:00:00Z"}
	h := NewHandler(nil, nil, info, false, 0)

	sysInfo := h.getSystemInfo()

	requiredKeys := []string{"version", "git_commit", "build_time", "uptime", "go_version", "goroutines"}
	for _, key := range requiredKeys {
		if _, ok := sysInfo[key]; !ok {
			t.Errorf("getSystemInfo() missing key %q", key)
		}
	}

	if sysInfo["version"] != "v2.0.0" {
		t.Errorf("version = %q, want %q", sysInfo["version"], "v2.0.0")
	}
	if sysInfo["git_commit"] != "deadbeef" {
		t.Errorf("git_commit = %q, want %q", sysInfo["git_commit"], "deadbeef")
	}
}
