// Package api 提供 OpenAPI 规范文件和 Swagger UI 文档服务。
package api

import (
	_ "embed"
	"net/http"

	"github.com/flowchartsman/swaggerui"
	"github.com/gin-gonic/gin"
)

//go:embed openapi.bundled.yaml
var specData []byte

// RegisterDocs 注册 Swagger UI 文档路由。
// 仅在非生产环境启用。
func RegisterDocs(r *gin.Engine, isProduction bool) {
	if isProduction {
		return
	}
	r.GET("/docs/*any", gin.WrapH(
		http.StripPrefix("/docs", swaggerui.Handler(specData)),
	))
}
