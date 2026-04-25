package routeassert

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Exists 断言路由已注册。
func Exists(t *testing.T, routes gin.RoutesInfo, method, path string) {
	t.Helper()
	for _, route := range routes {
		if route.Method == method && route.Path == path {
			return
		}
	}
	assert.Failf(t, "missing route", "expected route %s %s to be registered", method, path)
}

// NotExists 断言路由未注册。
func NotExists(t *testing.T, routes gin.RoutesInfo, method, path string) {
	t.Helper()
	for _, route := range routes {
		if route.Method == method && route.Path == path {
			assert.Failf(t, "unexpected route", "did not expect route %s %s to be registered", method, path)
			return
		}
	}
}
