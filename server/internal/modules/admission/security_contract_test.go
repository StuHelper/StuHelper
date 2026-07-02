package admission

import (
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	apigen "git.stuhelper.com/StuHelper/StuHelper/internal/api/gen"
)

// admission surface 在 gin 路由上的前缀。security 契约只覆盖本模块注册的路径，
// 与 F072（mobile-camera-handoffs 漏标 public / school-sso 误标 public）同域。
var admissionContractPrefixes = []string{
	"/api/v1/admission/",
	"/api/v1/bot/admission/",
	"/api/v1/bot/member-blacklist/",
}

var admissionPathParamPattern = regexp.MustCompile(`\{([^}]+)\}`)

// sentinelAuthAbortStatus 是一个绝不会被业务 handler 使用的状态码，
// 用来探测「会话鉴权中间件是否真的挂在该路由上」。
const sentinelAuthAbortStatus = 799

type admissionSecurityClass int

const (
	securityPublic admissionSecurityClass = iota
	securitySession
	securityServiceToken
)

// TestAdmissionSecurityContract 行为式锁定 OpenAPI security 声明与路由实际挂载的鉴权中间件一致：
//   - security: []        → 路由不得挂会话鉴权，匿名请求不应被 799/401/503 拦截；
//   - serviceTokenAuth    → 路由必须挂 requireBotCredential，无凭证请求返回 401；
//   - 其余（继承全局 cookieAuth/bearerAuth）→ 路由必须挂会话鉴权，匿名请求返回 799。
//
// 该测试能捕获 F072 类漂移：spec 与实现对某操作的公开/鉴权判断相反时必然失败。
func TestAdmissionSecurityContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	swagger, err := apigen.GetSwagger()
	require.NoError(t, err)

	router := newAdmissionSecurityContractRouter()

	counts := map[admissionSecurityClass]int{}
	for path, item := range swagger.Paths.Map() {
		ginPath := openAPIPathToGin(path)
		if !hasAdmissionContractPrefix(ginPath) {
			continue
		}
		for method, op := range item.Operations() {
			class := classifyOperationSecurity(swagger, op)
			counts[class]++

			status := probeRouteStatus(t, router, method, ginPath)
			assertSecurityContract(t, method, ginPath, class, status)
		}
	}

	// 防止分类逻辑悄悄退化为「全部归到某一类后无脑通过」：三类都必须有真实样本。
	require.Positivef(t, counts[securityPublic], "no public admission operations classified — F072 fixtures (mobile-camera-handoffs) missing?")
	require.Positivef(t, counts[securitySession], "no session-guarded admission operations classified")
	require.Positivef(t, counts[securityServiceToken], "no serviceToken bot operations classified")
}

func newAdmissionSecurityContractRouter() *gin.Engine {
	router := gin.New()
	// 公开路由会触达 nil-service handler 并 panic；静默兜底转成 500（≠ 799/401/503），
	// 既保证探测不中断，又不把预期内的 panic 噪声打到测试输出。
	router.Use(gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, _ any) {
		c.AbortWithStatus(http.StatusInternalServerError)
	}))

	sentinelAuthMW := func(c *gin.Context) {
		c.AbortWithStatus(sentinelAuthAbortStatus)
	}

	// 注入非 nil verifier：bot 路由在缺少 Authorization 头时走 401 分支（而非 nil → 503）。
	handler := &Handler{botCredentialVerifier: &fakeAdmissionBotCredentialVerifier{}}

	api := router.Group("/api/v1")
	handler.RegisterRoutes(api, sentinelAuthMW)
	handler.RegisterBotRoutes(api)
	return router
}

func classifyOperationSecurity(swagger *openapi3.T, op *openapi3.Operation) admissionSecurityClass {
	requirements := swagger.Security
	if op.Security != nil {
		requirements = *op.Security
	}

	if len(requirements) == 0 {
		return securityPublic
	}
	for _, requirement := range requirements {
		if _, ok := requirement["serviceTokenAuth"]; ok {
			return securityServiceToken
		}
	}
	return securitySession
}

func assertSecurityContract(
	t *testing.T,
	method, ginPath string,
	class admissionSecurityClass,
	status int,
) {
	t.Helper()
	route := method + " " + ginPath

	switch class {
	case securityPublic:
		require.NotEqualf(t, sentinelAuthAbortStatus, status, "%s declares security:[] but a session auth middleware is mounted", route)
		require.NotEqualf(t, http.StatusUnauthorized, status, "%s declares security:[] but bot credential middleware rejected anonymous request", route)
		require.NotEqualf(t, http.StatusServiceUnavailable, status, "%s declares security:[] but bot credential middleware is mounted", route)
	case securitySession:
		require.Equalf(t, sentinelAuthAbortStatus, status, "%s requires a session but no session auth middleware ran for anonymous request", route)
	case securityServiceToken:
		require.Equalf(t, http.StatusUnauthorized, status, "%s requires serviceTokenAuth but anonymous request was not rejected with 401", route)
	}
}

func probeRouteStatus(t *testing.T, router *gin.Engine, method, ginPath string) int {
	t.Helper()
	req := httptest.NewRequest(method, ginPathToRequestURL(ginPath), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp.Code
}

func openAPIPathToGin(path string) string {
	return admissionPathParamPattern.ReplaceAllString(path, ":$1")
}

// ginPathToRequestURL 用占位值填充 gin 路径参数，得到可发起请求的具体 URL。
func ginPathToRequestURL(ginPath string) string {
	segments := strings.Split(ginPath, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, ":") {
			segments[i] = "contract-param"
		}
	}
	return strings.Join(segments, "/")
}

func hasAdmissionContractPrefix(ginPath string) bool {
	for _, prefix := range admissionContractPrefixes {
		if strings.HasPrefix(ginPath, prefix) {
			return true
		}
	}
	return false
}
