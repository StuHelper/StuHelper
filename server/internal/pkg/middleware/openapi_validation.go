package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	legacyrouter "github.com/getkin/kin-openapi/routers/legacy"
	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/api/gen"
	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

type openAPIRequestValidator struct {
	router   routers.Router
	authFunc openapi3filter.AuthenticationFunc
}

// NewOpenAPIRequestValidationMiddleware 创建基于 OpenAPI 的请求校验中间件。
//
// 说明：
//   - 只校验“请求是否符合契约”（路径/方法/参数/请求体）
//   - 认证仍由现有 AuthMiddleware / OptionalAuthMiddleware 负责
//   - 安全声明通过 NoopAuthenticationFunc 跳过具体认证细节，避免与现有认证链重复
func NewOpenAPIRequestValidationMiddleware() (gin.HandlerFunc, error) {
	swagger, err := gen.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("load openapi spec: %w", err)
	}
	return newOpenAPIRequestValidationMiddleware(swagger, openapi3filter.NoopAuthenticationFunc)
}

func newOpenAPIRequestValidationMiddleware(
	swagger *openapi3.T,
	authFunc openapi3filter.AuthenticationFunc,
) (gin.HandlerFunc, error) {
	if swagger == nil {
		return nil, fmt.Errorf("openapi spec is nil")
	}
	// 运行时请求可能来自反向代理、自定义域名或测试环境；这里不按 servers 限制 Host，
	// 仅校验路径/方法/参数/请求体是否符合契约。
	swagger.Servers = nil
	if authFunc == nil {
		authFunc = openapi3filter.NoopAuthenticationFunc
	}
	if err := swagger.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("validate openapi spec: %w", err)
	}
	router, err := legacyrouter.NewRouter(swagger)
	if err != nil {
		return nil, fmt.Errorf("build openapi router: %w", err)
	}

	validator := &openAPIRequestValidator{
		router:   router,
		authFunc: authFunc,
	}

	return func(c *gin.Context) {
		if err := validator.validate(c); err != nil {
			writeOpenAPIValidationError(c, err)
			return
		}
		c.Next()
	}, nil
}

func (v *openAPIRequestValidator) validate(c *gin.Context) error {
	route, pathParams, err := v.router.FindRoute(c.Request)
	if err != nil {
		return err
	}

	input := &openapi3filter.RequestValidationInput{
		Request:    c.Request,
		PathParams: pathParams,
		Route:      route,
		Options: &openapi3filter.Options{
			AuthenticationFunc: v.authFunc,
		},
	}
	return openapi3filter.ValidateRequest(c.Request.Context(), input)
}

func writeOpenAPIValidationError(c *gin.Context, err error) {
	converted := openapi3filter.ConvertErrors(err)
	if vErr, ok := converted.(*openapi3filter.ValidationError); ok {
		switch vErr.Status {
		case http.StatusNotFound:
			response.NotFound(c, "resource not found", errs.ErrNotFound)
			return
		case http.StatusMethodNotAllowed:
			response.Error(c, http.StatusMethodNotAllowed, errs.ErrMethodNotAllowed, "method not allowed")
			return
		case http.StatusUnsupportedMediaType:
			response.Error(c, http.StatusUnsupportedMediaType, errs.ErrUnsupportedMedia, "unsupported media type")
			return
		case http.StatusUnprocessableEntity:
			response.ErrorWithDetails(c, http.StatusUnprocessableEntity, errs.ErrValidation, "request validation failed", buildValidationDetails(vErr))
			return
		default:
			response.ValidationError(c, "request validation failed", buildValidationDetails(vErr))
			return
		}
	}

	response.ValidationError(c, "request validation failed", gin.H{
		"reason": converted.Error(),
	})
}

func buildValidationDetails(vErr *openapi3filter.ValidationError) gin.H {
	details := gin.H{
		"reason": vErr.Title,
	}
	if vErr.Detail != "" {
		details["detail"] = vErr.Detail
	}
	if vErr.Source != nil {
		if vErr.Source.Parameter != "" {
			details["parameter"] = vErr.Source.Parameter
		}
		if vErr.Source.Pointer != "" {
			details["pointer"] = vErr.Source.Pointer
		}
	}
	return details
}
